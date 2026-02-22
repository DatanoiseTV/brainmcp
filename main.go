package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/philippgille/chromem-go"
	"google.golang.org/genai"
)

// App encapsulates the BrainMCP server state and dependencies.
type App struct {
	db         *chromem.DB
	collection *chromem.Collection
	client     *genai.Client
	dbPath     string
	testMode   bool
	modelName  string
	llmModel   string
	logger     *log.Logger
}

func main() {
	testMode := flag.Bool("t", false, "Run in interactive CLI test mode")
	modelFlag := flag.String("model", DefaultEmbeddingModel, "Gemini embedding model")
	llmFlag := flag.String("llm", DefaultLLMModel, "Gemini model for assisted search")
	flag.Parse()

	ctx := context.Background()

	// Initialize logger
	logger := log.New(os.Stderr, "[BrainMCP] ", log.LstdFlags|log.Lshortfile)

	// Validate Gemini API key
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		logger.Fatal("GEMINI_API_KEY environment variable is required")
	}

	// Initialize Gemini client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: geminiKey,
	})
	if err != nil {
		logger.Fatalf("Failed to create GenAI client: %v", err)
	}

	// Initialize vector database
	db := chromem.NewDB()
	app := &App{
		db:        db,
		client:    client,
		dbPath:    DefaultDBPath,
		testMode:  *testMode,
		modelName: *modelFlag,
		llmModel:  *llmFlag,
		logger:    logger,
	}

	// Create embedding function
	embFunc := app.makeGeminiEmbedder()

	// Initialize or retrieve collection
	col, err := db.GetOrCreateCollection(CollectionName, nil, embFunc)
	if err != nil {
		logger.Fatalf("Failed to create collection: %v", err)
	}
	app.collection = col

	// Load persisted memories if they exist
	if info, err := os.Stat(app.dbPath); err == nil && info.Size() > 0 {
		if err := db.ImportFromFile(app.dbPath, ""); err != nil {
			if *testMode {
				fmt.Printf("Note: Started fresh (DB import failed: %v)\n", err)
			} else {
				logger.Printf("Warning: Failed to load persisted memories: %v", err)
			}
		}
	}

	// Run in appropriate mode
	if *testMode {
		app.runInteractiveCLI(ctx)
		return
	}

	// Initialize MCP server
	s := server.NewMCPServer(ServerName, ServerVersion)

	// Register all tools
	s.AddTool(mcp.NewTool("remember",
		mcp.WithDescription("Stores or updates information with semantic vectors for long-term recall."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Unique ID for this memory")),
		mcp.WithString("content", mcp.Required(), mcp.Description("The text content to remember")),
		mcp.WithString("metadata", mcp.Description("Optional metadata")),
	), app.rememberHandler)

	s.AddTool(mcp.NewTool("search_memory",
		mcp.WithDescription("Search memory using semantic similarity. Returns raw snippets."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Natural language search query")),
	), app.searchHandler)

	s.AddTool(mcp.NewTool("ask_brain",
		mcp.WithDescription("LLM-assisted search. Processes your question, searches memory, and provides a conversational answer based on found facts."),
		mcp.WithString("question", mcp.Required(), mcp.Description("The question you want to ask your memory")),
	), app.askBrainHandler)

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

	// Start server
	logger.Printf("BrainMCP Server starting (version %s) on Stdio...", ServerVersion)
	if err := server.ServeStdio(s); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}
