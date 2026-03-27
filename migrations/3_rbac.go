package migrations

import (
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// ── 1. groups ──────────────────────────────────────
		groups := core.NewBaseCollection("groups")
		groups.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.JSONField{Name: "capabilities", Required: true},
			&core.BoolField{Name: "is_preset"},
			&core.TextField{Name: "description", Max: 500},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		groups.AddIndex("idx_groups_name", true, "name", "")
		if err := app.Save(groups); err != nil {
			return err
		}

		// ── 2. group_members ───────────────────────────────
		groupMembers := core.NewBaseCollection("group_members")
		groupMembers.Fields.Add(
			&core.RelationField{
				Name:          "group",
				CollectionId:  groups.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "user", Required: true, Max: 255},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		groupMembers.AddIndex("idx_group_members_group", false, "`group`", "")
		groupMembers.AddIndex("idx_group_members_user", false, "user", "")
		groupMembers.AddIndex("idx_group_members_unique", true, "`group`, user", "")
		if err := app.Save(groupMembers); err != nil {
			return err
		}

		// ── 3. api_keys ────────────────────────────────────
		apiKeys := core.NewBaseCollection("api_keys")
		apiKeys.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.TextField{Name: "key_prefix", Required: true, Max: 8},
			&core.TextField{Name: "key_hash", Required: true, Max: 255},
			&core.TextField{Name: "user", Required: true, Max: 255},
			&core.JSONField{Name: "capabilities", Required: true},
			&core.BoolField{Name: "enabled"},
			&core.DateField{Name: "last_used_at"},
			&core.DateField{Name: "expires_at"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		apiKeys.AddIndex("idx_api_keys_key_prefix", false, "key_prefix", "")
		apiKeys.AddIndex("idx_api_keys_user", false, "user", "")
		apiKeys.AddIndex("idx_api_keys_enabled", false, "enabled", "")
		if err := app.Save(apiKeys); err != nil {
			return err
		}

		// ── 4. Seed preset groups ──────────────────────────
		groupsCol, err := app.FindCollectionByNameOrId("groups")
		if err != nil {
			return err
		}

		presets := []struct {
			Name         string
			Description  string
			Capabilities []string
		}{
			{
				Name:        "Administrators",
				Description: "Full access to everything",
				Capabilities: []string{
					"servers.view", "servers.manage",
					"containers.view", "containers.manage",
					"images.view", "images.manage",
					"compose.view", "compose.manage",
					"networks.manage", "volumes.manage",
					"backups.view", "backups.manage",
					"alerts.view", "alerts.manage",
					"notifications.view", "notifications.manage",
					"settings.view", "settings.manage",
					"users.view", "users.manage",
					"audit.view", "traefik.view",
				},
			},
			{
				Name:        "Operators",
				Description: "Can operate containers and stacks, cannot change settings/alerts/users",
				Capabilities: []string{
					"servers.view", "containers.view", "containers.manage",
					"images.view", "images.manage",
					"compose.view", "compose.manage",
					"networks.manage", "volumes.manage",
					"backups.view", "backups.manage",
					"alerts.view", "notifications.view",
					"audit.view", "traefik.view",
				},
			},
			{
				Name:        "Viewers",
				Description: "Read-only access to all data",
				Capabilities: []string{
					"servers.view", "containers.view",
					"images.view", "compose.view",
					"backups.view", "alerts.view",
					"notifications.view", "audit.view",
					"traefik.view", "settings.view",
				},
			},
		}

		for _, p := range presets {
			rec := core.NewRecord(groupsCol)
			rec.Set("name", p.Name)
			rec.Set("capabilities", p.Capabilities)
			rec.Set("is_preset", true)
			rec.Set("description", p.Description)
			if err := app.Save(rec); err != nil {
				return err
			}
		}

		// ── 5. Seed existing superusers into Administrators ─
		adminGroup, err := app.FindFirstRecordByFilter("groups", "name = 'Administrators'")
		if err != nil {
			slog.Warn("could not find Administrators group to seed members", "error", err)
			return nil
		}

		superusers, err := app.FindRecordsByFilter("_superusers", "", "", 0, 0)
		if err != nil || len(superusers) == 0 {
			return nil // no superusers to seed
		}

		membersCol, err := app.FindCollectionByNameOrId("group_members")
		if err != nil {
			return err
		}

		for _, su := range superusers {
			member := core.NewRecord(membersCol)
			member.Set("group", adminGroup.Id)
			member.Set("user", su.Id)
			if err := app.Save(member); err != nil {
				slog.Warn("failed to add superuser to Administrators group", "user_id", su.Id, "error", err)
			}
		}

		return nil
	}, func(app core.App) error {
		// Down migration: drop collections in reverse order.
		for _, name := range []string{"api_keys", "group_members", "groups"} {
			col, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue
			}
			if err := app.Delete(col); err != nil {
				return err
			}
		}
		return nil
	})
}
