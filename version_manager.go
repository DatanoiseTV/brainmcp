package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// MemoryVersionManager handles versioning for memories.
type MemoryVersionManager struct {
	// In-memory version history (ideally would be in a proper database)
	histories map[string]*MemoryWithHistory
	filePath  string
}

// NewMemoryVersionManager creates a new version manager.
func NewMemoryVersionManager(filePath string) *MemoryVersionManager {
	return &MemoryVersionManager{
		histories: make(map[string]*MemoryWithHistory),
		filePath:  filePath,
	}
}

// AddVersion adds a new version to a memory's history.
func (m *MemoryVersionManager) AddVersion(memoryID, content, clientID, changeNote string, context string, tags []string) error {
	history, exists := m.histories[memoryID]
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
		m.histories[memoryID] = history
	}

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

	return nil
}

// GetVersion retrieves a specific version of a memory.
func (m *MemoryVersionManager) GetVersion(memoryID string, versionNumber int) (*MemoryVersion, error) {
	history, exists := m.histories[memoryID]
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
	history, exists := m.histories[memoryID]
	if !exists {
		return nil, fmt.Errorf("memory %q not found", memoryID)
	}
	return history, nil
}

// ExportMemories exports memories to a JSON file.
func (m *MemoryVersionManager) ExportMemories(filePath string, memoryIDs []string, clientID string, ctx *ContextManager) error {
	memories := []MemoryWithHistory{}

	if len(memoryIDs) == 0 {
		// Export all memories
		for _, history := range m.histories {
			memories = append(memories, *history)
		}
	} else {
		// Export specific memories
		for _, id := range memoryIDs {
			if history, exists := m.histories[id]; exists {
				memories = append(memories, *history)
			}
		}
	}

	exportData := ExportData{
		ExportedAt: time.Now(),
		ExportedBy: clientID,
		Memories:   memories,
		Contexts:   ctx.data.Contexts,
		Tags:       ctx.data.Tags,
		Version:    "1.0",
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// ImportMemories imports memories from a JSON file.
func (m *MemoryVersionManager) ImportMemories(filePath string) (*ExportData, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read import file: %w", err)
	}

	var exportData ExportData
	if err := json.Unmarshal(data, &exportData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal import data: %w", err)
	}

	// Load memories into history
	for _, memory := range exportData.Memories {
		m.histories[memory.ID] = &memory
	}

	return &exportData, nil
}

// DeleteMemoryHistory removes all version history for a memory.
func (m *MemoryVersionManager) DeleteMemoryHistory(memoryID string) error {
	if _, exists := m.histories[memoryID]; !exists {
		return fmt.Errorf("memory %q not found", memoryID)
	}

	delete(m.histories, memoryID)
	return nil
}

// GetAllHistories returns all memory histories (for backup/export).
func (m *MemoryVersionManager) GetAllHistories() map[string]*MemoryWithHistory {
	return m.histories
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

	for _, id := range memoryIDs {
		if history, exists := m.histories[id]; exists {
			for _, tag := range tags {
				// Check if tag already exists
				tagExists := false
				for _, existingTag := range history.Tags {
					if existingTag == tag {
						tagExists = true
						break
					}
				}
				if !tagExists {
					history.Tags = append(history.Tags, tag)
				}
			}
			history.UpdatedAt = time.Now()
			result.Successful++
		} else {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Memory %q not found", id))
		}
	}

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

	for _, id := range memoryIDs {
		if history, exists := m.histories[id]; exists {
			newTags := []string{}
			for _, existingTag := range history.Tags {
				shouldKeep := true
				for _, removeTag := range tags {
					if existingTag == removeTag {
						shouldKeep = false
						break
					}
				}
				if shouldKeep {
					newTags = append(newTags, existingTag)
				}
			}
			history.Tags = newTags
			history.UpdatedAt = time.Now()
			result.Successful++
		} else {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Memory %q not found", id))
		}
	}

	return result, nil
}
