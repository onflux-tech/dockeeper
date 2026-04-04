package executor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// BackupVolume creates a tar.gz of the volume mountpoint and uploads it to S3.
func (e *DockerExecutor) BackupVolume(ctx context.Context, mountpoint string, s3 S3Config, progress func(BackupProgressInfo)) error {
	progress(BackupProgressInfo{Status: "tarring", Message: "Creating archive from volume"})

	// Create a temp file for the tar.gz archive.
	tmpFile, err := os.CreateTemp("", "dockeeper-backup-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	// Create tar.gz archive.
	gzWriter := gzip.NewWriter(tmpFile)
	tarWriter := tar.NewWriter(gzWriter)

	err = filepath.Walk(mountpoint, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		header, hErr := tar.FileInfoHeader(info, "")
		if hErr != nil {
			return hErr
		}

		rel, relErr := filepath.Rel(mountpoint, path)
		if relErr != nil {
			return relErr
		}
		header.Name = filepath.ToSlash(rel)

		if wErr := tarWriter.WriteHeader(header); wErr != nil {
			return wErr
		}

		if info.IsDir() {
			return nil
		}

		f, fErr := os.Open(path)
		if fErr != nil {
			return fErr
		}
		defer func() { _ = f.Close() }()

		_, copyErr := io.Copy(tarWriter, f)
		return copyErr
	})
	if err != nil {
		_ = tarWriter.Close()
		_ = gzWriter.Close()
		_ = tmpFile.Close()
		return fmt.Errorf("tar volume %s: %w", mountpoint, err)
	}

	if err = tarWriter.Close(); err != nil {
		_ = gzWriter.Close()
		_ = tmpFile.Close()
		return fmt.Errorf("close tar writer: %w", err)
	}
	if err = gzWriter.Close(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("close gzip writer: %w", err)
	}
	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Get file size for progress tracking.
	stat, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("stat temp file: %w", err)
	}
	totalBytes := stat.Size()

	progress(BackupProgressInfo{
		Status:     "uploading",
		Message:    fmt.Sprintf("Uploading archive (%d bytes)", totalBytes),
		TotalBytes: totalBytes,
	})

	// Upload to S3.
	client, err := newS3Client(s3)
	if err != nil {
		return fmt.Errorf("create S3 client: %w", err)
	}

	uploadFile, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("open temp file for upload: %w", err)
	}
	defer func() { _ = uploadFile.Close() }()

	_, err = client.PutObject(ctx, s3.Bucket, s3.Key, uploadFile, totalBytes, minio.PutObjectOptions{
		ContentType: "application/gzip",
	})
	if err != nil {
		return fmt.Errorf("upload to S3: %w", err)
	}

	progress(BackupProgressInfo{
		Status:         "done",
		Message:        "Backup complete",
		BytesProcessed: totalBytes,
		TotalBytes:     totalBytes,
	})

	return nil
}

// RestoreVolume downloads a tar.gz from S3 and extracts it to the volume mountpoint.
func (e *DockerExecutor) RestoreVolume(ctx context.Context, mountpoint string, s3 S3Config, progress func(BackupProgressInfo)) error {
	progress(BackupProgressInfo{Status: "downloading", Message: "Downloading archive from S3"})

	client, err := newS3Client(s3)
	if err != nil {
		return fmt.Errorf("create S3 client: %w", err)
	}

	obj, err := client.GetObject(ctx, s3.Bucket, s3.Key, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("download from S3: %w", err)
	}
	defer func() { _ = obj.Close() }()

	// Get object size for progress.
	info, err := obj.Stat()
	if err != nil {
		return fmt.Errorf("stat S3 object: %w", err)
	}
	totalBytes := info.Size

	progress(BackupProgressInfo{
		Status:     "extracting",
		Message:    fmt.Sprintf("Extracting archive (%d bytes) to volume", totalBytes),
		TotalBytes: totalBytes,
	})

	gzReader, err := gzip.NewReader(obj)
	if err != nil {
		return fmt.Errorf("open gzip reader: %w", err)
	}
	defer func() { _ = gzReader.Close() }()

	tarReader := tar.NewReader(gzReader)

	for {
		header, tErr := tarReader.Next()
		if tErr == io.EOF {
			break
		}
		if tErr != nil {
			return fmt.Errorf("read tar entry: %w", tErr)
		}

		target := filepath.Join(mountpoint, filepath.FromSlash(header.Name))

		// Ensure the target is within the mountpoint (path traversal protection).
		if !isSubpath(mountpoint, target) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if mkErr := os.MkdirAll(target, os.FileMode(header.Mode)); mkErr != nil {
				return fmt.Errorf("mkdir %s: %w", target, mkErr)
			}
		case tar.TypeReg:
			if mkErr := os.MkdirAll(filepath.Dir(target), 0o755); mkErr != nil {
				return fmt.Errorf("mkdir parent %s: %w", target, mkErr)
			}
			f, fErr := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if fErr != nil {
				return fmt.Errorf("create file %s: %w", target, fErr)
			}
			if _, copyErr := io.Copy(f, tarReader); copyErr != nil {
				_ = f.Close()
				return fmt.Errorf("extract file %s: %w", target, copyErr)
			}
			_ = f.Close()
		}
	}

	progress(BackupProgressInfo{
		Status:         "done",
		Message:        "Restore complete",
		BytesProcessed: totalBytes,
		TotalBytes:     totalBytes,
	})

	return nil
}

func newS3Client(s3 S3Config) (*minio.Client, error) {
	return minio.New(s3.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3.AccessKey, s3.SecretKey, ""),
		Secure: s3.UseSSL,
		Region: s3.Region,
	})
}

// isSubpath checks that target is under base (prevents zip-slip / path traversal).
func isSubpath(base, target string) bool {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..")
}
