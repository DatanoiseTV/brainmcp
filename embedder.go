package main

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/philippgille/chromem-go"
	"google.golang.org/genai"
)

// makeGeminiEmbedder creates an embedding function using Gemini's embedding API.
// It automatically switches between RETRIEVAL_DOCUMENT for storage and RETRIEVAL_QUERY
// for searches based on whether the text is prefixed with QUERY_TASK:.
func (a *App) makeGeminiEmbedder() chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		taskType := TaskTypeDocument
		if strings.HasPrefix(text, QueryTaskPrefix) {
			taskType = TaskTypeQuery
			text = strings.TrimPrefix(text, QueryTaskPrefix)
		}

		contents := []*genai.Content{{Parts: []*genai.Part{{Text: text}}}}
		dim := int32(EmbeddingDimension)
		res, err := a.client.Models.EmbedContent(ctx, a.modelName, contents, &genai.EmbedContentConfig{
			TaskType:             taskType,
			OutputDimensionality: &dim,
		})
		if err != nil {
			return nil, fmt.Errorf("embedding failed: %w", err)
		}

		if len(res.Embeddings) == 0 {
			return nil, fmt.Errorf("no embeddings returned from Gemini API")
		}

		values := res.Embeddings[0].Values
		normalize(values)
		return values, nil
	}
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
