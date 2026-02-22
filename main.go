package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	ctx        *ContextManager
	clientID   string // Default client ID for server operations
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
		clientID:  "server",
	}

	// Initialize context manager for persistent contexts and tagging
	contextMgr := NewContextManager(ContextsDataPath)
	app.ctx = contextMgr

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

	// Context management tools
	s.AddTool(mcp.NewTool("create_context",
		mcp.WithDescription("Create a new named context to organize memories by topic or project."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Unique context identifier")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Human-readable context name")),
		mcp.WithString("description", mcp.Description("Optional description of the context")),
	), app.createContextHandler)

	s.AddTool(mcp.NewTool("list_contexts",
		mcp.WithDescription("List all named contexts in the brain."),
	), app.listContextsHandler)

	s.AddTool(mcp.NewTool("switch_context",
		mcp.WithDescription("Switch to a different context for organizing memories."),
		mcp.WithString("context_id", mcp.Required(), mcp.Description("The context ID to switch to")),
		mcp.WithString("client_id", mcp.Description("Optional client ID (uses server default if not provided)")),
	), app.switchContextHandler)

	s.AddTool(mcp.NewTool("share_context",
		mcp.WithDescription("Share a context with another client to enable collaboration."),
		mcp.WithString("context_id", mcp.Required(), mcp.Description("Context to share")),
		mcp.WithString("target_client_id", mcp.Required(), mcp.Description("Client ID to share with")),
	), app.shareContextHandler)

	// Tag management tools
	s.AddTool(mcp.NewTool("add_tag",
		mcp.WithDescription("Add a tag to a memory for categorization."),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("ID of the memory to tag")),
		mcp.WithString("tag", mcp.Required(), mcp.Description("Tag to add")),
	), app.addTagHandler)

	s.AddTool(mcp.NewTool("create_tag",
		mcp.WithDescription("Create a new tag definition for categorization."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Tag name")),
		mcp.WithString("description", mcp.Description("Optional description")),
		mcp.WithString("color", mcp.Description("Optional hex color for UI")),
	), app.createTagHandler)

	s.AddTool(mcp.NewTool("list_tags",
		mcp.WithDescription("List all available tags."),
	), app.listTagsHandler)

	s.AddTool(mcp.NewTool("search_by_tag",
		mcp.WithDescription("Search memories by tag."),
		mcp.WithString("tag", mcp.Required(), mcp.Description("Tag to search for")),
	), app.searchByTagHandler)

	s.AddTool(mcp.NewTool("save_to_disk",
		mcp.WithDescription("Explicitly persist the database and context state to disk."),
	), app.saveToDiskHandler)

	// Setup graceful shutdown on signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server
	logger.Printf("BrainMCP Server starting (version %s) on Stdio...", ServerVersion)
	go func() {
		sig := <-sigChan
		logger.Printf("Received signal %v, gracefully shutting down...", sig)
		app.gracefulShutdown()
		os.Exit(0)
	}()

	if err := server.ServeStdio(s); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}

// gracefulShutdown performs cleanup operations before server exit.
// It saves the database and context state to disk.
func (a *App) gracefulShutdown() {
	a.logger.Println("Saving database to disk...")

	// Save vector database
	if err := a.db.ExportToFile(a.dbPath, true, ""); err != nil {
		a.logger.Printf("Error saving vector database: %v", err)
	} else {
		a.logger.Println("Vector database saved successfully")
	}

	// Save context state
	if err := a.ctx.Save(); err != nil {
		a.logger.Printf("Error saving context state: %v", err)
	} else {
		a.logger.Println("Context state saved successfully")
	}

	a.logger.Println("Shutdown complete")
}
