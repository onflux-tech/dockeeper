package migrations

import (
	"slices"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(migrateHostProxyCapabilities, nil)
}

var hostProxyCapabilities = map[string][]string{
	"Administrators": {"proxies.view", "host.apps.view", "host.apps.manage"},
	"Operators":      {"proxies.view", "host.apps.view", "host.apps.manage"},
	"Viewers":        {"proxies.view", "host.apps.view"},
}

func migrateHostProxyCapabilities(app core.App) error {
	groups, err := app.FindRecordsByFilter("groups", "is_preset = true", "", 0, 0)
	if err != nil {
		return err
	}

	for _, group := range groups {
		extra, ok := hostProxyCapabilities[group.GetString("name")]
		if !ok {
			continue
		}
		capabilities := group.GetStringSlice("capabilities")
		changed := false
		for _, capability := range extra {
			if !slices.Contains(capabilities, capability) {
				capabilities = append(capabilities, capability)
				changed = true
			}
		}
		if !changed {
			continue
		}
		group.Set("capabilities", capabilities)
		if err := app.Save(group); err != nil {
			return err
		}
	}
	return nil
}
