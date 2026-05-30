# Help CLI — Design Spec

## Overview

A CLI tool (`help`) that provides semantic search over a local collection of help documents (markdown with YAML frontmatter), grouped by tags. Includes a Wails-based GUI for reading and searching docs.

## Architecture

Single Go module, monorepo structure:

```
help/
├── cmd/help/          # CLI entry point
├── embed/             # embedding + reranker (adapted from provenance)
├── index/             # SQLite index (build, query)
├── docs/              # help documents (markdown w/ frontmatter tags)
├── ui/                # Wails app (reader + search)
└── go.mod
```

## Document Format

Markdown files in `docs/` with YAML frontmatter:

```yaml
---
title: "Setting up SSH keys"
tags: [system, ssh]
---

# Setting up SSH keys
...
```

Tags are the category system. No fixed set — grows organically.

## Storage

- **Source of truth:** Markdown files in `docs/`
- **Search index:** SQLite at `~/.cache/help/index.db`
  - `documents` table: id, path, title, tags (JSON array), content, content_hash
  - `chunks` table: id, doc_id, heading, content, start_line, end_line
  - `vec_chunks` virtual table: 256-dim float embeddings keyed by chunk id
- Index is derived/rebuildable. On `help index`, hash each file, skip unchanged.

## Embedding Pipeline

Adapted from `/home/edmund/projects/provenance/cli/embed/`:

- **Embedder:** Model2Vec-quantized Qwen3-Embedding-0.6B (256-dim, ONNX)
- **Reranker:** Jina Reranker (ONNX)
- **Tokenizer:** Qwen2 tokenizer for embedder, custom tokenizer for reranker
- **Runtime:** ONNX Runtime (dynamically downloaded, cached at `~/.cache/help/models/`)
- **Vector search:** sqlite-vec extension

## Search Pipeline

`help search <query> [--tag=...] [--limit=10] [--json]`

1. Embed query with Qwen3 model
2. Vector search `vec_chunks` — top 30 candidates
3. Tag filter (pre-filter on document tags when specified)
4. Rerank with Jina reranker — return top N
5. Output: document title, matched section heading, snippet, relevance score

Output formats: human-readable (default), JSON for programmatic consumers.

## CLI Commands

```
help search <query> [--tag=...] [--limit=10] [--json]
help index [--force]
help list [--tag=...]
help read <doc-path>       # renders markdown to terminal via glamour
help tags                  # list all known tags
```

Models downloaded on first `help index`.

## Wails UI

Desktop app for reading and searching docs.

- **Search view:** Input bar, results as cards (title, tag pills, snippet). Click opens reader.
- **Reader view:** Rendered HTML from markdown (goldmark on Go side), sidebar with doc list filterable by tag.
- **Backend bindings:** `Search(query, tags, limit)`, `ListDocuments(tag)`, `GetDocument(path)`
- **Frontend:** Minimal JS. Markdown rendered to HTML server-side by Go (goldmark). Frontend just displays HTML + handles navigation.

Designed to be toggled via window manager hotkey (future work).

## Future Integration Points (deferred)

- **Hotkey toggle:** WM binding to show/hide the Wails window
- **System bar search:** JSON output from CLI piped to rofi/fuzzel/wofi
- **Foot command insertion:** CLI command mode that outputs a command string; external script sends to foot via wtype or similar
- **External doc ingestion:** man pages, --help output, tldr pages (not in initial scope)

## Dependencies

- `github.com/yalue/onnxruntime_go` — ONNX inference
- `github.com/CharLemAznable/qwen-tokenizer` — Qwen2 tokenization
- `github.com/ncruces/go-sqlite3` — SQLite (pure Go, no CGO)
- `github.com/asg017/sqlite-vec-go-bindings` — vector search
- `github.com/yuin/goldmark` — markdown to HTML
- `github.com/charmbracelet/glamour` — markdown to terminal
- `github.com/wailsapp/wails/v2` — desktop UI
- `github.com/spf13/cobra` or similar — CLI framework
