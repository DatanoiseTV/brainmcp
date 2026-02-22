# BrainMCP

BrainMCP is a Model Context Protocol (MCP) server that provides semantic long-term memory for LLMs with persistent context management and collaborative features. It allows AI systems to store information with vector embeddings, organize memories by context, and retrieve relevant content using natural language search.

Uses the Google Gemini GenAI SDK for high-quality embeddings and chromem-go for a lightweight, local vector database.

## Features

- **Semantic Search**: Retrieve memories based on conceptual meaning rather than keyword matching
- **Context Management**: Organize memories into named contexts (topics, projects, conversations)
- **Memory Tagging**: Categorize memories with tags for flexible organization
- **Client Collaboration**: Share contexts between different clients for multi-user scenarios
- **Persistent Storage**: Automatically saves and loads memory state from local binary and JSON files
- **Graceful Shutdown**: Saves all state on exit or SIGINT (Ctrl+C)
- **Optimized Embeddings**: Uses gemini-embedding-001 with Matryoshka Representation Learning (MRL) optimized at 768 dimensions
- **Task-Specific Optimization**: Uses RETRIEVAL_DOCUMENT for storage and RETRIEVAL_QUERY for searching
- **LLM-Assisted Synthesis**: Ask questions and receive conversational answers from stored memories
- **Interactive Test CLI**: Built-in terminal interface for testing without needing an MCP client
- **Modular Architecture**: Clean separation of concerns across multiple Go files

## Project Structure

- `main.go` - Application entry point, server initialization, graceful shutdown
- `types.go` - Data structures for contexts, tags, and sessions
- `constants.go` - Configuration and message constants
- `embedder.go` - Gemini embedding functions and vector normalization
- `handlers.go` - Original MCP tool handlers (remember, search, ask, delete, list, wipe)
- `context.go` - Context and tag management with persistence
- `context_handlers.go` - MCP handlers for context and tag operations
- `cli.go` - Interactive test mode CLI
- `Makefile` - Build and development tasks

## Prerequisites

- Go 1.25.1 or higher
- Google Gemini API Key

## Installation

```bash
git clone https://github.com/DatanoiseTV/brainmcp
cd brainmcp
make build
```

## Configuration

Set the Gemini API key as an environment variable:

```bash
export GEMINI_API_KEY="your-api-key-here"
```

Optional flags:
- `-model`: Embedding model (default: gemini-embedding-001)
- `-llm`: LLM model for synthesis (default: gemini-flash-lite-latest)
- `-t`: Run in interactive test mode

## Usage

### Interactive Test Mode

Test the memory system locally with context and tag support:

```bash
make test
```

Available commands in CLI:
- `remember <id> <content>` - Store a new memory in current context
- `search <query>` - Search through stored memories
- `ask <question>` - Ask a question and get conversational answers
- `list` - Show all stored memories
- `delete <id>` - Remove a specific memory
- `tag <memory_id> <tag>` - Add a tag to a memory
- `tags` - List all available tags
- `context list` - Show all contexts
- `context create <id> <name>` - Create a new context
- `context switch <id>` - Switch to a different context
- `save` - Explicitly persist state to disk
- `wipe` - Clear all memories
- `exit` - Close the application (auto-saves)

### MCP Server Mode

Run as an MCP server for use with AI clients:

```bash
make run
```

## MCP Tools Reference

### Memory Operations

**remember** - Store memories with semantic vectors
- `id` (required): Unique ID for this memory
- `content` (required): The text content to remember
- `metadata` (optional): Additional metadata

**search_memory** - Semantic similarity search
- `query` (required): Natural language search query

**ask_brain** - LLM-assisted question answering
- `question` (required): Question to answer from memories

**delete_memory** - Remove a memory by ID
- `id` (required): Memory ID to delete

**list_memories** - List all stored memories with snippets

**wipe_all_memories** - Clear entire brain (use with caution)

### Context Management

**create_context** - Create a new named context
- `id` (required): Unique context identifier
- `name` (required): Human-readable context name
- `description` (optional): Description of the context

**list_contexts** - Show all available contexts

**switch_context** - Change current context for a client
- `context_id` (required): Context ID to switch to
- `client_id` (optional): Client ID (uses server default if not provided)

**share_context** - Share a context with another client
- `context_id` (required): Context to share
- `target_client_id` (required): Client ID to share with

### Tag Management

**create_tag** - Create a new tag definition
- `name` (required): Tag name
- `description` (optional): Tag description
- `color` (optional): Hex color for UI

**add_tag** - Add a tag to a memory
- `memory_id` (required): Memory ID to tag
- `tag` (required): Tag to add

**list_tags** - Show all available tags

**search_by_tag** - Search memories by tag
- `tag` (required): Tag to search for

### Data Persistence

**save_to_disk** - Explicitly persist database and context state to disk

## Persistence

The system maintains two persistent stores:

1. **Vector Database** (`brain_memory.bin`)
   - Stores all memories and their embeddings
   - Auto-persisted on memory operations
   - Saved on graceful shutdown

2. **Context State** (`brain_contexts.json`)
   - Stores all contexts, tags, and client sessions
   - JSON format for human readability
   - Auto-persisted on context changes
   - Saved on graceful shutdown

Both files are automatically saved when:
- A memory is created, updated, or deleted
- A context is created, switched, or shared
- A tag is created or updated
- The `save_to_disk` tool is called
- The server receives SIGINT (Ctrl+C) or SIGTERM

## Development

Build the project:
```bash
make build
```

Format code:
```bash
make format
```

Run linter:
```bash
make lint
```

Clean build artifacts:
```bash
make clean
```

## Architecture Details

### Dual-Task Embeddings
Documents are embedded with RETRIEVAL_DOCUMENT task type while queries use RETRIEVAL_QUERY to maximize semantic matching accuracy.

### Context-Aware Memory
Each memory is tagged with its creation context and client ID, enabling multi-context support and client isolation when needed.

### Session Management
Client sessions are tracked with:
- Client ID for identification
- Current context association
- Last activity timestamp
- Shared context list for collaboration

### Tag Categorization
Tags enable flexible memory organization independent of contexts, allowing memories to be cross-referenced and discovered through multiple classification schemes.

## Version

1.4.0 - Added persistent context management, memory tagging, collaborative sharing, and graceful shutdown with Ctrl+C support



### MCP Mode (Standard)

To use BrainMCP with an MCP client like Claude Desktop, add the server to your configuration file (e.g., ~/Library/Application Support/Claude/claude_desktop_config.json):

```json
{
  "mcpServers": {
    "brainmcp": {
      "command": "/path/to/brainmcp",
      "env": {
        "GEMINI_API_KEY": "your-api-key-here"
      }
    }
  }
}
```

## Tools Provided

### remember
Stores information with semantic vectors for long-term recall.
- id (string, required): A unique identifier for the memory.
- content (string, required): The text content to be remembered.
- metadata (string, optional): Extra tags or JSON data associated with the memory.

### search_memory
Search memory using semantic similarity. Finds relevant concepts even if the wording is different.
- query (string, required): The natural language search query.

## Implementation Details

- Vector Database: Uses chromem-go for in-memory vector storage with file-based persistence (brain_memory.bin).
- Embedding Model: gemini-embedding-001.
- Normalization: Vectors are manually normalized to ensure high-quality cosine similarity results at 768 dimensions.
- Robustness: The search tool dynamically adjusts the number of results requested based on the current size of the document collection to avoid out-of-bounds errors.

## License

MIT