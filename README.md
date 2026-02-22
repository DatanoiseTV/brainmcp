# BrainMCP

BrainMCP is a Model Context Protocol (MCP) server that provides semantic long-term memory for LLMs. It allows AI systems to store information with vector embeddings and retrieve relevant context using natural language search.

Uses the Google Gemini GenAI SDK for high-quality embeddings and chromem-go for a lightweight, local vector database.

## Features

- **Semantic Search**: Retrieve memories based on conceptual meaning rather than keyword matching
- **Optimized Embeddings**: Uses gemini-embedding-001 with Matryoshka Representation Learning (MRL) optimized at 768 dimensions
- **Persistence**: Automatically saves and loads memory state from a local binary file
- **Task-Specific Optimization**: Uses RETRIEVAL_DOCUMENT for storage and RETRIEVAL_QUERY for searching to maximize accuracy
- **LLM-Assisted Synthesis**: Ask questions and receive conversational answers synthesized from stored memories
- **Interactive Test Mode**: Built-in CLI for testing without requiring an MCP client
- **Modular Architecture**: Clean separation of concerns across multiple Go files

## Project Structure

- `main.go` - Application entry point and server initialization
- `constants.go` - Configuration and message constants
- `embedder.go` - Gemini embedding functions and vector normalization
- `handlers.go` - MCP tool handlers for all operations
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

Test the memory system locally:

```bash
make test
```

Available commands:
- `remember <id> <content>` - Store a new memory
- `search <query>` - Search through stored memories
- `ask <question>` - Ask a question and get conversational answers
- `list` - Show all stored memories
- `delete <id>` - Remove a specific memory
- `wipe` - Clear all memories
- `exit` - Close the application

### MCP Server Mode

Run as an MCP server:

```bash
make run
```

The server listens on stdio and provides these tools:
- **remember** - Store memories with semantic vectors
- **search_memory** - Semantic search through memories
- **ask_brain** - LLM-assisted question answering
- **list_memories** - List all stored memories
- **delete_memory** - Remove memories
- **wipe_all_memories** - Clear entire memory database

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

## Implementation Details

The system uses a dual-task embedding strategy:
- Documents are embedded with RETRIEVAL_DOCUMENT task type
- Queries are embedded with RETRIEVAL_QUERY task type

This task-specific optimization ensures that searches find semantically relevant results even when query wording differs significantly from stored content.

## Version

1.3.0 - Modular refactor with improved error handling and documentation


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