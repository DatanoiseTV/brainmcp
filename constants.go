package main

// Embedding and model configuration constants
const (
	// Embedding model for generating vector representations
	DefaultEmbeddingModel = "gemini-embedding-001"
	// LLM model for assisted search and synthesis
	DefaultLLMModel = "gemini-flash-lite-latest"
	// Output dimensionality for embeddings (MRL optimized)
	EmbeddingDimension = 768
)

// Memory storage constants
const (
	// Default path for persisted memory database
	DefaultDBPath = "brain_memory.bin"
	// Collection name in the vector database
	CollectionName = "brain_memory"
)

// Embedding task type constants
const (
	// Task type for storing documents
	TaskTypeDocument = "RETRIEVAL_DOCUMENT"
	// Task type for querying
	TaskTypeQuery = "RETRIEVAL_QUERY"
	// Prefix to mark query tasks in the embedding function
	QueryTaskPrefix = "QUERY_TASK:"
)

// Search and retrieval constants
const (
	// Default number of results to return from semantic search
	DefaultSearchResults = 5
	// Maximum snippet length in list output
	MaxSnippetLength = 50
)

// Server configuration constants
const (
	// MCP server name
	ServerName = "brain-mcp"
	// Server version following semantic versioning
	ServerVersion = "1.3.0"
)

// UI/CLI messages
const (
	PrompStr = "brain> "
	WelcomeMsg = "=== BrainMCP Test Mode ==="
	HelpMsg = "Commands: remember <id> <msg> | search <q> | ask <q> | delete <id> | list | wipe | exit"
	UnknownCmdMsg = "Unknown command. Try: remember, search, ask, delete, list, wipe, exit"
)

// Error and status messages
const (
	NoMemoriesMsg = "I don't have any memories yet to answer that."
	EmptyBrainMsg = "Brain is empty."
	NoMemoriesStoredMsg = "No memories stored."
	BrainWipedMsg = "Brain completely wiped and reset."
)
