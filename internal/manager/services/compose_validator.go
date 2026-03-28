package services

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeValidationError represents a single validation issue in a compose YAML.
type ComposeValidationError struct {
	Line    int    `json:"line"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// allowedTopLevelKeys lists the valid top-level compose keys.
var allowedTopLevelKeys = map[string]bool{
	"version":  true,
	"services": true,
	"networks": true,
	"volumes":  true,
	"configs":  true,
	"secrets":  true,
	"name":     true,
}

// portRegex matches valid port mappings: host:container or container, with optional /proto.
var portRegex = regexp.MustCompile(`^\d+(?:-\d+)?(?::\d+(?:-\d+)?)?(?:/(tcp|udp))?$`)

// ValidateComposeYAML validates a docker-compose YAML and returns all errors found.
func ValidateComposeYAML(content []byte) []ComposeValidationError {
	if len(content) == 0 {
		return []ComposeValidationError{{Line: 1, Message: "empty compose file"}}
	}

	// Layer 1: YAML syntax validation.
	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return []ComposeValidationError{parseYAMLError(err)}
	}

	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return []ComposeValidationError{{Line: 1, Message: "invalid YAML document"}}
	}

	mapping := root.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return []ComposeValidationError{{Line: mapping.Line, Message: "compose file must be a YAML mapping"}}
	}

	var errors []ComposeValidationError

	// Layer 2: Structure validation — check top-level keys.
	hasServices := false
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		keyNode := mapping.Content[i]
		key := keyNode.Value

		if key == "services" {
			hasServices = true
		}

		// Allow x-* extension keys.
		if strings.HasPrefix(key, "x-") {
			continue
		}

		if !allowedTopLevelKeys[key] {
			errors = append(errors, ComposeValidationError{
				Line:    keyNode.Line,
				Message: fmt.Sprintf("unknown top-level key %q", key),
				Field:   key,
			})
		}
	}

	if !hasServices {
		errors = append(errors, ComposeValidationError{
			Line:    1,
			Message: "missing required \"services\" key",
			Field:   "services",
		})
		return errors
	}

	// Layer 3: Service validation.
	servicesNode := findMappingValue(mapping, "services")
	if servicesNode == nil || servicesNode.Kind != yaml.MappingNode {
		errors = append(errors, ComposeValidationError{
			Line:    1,
			Message: "\"services\" must be a mapping",
			Field:   "services",
		})
		return errors
	}

	for i := 0; i < len(servicesNode.Content)-1; i += 2 {
		svcKeyNode := servicesNode.Content[i]
		svcValueNode := servicesNode.Content[i+1]
		svcName := svcKeyNode.Value

		if svcValueNode.Kind != yaml.MappingNode {
			errors = append(errors, ComposeValidationError{
				Line:    svcKeyNode.Line,
				Message: fmt.Sprintf("service %q must be a mapping", svcName),
				Field:   "services." + svcName,
			})
			continue
		}

		errors = append(errors, validateService(svcName, svcValueNode)...)
	}

	return errors
}

func validateService(name string, node *yaml.Node) []ComposeValidationError {
	var errors []ComposeValidationError

	hasImage := false
	hasBuild := false

	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]

		switch keyNode.Value {
		case "image":
			hasImage = true
		case "build":
			hasBuild = true
		case "ports":
			errors = append(errors, validatePorts(name, valNode)...)
		case "volumes":
			errors = append(errors, validateVolumeMounts(name, valNode)...)
		}
	}

	if !hasImage && !hasBuild {
		errors = append(errors, ComposeValidationError{
			Line:    node.Line,
			Message: fmt.Sprintf("service %q must have \"image\" or \"build\"", name),
			Field:   "services." + name,
		})
	}

	return errors
}

func validatePorts(svcName string, node *yaml.Node) []ComposeValidationError {
	if node.Kind != yaml.SequenceNode {
		return nil
	}

	var errors []ComposeValidationError
	for _, item := range node.Content {
		port := item.Value
		if port == "" {
			// Could be a mapping-style port (long syntax), skip.
			continue
		}
		// Strip optional IP binding prefix (e.g. "127.0.0.1:8080:80").
		stripped := port
		parts := strings.SplitN(port, ":", 3)
		if len(parts) == 3 {
			stripped = parts[1] + ":" + parts[2]
		}
		if !portRegex.MatchString(stripped) {
			errors = append(errors, ComposeValidationError{
				Line:    item.Line,
				Message: fmt.Sprintf("invalid port format %q in service %q", port, svcName),
				Field:   fmt.Sprintf("services.%s.ports", svcName),
			})
		}
	}
	return errors
}

func validateVolumeMounts(svcName string, node *yaml.Node) []ComposeValidationError {
	if node.Kind != yaml.SequenceNode {
		return nil
	}

	var errors []ComposeValidationError
	for _, item := range node.Content {
		mount := item.Value
		if mount == "" {
			// Long syntax (mapping), skip simple validation.
			continue
		}
		// Short syntax must contain at least one ":"
		if !strings.Contains(mount, ":") {
			errors = append(errors, ComposeValidationError{
				Line:    item.Line,
				Message: fmt.Sprintf("volume mount %q in service %q must contain \":\" (source:target)", mount, svcName),
				Field:   fmt.Sprintf("services.%s.volumes", svcName),
			})
		}
	}
	return errors
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func parseYAMLError(err error) ComposeValidationError {
	msg := err.Error()
	// yaml.v3 errors contain "line N:" prefix.
	ve := ComposeValidationError{Line: 1, Message: msg}
	var line int
	if _, scanErr := fmt.Sscanf(msg, "yaml: line %d:", &line); scanErr == nil {
		ve.Line = line
	}
	return ve
}
