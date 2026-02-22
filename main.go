package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/philippgille/chromem-go"
	"google.golang.org/genai"
)

type App struct {
	db         *chromem.DB
	collection *chromem.Collection
	client     *genai.Client
	dbPath     string
	testMode   bool
	modelName  string
}

func main() {
	testMode := flag.Bool("t", false, "Run in interactive CLI test mode")
	modelFlag := flag.String("model", "gemini-embedding-001", "Gemini embedding model")
	flag.Parse()

	ctx := context.Background()
	dbPath := "brain_memory.bin"

	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: geminiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create GenAI client: %v", err)
	}

	db := chromem.NewDB()
	app := &App{
		db:        db,
		client:    client,
		dbPath:    dbPath,
		testMode:  *testMode,
		modelName: *modelFlag,
	}

	embFunc := app.makeGeminiEmbedder()
	col, err := db.GetOrCreateCollection("brain_memory", nil, embFunc)
	if err != nil {
		log.Fatal(err)
	}
	app.collection = col

	// Load existing memory from file if it exists and isn't empty
	if info, err := os.Stat(dbPath); err == nil && info.Size() > 0 {
		err = db.ImportFromFile(dbPath, "")
		if err != nil && app.testMode {
			fmt.Printf("Note: Started fresh (DB import failed: %v)\n", err)
		}
	}

	if *testMode {
		app.runInteractiveCLI(ctx)
		return
	}

	s := server.NewMCPServer("brain-mcp", "3.0.0")

	s.AddTool(mcp.NewTool("remember",
		mcp.WithDescription("Stores information with semantic vectors for long-term recall."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Unique ID for this memory")),
		mcp.WithString("content", mcp.Required(), mcp.Description("The text content to remember")),
		mcp.WithString("metadata", mcp.Description("Optional metadata")),
	), app.rememberHandler)

	s.AddTool(mcp.NewTool("search_memory",
		mcp.WithDescription("Search memory using semantic similarity."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Natural language search query")),
	), app.searchHandler)

	log.Println("BrainMCP Server starting on Stdio...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func (a *App) makeGeminiEmbedder() chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		taskType := "RETRIEVAL_DOCUMENT"
		if strings.HasPrefix(text, "QUERY_TASK:") {
			taskType = "RETRIEVAL_QUERY"
			text = strings.TrimPrefix(text, "QUERY_TASK:")
		}

		contents := []*genai.Content{{Parts: []*genai.Part{{Text: text}}}}
		dim := int32(768)

		res, err := a.client.Models.EmbedContent(ctx, a.modelName, contents, &genai.EmbedContentConfig{
			TaskType:             taskType,
			OutputDimensionality: &dim,
		})
		if err != nil {
			return nil, fmt.Errorf("gemini embedding failed: %w", err)
		}

		if len(res.Embeddings) == 0 {
			return nil, fmt.Errorf("gemini returned no embeddings")
		}

		values := res.Embeddings[0].Values
		normalize(values)
		return values, nil
	}
}

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

func (a *App) rememberHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("Invalid args"), nil
	}

	id, _ := args["id"].(string)
	content, _ := args["content"].(string)
	meta, _ := args["metadata"].(string)

	err := a.collection.AddDocuments(ctx, []chromem.Document{{
		ID:       id,
		Content:  content,
		Metadata: map[string]string{"extra": meta},
	}}, 1)

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Embedding failed: %v", err)), nil
	}

	_ = a.db.ExportToFile(a.dbPath, true, "")
	return mcp.NewToolResultText(fmt.Sprintf("Memory '%s' saved.", id)), nil
}

func (a *App) searchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("Invalid args"), nil
	}

	query, _ := args["query"].(string)

	// FIX: Use the .Count() method to see how many documents exist
	totalDocs := a.collection.Count()
	if totalDocs == 0 {
		return mcp.NewToolResultText("No relevant memories found (Your memory is currently empty)."), nil
	}

	// We can't ask for more documents than we have
	nResults := 3
	if totalDocs < nResults {
		nResults = totalDocs
	}

	results, err := a.collection.Query(ctx, "QUERY_TASK:"+query, nResults, nil, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("Found relevant memories:\n\n")
	for _, res := range results {
		sb.WriteString(fmt.Sprintf("[%s] (Score: %.2f)\n%s\n---\n", res.ID, 1-res.Similarity, res.Content))
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (a *App) runInteractiveCLI(ctx context.Context) {
	fmt.Println("=== BrainMCP Test Mode ===")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nbrain> ")
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 1 {
			continue
		}

		switch strings.ToLower(parts[0]) {
		case "exit":
			return
		case "remember":
			if len(parts) < 3 {
				fmt.Println("Usage: remember <id> <content>")
				continue
			}
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{"id": parts[1], "content": parts[2]}
			res, _ := a.rememberHandler(ctx, req)
			fmt.Println(res.Content[0].(mcp.TextContent).Text)
		case "search":
			if len(parts) < 2 {
				fmt.Println("Usage: search <query>")
				continue
			}
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{"query": strings.TrimPrefix(line, parts[0]+" ")}
			res, _ := a.searchHandler(ctx, req)
			fmt.Println(res.Content[0].(mcp.TextContent).Text)
		}
	}
}