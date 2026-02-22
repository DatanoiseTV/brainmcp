package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// timeNow returns current time (allows for mock in tests)
func timeNow() time.Time {
	return time.Now()
}

// MemoryVersionManager handles versioning for memories using BadgerDB for persistence.
type MemoryVersionManager struct {
	db       *badger.DB
	dbPath   string
}

// NewMemoryVersionManager creates a new version manager with BadgerDB backend.
func NewMemoryVersionManager(dirPath string) (*MemoryVersionManager, error) {
	opts := badger.DefaultOptions(dirPath).
		WithLoggingLevel(badger.ERROR)

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open version database: %w", err)
	}

	return &MemoryVersionManager{
		db:     db,
		dbPath: dirPath,
	}, nil
}

// Close closes the BadgerDB instance.
func (m *MemoryVersionManager) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}


// AddVersion adds a new version to a memory's history.
func (m *MemoryVersionManager) AddVersion(memoryID, content, clientID, changeNote string, context string, tags []string) error {
	return m.db.Update(func(txn *badger.Txn) error {
		// Fetch existing history
		history := &MemoryWithHistory{
			ID:             memoryID,
			CurrentVersion: 0,
			Versions:       []MemoryVersion{},
			Context:        context,
			Tags:           tags,
			CreatedAt:      timeNow(),
			UpdatedAt:      timeNow(),
			Metadata:       make(map[string]string),
		}

		// Try to get existing history
		item, err := txn.Get([]byte(memoryID))
		if err == nil {
			if err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, history)
			}); err != nil {
				return fmt.Errorf("failed to unmarshal history: %w", err)
			}
		} else if err != badger.ErrKeyNotFound {
			return err
		}

		// Add new version
		newVersion := MemoryVersion{
			VersionNumber: len(history.Versions) + 1,
			Content:       content,
			CreatedAt:     timeNow(),
			CreatedBy:     clientID,
			ChangeNote:    changeNote,
		}

		history.Versions = append(history.Versions, newVersion)
		history.CurrentVersion = newVersion.VersionNumber
		history.UpdatedAt = timeNow()
		history.Context = context
		history.Tags = tags

		// Store back to database
		data, err := json.Marshal(history)
		if err != nil {
			return fmt.Errorf("failed to marshal history: %w", err)
		}

		return txn.Set([]byte(memoryID), data)
	})
}


// GetVersion retrieves a specific version of a memory.
func (m *MemoryVersionManager) GetVersion(memoryID string, versionNumber int) (*MemoryVersion, error) {
	var history *MemoryWithHistory

	err := m.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(memoryID))
		if err != nil {
			return fmt.Errorf("memory %q not found", memoryID)
		}

		return item.Value(func(val []byte) error {
			history = &MemoryWithHistory{}
			return json.Unmarshal(val, history)
		})
	})

	if err != nil {
		return nil, err
	}

	if versionNumber < 1 || versionNumber > len(history.Versions) {
		return nil, fmt.Errorf("version %d not found for memory %q", versionNumber, memoryID)
	}

	return &history.Versions[versionNumber-1], nil
}

// GetHistory returns the full history of a memory.
func (m *MemoryVersionManager) GetHistory(memoryID string) (*MemoryWithHistory, error) {
	var history *MemoryWithHistory

	err := m.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(memoryID))
		if err != nil {
			return fmt.Errorf("memory %q not found", memoryID)
		}

		return item.Value(func(val []byte) error {
			history = &MemoryWithHistory{}
			return json.Unmarshal(val, history)
		})
	})

	return history, err
}


// ExportMemories exports memories to a JSON file.
func (m *MemoryVersionManager) ExportMemories(memoryIDs []string, includeVersions bool) *ExportData {
	memories := []MemoryWithHistory{}
	export := &ExportData{
		ExportedAt: timeNow(),
		ExportedBy: "system",
		Memories:   memories,
		Version:    "1.0",
	}

	m.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())

			// Filter by memory IDs if provided
			if len(memoryIDs) > 0 {
				found := false
				for _, id := range memoryIDs {
					if id == key {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			var history MemoryWithHistory
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &history)
			})

			if err == nil {
				// Strip versions if not requested
				if !includeVersions && len(history.Versions) > 0 {
					lastVersion := history.Versions[len(history.Versions)-1]
					history.Versions = []MemoryVersion{lastVersion}
				}
				export.Memories = append(export.Memories, history)
			}
		}

		return nil
	})

	return export
}

// ImportMemories imports memories from export data.
func (m *MemoryVersionManager) ImportMemories(export *ExportData) error {
	return m.db.Update(func(txn *badger.Txn) error {
		for _, memory := range export.Memories {
			data, err := json.Marshal(memory)
			if err != nil {
				return fmt.Errorf("failed to marshal memory %q: %w", memory.ID, err)
			}

			if err := txn.Set([]byte(memory.ID), data); err != nil {
				return fmt.Errorf("failed to store memory %q: %w", memory.ID, err)
			}
		}
		return nil
	})
}


// DeleteMemoryHistory removes all version history for a memory.
func (m *MemoryVersionManager) DeleteMemoryHistory(memoryID string) error {
	return m.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(memoryID))
		if err == badger.ErrKeyNotFound {
			return fmt.Errorf("memory %q not found", memoryID)
		}
		return err
	})
}

// GetAllHistories returns all memory histories (for backup/export).
func (m *MemoryVersionManager) GetAllHistories() map[string]*MemoryWithHistory {
	histories := make(map[string]*MemoryWithHistory)

	m.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())

			var history MemoryWithHistory
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &history)
			})

			if err == nil {
				histories[key] = &history
			}
		}

		return nil
	})

	return histories
}


// BatchCreateMemories creates multiple memories at once.
func (m *MemoryVersionManager) BatchCreateMemories(memories []struct {
	ID       string
	Content  string
	Context  string
	Tags     []string
	ClientID string
}) (BatchOperationResult, error) {
	result := BatchOperationResult{
		OperationType: "batch_create",
		Total:         len(memories),
		Successful:    0,
		Failed:        0,
		Errors:        []string{},
	}

	for _, mem := range memories {
		if err := m.AddVersion(mem.ID, mem.Content, mem.ClientID, "Batch import", mem.Context, mem.Tags); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create %q: %v", mem.ID, err))
		} else {
			result.Successful++
		}
	}

	return result, nil
}

// BatchDeleteMemories deletes multiple memories.
func (m *MemoryVersionManager) BatchDeleteMemories(memoryIDs []string) (BatchOperationResult, error) {
	result := BatchOperationResult{
		OperationType: "batch_delete",
		Total:         len(memoryIDs),
		Successful:    0,
		Failed:        0,
		Errors:        []string{},
	}

	for _, id := range memoryIDs {
		if err := m.DeleteMemoryHistory(id); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to delete %q: %v", id, err))
		} else {
			result.Successful++
		}
	}

	return result, nil
}

// BatchAddTags adds tags to multiple memories.
func (m *MemoryVersionManager) BatchAddTags(memoryIDs []string, tags []string) (BatchOperationResult, error) {
	result := BatchOperationResult{
		OperationType: "batch_add_tags",
		Total:         len(memoryIDs),
		Successful:    0,
		Failed:        0,
		Errors:        []string{},
	}

	err := m.db.Update(func(txn *badger.Txn) error {
		for _, id := range memoryIDs {
			item, err := txn.Get([]byte(id))
			if err != nil {
				if err == badger.ErrKeyNotFound {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("Memory %q not found", id))
					continue
				}
				return err
			}

			var history MemoryWithHistory
			err = item.Value(func(val []byte) error {
				return json.Unmarshal(val, &history)
			})
			if err != nil {
				result.Failed++
				continue
			}

			// Add new tags (dedup)
			for _, tag := range tags {
				found := false
				for _, existing := range history.Tags {
					if existing == tag {
						found = true
						break
					}
				}
				if !found {
					history.Tags = append(history.Tags, tag)
				}
			}
			history.UpdatedAt = timeNow()

			// Store back
			data, _ := json.Marshal(history)
			txn.Set([]byte(id), data)
			result.Successful++
		}
		return nil
	})

	if err != nil {
		result.Failed = len(memoryIDs)
		result.Successful = 0
	}

	return result, err
}

// BatchRemoveTags removes tags from multiple memories.
func (m *MemoryVersionManager) BatchRemoveTags(memoryIDs []string, tags []string) (BatchOperationResult, error) {
	result := BatchOperationResult{
		OperationType: "batch_remove_tags",
		Total:         len(memoryIDs),
		Successful:    0,
		Failed:        0,
		Errors:        []string{},
	}

	err := m.db.Update(func(txn *badger.Txn) error {
		for _, id := range memoryIDs {
			item, err := txn.Get([]byte(id))
			if err != nil {
				if err == badger.ErrKeyNotFound {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("Memory %q not found", id))
					continue
				}
				return err
			}

			var history MemoryWithHistory
			err = item.Value(func(val []byte) error {
				return json.Unmarshal(val, &history)
			})
			if err != nil {
				result.Failed++
				continue
			}

			// Remove tags
			newTags := []string{}
			for _, existing := range history.Tags {
				remove := false
				for _, tag := range tags {
					if existing == tag {
						remove = true
						break
					}
				}
				if !remove {
					newTags = append(newTags, existing)
				}
			}
			history.Tags = newTags
			history.UpdatedAt = timeNow()

			// Store back
			data, _ := json.Marshal(history)
			txn.Set([]byte(id), data)
			result.Successful++
		}
		return nil
	})

	if err != nil {
		result.Failed = len(memoryIDs)
		result.Successful = 0
	}

	return result, err
}
