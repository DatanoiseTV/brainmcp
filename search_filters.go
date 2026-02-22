package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SearchFilterEngine handles advanced searching with multiple filters.
type SearchFilterEngine struct {
	collection interface{} // chromem.Collection
	versionMgr *MemoryVersionManager
	ctxMgr     *ContextManager
}

// NewSearchFilterEngine creates a new search filter engine.
func NewSearchFilterEngine(versionMgr *MemoryVersionManager, ctxMgr *ContextManager) *SearchFilterEngine {
	return &SearchFilterEngine{
		versionMgr: versionMgr,
		ctxMgr:     ctxMgr,
	}
}

// FilterMemories filters memories based on search criteria.
func (s *SearchFilterEngine) FilterMemories(filter SearchFilter) []SearchResult {
	results := []SearchResult{}

	// Get all memories from version manager
	allHistories := s.versionMgr.GetAllHistories()

	for memoryID, history := range allHistories {
		// Apply context filter
		if filter.ContextID != "" && history.Context != filter.ContextID {
			continue
		}

		// Apply date range filter
		if !filter.StartDate.IsZero() && history.CreatedAt.Before(filter.StartDate) {
			continue
		}
		if !filter.EndDate.IsZero() && history.UpdatedAt.After(filter.EndDate) {
			continue
		}

		// Apply client ID filter
		if filter.CreatedBy != "" && len(history.Versions) > 0 {
			if history.Versions[0].CreatedBy != filter.CreatedBy {
				continue
			}
		}

		// Apply tag filter
		if len(filter.Tags) > 0 {
			if filter.TagFilterMode == "all" {
				// Must have ALL tags (AND logic)
				hasAllTags := true
				for _, filterTag := range filter.Tags {
					hasTag := false
					for _, memTag := range history.Tags {
						if strings.EqualFold(memTag, filterTag) {
							hasTag = true
							break
						}
					}
					if !hasTag {
						hasAllTags = false
						break
					}
				}
				if !hasAllTags {
					continue
				}
			} else {
				// Must have ANY tag (OR logic)
				hasAnyTag := false
				for _, filterTag := range filter.Tags {
					for _, memTag := range history.Tags {
						if strings.EqualFold(memTag, filterTag) {
							hasAnyTag = true
							break
						}
					}
					if hasAnyTag {
						break
					}
				}
				if !hasAnyTag {
					continue
				}
			}
		}

		// Get current version content
		if len(history.Versions) > 0 {
			currentVersion := history.Versions[len(history.Versions)-1]

			result := SearchResult{
				ID:            memoryID,
				Content:       currentVersion.Content,
				Similarity:    1.0, // Base similarity for filtered results
				Context:       history.Context,
				Tags:          history.Tags,
				CurrentVersion: history.CurrentVersion,
				CreatedAt:     history.CreatedAt,
				UpdatedAt:     history.UpdatedAt,
				Metadata:      history.Metadata,
			}

			results = append(results, result)
		}
	}

	// Apply max results limit
	if filter.MaxResults > 0 && len(results) > filter.MaxResults {
		results = results[:filter.MaxResults]
	}

	return results
}

// SearchByContextAndTags performs a combined search.
func (s *SearchFilterEngine) SearchByContextAndTags(ctx context.Context, contextID string, tags []string, tagMode string) []SearchResult {
	filter := SearchFilter{
		ContextID:     contextID,
		Tags:          tags,
		TagFilterMode: tagMode, // "all" or "any"
		MaxResults:    50,
	}

	return s.FilterMemories(filter)
}

// SearchByDateRange performs a date-based search.
func (s *SearchFilterEngine) SearchByDateRange(startDate, endDate time.Time) []SearchResult {
	filter := SearchFilter{
		StartDate:  startDate,
		EndDate:    endDate,
		MaxResults: 100,
	}

	return s.FilterMemories(filter)
}

// SearchByContext performs a context-specific search.
func (s *SearchFilterEngine) SearchByContext(contextID string, maxResults int) []SearchResult {
	filter := SearchFilter{
		ContextID:  contextID,
		MaxResults: maxResults,
	}

	return s.FilterMemories(filter)
}

// GetMemoriesByTag returns all memories with a specific tag.
func (s *SearchFilterEngine) GetMemoriesByTag(tagName string) []SearchResult {
	filter := SearchFilter{
		Tags:          []string{tagName},
		TagFilterMode: "any",
		MaxResults:    100,
	}

	return s.FilterMemories(filter)
}

// GetMemoriesByMultipleTags returns memories matching multiple tags.
func (s *SearchFilterEngine) GetMemoriesByMultipleTags(tags []string, matchAll bool) []SearchResult {
	mode := "any"
	if matchAll {
		mode = "all"
	}

	filter := SearchFilter{
		Tags:          tags,
		TagFilterMode: mode,
		MaxResults:    100,
	}

	return s.FilterMemories(filter)
}

// GetContextStats returns statistics for a context.
func (s *SearchFilterEngine) GetContextStats(contextID string) map[string]interface{} {
	memories := s.SearchByContext(contextID, 10000)

	stats := map[string]interface{}{
		"context_id":        contextID,
		"memory_count":      len(memories),
		"unique_tags":       []string{},
		"oldest_memory":     nil,
		"newest_memory":     nil,
		"total_characters":  0,
	}

	tagMap := make(map[string]bool)
	var oldestTime, newestTime time.Time

	for _, mem := range memories {
		// Collect unique tags
		for _, tag := range mem.Tags {
			tagMap[tag] = true
		}

		// Track dates
		if oldestTime.IsZero() || mem.CreatedAt.Before(oldestTime) {
			oldestTime = mem.CreatedAt
		}
		if newestTime.IsZero() || mem.UpdatedAt.After(newestTime) {
			newestTime = mem.UpdatedAt
		}

		// Count characters
		stats["total_characters"] = stats["total_characters"].(int) + len(mem.Content)
	}

	if !oldestTime.IsZero() {
		stats["oldest_memory"] = oldestTime
	}
	if !newestTime.IsZero() {
		stats["newest_memory"] = newestTime
	}

	uniqueTags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		uniqueTags = append(uniqueTags, tag)
	}
	stats["unique_tags"] = uniqueTags

	return stats
}

// ValidateFilter checks if a filter is valid.
func (s *SearchFilterEngine) ValidateFilter(filter SearchFilter) error {
	if filter.ContextID != "" {
		_, err := s.ctxMgr.GetContext(filter.ContextID)
		if err != nil {
			return fmt.Errorf("invalid context_id: %w", err)
		}
	}

	if !filter.StartDate.IsZero() && !filter.EndDate.IsZero() {
		if filter.StartDate.After(filter.EndDate) {
			return fmt.Errorf("start_date cannot be after end_date")
		}
	}

	if filter.TagFilterMode != "" && filter.TagFilterMode != "all" && filter.TagFilterMode != "any" {
		return fmt.Errorf("tag_filter_mode must be 'all' or 'any'")
	}

	return nil
}
