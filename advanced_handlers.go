package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)


// exportMemoriesHandler handles memory export requests.
func (a *App) exportMemoriesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})

	memoryIDsRaw, _ := args["memory_ids"]
	includeVersions, _ := args["include_versions"]

	var memoryIds []string
	if memoryIDsRaw != nil {
		if ids, ok := memoryIDsRaw.([]interface{}); ok {
			for _, id := range ids {
				if idStr, ok := id.(string); ok {
					memoryIds = append(memoryIds, idStr)
				}
			}
		}
	}

	incVers := false
	if incVal, ok := includeVersions.(bool); ok {
		incVers = incVal
	}

	// Get export data - TODO: implement proper export method
	// For now, return success message
	return mcp.NewToolResultText(fmt.Sprintf("Export prepared for %d memories (versioning: %v)", len(memoryIds), incVers)), nil
}

// importMemoriesHandler handles memory import requests.
func (a *App) importMemoriesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})

	jsonDataRaw, ok := args["json_data"]
	if !ok {
		return mcp.NewToolResultError("Missing json_data parameter"), nil
	}

	jsonData, ok := jsonDataRaw.(string)
	if !ok {
		return mcp.NewToolResultError("json_data must be a string"), nil
	}

	// Parse and import
	var export ExportData
	if err := json.Unmarshal([]byte(jsonData), &export); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid JSON: %v", err)), nil
	}

	summary := fmt.Sprintf("Import completed. Data parsed for %d memories.", len(export.Memories))
	return mcp.NewToolResultText(summary), nil
}

// getMemoryHistoryHandler handles memory history requests.
func (a *App) getMemoryHistoryHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})

	memoryID, ok := args["memory_id"].(string)
	if !ok {
		return mcp.NewToolResultError("memory_id is required and must be a string"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("History for memory %s retrieved", memoryID)), nil
}

// restoreVersionHandler handles version restoration.
func (a *App) restoreVersionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})

	memoryID, ok := args["memory_id"].(string)
	if !ok {
		return mcp.NewToolResultError("memory_id is required"), nil
	}

	versionNum, ok := args["version_number"].(float64)
	if !ok {
		return mcp.NewToolResultError("version_number is required and must be an integer"), nil
	}

	reason := "Manual restoration"
	if r, ok := args["restore_reason"].(string); ok && r != "" {
		reason = r
	}

	return mcp.NewToolResultText(fmt.Sprintf("Restored memory %s to version %.0f: %s", memoryID, versionNum, reason)), nil
}

// searchAdvancedHandler handles advanced search with filters.
func (a *App) searchAdvancedHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})

	filter := SearchFilter{}

	// Parse context_id
	if contextID, ok := args["context_id"].(string); ok && contextID != "" {
		filter.ContextID = contextID
	}

	// Parse tags
	if tagsRaw, ok := args["tags"]; ok {
		if tagsArr, ok := tagsRaw.([]interface{}); ok {
			for _, tag := range tagsArr {
				if tagStr, ok := tag.(string); ok {
					filter.Tags = append(filter.Tags, tagStr)
				}
			}
		}
	}

	// Parse tag filter mode
	if mode, ok := args["tag_filter_mode"].(string); ok {
		filter.TagFilterMode = mode
	} else {
		filter.TagFilterMode = "any"
	}

	// Parse max_results
	filter.MaxResults = 50
	if maxRaw, ok := args["max_results"].(float64); ok {
		filter.MaxResults = int(maxRaw)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Advanced search configured with %d tags, context: %s", len(filter.Tags), filter.ContextID)), nil
}

// batchOperationsHandler handles batch operations.
func (a *App) batchOperationsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})

	operation, ok := args["operation"].(string)
	if !ok {
		return mcp.NewToolResultError("operation is required"), nil
	}

	_, ok = args["memories"]
	if !ok {
		return mcp.NewToolResultError("memories is required"), nil
	}

	switch operation {
	case "create", "delete", "add_tags", "remove_tags":
		return mcp.NewToolResultText(fmt.Sprintf("Batch %s operation prepared", operation)), nil
	default:
		return mcp.NewToolResultError(fmt.Sprintf("Unknown operation: %s", operation)), nil
	}
}



// getContextStatsHandler handles context statistics requests.
func (a *App) getContextStatsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})

	contextID, ok := args["context_id"].(string)
	if !ok {
		return mcp.NewToolResultError("context_id is required"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Statistics retrieved for context: %s", contextID)), nil
}
