package main

import (
	"context"
	"fmt"

	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/genai"
	"github.com/philippgille/chromem-go"
)

// askBrainHandler handles the ask_brain tool - LLM-assisted search with synthesis.
// It searches for relevant memories and uses Gemini to provide a conversational answer.
func (a *App) askBrainHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, _ := request.Params.Arguments.(map[string]any)
	question, _ := args["question"].(string)

	if question = strings.TrimSpace(question); question == "" {
		return mcp.NewToolResultError("Question cannot be empty"), nil
	}

	count := a.vectorStore.Count()
	if count == 0 {
		return mcp.NewToolResultText(NoMemoriesMsg), nil
	}

	nResults := DefaultSearchResults
	if count < nResults {
		nResults = count
	}

	// Use the prefix to trigger RETRIEVAL_QUERY for better accuracy
	results, err := a.vectorStore.Query(ctx, QueryTaskPrefix+question, nResults, nil, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Memory retrieval failed: %v", err)), nil
	}

	var contextBuilder strings.Builder
	for _, res := range results {
		contextBuilder.WriteString(fmt.Sprintf("- Memory [%s]: %s\n", res.ID, res.Content))
	}

	prompt := fmt.Sprintf(`You are a personal memory assistant. Based ONLY on the retrieved memories provided below, answer the user's question. 
If the answer is not contained within the memories, politely state that you don't recall that information.

Retrieved Memories:
%s

User Question: %s`, contextBuilder.String(), question)

	resp, err := a.client.Models.GenerateContent(ctx, a.llmModel, genai.Text(prompt), nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("LLM synthesis failed: %v", err)), nil
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return mcp.NewToolResultText("Unable to generate an answer (check safety filters)."), nil
	}

	answer := resp.Candidates[0].Content.Parts[0].Text
	return mcp.NewToolResultText(answer), nil
}

// rememberHandler handles the remember tool - stores or updates memories with semantic embeddings.
func (a *App) rememberHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("Invalid arguments"), nil
	}

	id, _ := args["id"].(string)
	content, _ := args["content"].(string)
	meta, _ := args["metadata"].(string)

	if id = strings.TrimSpace(id); id == "" {
		return mcp.NewToolResultError("Memory ID cannot be empty"), nil
	}
	if content = strings.TrimSpace(content); content == "" {
		return mcp.NewToolResultError("Memory content cannot be empty"), nil
	}

	// Get client's current context
	currentContext, err := a.ctx.GetClientContext(a.clientID)
	if err != nil {
		currentContext = DefaultContextID
	}

	// Create metadata with context info
	metadata := map[string]string{
		"extra":    meta,
		"context":  currentContext,
		"client":   a.clientID,
	}

	err = a.vectorStore.AddDocuments(ctx, []chromem.Document{{
		ID:       id,
		Content:  content,
		Metadata: metadata,
	}}, 1)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to store memory: %v", err)), nil
	}

	// Update context memory count
	if err := a.ctx.IncrementMemoryCount(currentContext); err != nil {
		a.logger.Printf("Warning: Failed to update context count: %v", err)
	}

	// Save context state (vector store persists automatically)
	if err := a.ctx.Save(); err != nil {
		a.logger.Printf("Warning: Failed to save context state: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Memory '%s' saved in context '%s'.", id, currentContext)), nil
}

// rememberBatchHandler handles storing multiple memories at once.
func (a *App) rememberBatchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("Invalid arguments"), nil
	}

	memoriesRaw, _ := args["memories"].([]any)
	if len(memoriesRaw) == 0 {
		return mcp.NewToolResultError("No memories provided"), nil
	}

	// Get client's current context
	currentContext, err := a.ctx.GetClientContext(a.clientID)
	if err != nil {
		currentContext = DefaultContextID
	}

	documents := make([]chromem.Document, 0, len(memoriesRaw))
	for _, m := range memoriesRaw {
		mem, ok := m.(map[string]any)
		if !ok {
			continue
		}

		id, _ := mem["id"].(string)
		content, _ := mem["content"].(string)
		meta, _ := mem["metadata"].(string)

		if id = strings.TrimSpace(id); id == "" {
			continue
		}
		if content = strings.TrimSpace(content); content == "" {
			continue
		}

		metadata := map[string]string{
			"extra":   meta,
			"context": currentContext,
			"client":  a.clientID,
		}

		documents = append(documents, chromem.Document{
			ID:       id,
			Content:  content,
			Metadata: metadata,
		})
	}

	if len(documents) == 0 {
		return mcp.NewToolResultError("No valid memories to store"), nil
	}

	err = a.vectorStore.AddDocuments(ctx, documents, 4) // Concurrency 4 for batch
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to store batch: %v", err)), nil
	}

	// Update context memory count
	for range documents {
		if err := a.ctx.IncrementMemoryCount(currentContext); err != nil {
			a.logger.Printf("Warning: Failed to update context count: %v", err)
		}
	}

	// Save context state
	if err := a.ctx.Save(); err != nil {
		a.logger.Printf("Warning: Failed to save context state: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully stored %d memories in context '%s'.", len(documents), currentContext)), nil
}

// searchHandler handles the search_memory tool - semantic similarity search.
func (a *App) searchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("Invalid arguments"), nil
	}

	query, _ := args["query"].(string)
	if query = strings.TrimSpace(query); query == "" {
		return mcp.NewToolResultError("Search query cannot be empty"), nil
	}

	totalDocs := a.vectorStore.Count()
	if totalDocs == 0 {
		return mcp.NewToolResultText(NoMemoriesMsg), nil
	}

	nResults := DefaultSearchResults
	if totalDocs < nResults {
		nResults = totalDocs
	}

	results, err := a.vectorStore.Query(ctx, QueryTaskPrefix+query, nResults, nil, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("Relevant memories:\n\n")
	for _, res := range results {
		sb.WriteString(fmt.Sprintf("[%s] (Sim: %.2f)\n%s\n---\n", res.ID, 1-res.Similarity, res.Content))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// deleteHandler handles the delete_memory tool - removes a specific memory by ID.
func (a *App) deleteHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, _ := request.Params.Arguments.(map[string]any)
	id, _ := args["id"].(string)

	if id = strings.TrimSpace(id); id == "" {
		return mcp.NewToolResultError("Memory ID cannot be empty"), nil
	}

	err := a.vectorStore.Delete(ctx, nil, nil, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Delete failed: %v", err)), nil
	}

	// Update context memory count
	currentContext, err := a.ctx.GetClientContext(a.clientID)
	if err == nil {
		if err := a.ctx.DecrementMemoryCount(currentContext); err != nil {
			a.logger.Printf("Warning: Failed to update context count: %v", err)
		}
	}

	// Save both database and context state
	if err := a.ctx.Save(); err != nil {
		a.logger.Printf("Warning: Failed to save context state: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Memory '%s' deleted.", id)), nil
}

// listHandler handles the list_memories tool - returns all stored memory IDs and snippets.
func (a *App) listHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	count := a.vectorStore.Count()
	if count == 0 {
		return mcp.NewToolResultText(EmptyBrainMsg), nil
	}

	results, err := a.vectorStore.Query(ctx, " ", count, nil, nil)
	if err != nil {
		return mcp.NewToolResultError("Could not retrieve memory list"), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Brain contains %d memories:\n", count))
	for _, res := range results {
		snippet := res.Content
		if len(snippet) > MaxSnippetLength {
			snippet = snippet[:MaxSnippetLength-3] + "..."
		}
		sb.WriteString(fmt.Sprintf("- %s: %s\n", res.ID, snippet))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// wipeHandler handles the wipe_all_memories tool - completely clears the brain database.
func (a *App) wipeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := a.vectorStore.ClearAll(ctx); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to wipe memories: %v", err)), nil
	}

	// Reset context memory counts
	contexts := a.ctx.ListContexts()
	for _, c := range contexts {
		c.MemoryCount = 0
	}

	// Save reset state
	if err := a.ctx.Save(); err != nil {
		a.logger.Printf("Warning: Failed to save context state: %v", err)
	}

	return mcp.NewToolResultText(BrainWipedMsg), nil
}
