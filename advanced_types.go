package main

import (
	"time"
)

// MemoryVersion represents a single version of a memory.
type MemoryVersion struct {
	VersionNumber int       `json:"version_number"` // Version number (1, 2, 3...)
	Content       string    `json:"content"`        // Memory content at this version
	CreatedAt     time.Time `json:"created_at"`    // When this version was created
	CreatedBy     string    `json:"created_by"`    // Client ID that created this version
	ChangeNote    string    `json:"change_note"`   // Optional note about what changed
}

// MemoryWithHistory extends memory with version history.
type MemoryWithHistory struct {
	ID            string            `json:"id"`             // Memory ID
	CurrentVersion int              `json:"current_version"` // Current version number
	Versions      []MemoryVersion   `json:"versions"`       // All versions in order
	Context       string            `json:"context"`        // Current context
	Tags          []string          `json:"tags"`          // Current tags
	CreatedAt     time.Time        `json:"created_at"`    // Original creation time
	UpdatedAt     time.Time        `json:"updated_at"`    // Last update time
	Metadata      map[string]string `json:"metadata"`      // Additional metadata
}

// ExportData represents a complete export of memories.
type ExportData struct {
	ExportedAt  time.Time              `json:"exported_at"`
	ExportedBy  string                 `json:"exported_by"`
	Memories    []MemoryWithHistory    `json:"memories"`
	Contexts    map[string]*Context    `json:"contexts"`
	Tags        map[string]*Tag        `json:"tags"`
	Version     string                 `json:"version"` // Export format version
}

// BatchOperation represents a batch operation on memories.
type BatchOperation struct {
	OperationType string   `json:"operation_type"` // "create", "delete", "tag", "untag"
	MemoryIDs     []string `json:"memory_ids"`     // Memory IDs to operate on
	Content       string   `json:"content"`        // For create operations
	TagName       string   `json:"tag_name"`       // For tag operations
	Metadata      map[string]string `json:"metadata"` // For create operations
	ClientID      string   `json:"client_id"`      // Who initiated this
	CreatedAt     time.Time `json:"created_at"`   // When operation was created
}

// BatchOperationResult represents the result of a batch operation.
type BatchOperationResult struct {
	OperationType string `json:"operation_type"`
	Total         int    `json:"total"`        // Total items targeted
	Successful    int    `json:"successful"`   // Successfully processed
	Failed        int    `json:"failed"`       // Failed items
	Errors        []string `json:"errors"`    // Error messages
	OperationID   string `json:"operation_id"` // Unique operation ID
}

// SearchFilter represents filtering criteria for searching memories.
type SearchFilter struct {
	Query           string    `json:"query"`            // Search query
	ContextID       string    `json:"context_id"`       // Filter by context
	Tags            []string  `json:"tags"`            // Filter by tags (AND/OR logic)
	StartDate       time.Time `json:"start_date"`      // Filter by date range start
	EndDate         time.Time `json:"end_date"`        // Filter by date range end
	CreatedBy       string    `json:"created_by"`      // Filter by client ID
	MaxResults      int       `json:"max_results"`     // Limit results
	TagFilterMode   string    `json:"tag_filter_mode"` // "all" (AND) or "any" (OR)
}

// SearchResult represents a search result with metadata.
type SearchResult struct {
	ID            string            `json:"id"`
	Content       string            `json:"content"`
	Similarity    float32           `json:"similarity"`
	Context       string            `json:"context"`
	Tags          []string          `json:"tags"`
	CurrentVersion int              `json:"current_version"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Metadata      map[string]string `json:"metadata"`
}
