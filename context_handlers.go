package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// createContextHandler creates a new named context.
func (a *App) createContextHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, _ := request.Params.Arguments.(map[string]any)
	id, _ := args["id"].(string)
	name, _ := args["name"].(string)
	description, _ := args["description"].(string)

	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)

	if id == "" {
		return mcp.NewToolResultError("Context ID cannot be empty"), nil
	}
	if name == "" {
		return mcp.NewToolResultError("Context name cannot be empty"), nil
	}

	if err := a.ctx.CreateContext(id, name, description); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create context: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Context '%s' (%s) created successfully.", name, id)), nil
}

// listContextsHandler lists all available contexts.
func (a *App) listContextsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	contexts := a.ctx.ListContexts()
	if len(contexts) == 0 {
		return mcp.NewToolResultText("No contexts found."), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Available contexts (%d total):\n\n", len(contexts)))
	for _, c := range contexts {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", c.ID, c.Name))
		if c.Description != "" {
			sb.WriteString(fmt.Sprintf("  Description: %s\n", c.Description))
		}
		sb.WriteString(fmt.Sprintf("  Memories: %d\n", c.MemoryCount))
		sb.WriteString("\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// switchContextHandler switches the current context for a client.
func (a *App) switchContextHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, _ := request.Params.Arguments.(map[string]any)
	contextID, _ := args["context_id"].(string)
	clientID, _ := args["client_id"].(string)

	contextID = strings.TrimSpace(contextID)
	if contextID == "" {
		return mcp.NewToolResultError("Context ID cannot be empty"), nil
	}

	// Use provided client ID or default
	if clientID = strings.TrimSpace(clientID); clientID == "" {
		clientID = a.clientID
	}

	// Register session if needed
	if _, err := a.ctx.GetSession(clientID); err != nil {
		if err := a.ctx.RegisterSession(clientID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to register session: %v", err)), nil
		}
	}

	if err := a.ctx.SwitchContext(clientID, contextID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to switch context: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Switched to context '%s'.", contextID)), nil
}

// shareContextHandler shares a context with another client.
func (a *App) shareContextHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, _ := request.Params.Arguments.(map[string]any)
	contextID, _ := args["context_id"].(string)
	targetClientID, _ := args["target_client_id"].(string)

	contextID = strings.TrimSpace(contextID)
	targetClientID = strings.TrimSpace(targetClientID)

	if contextID == "" {
		return mcp.NewToolResultError("Context ID cannot be empty"), nil
	}
	if targetClientID == "" {
		return mcp.NewToolResultError("Target client ID cannot be empty"), nil
	}

	// Ensure target session exists
	if _, err := a.ctx.GetSession(targetClientID); err != nil {
		if err := a.ctx.RegisterSession(targetClientID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to register target session: %v", err)), nil
		}
	}

	if err := a.ctx.ShareContext(a.clientID, targetClientID, contextID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to share context: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Context '%s' shared with client '%s'.", contextID, targetClientID)), nil
}

// createTagHandler creates a new tag for categorization.
func (a *App) createTagHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, _ := request.Params.Arguments.(map[string]any)
	name, _ := args["name"].(string)
	description, _ := args["description"].(string)
	color, _ := args["color"].(string)

	name = strings.TrimSpace(name)
	if name == "" {
		return mcp.NewToolResultError("Tag name cannot be empty"), nil
	}

	if err := a.ctx.CreateTag(name, description, color); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create tag: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Tag '%s' created successfully.", name)), nil
}

// listTagsHandler lists all available tags.
func (a *App) listTagsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tags := a.ctx.ListTags()
	if len(tags) == 0 {
		return mcp.NewToolResultText("No tags found."), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Available tags (%d total):\n\n", len(tags)))
	for _, tag := range tags {
		sb.WriteString(fmt.Sprintf("- %s", tag.Name))
		if tag.Color != "" {
			sb.WriteString(fmt.Sprintf(" (color: %s)", tag.Color))
		}
		sb.WriteString("\n")
		if tag.Description != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", tag.Description))
		}
		sb.WriteString(fmt.Sprintf("  Memories: %d\n\n", tag.MemoryCount))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// addTagHandler adds a tag to an existing memory.
func (a *App) addTagHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, _ := request.Params.Arguments.(map[string]any)
	memoryID, _ := args["memory_id"].(string)
	tag, _ := args["tag"].(string)

	memoryID = strings.TrimSpace(memoryID)
	tag = strings.TrimSpace(tag)

	if memoryID == "" {
		return mcp.NewToolResultError("Memory ID cannot be empty"), nil
	}
	if tag == "" {
		return mcp.NewToolResultError("Tag cannot be empty"), nil
	}

	tag = strings.ToLower(tag)

	// Verify tag exists or create it
	if _, err := a.ctx.GetTag(tag); err != nil {
		if err := a.ctx.CreateTag(tag, "", ""); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create tag: %v", err)), nil
		}
	}

	// Retrieve the existing memory to update its metadata
	memory, err := a.collection.GetByID(ctx, memoryID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Memory not found: %v", err)), nil
	}

	// Update the tags field in metadata (comma-separated)
	if memory.Metadata == nil {
		memory.Metadata = make(map[string]string)
	}

	currentTags := memory.Metadata["tags"]
	var tags []string
	if currentTags != "" {
		tags = strings.Split(currentTags, ",")
	}

	// Check if tag already exists
	tagExists := false
	for _, t := range tags {
		if strings.TrimSpace(t) == tag {
			tagExists = true
			break
		}
	}

	if !tagExists {
		tags = append(tags, tag)
		memory.Metadata["tags"] = strings.Join(tags, ",")

		// Delete the old memory and re-add with updated metadata
		if err := a.collection.Delete(ctx, nil, nil, memoryID); err != nil {
			a.logger.Printf("Warning: Failed to delete old memory during tag update: %v", err)
		}

		if err := a.collection.AddDocument(ctx, memory); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update memory with tag: %v", err)), nil
		}

		// Persist the updated database
		if err := a.db.ExportToFile(a.dbPath, true, ""); err != nil {
			a.logger.Printf("Warning: Failed to persist memory update to disk: %v", err)
		}

		if err := a.ctx.IncrementTagCount(tag); err != nil {
			a.logger.Printf("Warning: Failed to increment tag count: %v", err)
		}
	}

	if err := a.ctx.Save(); err != nil {
		a.logger.Printf("Warning: Failed to save context state: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Tag '%s' added to memory '%s'.", tag, memoryID)), nil
}

// searchByTagHandler searches for memories by tag.
func (a *App) searchByTagHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, _ := request.Params.Arguments.(map[string]any)
	tagName, _ := args["tag"].(string)

	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return mcp.NewToolResultError("Tag cannot be empty"), nil
	}

	tagName = strings.ToLower(tagName)

	// Verify tag exists
	if _, err := a.ctx.GetTag(tagName); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Tag not found: %v", err)), nil
	}

	// Query all memories and filter by tag
	totalDocs := a.collection.Count()
	if totalDocs == 0 {
		return mcp.NewToolResultText(EmptyBrainMsg), nil
	}

	results, err := a.collection.Query(ctx, " ", totalDocs, nil, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
	}

	var sb strings.Builder
	matchCount := 0

	for _, res := range results {
		// Check if memory has the tag in metadata
		if tags, ok := res.Metadata["tags"]; ok && strings.Contains(tags, tagName) {
			if matchCount == 0 {
				sb.WriteString(fmt.Sprintf("Memories tagged with '%s':\n\n", tagName))
			}
			matchCount++
			sb.WriteString(fmt.Sprintf("[%s]\n%s\n---\n", res.ID, res.Content))
		}
	}

	if matchCount == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No memories found with tag '%s'.", tagName)), nil
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// saveToDiskHandler persists the database and context state to disk.
func (a *App) saveToDiskHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Save vector database
	if err := a.db.ExportToFile(a.dbPath, true, ""); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to save vector database: %v", err)), nil
	}

	// Save context state
	if err := a.ctx.Save(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to save context state: %v", err)), nil
	}

	return mcp.NewToolResultText("Database and context state saved successfully to disk."), nil
}
