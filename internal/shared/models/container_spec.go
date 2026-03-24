package models

import (
	"fmt"
	"regexp"
	"strings"
)

// validNameRe allows alphanumeric, dashes, underscores and dots (Docker convention).
var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// validRestartPolicies lists allowed Docker restart policies.
var validRestartPolicies = map[string]bool{
	"":               true,
	"no":             true,
	"always":         true,
	"unless-stopped": true,
	"on-failure":     true,
}

// PortMapping defines a host-to-container port binding.
type PortMapping struct {
	HostPort      string `json:"host_port"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol"` // tcp, udp — defaults to tcp
}

// VolumeMount defines a volume or bind mount for the container.
type VolumeMount struct {
	Source   string `json:"source"`    // host path or volume name
	Target   string `json:"target"`    // container path
	ReadOnly bool   `json:"read_only"`
}

// ContainerSpec holds the full specification for creating a new container.
type ContainerSpec struct {
	ServerID      string            `json:"server_id"`
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	Ports         []PortMapping     `json:"ports"`
	Env           map[string]string `json:"env"`
	Volumes       []VolumeMount     `json:"volumes"`
	Labels        map[string]string `json:"labels"`
	Network       string            `json:"network"`
	CPULimit      float64           `json:"cpu_limit"`       // in CPU cores (e.g. 0.5 = half a core)
	MemLimit      int64             `json:"mem_limit"`        // in bytes
	PidsLimit     int64             `json:"pids_limit"`
	RestartPolicy string            `json:"restart_policy"`
	Command       []string          `json:"command"`
	Entrypoint    []string          `json:"entrypoint"`
	AutoStart     bool              `json:"auto_start"`
}

// Validate checks the container spec for required fields and valid values.
// Returns a list of human-readable validation errors.
func (s *ContainerSpec) Validate() []string {
	var errs []string

	if s.ServerID == "" {
		errs = append(errs, "server_id is required")
	}

	if s.Name == "" {
		errs = append(errs, "name is required")
	} else if !validNameRe.MatchString(s.Name) {
		errs = append(errs, "name must start with alphanumeric and contain only [a-zA-Z0-9._-]")
	} else if len(s.Name) > 128 {
		errs = append(errs, "name must be 128 characters or less")
	}

	if s.Image == "" {
		errs = append(errs, "image is required")
	}

	if !validRestartPolicies[s.RestartPolicy] {
		errs = append(errs, fmt.Sprintf("invalid restart_policy: %q (must be one of: no, always, unless-stopped, on-failure)", s.RestartPolicy))
	}

	if s.CPULimit < 0 {
		errs = append(errs, "cpu_limit must be >= 0")
	}
	if s.MemLimit < 0 {
		errs = append(errs, "mem_limit must be >= 0")
	}
	if s.PidsLimit < 0 {
		errs = append(errs, "pids_limit must be >= 0")
	}

	for i, p := range s.Ports {
		if p.ContainerPort == "" {
			errs = append(errs, fmt.Sprintf("ports[%d].container_port is required", i))
		}
		proto := strings.ToLower(p.Protocol)
		if proto == "" {
			proto = "tcp"
		}
		if proto != "tcp" && proto != "udp" {
			errs = append(errs, fmt.Sprintf("ports[%d].protocol must be tcp or udp", i))
		}
	}

	for i, v := range s.Volumes {
		if v.Target == "" {
			errs = append(errs, fmt.Sprintf("volumes[%d].target is required", i))
		}
	}

	return errs
}
