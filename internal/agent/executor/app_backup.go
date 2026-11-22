package executor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const (
	appBackupManifestName = "dockeeper-manifest.json"
	appBackupFilesPrefix  = "files/"
	defaultAppBackupMax   = int64(20 << 30)
	maxAppBackupPaths     = 20
	appBackupTimeout      = 30 * time.Minute
)

var (
	ErrUnsafePath         = errors.New("unsafe backup path")
	ErrNoBackupPaths      = errors.New("backup paths are required")
	ErrTooManyBackupPaths = errors.New("too many backup paths")
	ErrBackupTooLarge     = errors.New("backup exceeds max size")
	ErrInvalidManifest    = errors.New("invalid backup manifest")
)

var deniedBackupPaths = []string{
	"/proc", "/sys", "/dev", "/boot", "/etc/shadow", "/etc/passwd", "/etc/sudoers",
}

type Manifest struct {
	App       string   `json:"app"`
	Source    string   `json:"source"`
	Server    string   `json:"server"`
	Paths     []string `json:"paths"`
	Excludes  []string `json:"exclude,omitempty"`
	CreatedAt string   `json:"created_at"`
	Version   string   `json:"version"`
	Checksum  string   `json:"checksum"`
}

type AppBackupExecutor struct {
	logger      *slog.Logger
	hostActions *HostActionExecutor
	tempDir     string
	version     string
}

func NewAppBackupExecutor(logger *slog.Logger, hostActions *HostActionExecutor) *AppBackupExecutor {
	return &AppBackupExecutor{
		logger:      logger,
		hostActions: hostActions,
		tempDir:     os.TempDir(),
		version:     "dev",
	}
}

func (e *AppBackupExecutor) Backup(ctx context.Context, req *agentpb.BackupAppRequest, progress func(BackupProgressInfo)) error {
	paths := req.GetPaths()
	if err := validateBackupPaths(paths); err != nil {
		return err
	}

	maxBytes := int64(req.GetMaxBytes())
	if maxBytes <= 0 {
		maxBytes = defaultAppBackupMax
	}

	manifest := Manifest{
		App:       req.GetAppName(),
		Source:    req.GetAppSource(),
		Server:    req.GetServerId(),
		Paths:     paths,
		Excludes:  req.GetExclude(),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Version:   e.version,
	}

	progress(BackupProgressInfo{Status: "tarring", Message: "Creating archive from app paths"})

	archive, signed, err := e.buildArchive(manifest, maxBytes)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(archive) }()

	stat, err := os.Stat(archive)
	if err != nil {
		return fmt.Errorf("stat archive: %w", err)
	}

	progress(BackupProgressInfo{
		Status:     "uploading",
		Message:    fmt.Sprintf("Uploading archive (%d bytes)", stat.Size()),
		TotalBytes: stat.Size(),
	})

	client, err := newS3Client(protoS3Config(req.GetS3()))
	if err != nil {
		return fmt.Errorf("create S3 client: %w", err)
	}

	file, err := os.Open(archive)
	if err != nil {
		return fmt.Errorf("open archive for upload: %w", err)
	}
	defer func() { _ = file.Close() }()

	_, err = client.PutObject(ctx, req.GetS3().GetBucket(), req.GetS3().GetKey(), file, stat.Size(), minio.PutObjectOptions{
		ContentType: "application/gzip",
		UserMetadata: map[string]string{
			"manifest-checksum": signed.Checksum,
		},
	})
	if err != nil {
		return fmt.Errorf("upload to S3: %w", err)
	}

	progress(BackupProgressInfo{
		Status:         "done",
		Message:        "Backup complete",
		BytesProcessed: stat.Size(),
		TotalBytes:     stat.Size(),
	})

	return nil
}

func (e *AppBackupExecutor) Restore(ctx context.Context, req *agentpb.RestoreAppRequest, progress func(BackupProgressInfo)) error {
	allowed := req.GetPaths()
	if err := validateBackupPaths(allowed); err != nil {
		return err
	}

	progress(BackupProgressInfo{Status: "downloading", Message: "Downloading archive from S3"})

	client, err := newS3Client(protoS3Config(req.GetS3()))
	if err != nil {
		return fmt.Errorf("create S3 client: %w", err)
	}

	obj, err := client.GetObject(ctx, req.GetS3().GetBucket(), req.GetS3().GetKey(), minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("download from S3: %w", err)
	}
	defer func() { _ = obj.Close() }()

	info, err := obj.Stat()
	if err != nil {
		return fmt.Errorf("stat S3 object: %w", err)
	}

	archivePath := filepath.Join(e.tempDir, fmt.Sprintf("dockeeper-restore-%d.tar.gz", time.Now().UnixNano()))
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create restore file: %w", err)
	}
	defer func() { _ = os.Remove(archivePath) }()

	if _, err := io.Copy(archiveFile, obj); err != nil {
		_ = archiveFile.Close()
		return fmt.Errorf("download archive: %w", err)
	}
	if err := archiveFile.Close(); err != nil {
		return fmt.Errorf("close restore file: %w", err)
	}

	manifest, err := readManifest(archivePath)
	if err != nil {
		return err
	}
	if err := validateManifest(&manifest, allowed); err != nil {
		return err
	}

	progress(BackupProgressInfo{Status: "extracting", Message: "Extracting archive", TotalBytes: info.Size})

	safety, err := e.createSafetyArchive(manifest.Paths)
	if err != nil {
		return fmt.Errorf("create safety archive: %w", err)
	}
	if safety != "" {
		defer func() { _ = os.Remove(safety) }()
	}

	started := false
	if e.hostActions != nil {
		if _, _, err := e.hostActions.Execute(ctx, req.GetAppSource(), req.GetAppName(), "stop"); err != nil {
			return fmt.Errorf("stop app before restore: %w", err)
		}
		started = true
	}

	restoreErr := restoreArchive(archivePath, manifest.Paths)
	if restoreErr != nil && safety != "" {
		_ = restoreArchive(safety, manifest.Paths)
	}
	if started {
		if _, _, startErr := e.hostActions.Execute(ctx, req.GetAppSource(), req.GetAppName(), "start"); startErr != nil && restoreErr == nil {
			restoreErr = fmt.Errorf("start app after restore: %w", startErr)
		}
	}
	if restoreErr != nil {
		return restoreErr
	}

	progress(BackupProgressInfo{
		Status:         "done",
		Message:        "Restore complete",
		BytesProcessed: info.Size,
		TotalBytes:     info.Size,
	})

	return nil
}

func (e *AppBackupExecutor) buildArchive(manifest Manifest, maxBytes int64) (string, Manifest, error) {
	if err := validateBackupPaths(manifest.Paths); err != nil {
		return "", Manifest{}, err
	}

	tmp, err := os.CreateTemp(e.tempDir, "dockeeper-app-backup-*.tar.gz")
	if err != nil {
		return "", Manifest{}, fmt.Errorf("create temp archive: %w", err)
	}
	tmpPath := tmp.Name()

	gz := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gz)

	cleanup := func(err error) (string, Manifest, error) {
		_ = tw.Close()
		_ = gz.Close()
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", Manifest{}, err
	}

	signed := manifest
	signed.Checksum = manifestChecksum(signed)
	encoded, err := json.Marshal(signed)
	if err != nil {
		return cleanup(fmt.Errorf("encode manifest: %w", err))
	}
	if err := writeTarBytes(tw, appBackupManifestName, encoded); err != nil {
		return cleanup(err)
	}

	var written int64
	for rootIndex, root := range manifest.Paths {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if excludedPath(path, root, manifest.Excludes) {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSocket != 0 || info.Mode()&os.ModeDevice != 0 {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				resolved, err := filepath.EvalSymlinks(path)
				if err != nil {
					return nil
				}
				if !isSubpath(root, resolved) {
					return fmt.Errorf("%w: symlink %s escapes %s", ErrUnsafePath, path, root)
				}
			}
			if written+info.Size() > maxBytes {
				return ErrBackupTooLarge
			}

			link := ""
			if entry.Type()&os.ModeSymlink != 0 {
				link, _ = os.Readlink(path)
			}
			header, err := tar.FileInfoHeader(info, link)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			header.Name = fmt.Sprintf("%s%d", appBackupFilesPrefix, rootIndex)
			if rel != "." {
				header.Name += "/" + filepath.ToSlash(rel)
			}
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() { _ = file.Close() }()
			n, err := io.Copy(tw, file)
			written += n
			return err
		})
		if err != nil {
			return cleanup(err)
		}
	}

	if err := tw.Close(); err != nil {
		return cleanup(err)
	}
	if err := gz.Close(); err != nil {
		return cleanup(err)
	}
	if err := tmp.Close(); err != nil {
		return cleanup(err)
	}

	return tmpPath, signed, nil
}

func (e *AppBackupExecutor) createSafetyArchive(paths []string) (string, error) {
	existing := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			existing = append(existing, path)
		}
	}
	if len(existing) == 0 {
		return "", nil
	}
	manifest := Manifest{
		Paths:     existing,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Version:   e.version,
	}
	archive, _, err := e.buildArchive(manifest, defaultAppBackupMax)
	if err != nil {
		return "", err
	}
	return archive, nil
}

func readManifest(archivePath string) (Manifest, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return Manifest{}, fmt.Errorf("open archive: %w", err)
	}
	defer func() { _ = file.Close() }()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return Manifest{}, fmt.Errorf("open gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Manifest{}, fmt.Errorf("read archive: %w", err)
		}
		if header.Name != appBackupManifestName {
			continue
		}
		body, err := io.ReadAll(io.LimitReader(tr, 1<<20))
		if err != nil {
			return Manifest{}, fmt.Errorf("read manifest: %w", err)
		}
		var manifest Manifest
		if err := json.Unmarshal(body, &manifest); err != nil {
			return Manifest{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
		if manifest.Checksum != manifestChecksum(manifest) {
			return Manifest{}, fmt.Errorf("%w: checksum mismatch", ErrInvalidManifest)
		}
		return manifest, nil
	}
	return Manifest{}, fmt.Errorf("%w: manifest missing", ErrInvalidManifest)
}

func restoreArchive(archivePath string, paths []string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer func() { _ = file.Close() }()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		if header.Name == appBackupManifestName || !strings.HasPrefix(header.Name, appBackupFilesPrefix) {
			continue
		}

		parts := strings.SplitN(strings.TrimPrefix(header.Name, appBackupFilesPrefix), "/", 2)
		rootIndex, err := strconv.Atoi(parts[0])
		if err != nil || rootIndex < 0 || rootIndex >= len(paths) {
			return fmt.Errorf("%w: %s", ErrUnsafePath, header.Name)
		}
		target := paths[rootIndex]
		if len(parts) == 2 && parts[1] != "" {
			target = filepath.Join(paths[rootIndex], filepath.FromSlash(parts[1]))
		}
		if !withinAnyPath(paths, target) {
			return fmt.Errorf("%w: %s", ErrUnsafePath, target)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
		case tar.TypeSymlink:
			if !withinAnyPath(paths, filepath.Dir(target)) {
				return fmt.Errorf("%w: %s", ErrUnsafePath, target)
			}
			_ = os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("symlink %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("mkdir parent %s: %w", target, err)
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create %s: %w", target, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return fmt.Errorf("write %s: %w", target, err)
			}
			if err := out.Close(); err != nil {
				return fmt.Errorf("close %s: %w", target, err)
			}
		}
	}
	return nil
}

func validateBackupPaths(paths []string) error {
	if len(paths) == 0 {
		return ErrNoBackupPaths
	}
	if len(paths) > maxAppBackupPaths {
		return ErrTooManyBackupPaths
	}
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			return fmt.Errorf("%w: %s", ErrUnsafePath, path)
		}
		clean := filepath.Clean(path)
		if clean != path || clean == "/" {
			return fmt.Errorf("%w: %s", ErrUnsafePath, path)
		}
		resolved := clean
		if r, err := filepath.EvalSymlinks(clean); err == nil {
			resolved = r
		}
		for _, denied := range deniedBackupPaths {
			if resolved == denied || strings.HasPrefix(resolved, denied+"/") {
				return fmt.Errorf("%w: %s", ErrUnsafePath, path)
			}
		}
	}
	return nil
}

func validateManifest(manifest *Manifest, allowed []string) error {
	if len(manifest.Paths) == 0 {
		return fmt.Errorf("%w: no paths", ErrInvalidManifest)
	}
	if err := validateBackupPaths(manifest.Paths); err != nil {
		return err
	}
	for _, path := range manifest.Paths {
		found := false
		for _, allowedPath := range allowed {
			if path == allowedPath {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: path %s is not allowed", ErrUnsafePath, path)
		}
	}
	return nil
}

func manifestChecksum(manifest Manifest) string {
	manifest.Checksum = ""
	encoded, _ := json.Marshal(manifest)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func writeTarBytes(tw *tar.Writer, name string, body []byte) error {
	header := &tar.Header{
		Name:    name,
		Mode:    0o600,
		Size:    int64(len(body)),
		ModTime: time.Now().UTC(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write manifest header: %w", err)
	}
	if _, err := tw.Write(body); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func excludedPath(path, root string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, rel); matched {
			return true
		}
	}
	return false
}

func withinAnyPath(paths []string, target string) bool {
	for _, path := range paths {
		if isSubpath(path, target) || target == path {
			return true
		}
	}
	return false
}

func protoS3Config(pb *agentpb.S3Config) S3Config {
	if pb == nil {
		return S3Config{}
	}
	return S3Config{
		Endpoint:  pb.GetEndpoint(),
		Bucket:    pb.GetBucket(),
		Key:       pb.GetKey(),
		AccessKey: pb.GetAccessKey(),
		SecretKey: pb.GetSecretKey(),
		Region:    pb.GetRegion(),
		UseSSL:    pb.GetUseSsl(),
	}
}
