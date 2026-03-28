package store

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// BatchItem represents a single item in a batch operation.
type BatchItem struct {
	ContainerID string `json:"container_id"`
	ServerID    string `json:"server_id"`
	Status      string `json:"status"` // pending | running | success | failed | cancelled
	Error       string `json:"error,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// BatchStore handles persistence for batch job records.
type BatchStore struct {
	app core.App
}

// NewBatchStore creates a new BatchStore.
func NewBatchStore(app core.App) *BatchStore {
	return &BatchStore{app: app}
}

// CreateJob persists a new batch job and returns the record.
func (s *BatchStore) CreateJob(action, createdBy string, items []BatchItem) (*core.Record, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollBatchJobs)
	if err != nil {
		return nil, err
	}

	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}

	rec := core.NewRecord(col)
	rec.Set("action", action)
	rec.Set("items", string(itemsJSON))
	rec.Set("status", "pending")
	rec.Set("total", len(items))
	rec.Set("succeeded", 0)
	rec.Set("failed", 0)
	rec.Set("cancelled", 0)
	rec.Set("created_by", createdBy)

	if err := s.app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// GetJob returns a batch job by ID.
func (s *BatchStore) GetJob(id string) (*core.Record, error) {
	return s.app.FindRecordById(constants.CollBatchJobs, id)
}

// ListJobs returns recent batch jobs, optionally filtered by status.
func (s *BatchStore) ListJobs(status string, page, perPage int) ([]*core.Record, error) {
	filter := ""
	var params []dbx.Params
	if status != "" {
		filter = "status = {:status}"
		params = append(params, dbx.Params{"status": status})
	}
	if perPage <= 0 {
		perPage = 50
	}
	offset := 0
	if page > 1 {
		offset = (page - 1) * perPage
	}

	if filter != "" {
		return s.app.FindRecordsByFilter(constants.CollBatchJobs, filter, "-created", perPage, offset, params[0])
	}
	return s.app.FindRecordsByFilter(constants.CollBatchJobs, "", "-created", perPage, offset)
}

// UpdateJob updates specific fields on a batch job record.
func (s *BatchStore) UpdateJob(id string, fields map[string]any) error {
	rec, err := s.app.FindRecordById(constants.CollBatchJobs, id)
	if err != nil {
		return err
	}
	for k, v := range fields {
		rec.Set(k, v)
	}
	return s.app.Save(rec)
}

// UpdateItemStatus updates a single item's status within the batch job items JSON.
func (s *BatchStore) UpdateItemStatus(jobID, containerID, status, errMsg, timestamp string) error {
	rec, err := s.app.FindRecordById(constants.CollBatchJobs, jobID)
	if err != nil {
		return err
	}

	var items []BatchItem
	raw := rec.GetString("items")
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return err
	}

	for i, item := range items {
		if item.ContainerID == containerID {
			items[i].Status = status
			if errMsg != "" {
				items[i].Error = errMsg
			}
			if status == "running" {
				items[i].StartedAt = timestamp
			}
			if status == "success" || status == "failed" || status == "cancelled" {
				items[i].CompletedAt = timestamp
			}
			break
		}
	}

	updated, err := json.Marshal(items)
	if err != nil {
		return err
	}
	rec.Set("items", string(updated))
	return s.app.Save(rec)
}

// IncrementCounter atomically increments a counter field (succeeded, failed, cancelled).
func (s *BatchStore) IncrementCounter(jobID, field string) error {
	rec, err := s.app.FindRecordById(constants.CollBatchJobs, jobID)
	if err != nil {
		return err
	}
	current := rec.GetInt(field)
	rec.Set(field, current+1)
	return s.app.Save(rec)
}

// GetItems parses and returns the items from a batch job record.
func (s *BatchStore) GetItems(rec *core.Record) ([]BatchItem, error) {
	var items []BatchItem
	raw := rec.GetString("items")
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}
	return items, nil
}
