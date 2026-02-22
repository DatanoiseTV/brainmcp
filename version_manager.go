package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MemoryVersionManager handles versioning for memories using simple JSON storage.
// This implementation avoids external database locking, allowing multiple instances
// to safely access version history concurrently.
type MemoryVersionManager struct {
	mu        sync.RWMutex
	versionDB map[string]*MemoryWithHistory
	filePath  string
	logger    *log.Logger
}

// NewMemoryVersionManager creates a new version manager with JSON-based storage.
// Pass a logger to enable activity logging, or nil to disable logging.
func NewMemoryVersionManager(dirPath string, logger *log.Logger) (*MemoryVersionManager, error) {
	// Ensure directory exists
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create version directory: %w", err)
	}

	// If no logger provided, use discard
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	filePath := filepath.Join(dirPath, "memory_versions.json")
	mvm := &MemoryVersionManager{
		versionDB: make(map[string]*MemoryWithHistory),
		filePath:  filePath,
		logger:    logger,
	}

	// Load existing version history if it exists
	if err := mvm.load(); err != nil && !os.IsNotExist(err) {
		// Log but don't fail - start fresh if corrupted
		mvm.logger.Printf("Warning: Failed to load version history: %v. Starting fresh.", err)
	} else if err == nil {
		mvm.logger.Printf("Loaded %d versioned memories from disk", len(mvm.versionDB))
	}

	return mvm, nil
}

// load reads version history from disk (internal, not thread-safe caller must lock).
func (m *MemoryVersionManager) load() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, &m.versionDB)
}

// save writes version history to disk atomically (internal, not thread-safe - caller must lock).
func (m *MemoryVersionManager) save() error {
	data, err := json.MarshalIndent(m.versionDB, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version history: %w", err)
	}

	// Write to temporary file first, then rename (atomic)
	tmpPath := m.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	if err := os.Rename(tmpPath, m.filePath); err != nil {
		return fmt.Errorf("failed to finalize version file: %w", err)
	}

	m.logger.Printf("Persisted %d versioned memories to disk", len(m.versionDB))
	return nil
}

// Close syncs any pending writes and closes the manager.
func (m *MemoryVersionManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.save()
}


// AddVersion adds a new version to a memory's history.
func (m *MemoryVersionManager) AddVersion(memoryID, content, clientID, changeNote string, context string, tags []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Fetch or create history
	history, exists := m.versionDB[memoryID]
	if !exists {
		history = &MemoryWithHistory{
			ID:             memoryID,
			CurrentVersion: 0,
			Versions:       []MemoryVersion{},
			Context:        context,
			Tags:           tags,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			Metadata:       make(map[string]string),
		}
		m.logger.Printf("Creating new version history for memory %q", memoryID)
	}

	// Add new version
	newVersion := MemoryVersion{
		VersionNumber: len(history.Versions) + 1,
		Content:       content,
		CreatedAt:     time.Now(),
		CreatedBy:     clientID,
		ChangeNote:    changeNote,
	}

	history.Versions = append(history.Versions, newVersion)
	history.CurrentVersion = newVersion.VersionNumber
	history.UpdatedAt = time.Now()
	history.Context = context
	history.Tags = tags

	m.versionDB[memoryID] = history
	m.logger.Printf("Added version %d to memory %q (client: %s)", newVersion.VersionNumber, memoryID, clientID)

	// Persist to disk
	return m.save()
}


// GetVersion retrieves a specific version of a memory.
func (m *MemoryVersionManager) GetVersion(memoryID string, versionNumber int) (*MemoryVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history, exists := m.versionDB[memoryID]
	if !exists {
		return nil, fmt.Errorf("memory %q not found", memoryID)
	}

	if versionNumber < 1 || versionNumber > len(history.Versions) {
		return nil, fmt.Errorf("version %d not found for memory %q", versionNumber, memoryID)
	}

	return &history.Versions[versionNumber-1], nil
}

// GetHistory returns the full history of a memory.
func (m *MemoryVersionManager) GetHistory(memoryID string) (*MemoryWithHistory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history, exists := m.versionDB[memoryID]
	if !exists {
		return nil, fmt.Errorf("memory %q not found", memoryID)
	}

	m.logger.Printf("Retrieved history for memory %q (%d versions)", memoryID, len(history.Versions))
	return history, nil
}


// ExportMemories exports memories to a JSON structure (doesn't write to file).
func (m *MemoryVersionManager) ExportMemories(memoryIDs []string, includeVersions bool) *ExportData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	memories := []MemoryWithHistory{}
	export := &ExportData{
		ExportedAt: time.Now(),
		ExportedBy: "system",
		Memories:   memories,
		Version:    "1.0",
	}

	for id, history := range m.versionDB {
		// Filter by memory IDs if provided
		if len(memoryIDs) > 0 {
			found := false
			for _, filterID := range memoryIDs {
				if filterID == id {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Make a copy
		copyHistory := *history
		// Strip versions if not requested
		if !includeVersions && len(copyHistory.Versions) > 0 {
			lastVersion := copyHistory.Versions[len(copyHistory.Versions)-1]
			copyHistory.Versions = []MemoryVersion{lastVersion}
		}
		export.Memories = append(export.Memories, copyHistory)
	}

	return export
}

// ImportMemories imports memories from export data.
func (m *MemoryVersionManager) ImportMemories(export *ExportData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, memory := range export.Memories {
		m.versionDB[memory.ID] = &memory
	}

	return m.save()
}


// DeleteMemoryHistory removes all version history for a memory.
func (m *MemoryVersionManager) DeleteMemoryHistory(memoryID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.versionDB[memoryID]; !exists {
		return fmt.Errorf("memory %q not found", memoryID)
	}

	delete(m.versionDB, memoryID)
	m.logger.Printf("Deleted history for memory %q", memoryID)
	return m.save()
}

// GetAllHistories returns all memory histories (for backup/export).
func (m *MemoryVersionManager) GetAllHistories() map[string]*MemoryWithHistory {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*MemoryWithHistory)
	for id, history := range m.versionDB {
		copyHistory := *history
		result[id] = &copyHistory
	}

	return result
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

	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Printf("Starting batch create operation for %d memories", len(memories))

	for _, mem := range memories {
		history := &MemoryWithHistory{
			ID:             mem.ID,
			CurrentVersion: 1,
			Context:        mem.Context,
			Tags:           mem.Tags,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			Metadata:       make(map[string]string),
			Versions: []MemoryVersion{
				{
					VersionNumber: 1,
					Content:       mem.Content,
					CreatedAt:     time.Now(),
					CreatedBy:     mem.ClientID,
					ChangeNote:    "Batch import",
				},
			},
		}
		m.versionDB[mem.ID] = history
		result.Successful++
	}

	// Save all at once
	if err := m.save(); err != nil {
		result.Failed = result.Successful
		result.Successful = 0
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to save: %v", err))
		m.logger.Printf("Batch create failed: %v", err)
		return result, err
	}

	m.logger.Printf("Batch create completed: %d successful, %d failed", result.Successful, result.Failed)
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

	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Printf("Starting batch delete operation for %d memories", len(memoryIDs))

	for _, id := range memoryIDs {
		if _, exists := m.versionDB[id]; exists {
			delete(m.versionDB, id)
			result.Successful++
		} else {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Memory %q not found", id))
		}
	}

	// Save all at once
	if err := m.save(); err != nil {
		result.Failed = result.Total
		result.Successful = 0
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to save: %v", err))
		m.logger.Printf("Batch delete failed: %v", err)
		return result, err
	}

	m.logger.Printf("Batch delete completed: %d successful, %d failed", result.Successful, result.Failed)
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

	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Printf("Starting batch add tags operation: %d memories, %d tags", len(memoryIDs), len(tags))

	for _, id := range memoryIDs {
		history, exists := m.versionDB[id]
		if !exists {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Memory %q not found", id))
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
		history.UpdatedAt = time.Now()
		result.Successful++
	}

	// Save all at once
	if err := m.save(); err != nil {
		result.Failed = result.Total
		result.Successful = 0
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to save: %v", err))
		m.logger.Printf("Batch add tags failed: %v", err)
		return result, err
	}

	m.logger.Printf("Batch add tags completed: %d successful, %d failed", result.Successful, result.Failed)
	return result, nil
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

	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Printf("Starting batch remove tags operation: %d memories, %d tags", len(memoryIDs), len(tags))

	for _, id := range memoryIDs {
		history, exists := m.versionDB[id]
		if !exists {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Memory %q not found", id))
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
		history.UpdatedAt = time.Now()
		result.Successful++
	}

	// Save all at once
	if err := m.save(); err != nil {
		result.Failed = result.Total
		result.Successful = 0
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to save: %v", err))
		m.logger.Printf("Batch remove tags failed: %v", err)
		return result, err
	}

	m.logger.Printf("Batch remove tags completed: %d successful, %d failed", result.Successful, result.Failed)
	return result, nil
}
