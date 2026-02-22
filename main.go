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

	s := server.NewMCPServer("brain-mcp", "1.1.0")

	// --- Tool Registration ---

	s.AddTool(mcp.NewTool("remember",
		mcp.WithDescription("Stores or updates information with semantic vectors for long-term recall."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Unique ID for this memory")),
		mcp.WithString("content", mcp.Required(), mcp.Description("The text content to remember")),
		mcp.WithString("metadata", mcp.Description("Optional metadata")),
	), app.rememberHandler)

	s.AddTool(mcp.NewTool("search_memory",
		mcp.WithDescription("Search memory using semantic similarity."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Natural language search query")),
	), app.searchHandler)

	s.AddTool(mcp.NewTool("delete_memory",
		mcp.WithDescription("Removes a specific memory from the brain by its ID."),
		mcp.WithString("id", mcp.Required(), mcp.Description("The unique ID of the memory to delete")),
	), app.deleteHandler)

	s.AddTool(mcp.NewTool("list_memories",
		mcp.WithDescription("Returns a list of all stored memory IDs and a snippet of their content."),
	), app.listHandler)

	s.AddTool(mcp.NewTool("wipe_all_memories",
		mcp.WithDescription("Completely clears the brain. Use with caution."),
	), app.wipeHandler)

	log.Println("BrainMCP Server starting on Stdio...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// --- SDK Embedder ---

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
			return nil, err
		}
		if len(res.Embeddings) == 0 {
			return nil, fmt.Errorf("no embeddings returned")
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

// --- Handlers ---

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
	totalDocs := a.collection.Count()
	if totalDocs == 0 {
		return mcp.NewToolResultText("Brain is empty."), nil
	}
	nResults := 5
	if totalDocs < nResults {
		nResults = totalDocs
	}
	results, err := a.collection.Query(ctx, "QUERY_TASK:"+query, nResults, nil, nil)
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

func (a *App) deleteHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, _ := request.Params.Arguments.(map[string]any)
	id, _ := args["id"].(string)

	// FIX: Passing id as a single string instead of []string{id}
	err := a.collection.Delete(ctx, nil, nil, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Delete failed: %v", err)), nil
	}
	_ = a.db.ExportToFile(a.dbPath, true, "")
	return mcp.NewToolResultText(fmt.Sprintf("Memory '%s' deleted.", id)), nil
}

func (a *App) listHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	count := a.collection.Count()
	if count == 0 {
		return mcp.NewToolResultText("No memories stored."), nil
	}

	results, err := a.collection.Query(ctx, " ", count, nil, nil)
	if err != nil {
		return mcp.NewToolResultError("Could not retrieve memory list"), nil
	}
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Brain contains %d memories:\n", count))
	for _, res := range results {
		snippet := res.Content
		if len(snippet) > 50 {
			snippet = snippet[:47] + "..."
		}
		sb.WriteString(fmt.Sprintf("- %s: %s\n", res.ID, snippet))
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (a *App) wipeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	a.db.DeleteCollection("brain_memory")
	embFunc := a.makeGeminiEmbedder()
	col, _ := a.db.GetOrCreateCollection("brain_memory", nil, embFunc)
	a.collection = col
	os.Remove(a.dbPath)
	return mcp.NewToolResultText("Brain completely wiped and reset."), nil
}

// --- Interactive CLI ---

func (a *App) runInteractiveCLI(ctx context.Context) {
	fmt.Println("=== BrainMCP Test Mode ===")
	fmt.Println("Commands: remember <id> <msg> | search <q> | delete <id> | list | wipe | exit")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nbrain> ")
		if !scanner.Scan() { break }
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 0 { continue }

		cmd := strings.ToLower(parts[0])
		switch cmd {
		case "exit": return
		case "list":
			res, _ := a.listHandler(ctx, mcp.CallToolRequest{})
			fmt.Println(res.Content[0].(mcp.TextContent).Text)
		case "wipe":
			res, _ := a.wipeHandler(ctx, mcp.CallToolRequest{})
			fmt.Println(res.Content[0].(mcp.TextContent).Text)
		case "remember":
			if len(parts) < 3 { continue }
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{"id": parts[1], "content": strings.Join(parts[2:], " ")}
			res, _ := a.rememberHandler(ctx, req)
			fmt.Println(res.Content[0].(mcp.TextContent).Text)
		case "search":
			if len(parts) < 2 { continue }
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{"query": strings.Join(parts[1:], " ")}
			res, _ := a.searchHandler(ctx, req)
			fmt.Println(res.Content[0].(mcp.TextContent).Text)
		case "delete":
			if len(parts) < 2 { continue }
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{"id": parts[1]}
			res, _ := a.deleteHandler(ctx, req)
			fmt.Println(res.Content[0].(mcp.TextContent).Text)
		}
	}
}