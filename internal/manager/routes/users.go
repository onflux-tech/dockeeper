package routes

import (
	"net/http"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterUserRoutes adds user and group management endpoints.
func RegisterUserRoutes(se *core.ServeEvent, app core.App, rbacStore *store.RBACStore, rbac *middleware.RBAC) {
	users := se.Router.Group("/api/dk/users")
	groups := se.Router.Group("/api/dk/groups")

	// Users CRUD.
	users.GET("", handleListUsers(app, rbacStore)).BindFunc(rbac.RequireCapability(constants.CapUsersView))
	users.POST("", handleCreateUser(app)).BindFunc(rbac.RequireCapability(constants.CapUsersManage))
	users.PATCH("/{id}", handleUpdateUser(app)).BindFunc(rbac.RequireCapability(constants.CapUsersManage))
	users.DELETE("/{id}", handleDeleteUser(app, rbacStore)).BindFunc(rbac.RequireCapability(constants.CapUsersManage))

	// Groups CRUD.
	groups.GET("", handleListGroups(rbacStore)).BindFunc(rbac.RequireCapability(constants.CapUsersView))
	groups.POST("", handleCreateGroup(rbacStore)).BindFunc(rbac.RequireCapability(constants.CapUsersManage))
	groups.PUT("/{id}", handleUpdateGroup(rbacStore)).BindFunc(rbac.RequireCapability(constants.CapUsersManage))
	groups.DELETE("/{id}", handleDeleteGroup(rbacStore)).BindFunc(rbac.RequireCapability(constants.CapUsersManage))

	// Group membership management.
	groups.POST("/{id}/members", handleAddMember(rbacStore)).BindFunc(rbac.RequireCapability(constants.CapUsersManage))
	groups.DELETE("/{id}/members/{userId}", handleRemoveMember(rbacStore)).BindFunc(rbac.RequireCapability(constants.CapUsersManage))
}

func handleListUsers(app core.App, rbacStore *store.RBACStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Auth collections need AutodateField registered to support sort by "created".
		dkUsers, _ := app.FindRecordsByFilter(constants.CollDkUsers, "", "-created", 200, 0)
		superusers, _ := app.FindRecordsByFilter(core.CollectionNameSuperusers, "", "-created", 200, 0)

		// Preload all groups for name lookup.
		allGroups, _ := rbacStore.GetGroups()
		groupNames := make(map[string]string, len(allGroups))
		for _, g := range allGroups {
			groupNames[g.Id] = g.GetString("name")
		}

		// Preload all memberships.
		allMembers, _ := app.FindRecordsByFilter(constants.CollGroupMembers, "", "", 0, 0)
		userGroups := make(map[string][]map[string]string) // userID -> [{id, name}]
		for _, m := range allMembers {
			uid := m.GetString("user")
			gid := m.GetString("group")
			gname := groupNames[gid]
			if gname == "" {
				continue
			}
			userGroups[uid] = append(userGroups[uid], map[string]string{"id": gid, "name": gname})
		}

		users := make([]map[string]any, 0, len(dkUsers)+len(superusers))

		// Add superusers first.
		for _, r := range superusers {
			if r.Email() == core.DefaultInstallerEmail {
				continue
			}
			groups := userGroups[r.Id]
			if groups == nil {
				groups = []map[string]string{}
			}
			u := userExport(r)
			u["groups"] = groups
			u["is_superuser"] = true
			u["enabled"] = true
			users = append(users, u)
		}

		// Add dk_users.
		for _, r := range dkUsers {
			groups := userGroups[r.Id]
			if groups == nil {
				groups = []map[string]string{}
			}
			u := userExport(r)
			u["groups"] = groups
			u["is_superuser"] = false
			users = append(users, u)
		}

		return e.JSON(http.StatusOK, users)
	}
}

// userExport builds a clean map from a PB auth record, extracting only the fields the frontend needs.
func userExport(r *core.Record) map[string]any {
	pub := r.PublicExport()
	name, _ := pub["name"].(string)
	return map[string]any{
		"id":      r.Id,
		"email":   r.Email(),
		"enabled": pub["enabled"],
		"name":    name,
		"created": pub["created"],
		"updated": pub["updated"],
	}
}

func handleCreateUser(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Name     string `json:"name"`
		}
		if err := e.BindBody(&body); err != nil || body.Email == "" || body.Password == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		}

		// Create user in dk_users collection (NOT _superusers).
		col, err := app.FindCachedCollectionByNameOrId(constants.CollDkUsers)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "dk_users collection not found"})
		}

		rec := core.NewRecord(col)
		rec.SetEmail(body.Email)
		rec.SetPassword(body.Password)
		rec.Set("name", body.Name)
		rec.Set("enabled", true)

		if err := app.Save(rec); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusCreated, map[string]any{
			"id":    rec.Id,
			"email": rec.Email(),
		})
	}
}

func handleUpdateUser(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		userID := e.Request.PathValue("id")

		// Try dk_users first, then _superusers.
		rec, err := app.FindRecordById(constants.CollDkUsers, userID)
		if err != nil {
			rec, err = app.FindRecordById(core.CollectionNameSuperusers, userID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
			}
		}

		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Name     string `json:"name"`
			Enabled  *bool  `json:"enabled"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		if body.Email != "" {
			rec.SetEmail(body.Email)
		}
		if body.Password != "" {
			rec.SetPassword(body.Password)
		}
		// Only set name/enabled for dk_users (not superusers).
		if !rec.IsSuperuser() {
			if body.Name != "" {
				rec.Set("name", body.Name)
			}
			if body.Enabled != nil {
				rec.Set("enabled", *body.Enabled)
			}
		}

		if err := app.Save(rec); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"id":    rec.Id,
			"email": rec.Email(),
		})
	}
}

func handleDeleteUser(app core.App, rbacStore *store.RBACStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		userID := e.Request.PathValue("id")

		// Try dk_users first.
		rec, err := app.FindRecordById(constants.CollDkUsers, userID)
		if err != nil {
			// Try _superusers.
			rec, err = app.FindRecordById(core.CollectionNameSuperusers, userID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
			}

			// Prevent deleting the last superuser.
			superusers, _ := app.FindRecordsByFilter(core.CollectionNameSuperusers, "", "", 0, 0)
			realCount := 0
			for _, su := range superusers {
				if su.Email() != core.DefaultInstallerEmail {
					realCount++
				}
			}
			if realCount <= 1 {
				return e.JSON(http.StatusConflict, map[string]string{"error": "cannot delete the last admin"})
			}
		}

		// Clean up group memberships.
		members, _ := app.FindRecordsByFilter(
			constants.CollGroupMembers,
			"user = {:userId}",
			"", 0, 0,
			dbx.Params{"userId": userID},
		)
		for _, m := range members {
			_ = app.Delete(m)
		}

		rbacStore.InvalidateUserCache(userID)

		if err := app.Delete(rec); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func handleListGroups(rbacStore *store.RBACStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		groups, err := rbacStore.GetGroups()
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, groups)
	}
}

func handleCreateGroup(rbacStore *store.RBACStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body struct {
			Name         string   `json:"name"`
			Description  string   `json:"description"`
			Capabilities []string `json:"capabilities"`
		}
		if err := e.BindBody(&body); err != nil || strings.TrimSpace(body.Name) == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
		}

		rec, err := rbacStore.CreateGroup(body.Name, body.Description, body.Capabilities, false)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusCreated, rec)
	}
}

func handleUpdateGroup(rbacStore *store.RBACStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		groupID := e.Request.PathValue("id")

		group, err := rbacStore.GetGroupByID(groupID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "group not found"})
		}

		if group.GetBool("is_preset") {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "preset groups cannot be modified"})
		}

		var body struct {
			Name         string   `json:"name"`
			Description  string   `json:"description"`
			Capabilities []string `json:"capabilities"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		name := body.Name
		if name == "" {
			name = group.GetString("name")
		}

		rec, err := rbacStore.UpdateGroup(groupID, name, body.Description, body.Capabilities)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, rec)
	}
}

func handleDeleteGroup(rbacStore *store.RBACStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		groupID := e.Request.PathValue("id")

		group, err := rbacStore.GetGroupByID(groupID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "group not found"})
		}

		if group.GetBool("is_preset") {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "preset groups cannot be deleted"})
		}

		if err := rbacStore.DeleteGroup(groupID); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func handleAddMember(rbacStore *store.RBACStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		groupID := e.Request.PathValue("id")

		var body struct {
			UserID string `json:"user_id"`
		}
		if err := e.BindBody(&body); err != nil || body.UserID == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "user_id is required"})
		}

		rec, err := rbacStore.AddMember(groupID, body.UserID)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusCreated, rec)
	}
}

func handleRemoveMember(rbacStore *store.RBACStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		groupID := e.Request.PathValue("id")
		userID := e.Request.PathValue("userId")

		// Last admin protection.
		group, err := rbacStore.GetGroupByID(groupID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "group not found"})
		}

		if group.GetString("name") == "Administrators" {
			count, err := rbacStore.CountAdminGroupMembers()
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			if count <= 1 {
				return e.JSON(http.StatusConflict, map[string]string{"error": "cannot remove the last administrator"})
			}
		}

		if err := rbacStore.RemoveMember(groupID, userID); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "removed"})
	}
}
