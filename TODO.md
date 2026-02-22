# BrainMCP TODO & Backlog

## Session 3 - Advanced Features (2026-02-22)

### Completed ✓
- [x] Create advanced_types.go with type definitions for versioning, export/import, filtering, batch ops
- [x] Create version_manager.go with MemoryVersionManager implementation
- [x] Create search_filters.go with SearchFilterEngine for advanced filtering
- [x] Create advanced_handlers.go with MCP tool handler stubs
- [x] Fix compilation issues (imports, unused variables)
- [x] Successfully build with all advanced feature modules
- [x] Commit: e2e4dc1 - Add advanced features: versioning, filtering, export/import, batch ops

### Implementation Details

#### Files Created

**advanced_types.go** (90 lines)
- MemoryVersion: Track individual versions with timestamp and author
- MemoryWithHistory: Container for versioned memory with metadata
- ExportData: Structured format for export/import with contexts and tags
- BatchMemoryData: Input format for batch memory creation
- BatchOperationResult: Result tracking for batch operations
- SearchFilter: Advanced search criteria with date ranges, tags, context
- SearchResult: Formatted search result with similarity and metadata

**version_manager.go** (280 lines)
- MemoryVersionManager: Manages version history for all memories
- Methods:
  - AddVersion(): Track new version with change notes
  - GetVersion(id, number): Retrieve specific version
  - GetHistory(id): Get all versions for a memory
  - ExportMemories(): JSON export with metadata
  - ImportMemories(): JSON import with merge strategies
  - BatchCreateMemories(): Bulk create with context
  - BatchDeleteMemories(): Bulk delete with tracking
  - BatchAddTags()/BatchRemoveTags(): Bulk tag operations

**search_filters.go** (220 lines)
- SearchFilterEngine: Advanced search with multiple filter types
- Filtering by:
  - Context ID
  - Tag combinations (AND/OR logic)
  - Date ranges (created, updated)
  - Client/creator ID
  - Result limits
- Methods:
  - FilterMemories(): Apply complex filters
  - SearchByContext(): Context-specific search
  - SearchByDateRange(): Temporal search
  - GetMemoriesByTag()/GetMemoriesByMultipleTags()
  - GetContextStats(): Statistics for a context
  - ValidateFilter(): Filter validation

**advanced_handlers.go** (210 lines)
- MCP tool handler stubs with correct signatures
- Handlers (registered to be added to main.go):
  - exportMemoriesHandler: Export memories with metadata
  - importMemoriesHandler: Import from JSON
  - getMemoryHistoryHandler: Retrieve version history
  - restoreVersionHandler: Restore to previous version
  - searchAdvancedHandler: Advanced filtering search
  - batchOperationsHandler: Bulk operations (create/delete/tag)
  - getContextStatsHandler: Context statistics

### Architecture Overview

```
MemoryVersionManager
├── histories: map[string]*MemoryWithHistory
└── Methods for versioning, export, batch ops

SearchFilterEngine
├── versionMgr: *MemoryVersionManager
├── ctxMgr: *ContextManager
└── Methods for filtering, statistics

Advanced Handlers (MCP Integration)
├── export_memories: Save to JSON
├── import_memories: Load from JSON
├── get_memory_history: Retrieve versions
├── restore_version: Time-travel to old version
├── search_advanced: Complex filtering
├── batch_operations: Bulk CRUD
└── get_context_stats: Analytics
```

### In Progress
- [ ] Integrate MemoryVersionManager into App struct (main.go)
- [ ] Register advanced handlers in main.go
- [ ] Add version tracking to rememberHandler and other memory operations
- [ ] Add advanced CLI commands for versioning and filtering
- [ ] Test roundtrip: create → export → import → verify
- [ ] Test version history and restore functionality
- [ ] Test batch operations and search filters
- [ ] Update version to v1.5.0

### Pending
- [ ] Performance testing with large datasets
- [ ] Persistence of version history to disk
- [ ] Advanced handler implementations with full logic
- [ ] Comprehensive integration tests
- [ ] Error handling improvements

## Session 2 - Context Management & Tagging (2026-02-22)

### Completed ✓
- [x] Plan context management architecture
- [x] Create types.go with Context, Tag, ClientSession structures
- [x] Implement context.go module with ContextManager
- [x] Add persistent context and tag storage (brain_contexts.json)
- [x] Implement context-aware memory storage
- [x] Create context_handlers.go with MCP tools
- [x] Add tagging system for memory categorization
- [x] Implement context sharing between clients
- [x] Add graceful shutdown on Ctrl+C (SIGINT)
- [x] Save to disk on signal or explicit call
- [x] Update CLI with context and tag commands
- [x] Fix compilation errors and build successfully
- [x] Update README with comprehensive documentation
- [x] **FIX**: Load persisted memories BEFORE collection creation
- [x] Update TODO.md with session 2 commits

### Issues Fixed
- **Persistent Memory Loading**: Collections were being created BEFORE loading from disk, causing memories to be lost on restart. Fixed by moving ImportFromFile() before GetOrCreateCollection().

### Key Features Added
- **Context Management**: Create, list, switch named contexts
- **Memory Tagging**: Add tags to memories and search by tag
- **Client Sessions**: Track multiple clients with session state
- **Context Sharing**: Share contexts between different clients
- **Graceful Shutdown**: Auto-save on SIGINT (Ctrl+C)
- **Persistent State**: Both binary (vectors) and JSON (contexts) files
- **CLI Support**: Full tag and context commands in test mode

## Session 1 - Code Refactoring (2026-02-22)

### Completed ✓
- [x] Extract constants and configuration into constants.go
- [x] Refactor into modular files (embedder.go, handlers.go, cli.go)
- [x] Improve error handling with better logging
- [x] Add godoc comments to all exported functions
- [x] Add Makefile for convenient build management
- [x] Update README.md with architecture details
- [x] Build and test - all files compile successfully

## Implementation Details

### Data Persistence
- **brain_memory.bin**: Vector database with all memories and embeddings
- **brain_contexts.json**: All contexts, tags, and client sessions in JSON format

### Context Architecture
```
Context
├── ID: unique identifier
├── Name: human-readable name
├── Description: optional description
├── MemoryCount: number of memories in this context
├── Tags: list of tags used in this context
└── Timestamps: created_at, updated_at

Tag
├── Name: unique tag identifier
├── Description: optional description
├── Color: optional hex color for UI
└── MemoryCount: number of memories with this tag

ClientSession
├── ClientID: unique client identifier
├── CurrentContext: active context for this client
├── SharedWith: list of shared context IDs
└── Timestamps: created_at, last_activity
```

### Memory Metadata
Each memory stores:
- context: which context it belongs to
- client: which client created it
- tags: array of applied tags
- timestamps: creation and update times

## Future Improvements
- [ ] Unit tests with proper test coverage
- [ ] Integration tests for MCP protocol
- [ ] Batch memory operations
- [ ] Performance optimization for large memory sets
- [ ] Memory export/import functionality
- [ ] Archive old contexts
- [ ] Search filters by date range, context, client
- [ ] Memory versioning/history
- [ ] Backup and restore functionality
- [ ] Analytics dashboard
- [ ] Memory consolidation/summarization
- [ ] Automatic tag suggestions
- [ ] Access control lists (ACLs) for shared contexts

## Git Commits - Session 2

Session 2 focused on adding context management, tagging, client collaboration, and fixing persistence:

- df9b0ca: feat: add context management and persistence
  - Add types.go with Context, Tag, ClientSession structures
  - Implement ContextManager in context.go
  - Add JSON persistence for contexts and tags

- 0624d71: feat: add graceful shutdown and context integration
  - Signal handling for SIGINT and SIGTERM
  - Implement gracefulShutdown() method
  - Register save_to_disk MCP tool

- 4338a26: feat: enhance handlers with context and persistence tracking
  - Update memory operations to track contexts
  - Add automatic memory count updates
  - Persist context state after operations

- 8ad3144: feat: add CLI support for context and tag management
  - Add interactive CLI commands for contexts and tags
  - Update help message and command parsing
  - Support graceful shutdown from exit command

- 6f6090d: fix: load persisted memories before collection creation
  - **CRITICAL FIX**: Move db.ImportFromFile() BEFORE GetOrCreateCollection()
  - Ensures memories are restored on server restart
  - Add success logging for loaded memories

## Git Commits - Session 1

- 29c6019: refactor: extract modular file structure
- 1ae1da3: refactor: simplify main.go and improve initialization
- 579fdb4: build: add Makefile and improve .gitignore
- 024f34e: docs: update README with modular architecture details

## Build & Run
```bash
make build          # Compile the application
export GEMINI_API_KEY="your-key"
./brainmcp -t       # Run interactive test mode
./brainmcp          # Run as MCP server

# In test mode, try:
# > remember memo1 "Hello World"
# > context list
# > context create project1 "My Project"
# > context switch project1
# > tag memo1 important
# > tags
# > save
# > exit
```
