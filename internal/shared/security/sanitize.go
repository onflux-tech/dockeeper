package security

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// namePattern matches safe Docker resource names: starts with alphanumeric, then alphanumeric, _, ., -
	namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

	// imageRefPattern matches Docker image references like: registry.io/name:tag or name@sha256:hex
	imageRefPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_./:@-]*$`)

	// sha256DigestPattern matches the @sha256:hex suffix
	sha256DigestPattern = regexp.MustCompile(`@sha256:[a-f0-9]{64}$`)
)

const (
	maxNameLength     = 128
	maxImageRefLength = 256
)

// ValidateContainerName validates a container name for safe characters.
func ValidateContainerName(name string) error {
	return validateDockerName(name, "container name")
}

// ValidateVolumeName validates a volume name for safe characters.
func ValidateVolumeName(name string) error {
	return validateDockerName(name, "volume name")
}

// ValidateNetworkName validates a network name for safe characters.
func ValidateNetworkName(name string) error {
	return validateDockerName(name, "network name")
}

func validateDockerName(name, label string) error {
	if name == "" {
		return fmt.Errorf("invalid %s: cannot be empty", label)
	}
	if len(name) > maxNameLength {
		return fmt.Errorf("invalid %s: exceeds maximum length of %d characters", label, maxNameLength)
	}
	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid %s: contains disallowed characters", label)
	}
	return nil
}

// ValidateDockerImageRef validates a Docker image reference.
func ValidateDockerImageRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("invalid image reference: cannot be empty")
	}
	if len(ref) > maxImageRefLength {
		return fmt.Errorf("invalid image reference: exceeds maximum length of %d characters", maxImageRefLength)
	}
	if !imageRefPattern.MatchString(ref) {
		return fmt.Errorf("invalid image reference: contains disallowed characters")
	}
	// If it has a digest, validate it's a proper sha256.
	if strings.Contains(ref, "@") && !sha256DigestPattern.MatchString(ref) {
		return fmt.Errorf("invalid image reference: malformed digest")
	}
	return nil
}

// SanitizeFilterParam escapes special PocketBase filter characters to prevent filter injection.
// This is a safety net — parameterized queries ({:param}) should be preferred.
func SanitizeFilterParam(param string) string {
	// These characters have special meaning in PocketBase filter expressions.
	replacer := strings.NewReplacer(
		"'", "''",
		"\\", "\\\\",
	)
	return replacer.Replace(param)
}
