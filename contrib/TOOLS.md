# BrainMCP — Agent Tools Reference

> **Mandatory for all agents**: BrainMCP is the **central memory, knowledge base, and cross-agent communication layer** for every development task. Before starting any work, check memory for existing context. After completing work, persist what you learned. This ensures continuity across sessions and agents.

---

## Core Principle

Every agent **must** follow this loop:

```
1. CHECK   → Search memory for existing context before starting
2. WORK    → Execute the task
3. PERSIST → Store results, decisions, and discoveries before finishing
```

Never begin a task cold. Never end a task without saving what you learned.
Remember also specifics about a project.

---

## Workflow Checklist

At the **start** of every task:
- [ ] `search_memory` for the current task, component, or feature
- [ ] `ask_brain` for relevant background or prior decisions
- [ ] `list_contexts` to check if a relevant project context exists
- [ ] `switch_context` to the appropriate project context if one exists

At the **end** of every task:
- [ ] `remember` key decisions, solutions, and architectural notes
- [ ] `add_tag` to categorize new memories for future retrieval
- [ ] `save_to_disk` to explicitly persist state

---

## Tool Reference

### Memory Operations

#### `remember`
Store a memory with semantic embeddings for future retrieval.

```
id       – Unique string ID (e.g. "auth-jwt-decision-2024-01")
content  – The knowledge to store (be specific and detailed)
metadata – Optional JSON or tag string for extra context
```

**When to use**: Architectural decisions, bug fixes and their root causes, implementation patterns, configuration choices, API behavior discoveries, anything a future agent would need to know.

**Best practices**:
- Use descriptive IDs that hint at content: `"feature-x-approach"`, `"bug-payment-null-fix"`
- Write content as if explaining to a new developer — include *why*, not just *what*
- Store negative knowledge too: "We tried X and it failed because Y"

---

#### `search_memory`
Semantic similarity search across all stored memories.

```
query – Natural language description of what you're looking for
```

**When to use**: At the start of every task, when encountering an unfamiliar component, when debugging, when making architectural decisions.

**Best practices**:
- Try multiple phrasings if the first search is sparse
- Search for the problem domain, not just the specific symptom
- Search for related components that might have relevant context

---

#### `ask_brain`
Ask a natural language question and receive a synthesized answer from stored memories.

```
question – A direct question about something stored in memory
```

**When to use**: When you need a reasoned answer combining multiple memories, rather than raw search results. Good for "How does X work?", "What was decided about Y?", "Why do we use Z?"

---

#### `list_memories`
List all stored memory IDs with content snippets.

**When to use**: Auditing what's been stored, finding IDs for deletion, getting an overview of accumulated knowledge.

---

#### `delete_memory`
Remove a specific memory by ID.

```
id – The memory ID to delete
```

**When to use**: Removing outdated information that could mislead future agents (e.g., after a major refactor supersedes old decisions).

---

#### `wipe_all_memories`
⚠️ **Destructive.** Clears the entire memory store. Use only when explicitly instructed.

---

### Context Management

Contexts let you organize memories by project, feature, or workstream. Memories are scoped to the active context.

#### `create_context`
Create a new named context.

```
id          – Unique identifier (e.g. "auth-service", "v2-migration")
name        – Human-readable name
description – What this context is for (optional but recommended)
```

---

#### `list_contexts`
List all available contexts. Check this at the start of a task to see what scopes exist.

---

#### `switch_context`
Change the active context for the current session.

```
context_id – The context to switch to
client_id  – Optional; uses server default if omitted
```

**When to use**: When starting work on a specific project or feature. All subsequent `remember` and `search_memory` calls will be scoped to this context.

---

#### `share_context`
Share a context with another client/agent for collaboration.

```
context_id       – The context to share
target_client_id – The other agent's client ID
```

**When to use**: When handing off work to another agent, or enabling multiple agents to collaborate on the same project.

---

### Tag Management

Tags provide cross-context categorization — useful for retrieving memories by type regardless of which project they belong to.

#### `create_tag`
Define a new tag.

```
name        – Tag name (e.g. "decision", "bug-fix", "architecture", "todo")
description – What this tag means
color       – Optional hex color for UI
```

**Recommended standard tags to create on first setup**:
- `decision` — Architectural or design decisions
- `bug-fix` — Documented bug fixes and root causes  
- `architecture` — System design notes
- `todo` — Work flagged for future agents
- `deprecated` — Patterns or code that should no longer be used
- `api` — External API behavior and quirks

---

#### `add_tag`
Tag an existing memory.

```
memory_id – The memory to tag
tag       – Tag name to apply
```

---

#### `list_tags`
List all defined tags.

---

#### `search_by_tag`
Retrieve all memories with a given tag.

```
tag – The tag to search by
```

**When to use**: "Show me all architectural decisions", "Find all known bugs", "What's been flagged as deprecated?"

---

### Persistence

#### `save_to_disk`
Explicitly persist both the vector database (`brain_memory.bin`) and context state (`brain_contexts.json`) to disk.

**When to use**: At the end of any session, after a batch of `remember` calls, before switching tasks. State is also auto-saved on memory/context changes and on server shutdown, but calling this explicitly ensures nothing is lost.

---

## Context Conventions

Use consistent context IDs across agents to ensure shared access:

| Context ID pattern | Use for |
|---|---|
| `project-<name>` | Top-level project or repo |
| `feature-<name>` | A specific feature branch or epic |
| `service-<name>` | A specific microservice or module |
| `sprint-<date>` | Sprint-scoped work |
| `global` | Cross-cutting knowledge with no single owner |

---

## Memory ID Conventions

Use structured IDs so memories are easy to find and manage:

```
<domain>-<topic>-<descriptor>
<domain>-<topic>-<YYYY-MM-DD>

Examples:
  auth-jwt-expiry-decision
  payments-stripe-webhook-bug-fix
  infra-docker-base-image-2024-03
  api-rate-limit-behavior
```

---

## Multi-Agent Handoff Protocol

When handing off work to another agent:

1. `remember` a handoff note with ID `handoff-<feature>-<date>` summarizing:
   - What was completed
   - What remains
   - Any blockers or open questions
   - Relevant memory IDs to review
2. `add_tag` with `todo` on any flagged items
3. `share_context` with the receiving agent's client ID
4. `save_to_disk`

The receiving agent should:
1. `search_memory` for `handoff` to find the note
2. `search_by_tag` with `todo` to find open items
3. Review referenced memory IDs before starting

---

## Quick Reference

```
Start of task:   search_memory → ask_brain → list_contexts → switch_context
During task:     remember (decisions) → add_tag (categorize)
End of task:     remember (results) → save_to_disk
Handoff:         remember (handoff note) → share_context → save_to_disk
Find old work:   search_memory / ask_brain / search_by_tag
```
