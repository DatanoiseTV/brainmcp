# BrainMCP

BrainMCP is a Model Context Protocol (MCP) server that provides semantic long-term memory for LLMs. It allows AI models to store information with vector embeddings and retrieve relevant context later using natural language search.

It uses the Google Gemini GenAI SDK for high-quality embeddings and chromem-go for a lightweight, local vector database.

## Features

- Semantic Search: Retrieve memories based on conceptual meaning rather than keyword matching.
- Optimized Embeddings: Uses the gemini-embedding-001 model with Matryoshka Representation Learning (MRL) optimized at 768 dimensions.
- Persistence: Automatically saves and loads your memory state from a local binary file.
- Task-Specific Optimization: Uses RETRIEVAL_DOCUMENT for storage and RETRIEVAL_QUERY for searching to maximize accuracy.
- Interactive Test Mode: A built-in CLI to verify embeddings and search results without an MCP client.

## Prerequisites

- Go 1.22 or higher.
- A Google Gemini API Key.

## Installation

```bash
git clone https://github.com/DatanoiseTV/brainmcp
cd brainmcp
go build -o brainmcp main.go
```

## Configuration

The server requires a Gemini API key. Set it as an environment variable:

```bash
export GEMINI_API_KEY="your-api-key-here"
```

## Usage

### Interactive Test Mode

You can run the server in a local terminal to test its memory capabilities manually:

```bash
./brainmcp -t
```

Commands in test mode:
- remember [id] [content]: Store a new memory.
- search [query]: Search through stored memories.
- exit: Close the application.

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