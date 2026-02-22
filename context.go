package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ContextManager handles persistent context, tags, and client sessions.
type ContextManager struct {
	mu       sync.RWMutex
	data     *ContextData
	dataPath string
	logger   interface{} // Will be assigned from App
}

// NewContextManager creates a new context manager and loads persisted state.
func NewContextManager(dataPath string) *ContextManager {
	cm := &ContextManager{
		dataPath: dataPath,
		data: &ContextData{
			Contexts: make(map[string]*Context),
			Tags:     make(map[string]*Tag),
			Sessions: make(map[string]*ClientSession),
			Version:  "1.0",
		},
	}

	// Load persisted state if it exists
	if err := cm.Load(); err != nil {
		// Start fresh if load fails
		cm.initializeDefaults()
	}

	return cm
}

// initializeDefaults creates the default "general" context.
func (cm *ContextManager) initializeDefaults() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.data.Contexts[DefaultContextID]; !exists {
		cm.data.Contexts[DefaultContextID] = &Context{
			ID:          DefaultContextID,
			Name:        DefaultContextName,
			Description: "Default context for memories",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			MemoryCount: 0,
			Tags:        []string{},
		}
	}
}

// CreateContext creates a new named context.
func (cm *ContextManager) CreateContext(id, name, description string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.data.Contexts[id]; exists {
		return fmt.Errorf("context %q already exists", id)
	}

	cm.data.Contexts[id] = &Context{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		MemoryCount: 0,
		Tags:        []string{},
	}

	return cm.Save()
}

// GetContext retrieves a context by ID.
func (cm *ContextManager) GetContext(id string) (*Context, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	ctx, exists := cm.data.Contexts[id]
	if !exists {
		return nil, fmt.Errorf("context %q not found", id)
	}

	return ctx, nil
}

// ListContexts returns all contexts.
func (cm *ContextManager) ListContexts() []*Context {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	contexts := make([]*Context, 0, len(cm.data.Contexts))
	for _, ctx := range cm.data.Contexts {
		contexts = append(contexts, ctx)
	}

	return contexts
}

// DeleteContext removes a context.
func (cm *ContextManager) DeleteContext(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if id == DefaultContextID {
		return fmt.Errorf("cannot delete default context")
	}

	if _, exists := cm.data.Contexts[id]; !exists {
		return fmt.Errorf("context %q not found", id)
	}

	delete(cm.data.Contexts, id)
	return cm.Save()
}

// CreateTag creates a new tag for categorization.
func (cm *ContextManager) CreateTag(name, description, color string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	if _, exists := cm.data.Tags[name]; exists {
		return fmt.Errorf("tag %q already exists", name)
	}

	cm.data.Tags[name] = &Tag{
		Name:        name,
		Description: description,
		Color:       color,
		MemoryCount: 0,
	}

	return cm.Save()
}

// GetTag retrieves a tag by name.
func (cm *ContextManager) GetTag(name string) (*Tag, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	name = strings.ToLower(name)
	tag, exists := cm.data.Tags[name]
	if !exists {
		return nil, fmt.Errorf("tag %q not found", name)
	}

	return tag, nil
}

// ListTags returns all tags.
func (cm *ContextManager) ListTags() []*Tag {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	tags := make([]*Tag, 0, len(cm.data.Tags))
	for _, tag := range cm.data.Tags {
		tags = append(tags, tag)
	}

	return tags
}

// DeleteTag removes a tag.
func (cm *ContextManager) DeleteTag(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	name = strings.ToLower(name)
	if _, exists := cm.data.Tags[name]; !exists {
		return fmt.Errorf("tag %q not found", name)
	}

	delete(cm.data.Tags, name)
	return cm.Save()
}

// RegisterSession creates a new client session.
func (cm *ContextManager) RegisterSession(clientID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.data.Sessions) >= MaxConcurrentClients {
		return fmt.Errorf("maximum concurrent clients reached")
	}

	cm.data.Sessions[clientID] = &ClientSession{
		ClientID:       clientID,
		CurrentContext: DefaultContextID,
		CreatedAt:      time.Now(),
		LastActivity:   time.Now(),
		SharedWith:     []string{},
	}

	return cm.Save()
}

// UnregisterSession removes a client session.
func (cm *ContextManager) UnregisterSession(clientID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.data.Sessions[clientID]; !exists {
		return fmt.Errorf("session %q not found", clientID)
	}

	delete(cm.data.Sessions, clientID)
	return cm.Save()
}

// GetSession retrieves a client session.
func (cm *ContextManager) GetSession(clientID string) (*ClientSession, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	session, exists := cm.data.Sessions[clientID]
	if !exists {
		return nil, fmt.Errorf("session %q not found", clientID)
	}

	return session, nil
}

// SwitchContext changes the current context for a client.
func (cm *ContextManager) SwitchContext(clientID, contextID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.data.Sessions[clientID]
	if !exists {
		return fmt.Errorf("session %q not found", clientID)
	}

	if _, exists := cm.data.Contexts[contextID]; !exists {
		return fmt.Errorf("context %q not found", contextID)
	}

	session.CurrentContext = contextID
	session.LastActivity = time.Now()

	return cm.Save()
}

// GetClientContext returns the current context for a client.
func (cm *ContextManager) GetClientContext(clientID string) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	session, exists := cm.data.Sessions[clientID]
	if !exists {
		// Auto-register session if not found
		cm.mu.RUnlock()
		cm.RegisterSession(clientID)
		cm.mu.RLock()
		return DefaultContextID, nil
	}

	return session.CurrentContext, nil
}

// ShareContext grants another client access to a context.
func (cm *ContextManager) ShareContext(ownerClientID, targetClientID, contextID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Verify context exists
	if _, exists := cm.data.Contexts[contextID]; !exists {
		return fmt.Errorf("context %q not found", contextID)
	}

	// Get target session
	session, exists := cm.data.Sessions[targetClientID]
	if !exists {
		return fmt.Errorf("target session %q not found", targetClientID)
	}

	// Check if already shared
	for _, id := range session.SharedWith {
		if id == contextID {
			return fmt.Errorf("context already shared with this client")
		}
	}

	session.SharedWith = append(session.SharedWith, contextID)
	return cm.Save()
}

// IncrementMemoryCount increments the memory count for a context.
func (cm *ContextManager) IncrementMemoryCount(contextID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ctx, exists := cm.data.Contexts[contextID]
	if !exists {
		return fmt.Errorf("context %q not found", contextID)
	}

	ctx.MemoryCount++
	ctx.UpdatedAt = time.Now()

	return nil // Don't save on every increment, batched save
}

// IncrementTagCount increments the memory count for a tag.
func (cm *ContextManager) IncrementTagCount(tagName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	tagName = strings.ToLower(tagName)
	tag, exists := cm.data.Tags[tagName]
	if !exists {
		return fmt.Errorf("tag %q not found", tagName)
	}

	tag.MemoryCount++
	return nil // Don't save on every increment, batched save
}

// DecrementMemoryCount decrements the memory count for a context.
func (cm *ContextManager) DecrementMemoryCount(contextID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ctx, exists := cm.data.Contexts[contextID]
	if !exists {
		return fmt.Errorf("context %q not found", contextID)
	}

	if ctx.MemoryCount > 0 {
		ctx.MemoryCount--
	}
	ctx.UpdatedAt = time.Now()

	return nil // Don't save on every decrement, batched save
}

// UpdateActivity updates the last activity time for a session.
func (cm *ContextManager) UpdateActivity(clientID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if session, exists := cm.data.Sessions[clientID]; exists {
		session.LastActivity = time.Now()
	}
}

// Save persists the context data to disk.
func (cm *ContextManager) Save() error {
	data, err := json.MarshalIndent(cm.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context data: %w", err)
	}

	if err := os.WriteFile(cm.dataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write context data: %w", err)
	}

	return nil
}

// Load restores the context data from disk.
func (cm *ContextManager) Load() error {
	data, err := os.ReadFile(cm.dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			cm.initializeDefaults()
			return nil
		}
		return fmt.Errorf("failed to read context data: %w", err)
	}

	if err := json.Unmarshal(data, cm.data); err != nil {
		return fmt.Errorf("failed to unmarshal context data: %w", err)
	}

	// Ensure defaults exist
	if _, exists := cm.data.Contexts[DefaultContextID]; !exists {
		cm.initializeDefaults()
	}

	return nil
}

// GetMemoryMetadata creates metadata for a new memory with proper initialization.
func (cm *ContextManager) GetMemoryMetadata(clientID, contextID string, tags []string) *MemoryMetadata {
	normalizedTags := make([]string, len(tags))
	for i, tag := range tags {
		normalizedTags[i] = strings.ToLower(tag)
	}

	return &MemoryMetadata{
		Context:    contextID,
		Tags:       normalizedTags,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ClientID:   clientID,
		SharedWith: []string{},
	}
}
