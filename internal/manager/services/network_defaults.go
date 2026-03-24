package services

import "strings"

var protectedDockerNetworks = map[string]struct{}{
	"bridge": {},
	"host":   {},
	"none":   {},
}

func isProtectedDockerNetwork(name string) bool {
	_, ok := protectedDockerNetworks[strings.ToLower(strings.TrimSpace(name))]
	return ok
}