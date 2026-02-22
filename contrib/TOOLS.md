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

---

## Context Window Management

BrainMCP is your **external long-term memory**. Use it aggressively to avoid hitting context window limits and to keep conversations focused.

### The Problem

LLMs have finite context windows. Long coding sessions accumulate file contents, error traces, discussion, and decisions until performance degrades or the window fills entirely. BrainMCP solves this by acting as an **offload layer** — you compress and store what you've learned so future turns (or agents) can retrieve just what's needed, rather than repeating everything from scratch.

### Strategy: Offload Early, Retrieve Lazily

**Don't wait until the window is full.** Store continuously:

- After understanding a codebase component → `remember` the key facts about it
- After a decision is made → `remember` it immediately, don't rely on scroll-back
- After resolving a bug → `remember` the root cause and fix
- After reading a long file → `remember` a compressed summary so you never need to re-read it

**Retrieve lazily** — don't load everything upfront. Use `search_memory` to pull in only the facts relevant to the current sub-task.

### What to Offload vs. Keep In-Context

| Keep in context | Offload to BrainMCP |
|---|---|
| The current file being edited | Summaries of files already reviewed |
| The immediate error or test output | Root cause analysis after it's understood |
| The active sub-task instructions | Prior sub-tasks and their outcomes |
| Live function signatures being written | Finalized APIs and their contracts |
| The last 2–3 turns of dialogue | Decisions made earlier in the session |

### Context Window Recovery Pattern

When you sense the context is getting long or cluttered:

1. `remember` a session checkpoint (see Session Checkpointing below)
2. Ask the human to start a fresh conversation
3. In the new conversation, `search_memory` for the checkpoint to resume cleanly

This gives you effectively **unlimited working memory** across arbitrarily long tasks.

---

## Workflow Checklist

At the **start** of every task:
- [ ] `search_memory` for the current task, component, or feature
- [ ] `ask_brain` for relevant background or prior decisions
- [ ] `list_contexts` to check if a relevant project context exists
- [ ] `switch_context` to the appropriate project context if one exists
- [ ] `search_by_tag "todo"` to find any open items from prior sessions

At the **end** of every task:
- [ ] `remember` key decisions, solutions, and architectural notes
- [ ] `add_tag` to categorize new memories for future retrieval
- [ ] `save_to_disk` to explicitly persist state

---

## Programming-Specific Task Patterns

### Codebase Onboarding

When starting work on an unfamiliar codebase or repository:

1. `search_memory` for the project name — check if prior onboarding notes exist
2. If no notes exist, explore the repo and then `remember` the following as separate memories:
   - **Project overview**: purpose, language, main entry points
   - **Architecture**: how major modules/services relate to each other
   - **Data flow**: how data moves through the system end-to-end
   - **Key files**: what each important file does and why
   - **Dev setup**: how to build, run, and test locally
   - **Known gotchas**: anything surprising or non-obvious about the codebase
3. Tag all of these with `architecture` and the project name
4. `save_to_disk`

Next time you (or another agent) touches this repo, onboarding takes seconds instead of re-reading everything.

---

### Code Review

Before reviewing a PR or changeset:

1. `search_memory` for the affected modules to recall their design contracts
2. `ask_brain "What are the known issues or constraints in <module>?"`
3. After the review, `remember` a summary:
   - What was changed and why
   - Any concerns raised
   - Decisions made during review
   - Follow-up work needed
4. Tag with `code-review` and the feature/PR name

---

### Debugging a Bug

1. `search_memory` for the error message, symptom, or affected module
2. `ask_brain "Has this error or something similar been seen before?"`
3. If the bug is novel, work through it and then `remember`:
   - The symptom and how to reproduce it
   - The root cause (be specific — what condition triggered it)
   - The fix applied
   - Any related areas that might have the same issue
4. Tag with `bug-fix` and the affected service/module

This builds a bug knowledge base. Future agents won't waste time re-investigating the same issues.

---

### Implementing a Feature

1. `search_memory` for related features, patterns, or prior decisions that might affect implementation
2. `ask_brain "How is <related system> implemented?"` before writing new code that touches it
3. Before writing, `remember` your implementation plan so you can recover it if the context resets
4. While implementing, `remember` any significant mid-course corrections or discoveries
5. After completing, `remember`:
   - What was built and where it lives
   - Key design decisions made and why
   - How to test or verify it
   - Any known limitations or follow-up work
6. Tag with `feature` and the feature name

---

### Refactoring

Refactoring is high-risk for context loss because it touches many files across many turns.

1. `remember` the refactoring plan before starting (ID: `refactor-<n>-plan`)
2. After each logical step, `remember` what was done and what remains (ID: `refactor-<n>-progress`)
3. `remember` the *old* design before deleting or changing it — you may need to reference it
4. Tag old patterns as `deprecated` so agents know not to replicate them
5. When complete, `remember` a summary of what changed and why, and delete the progress memory

---

### Writing Tests

1. `search_memory` for the module under test to understand its behavior contract
2. `ask_brain "What are the edge cases for <feature>?"` — previous agents may have documented them
3. After writing tests, `remember`:
   - What scenarios are covered
   - What is explicitly *not* tested and why
   - Any flaky behavior or test setup quirks
4. Tag with `testing`

---

### Dependency / Library Research

When evaluating or integrating a third-party library:

1. `search_memory` for the library name — check if it was evaluated before
2. After research, `remember`:
   - What the library does and why it was chosen (or rejected)
   - Version in use and any version-specific gotchas
   - Known bugs, limitations, or workarounds discovered
   - Key usage patterns for this codebase
3. Tag with `dependency`

This prevents re-evaluating the same libraries repeatedly and documents why things were chosen.

---

### API Integration

When integrating an external API:

1. `search_memory` for the API/service name
2. After integration work, `remember`:
   - Authentication method and where credentials are managed
   - Endpoints used and their behavior (especially undocumented quirks)
   - Rate limits, retry behavior, and error handling patterns
   - Any gotchas discovered during integration
3. Tag with `api` and the service name (e.g., `stripe`, `github`, `openai`)

---

### Database / Schema Work

1. `search_memory` for the table or collection before modifying it
2. After schema changes, `remember`:
   - What changed and why
   - Migration approach taken
   - Any data that was backfilled or transformed
   - Indexes added and the queries they support
3. Tag with `database` and `migration` if applicable

---

### Environment & Configuration

1. `remember` environment-specific configuration that isn't obvious from the code:
   - Required env vars and what they control
   - Non-obvious default values
   - Differences between dev/staging/prod setups
   - Secrets management approach
2. Tag with `config` and the environment (e.g., `production`, `staging`)

This is critical information that often only lives in someone's head or a Slack thread.

---

### Performance Optimization

1. `search_memory` for the component being optimized — were prior optimizations attempted?
2. `remember` before and after:
   - Baseline measurements
   - What was changed
   - Results achieved
   - Why the approach was chosen over alternatives tried
3. Tag with `performance`

---

### Security Review

1. `search_memory` for prior security findings in the affected area
2. After a security review or fix, `remember`:
   - The vulnerability class and where it was found
   - The fix applied
   - Any other areas flagged for follow-up
3. Tag with `security`

---

### Long File / Large Codebase Summarization

When a file or set of files is too large to keep in context repeatedly:

1. Read the file fully once
2. `remember` a structured summary (ID: `file-summary-<filename>`):
   - Purpose of the file
   - Key exports, functions, or types defined
   - Dependencies it has on other modules
   - Any important behavior, side effects, or constraints
3. Tag with `architecture`
4. In future turns, load the summary from memory instead of re-reading the file

Use `ask_brain "What does <filename> do?"` to retrieve it naturally.

---

### Session Checkpointing

Use this to safely pause and resume long tasks, or to recover from context window pressure.

**To save a checkpoint:**

```
remember(
  id: "checkpoint-<task>-<date>",
  content: "
    Task: <what you are doing>
    Status: <where you are in the task>
    Completed: <what has been done>
    Remaining: <what still needs to be done>
    Blockers: <anything stuck or unclear>
    Key decisions so far: <list>
    Files modified: <list>
    Next action: <exactly what to do next>
  "
)
add_tag(memory_id: "checkpoint-<task>-<date>", tag: "checkpoint")
save_to_disk()
```

**To resume from a checkpoint:**

```
search_memory("checkpoint <task>")
# or
search_by_tag("checkpoint")
```

Read the checkpoint and continue from "Next action". Delete the checkpoint memory once the task is complete.

---

### Global Context Bootstrap

Use BrainMCP as a **global context** that any new conversation can load instantly. On first setup, or after major project milestones, store a global state memory:

```
remember(
  id: "global-context-<date>",
  content: "
    Projects active: <list with brief descriptions>
    Current focus: <what's being worked on now>
    Tech stack: <languages, frameworks, infra>
    Key conventions: <coding standards, patterns in use>
    Important constraints: <performance budgets, compliance, etc.>
    Agents/tools in use: <what other agents or MCP servers are active>
    Last updated: <date>
  "
)
add_tag(memory_id: "global-context-<date>", tag: "global")
save_to_disk()
```

Any new conversation can start with:
```
search_by_tag("global")
```
...and immediately have the full project picture without any human re-explanation. Update this memory whenever the project state changes significantly.

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
- Prefer one focused memory per concept over one giant memory covering many things — semantic search retrieves relevant chunks better when memories are scoped

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
- Use as a first instinct before reading files — the answer may already be stored

---

#### `ask_brain`
Ask a natural language question and receive a synthesized answer from stored memories.

```
question – A direct question about something stored in memory
```

**When to use**: When you need a reasoned answer combining multiple memories, rather than raw search results. Good for "How does X work?", "What was decided about Y?", "Why do we use Z?", "What do I know about module X?"

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

**When to use**: Removing outdated information that could mislead future agents (e.g., after a major refactor supersedes old decisions). Always replace stale memories rather than leaving them to contradict newer ones.

---

#### `wipe_all_memories`
⚠️ **Destructive.** Clears the entire memory store. Use only when explicitly instructed.

---

### Context Management

Contexts let you organize memories by project, feature, or workstream. Memories are scoped to the active context.

#### `create_context`
```
id          – Unique identifier (e.g. "auth-service", "v2-migration")
name        – Human-readable name
description – What this context is for (optional but recommended)
```

#### `list_contexts`
List all available contexts. Check this at the start of a task to see what scopes exist.

#### `switch_context`
```
context_id – The context to switch to
client_id  – Optional; uses server default if omitted
```

#### `share_context`
```
context_id       – The context to share
target_client_id – The other agent's client ID
```

---

### Tag Management

#### `create_tag`
```
name        – Tag name
description – What this tag means
color       – Optional hex color for UI
```

**Recommended standard tags:**

| Tag | Use for |
|---|---|
| `decision` | Architectural or design decisions |
| `bug-fix` | Documented bug fixes and root causes |
| `architecture` | System design and file structure notes |
| `todo` | Work flagged for future agents |
| `deprecated` | Patterns or code that should no longer be used |
| `api` | External API behavior and quirks |
| `config` | Environment and configuration knowledge |
| `database` | Schema, migration, and query notes |
| `security` | Security findings and fixes |
| `performance` | Optimization work and benchmarks |
| `testing` | Test coverage notes and edge cases |
| `dependency` | Library evaluation and integration notes |
| `feature` | Feature implementation summaries |
| `code-review` | Review outcomes and decisions |
| `checkpoint` | In-progress task checkpoints for resumption |
| `global` | Global project context, loaded at session start |
| `migration` | Data or schema migrations |

#### `add_tag` / `list_tags` / `search_by_tag`
Standard tag operations — see tool reference header for parameters.

---

### Persistence

#### `save_to_disk`
Explicitly persist the vector database and context state to disk. Call at the end of every session and after saving checkpoints.

---

## Context & Memory ID Conventions

**Context IDs:**

| Pattern | Use for |
|---|---|
| `project-<n>` | Top-level project or repo |
| `feature-<n>` | A specific feature branch or epic |
| `service-<n>` | A specific microservice or module |
| `sprint-<date>` | Sprint-scoped work |
| `global` | Cross-cutting knowledge with no single owner |

**Memory IDs:**

```
<domain>-<topic>-<descriptor>
<domain>-<topic>-<YYYY-MM-DD>

Examples:
  auth-jwt-expiry-decision
  payments-stripe-webhook-bug-fix
  infra-docker-base-image-2024-03
  api-rate-limit-behavior
  refactor-user-service-plan
  checkpoint-oauth-migration-2024-06-01
  global-context-2024-06
  file-summary-handlers-go
  db-users-schema-v3
  test-checkout-edge-cases
```

---

## Multi-Agent Handoff Protocol

**Handing off:**
1. `remember` a handoff note (`handoff-<feature>-<date>`) — completed, remaining, blockers, key memory IDs
2. `add_tag` with `todo` on open items
3. `share_context` with the receiving agent
4. `save_to_disk`

**Receiving a handoff:**
1. `search_memory "handoff"` to find the note
2. `search_by_tag "todo"` for open items
3. Review referenced memory IDs before starting

---

## Quick Reference

```
New session start:     search_by_tag("global") → search_memory → switch_context
Codebase onboarding:  search_memory → explore → remember (arch + files + gotchas)
Debugging:            search_memory(error) → ask_brain → remember(root cause + fix)
Feature work:         search_memory → remember(plan) → implement → remember(summary)
Refactoring:          remember(plan) → remember(progress) → tag deprecated → remember(summary)
Large file:           read once → remember(summary) → use ask_brain in future
Long task pause:      remember(checkpoint) → add_tag(checkpoint) → save_to_disk
Resume from pause:    search_by_tag("checkpoint") → read → continue
Handoff:              remember(handoff) → share_context → save_to_disk
End of session:       remember(outcomes) → save_to_disk
```
