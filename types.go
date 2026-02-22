package main

import (
	"time"
)

// Context represents a named context that organizes and groups memories.
// It allows users to manage different conversation topics or projects.
type Context struct {
	ID          string    `json:"id"`          // Unique context identifier
	Name        string    `json:"name"`        // Human-readable context name
	Description string    `json:"description"` // Optional description of the context
	CreatedAt   time.Time `json:"created_at"`  // When the context was created
	UpdatedAt   time.Time `json:"updated_at"`  // Last update time
	MemoryCount int       `json:"memory_count"` // Number of memories in this context
	Tags        []string  `json:"tags"`        // Tags associated with this context
}

// Tag represents a label for categorizing memories.
type Tag struct {
	Name        string `json:"name"`        // Tag identifier (unique)
	Description string `json:"description"` // Optional description
	Color       string `json:"color"`       // Optional hex color for UI
	MemoryCount int    `json:"memory_count"` // Memories tagged with this
}

// ClientSession represents a client connected to the server.
// Multiple clients can share context through session identifiers.
type ClientSession struct {
	ClientID      string    `json:"client_id"`       // Unique client identifier
	CurrentContext string   `json:"current_context"` // Currently active context ID
	CreatedAt     time.Time `json:"created_at"`     // Session start time
	LastActivity  time.Time `json:"last_activity"`  // Last activity timestamp
	SharedWith    []string  `json:"shared_with"`    // List of other client IDs with access
}

// ContextData represents the full state that needs to be persisted.
type ContextData struct {
	Contexts map[string]*Context  `json:"contexts"`  // All contexts by ID
	Tags     map[string]*Tag      `json:"tags"`      // All tags by name
	Sessions map[string]*ClientSession `json:"sessions"` // Active sessions
	Version  string               `json:"version"`   // Data format version
}

// MemoryMetadata extends the metadata stored with each memory.
type MemoryMetadata struct {
	Context    string    `json:"context"`     // Context this memory belongs to
	Tags       []string  `json:"tags"`        // Tags for categorization
	CreatedAt  time.Time `json:"created_at"`  // When created
	UpdatedAt  time.Time `json:"updated_at"`  // Last update
	ClientID   string    `json:"client_id"`   // Client that created it
	SharedWith []string  `json:"shared_with"` // Clients with access
}
