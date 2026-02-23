package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"

	"github.com/philippgille/chromem-go"
	"google.golang.org/genai"
)

// makeGeminiEmbedder creates an embedding function using Gemini's embedding API.
func makeGeminiEmbedder(modelName string, client *genai.Client, logger interface{}) chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		embs, err := batchEmbedGemini(ctx, client, modelName, []string{text})
		if err != nil {
			return nil, err
		}
		return embs[0], nil
	}
}

func batchEmbedGemini(ctx context.Context, client *genai.Client, modelName string, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Batching is currently implemented via parallel calls to EmbedContent
	// as the SDK's EmbedContent takes one set of contents at a time.
	
	results := make([][]float32, len(texts))
	for i, text := range texts {
		taskType := TaskTypeDocument
		if strings.HasPrefix(text, QueryTaskPrefix) {
			taskType = TaskTypeQuery
			text = strings.TrimPrefix(text, QueryTaskPrefix)
		}

		contents := []*genai.Content{{Parts: []*genai.Part{{Text: text}}}}
		dim := int32(EmbeddingDimension)
		res, err := client.Models.EmbedContent(ctx, modelName, contents, &genai.EmbedContentConfig{
			TaskType:             taskType,
			OutputDimensionality: &dim,
		})
		if err != nil {
			return nil, fmt.Errorf("embedding failed at index %d: %w", i, err)
		}
		if len(res.Embeddings) == 0 {
			return nil, fmt.Errorf("no embeddings returned at index %d", i)
		}
		normalize(res.Embeddings[0].Values)
		results[i] = res.Embeddings[0].Values
	}
	return results, nil
}

// makeLMStudioEmbedder creates an embedding function using LM Studio's OpenAI-compatible API.
func makeLMStudioEmbedder(baseURL, modelName string, logger *log.Logger) chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		embs, err := batchEmbedLMStudio(ctx, baseURL, modelName, []string{text})
		if err != nil {
			return nil, err
		}
		return embs[0], nil
	}
}

func batchEmbedLMStudio(ctx context.Context, baseURL, modelName string, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	url := strings.TrimSuffix(baseURL, "/") + "/embeddings"
	requestBody, err := json.Marshal(map[string]interface{}{
		"model": modelName,
		"input": texts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Data) != len(texts) {
		return nil, fmt.Errorf("returned embedding count mismatch: expected %d, got %d", len(texts), len(result.Data))
	}

	results := make([][]float32, len(texts))
	for i, d := range result.Data {
		normalize(d.Embedding)
		results[i] = d.Embedding
	}
	return results, nil
}

// normalize performs L2 normalization on a vector of float32 values.
// This ensures embeddings are on the unit sphere, which improves similarity search accuracy.
func normalize(v []float32) {
	var sum float64
	for _, val := range v {
		sum += float64(val * val)
	}
	magnitude := float32(math.Sqrt(sum))
	if magnitude <= 0 {
		return
	}
	for i := range v {
		v[i] /= magnitude
	}
}
