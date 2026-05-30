# Help CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a CLI (`help`) that provides semantic search over local markdown help documents, with a Wails desktop reader/viewer.

**Architecture:** Go monorepo. Markdown files with YAML frontmatter are the source of truth. SQLite + sqlite-vec stores the search index (embeddings + metadata). Qwen3 Model2Vec embedder + Jina reranker via ONNX Runtime (adapted from provenance project). Wails v2 desktop app renders markdown to HTML server-side via goldmark.

**Tech Stack:** Go 1.25+, ONNX Runtime, sqlite-vec, goldmark, glamour, cobra, Wails v2, vanilla JS frontend, base16 color theming.

**Reference code:**
- Embed pipeline: `/home/edmund/projects/provenance/cli/embed/`
- DB/search patterns: `/home/edmund/projects/provenance/cli/db/db.go`
- Wails structure: `/home/edmund/brightworker/charon/charon-ui/`

---

## File Structure

```
help/
├── cmd/help/main.go           # CLI entry point (cobra root command)
├── cmd/help/search.go         # search subcommand
├── cmd/help/index.go          # index subcommand
├── cmd/help/list.go           # list subcommand
├── cmd/help/read.go           # read subcommand (terminal)
├── cmd/help/tags.go           # tags subcommand
├── embed/embed.go             # model paths, cache, setup
├── embed/tokenize.go          # qwen2 tokenizer
├── embed/inference.go         # ONNX embedding generation
├── embed/download.go          # ONNX runtime download
├── embed/rerank.go            # jina reranker
├── embed/models/model.onnx    # qwen3 embedding model (copy from provenance)
├── embed/models/jina-reranker/model.onnx       # reranker model
├── embed/models/jina-reranker/tokenizer.json   # reranker tokenizer
├── index/db.go                # SQLite schema, open/close
├── index/build.go             # indexing: parse docs, chunk, embed, store
├── index/search.go            # query: embed, vector search, rerank
├── index/parse.go             # markdown frontmatter parsing, chunking
├── index/parse_test.go        # parser tests
├── index/search_test.go       # search integration tests
├── docs/                      # help documents (user content)
├── docs/example.md            # sample doc for testing
├── ui/main.go                 # Wails app entry point
├── ui/app.go                  # Wails backend bindings
├── ui/frontend/index.html     # single HTML page
├── ui/frontend/app.js         # minimal JS: navigation, search input
├── ui/frontend/style.css      # base16 themed styles
├── ui/frontend/build.mjs      # esbuild config (trivial)
├── ui/frontend/package.json   # frontend deps (none or near-none)
├── ui/wails.json              # Wails config
├── config/config.go           # configuration (docs dir, theme, db path)
├── go.mod
├── go.sum
└── Makefile                   # build targets
```

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `cmd/help/main.go`
- Create: `config/config.go`
- Create: `docs/example.md`

- [ ] **Step 1: Initialize Go module**

```bash
cd /home/edmund/wonderinstruments/help
go mod init wonderinstruments.com/help
```

- [ ] **Step 2: Create Makefile**

Create `Makefile`:
```makefile
.PHONY: build run index search ui

build:
	go build -o bin/help ./cmd/help

run:
	go run ./cmd/help

index:
	go run ./cmd/help index

search:
	go run ./cmd/help search $(QUERY)

ui:
	cd ui && wails dev
```

- [ ] **Step 3: Create config package**

Create `config/config.go`:
```go
package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	DocsDir  string
	CacheDir string
	DBPath   string
	Theme    string // base16 theme name
}

func Default() Config {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", "help")
	return Config{
		DocsDir:  filepath.Join(home, "wonderinstruments", "help", "docs"),
		CacheDir: cacheDir,
		DBPath:   filepath.Join(cacheDir, "index.db"),
		Theme:    "default-dark",
	}
}

func (c Config) EnsureCacheDir() error {
	return os.MkdirAll(c.CacheDir, 0755)
}
```

- [ ] **Step 4: Create CLI entry point with cobra**

Create `cmd/help/main.go`:
```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "help",
	Short: "Semantic search over help documents",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Create example document**

Create `docs/example.md`:
```markdown
---
title: "Git Rebase Workflow"
tags: [cli, git]
---

# Git Rebase Workflow

## Interactive Rebase

To rewrite recent commits:

    git rebase -i HEAD~3

This opens an editor where you can reorder, squash, or edit commits.

## Rebase onto main

To bring your branch up to date:

    git fetch origin
    git rebase origin/main

If conflicts arise, resolve them and run:

    git rebase --continue
```

- [ ] **Step 6: Add cobra dependency and verify build**

```bash
cd /home/edmund/wonderinstruments/help
go get github.com/spf13/cobra
go build ./cmd/help
```

- [ ] **Step 7: Commit**

```bash
git add .
git commit -m "feat: scaffold help CLI with config and cobra"
```

---

## Task 2: Embed Package (adapted from provenance)

**Files:**
- Create: `embed/embed.go`
- Create: `embed/tokenize.go`
- Create: `embed/inference.go`
- Create: `embed/download.go`
- Create: `embed/rerank.go`
- Copy: `embed/models/` (from provenance)

- [ ] **Step 1: Copy model files from provenance**

```bash
mkdir -p /home/edmund/wonderinstruments/help/embed/models/jina-reranker
cp /home/edmund/projects/provenance/cli/embed/models/model.onnx /home/edmund/wonderinstruments/help/embed/models/
cp /home/edmund/projects/provenance/cli/embed/models/jina-reranker/model.onnx /home/edmund/wonderinstruments/help/embed/models/jina-reranker/
cp /home/edmund/projects/provenance/cli/embed/models/jina-reranker/tokenizer.json /home/edmund/wonderinstruments/help/embed/models/jina-reranker/
```

- [ ] **Step 2: Create embed/embed.go**

```go
package embed

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

const EmbeddingDim = 256

//go:embed models/model.onnx
var embeddedModel []byte

func GetCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".cache", "help", "models"), nil
}

func GetModelPath() (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "model.onnx"), nil
}

func EnsureModel() error {
	modelPath, err := GetModelPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(modelPath); err == nil {
		return nil
	}
	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	if err := os.WriteFile(modelPath, embeddedModel, 0644); err != nil {
		return fmt.Errorf("failed to write model file: %w", err)
	}
	return nil
}

func RuntimeReady() (bool, error) {
	libPath, err := GetONNXRuntimeLibPath()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return false, nil
	}
	modelPath, err := GetModelPath()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return false, nil
	}
	return true, nil
}
```

- [ ] **Step 3: Create embed/tokenize.go**

```go
package embed

import (
	"fmt"

	"github.com/CharLemAznable/qwen-tokenizer"
)

var globalTokenizer *tokenizer.Tokenizer

func InitTokenizer() error {
	globalTokenizer = &tokenizer.Tokenizer{}
	return nil
}

func Tokenize(text string) ([]int64, error) {
	if globalTokenizer == nil {
		return nil, fmt.Errorf("tokenizer not initialized")
	}
	ids := globalTokenizer.EncodeOrdinary(text)
	result := make([]int64, len(ids))
	for i, id := range ids {
		result[i] = int64(id)
	}
	return result, nil
}
```

- [ ] **Step 4: Create embed/inference.go**

```go
package embed

import (
	"encoding/binary"
	"fmt"
	"math"

	ort "github.com/yalue/onnxruntime_go"
)

var ortInitialized bool

func InitONNX() error {
	if ortInitialized {
		return nil
	}
	libPath, err := GetONNXRuntimeLibPath()
	if err != nil {
		return err
	}
	ort.SetSharedLibraryPath(libPath)
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("failed to initialize ONNX environment: %w", err)
	}
	ortInitialized = true
	return nil
}

func GenerateEmbedding(text string) ([]float32, error) {
	tokenIDs, err := Tokenize(text)
	if err != nil {
		return nil, err
	}
	modelPath, err := GetModelPath()
	if err != nil {
		return nil, err
	}

	numTokens := int64(len(tokenIDs))
	inputIDsShape := ort.NewShape(numTokens)
	offsetsShape := ort.NewShape(1)

	inputIDsTensor, err := ort.NewTensor(inputIDsShape, tokenIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	offsets := []int64{0}
	offsetsTensor, err := ort.NewTensor(offsetsShape, offsets)
	if err != nil {
		return nil, fmt.Errorf("failed to create offsets tensor: %w", err)
	}
	defer offsetsTensor.Destroy()

	outputShape := ort.NewShape(1, int64(EmbeddingDim))
	outputData := make([]byte, outputShape.FlattenedSize()*2)
	outputTensor, err := ort.NewCustomDataTensor(outputShape, outputData, ort.TensorElementDataTypeFloat16)
	if err != nil {
		return nil, fmt.Errorf("failed to create output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	session, err := ort.NewAdvancedSession(
		modelPath,
		[]string{"input_ids", "offsets"},
		[]string{"embeddings"},
		[]ort.Value{inputIDsTensor, offsetsTensor},
		[]ort.Value{outputTensor},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Destroy()

	if err := session.Run(); err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	return float16ToFloat32(outputData), nil
}

func float16ToFloat32(data []byte) []float32 {
	result := make([]float32, len(data)/2)
	for i := 0; i < len(result); i++ {
		bits := binary.LittleEndian.Uint16(data[i*2 : i*2+2])
		result[i] = float16BitsToFloat32(bits)
	}
	return result
}

func float16BitsToFloat32(h uint16) float32 {
	sign := uint32((h >> 15) & 0x1)
	exp := uint32((h >> 10) & 0x1f)
	mant := uint32(h & 0x3ff)

	var f uint32
	if exp == 0 {
		if mant == 0 {
			f = sign << 31
		} else {
			exp = 1
			for (mant & 0x400) == 0 {
				mant <<= 1
				exp--
			}
			mant &= 0x3ff
			f = (sign << 31) | ((exp + 127 - 15) << 23) | (mant << 13)
		}
	} else if exp == 31 {
		f = (sign << 31) | (0xff << 23) | (mant << 13)
	} else {
		f = (sign << 31) | ((exp + 127 - 15) << 23) | (mant << 13)
	}

	return math.Float32frombits(f)
}

func CleanupONNX() {
	if ortInitialized {
		ort.DestroyEnvironment()
		ortInitialized = false
	}
}
```

- [ ] **Step 5: Create embed/download.go**

```go
package embed

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const ONNXRuntimeVersion = "1.24.1"

func GetONNXRuntimeURL() (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var platform string
	switch {
	case goos == "linux" && goarch == "amd64":
		platform = "linux-x64"
	case goos == "linux" && goarch == "arm64":
		platform = "linux-aarch64"
	case goos == "darwin" && goarch == "amd64":
		platform = "osx-x86_64"
	case goos == "darwin" && goarch == "arm64":
		platform = "osx-arm64"
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", goos, goarch)
	}

	return fmt.Sprintf(
		"https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-%s-%s.tgz",
		ONNXRuntimeVersion, platform, ONNXRuntimeVersion,
	), nil
}

func GetONNXRuntimeLibPath() (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}

	var libName string
	switch runtime.GOOS {
	case "linux":
		libName = "libonnxruntime.so"
	case "darwin":
		libName = "libonnxruntime.dylib"
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	return filepath.Join(cacheDir, libName), nil
}

func DownloadONNXRuntime() error {
	libPath, err := GetONNXRuntimeLibPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(libPath); err == nil {
		return nil
	}

	url, err := GetONNXRuntimeURL()
	if err != nil {
		return err
	}

	fmt.Println("Downloading ONNX Runtime...")
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		isVersionedLib := (strings.Contains(header.Name, "libonnxruntime.so.") ||
			strings.Contains(header.Name, "libonnxruntime.") && strings.HasSuffix(header.Name, ".dylib")) &&
			header.Typeflag == tar.TypeReg && header.Size > 0

		if isVersionedLib {
			out, err := os.Create(libPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			break
		}
	}

	return nil
}

func SetupRuntime() error {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	if err := DownloadONNXRuntime(); err != nil {
		return fmt.Errorf("failed to download ONNX runtime: %w", err)
	}
	if err := EnsureModel(); err != nil {
		return fmt.Errorf("failed to extract model: %w", err)
	}
	return nil
}
```

- [ ] **Step 6: Create embed/rerank.go**

```go
package embed

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

const RerankMaxLength = 128

//go:embed models/jina-reranker/model.onnx
var embeddedRerankerModel []byte

//go:embed models/jina-reranker/tokenizer.json
var embeddedRerankerTokenizer []byte

var rerankerTok *tokenizer.Tokenizer

func GetRerankerModelPath() (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "jina-reranker.onnx"), nil
}

func getRerankerTokenizerPath() (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "jina-reranker-tokenizer.json"), nil
}

func EnsureRerankerModel() error {
	modelPath, err := GetRerankerModelPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(modelPath); err == nil {
		return nil
	}
	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	if err := os.WriteFile(modelPath, embeddedRerankerModel, 0644); err != nil {
		return fmt.Errorf("failed to write reranker model: %w", err)
	}
	return nil
}

func ensureRerankerTokenizerFile() (string, error) {
	tokPath, err := getRerankerTokenizerPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(tokPath); err == nil {
		return tokPath, nil
	}
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}
	if err := os.WriteFile(tokPath, embeddedRerankerTokenizer, 0644); err != nil {
		return "", fmt.Errorf("failed to write reranker tokenizer: %w", err)
	}
	return tokPath, nil
}

func InitRerankerTokenizer() error {
	if rerankerTok != nil {
		return nil
	}
	tokPath, err := ensureRerankerTokenizerFile()
	if err != nil {
		return err
	}
	tok, err := pretrained.FromFile(tokPath)
	if err != nil {
		return fmt.Errorf("failed to load reranker tokenizer from %s: %w", tokPath, err)
	}
	rerankerTok = tok
	return nil
}

func CleanupRerankerTokenizer() {
	rerankerTok = nil
}

type RerankResult struct {
	Index int
	Score float32
}

func Rerank(query string, documents []string) ([]float32, error) {
	if len(documents) == 0 {
		return nil, nil
	}
	if rerankerTok == nil {
		return nil, fmt.Errorf("reranker tokenizer not initialized")
	}

	modelPath, err := GetRerankerModelPath()
	if err != nil {
		return nil, err
	}

	batchSize := len(documents)
	inputIDs := make([]int64, 0, batchSize*RerankMaxLength)
	attentionMask := make([]int64, 0, batchSize*RerankMaxLength)

	for _, doc := range documents {
		ids, mask, err := tokenizeQueryDocPair(query, doc, RerankMaxLength)
		if err != nil {
			return nil, fmt.Errorf("failed to tokenize pair: %w", err)
		}
		inputIDs = append(inputIDs, ids...)
		attentionMask = append(attentionMask, mask...)
	}

	inputShape := ort.NewShape(int64(batchSize), int64(RerankMaxLength))

	inputIDsTensor, err := ort.NewTensor(inputShape, inputIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	attMaskTensor, err := ort.NewTensor(inputShape, attentionMask)
	if err != nil {
		return nil, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	defer attMaskTensor.Destroy()

	outputShape := ort.NewShape(int64(batchSize), 1)
	outputData := make([]float32, batchSize)
	outputTensor, err := ort.NewTensor(outputShape, outputData)
	if err != nil {
		return nil, fmt.Errorf("failed to create output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	session, err := ort.NewAdvancedSession(
		modelPath,
		[]string{"input_ids", "attention_mask"},
		[]string{"logits"},
		[]ort.Value{inputIDsTensor, attMaskTensor},
		[]ort.Value{outputTensor},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reranker session: %w", err)
	}
	defer session.Destroy()

	if err := session.Run(); err != nil {
		return nil, fmt.Errorf("reranker inference failed: %w", err)
	}

	return outputData, nil
}

func tokenizeQueryDocPair(query, doc string, maxLen int) ([]int64, []int64, error) {
	encoding, err := rerankerTok.EncodePair(query, doc, true)
	if err != nil {
		return nil, nil, err
	}

	ids := encoding.GetIds()
	if len(ids) > maxLen {
		ids = ids[:maxLen]
	}

	attMask := make([]int64, maxLen)
	for i := 0; i < len(ids); i++ {
		attMask[i] = 1
	}

	paddedIDs := make([]int64, maxLen)
	for i, id := range ids {
		paddedIDs[i] = int64(id)
	}
	for i := len(ids); i < maxLen; i++ {
		paddedIDs[i] = 1 // <pad>
	}

	return paddedIDs, attMask, nil
}

func RerankerReady() (bool, error) {
	libPath, err := GetONNXRuntimeLibPath()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return false, nil
	}
	modelPath, err := GetRerankerModelPath()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return false, nil
	}
	return true, nil
}
```

- [ ] **Step 7: Add dependencies**

```bash
cd /home/edmund/wonderinstruments/help
go get github.com/yalue/onnxruntime_go
go get github.com/CharLemAznable/qwen-tokenizer
go get github.com/sugarme/tokenizer
```

- [ ] **Step 8: Verify embed package compiles**

```bash
go build ./embed/
```

- [ ] **Step 9: Commit**

```bash
git add embed/
git commit -m "feat: add embed package (qwen3 embedder + jina reranker)"
```

---

## Task 3: Index Package — Schema and Parsing

**Files:**
- Create: `index/db.go`
- Create: `index/parse.go`
- Create: `index/parse_test.go`

- [ ] **Step 1: Write parse_test.go**

```go
package index

import "testing"

func TestParseFrontmatter(t *testing.T) {
	content := `---
title: "SSH Keys"
tags: [system, ssh]
---

# SSH Keys

## Generating a key

Run ssh-keygen to create a new key pair.

## Adding to agent

Use ssh-add to register the key.
`
	doc, err := ParseDocument("docs/ssh.md", []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "SSH Keys" {
		t.Errorf("got title %q", doc.Title)
	}
	if len(doc.Tags) != 2 || doc.Tags[0] != "system" || doc.Tags[1] != "ssh" {
		t.Errorf("got tags %v", doc.Tags)
	}
	if len(doc.Chunks) != 3 {
		t.Errorf("got %d chunks, want 3", len(doc.Chunks))
	}
	if doc.Chunks[0].Heading == "" {
		// First chunk is the intro (title heading)
	}
	if doc.Chunks[1].Heading != "Generating a key" {
		t.Errorf("chunk 1 heading: %q", doc.Chunks[1].Heading)
	}
}

func TestParseFrontmatterNoTags(t *testing.T) {
	content := `---
title: "Simple Doc"
---

Just some content.
`
	doc, err := ParseDocument("test.md", []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "Simple Doc" {
		t.Errorf("got title %q", doc.Title)
	}
	if len(doc.Tags) != 0 {
		t.Errorf("got tags %v", doc.Tags)
	}
	if len(doc.Chunks) != 1 {
		t.Errorf("got %d chunks, want 1", len(doc.Chunks))
	}
}
```

- [ ] **Step 2: Run tests, verify they fail**

```bash
go test ./index/ -v
```

Expected: compilation error (ParseDocument not defined).

- [ ] **Step 3: Create index/parse.go**

```go
package index

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Document struct {
	Path        string
	Title       string
	Tags        []string
	Content     string
	ContentHash string
	Chunks      []Chunk
}

type Chunk struct {
	Heading   string
	Content   string
	StartLine int
	EndLine   int
}

type frontmatter struct {
	Title string   `yaml:"title"`
	Tags  []string `yaml:"tags"`
}

func ParseDocument(path string, data []byte) (*Document, error) {
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	chunks := chunkByHeading(body)
	hash := fmt.Sprintf("%x", sha256.Sum256(data))

	return &Document{
		Path:        path,
		Title:       fm.Title,
		Tags:        fm.Tags,
		Content:     body,
		ContentHash: hash,
		Chunks:      chunks,
	}, nil
}

func splitFrontmatter(data []byte) (frontmatter, string, error) {
	var fm frontmatter
	scanner := bufio.NewScanner(bytes.NewReader(data))

	// Expect opening ---
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return fm, string(data), nil // no frontmatter
	}

	var fmBuf bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		fmBuf.WriteString(line)
		fmBuf.WriteByte('\n')
	}

	if err := yaml.Unmarshal(fmBuf.Bytes(), &fm); err != nil {
		return fm, "", fmt.Errorf("invalid frontmatter YAML: %w", err)
	}

	// Rest is body
	var bodyBuf bytes.Buffer
	for scanner.Scan() {
		bodyBuf.WriteString(scanner.Text())
		bodyBuf.WriteByte('\n')
	}

	return fm, bodyBuf.String(), nil
}

func chunkByHeading(body string) []Chunk {
	var chunks []Chunk
	var current Chunk
	var currentLines []string
	lineNum := 1
	startLine := 1

	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect markdown headings (## level or higher for chunking)
		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") {
			// Save previous chunk if it has content
			if len(currentLines) > 0 {
				current.Content = strings.TrimSpace(strings.Join(currentLines, "\n"))
				current.StartLine = startLine
				current.EndLine = lineNum - 1
				if current.Content != "" {
					chunks = append(chunks, current)
				}
			}
			// Start new chunk
			heading := trimmed
			heading = strings.TrimLeft(heading, "# ")
			current = Chunk{Heading: heading}
			currentLines = nil
			startLine = lineNum
		}

		currentLines = append(currentLines, line)
		lineNum++
	}

	// Final chunk
	if len(currentLines) > 0 {
		current.Content = strings.TrimSpace(strings.Join(currentLines, "\n"))
		current.StartLine = startLine
		current.EndLine = lineNum - 1
		if current.Content != "" {
			chunks = append(chunks, current)
		}
	}

	return chunks
}
```

- [ ] **Step 4: Add yaml dependency and run tests**

```bash
go get gopkg.in/yaml.v3
go test ./index/ -v
```

Expected: PASS

- [ ] **Step 5: Create index/db.go**

```go
package index

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"

	_ "github.com/asg017/sqlite-vec-go-bindings/ncruces"
	_ "github.com/ncruces/go-sqlite3/driver"

	"wonderinstruments.com/help/embed"
)

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS documents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			tags TEXT NOT NULL DEFAULT '[]',
			content TEXT NOT NULL,
			content_hash TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			doc_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
			heading TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			start_line INTEGER NOT NULL,
			end_line INTEGER NOT NULL
		)`,
		fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunks USING vec0(
			chunk_id INTEGER PRIMARY KEY,
			embedding FLOAT[%d]
		)`, embed.EmbeddingDim),
	}
	for _, stmt := range stmts {
		if _, err := db.conn.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, stmt)
		}
	}
	return nil
}

func (db *DB) UpsertDocument(doc *Document) (int64, error) {
	// Check if document exists and hash matches
	var existingID int64
	var existingHash string
	err := db.conn.QueryRow(
		"SELECT id, content_hash FROM documents WHERE path = ?", doc.Path,
	).Scan(&existingID, &existingHash)

	if err == nil && existingHash == doc.ContentHash {
		return existingID, nil // unchanged
	}

	if err == nil {
		// Document exists but changed — delete and re-insert
		db.conn.Exec("DELETE FROM vec_chunks WHERE chunk_id IN (SELECT id FROM chunks WHERE doc_id = ?)", existingID)
		db.conn.Exec("DELETE FROM chunks WHERE doc_id = ?", existingID)
		db.conn.Exec("DELETE FROM documents WHERE id = ?", existingID)
	}

	tagsJSON := "[]"
	if len(doc.Tags) > 0 {
		tagsJSON = `["` + joinTags(doc.Tags) + `"]`
	}

	res, err := db.conn.Exec(
		"INSERT INTO documents (path, title, tags, content, content_hash) VALUES (?, ?, ?, ?, ?)",
		doc.Path, doc.Title, tagsJSON, doc.Content, doc.ContentHash,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func joinTags(tags []string) string {
	result := ""
	for i, t := range tags {
		if i > 0 {
			result += `","`
		}
		result += t
	}
	return result
}

func (db *DB) InsertChunk(docID int64, chunk *Chunk) (int64, error) {
	res, err := db.conn.Exec(
		"INSERT INTO chunks (doc_id, heading, content, start_line, end_line) VALUES (?, ?, ?, ?, ?)",
		docID, chunk.Heading, chunk.Content, chunk.StartLine, chunk.EndLine,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) InsertChunkEmbedding(chunkID int64, embedding []float32) error {
	_, err := db.conn.Exec(
		"INSERT INTO vec_chunks (chunk_id, embedding) VALUES (?, ?)",
		chunkID, serializeEmbedding(embedding),
	)
	return err
}

type SearchResult struct {
	ChunkID  int64
	Distance float32
	DocID    int64
	DocPath  string
	DocTitle string
	Tags     string
	Heading  string
	Content  string
}

func (db *DB) SearchChunks(queryEmbedding []float32, limit int) ([]SearchResult, error) {
	rows, err := db.conn.Query(`
		SELECT v.chunk_id, v.distance, c.doc_id, c.heading, c.content,
		       d.path, d.title, d.tags
		FROM vec_chunks v
		JOIN chunks c ON c.id = v.chunk_id
		JOIN documents d ON d.id = c.doc_id
		WHERE v.embedding MATCH ?
		ORDER BY v.distance
		LIMIT ?
	`, serializeEmbedding(queryEmbedding), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ChunkID, &r.Distance, &r.DocID, &r.Heading, &r.Content, &r.DocPath, &r.DocTitle, &r.Tags); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) SearchChunksWithTag(queryEmbedding []float32, tag string, limit int) ([]SearchResult, error) {
	rows, err := db.conn.Query(`
		SELECT v.chunk_id, v.distance, c.doc_id, c.heading, c.content,
		       d.path, d.title, d.tags
		FROM vec_chunks v
		JOIN chunks c ON c.id = v.chunk_id
		JOIN documents d ON d.id = c.doc_id
		WHERE v.embedding MATCH ?
		  AND d.tags LIKE ?
		ORDER BY v.distance
		LIMIT ?
	`, serializeEmbedding(queryEmbedding), `%"`+tag+`"%`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ChunkID, &r.Distance, &r.DocID, &r.Heading, &r.Content, &r.DocPath, &r.DocTitle, &r.Tags); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) ListDocuments(tag string) ([]Document, error) {
	query := "SELECT path, title, tags FROM documents"
	var args []interface{}
	if tag != "" {
		query += ` WHERE tags LIKE ?`
		args = append(args, `%"`+tag+`"%`)
	}
	query += " ORDER BY title"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		var tags string
		if err := rows.Scan(&d.Path, &d.Title, &tags); err != nil {
			return nil, err
		}
		d.Tags = parseTags(tags)
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func (db *DB) ListTags() ([]string, error) {
	rows, err := db.conn.Query("SELECT DISTINCT tags FROM documents")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tagSet := make(map[string]bool)
	for rows.Next() {
		var tagsJSON string
		if err := rows.Scan(&tagsJSON); err != nil {
			return nil, err
		}
		for _, t := range parseTags(tagsJSON) {
			tagSet[t] = true
		}
	}

	var tags []string
	for t := range tagSet {
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func parseTags(tagsJSON string) []string {
	// Simple JSON array parse: ["tag1","tag2"]
	tagsJSON = strings.TrimSpace(tagsJSON)
	if tagsJSON == "[]" || tagsJSON == "" {
		return nil
	}
	tagsJSON = strings.TrimPrefix(tagsJSON, `[`)
	tagsJSON = strings.TrimSuffix(tagsJSON, `]`)
	parts := strings.Split(tagsJSON, ",")
	var tags []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}

func serializeEmbedding(embedding []float32) []byte {
	buf := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}
```

Note: add missing `"strings"` import to the imports block.

- [ ] **Step 6: Add sqlite dependencies**

```bash
go get github.com/ncruces/go-sqlite3
go get github.com/asg017/sqlite-vec-go-bindings
```

- [ ] **Step 7: Run parse tests**

```bash
go test ./index/ -run TestParse -v
```

Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add index/
git commit -m "feat: add index package with parsing and sqlite schema"
```

---

## Task 4: Index Build Command

**Files:**
- Create: `index/build.go`
- Create: `cmd/help/index.go`

- [ ] **Step 1: Create index/build.go**

```go
package index

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"wonderinstruments.com/help/embed"
)

func BuildIndex(docsDir string, dbPath string) error {
	db, err := Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	// Walk docs directory for .md files
	var files []string
	err = filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk docs: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No .md files found in", docsDir)
		return nil
	}

	// Initialize embedding pipeline
	if err := embed.SetupRuntime(); err != nil {
		return fmt.Errorf("setup runtime: %w", err)
	}
	if err := embed.InitONNX(); err != nil {
		return fmt.Errorf("init onnx: %w", err)
	}
	defer embed.CleanupONNX()
	if err := embed.InitTokenizer(); err != nil {
		return fmt.Errorf("init tokenizer: %w", err)
	}

	fmt.Printf("Indexing %d files...\n", len(files))

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", path, err)
			continue
		}

		// Use relative path from docsDir
		relPath, _ := filepath.Rel(docsDir, path)

		doc, err := ParseDocument(relPath, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", path, err)
			continue
		}

		docID, err := db.UpsertDocument(doc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", path, err)
			continue
		}

		// docID == existing unchanged doc
		if docID == 0 {
			continue
		}

		// Index chunks
		for i := range doc.Chunks {
			chunk := &doc.Chunks[i]
			chunkID, err := db.InsertChunk(docID, chunk)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  chunk error: %v\n", err)
				continue
			}

			// Generate embedding for chunk content
			text := chunk.Content
			if chunk.Heading != "" {
				text = chunk.Heading + "\n" + text
			}
			emb, err := embed.GenerateEmbedding(text)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  embed error: %v\n", err)
				continue
			}

			if err := db.InsertChunkEmbedding(chunkID, emb); err != nil {
				fmt.Fprintf(os.Stderr, "  vec error: %v\n", err)
			}
		}

		fmt.Printf("  indexed: %s (%d chunks)\n", relPath, len(doc.Chunks))
	}

	fmt.Println("Done.")
	return nil
}
```

- [ ] **Step 2: Create cmd/help/index.go**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"wonderinstruments.com/help/config"
	"wonderinstruments.com/help/index"
)

var forceReindex bool

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Build or rebuild the search index",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Default()
		if err := cfg.EnsureCacheDir(); err != nil {
			return err
		}

		if forceReindex {
			os.Remove(cfg.DBPath)
		}

		return index.BuildIndex(cfg.DocsDir, cfg.DBPath)
	},
}

func init() {
	indexCmd.Flags().BoolVar(&forceReindex, "force", false, "Force full reindex")
	rootCmd.AddCommand(indexCmd)
}
```

- [ ] **Step 3: Verify build**

```bash
go build ./cmd/help
```

- [ ] **Step 4: Commit**

```bash
git add index/build.go cmd/help/index.go
git commit -m "feat: add index build command"
```

---

## Task 5: Search Command

**Files:**
- Create: `index/search.go`
- Create: `cmd/help/search.go`

- [ ] **Step 1: Create index/search.go**

```go
package index

import (
	"fmt"
	"sort"

	"wonderinstruments.com/help/embed"
)

type RankedResult struct {
	SearchResult
	Score float32
}

func Search(dbPath string, query string, tag string, limit int, rerank bool) ([]RankedResult, error) {
	db, err := Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	// Initialize embedding
	if err := embed.InitONNX(); err != nil {
		return nil, err
	}
	if err := embed.InitTokenizer(); err != nil {
		return nil, err
	}

	// Embed query
	queryEmb, err := embed.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	// Vector search — fetch more candidates if reranking
	candidateLimit := limit
	if rerank {
		candidateLimit = 30
	}

	var results []SearchResult
	if tag != "" {
		results, err = db.SearchChunksWithTag(queryEmb, tag, candidateLimit)
	} else {
		results, err = db.SearchChunks(queryEmb, candidateLimit)
	}
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	// Convert to ranked results
	ranked := make([]RankedResult, len(results))
	for i, r := range results {
		ranked[i] = RankedResult{
			SearchResult: r,
			Score:        1.0 - r.Distance, // cosine similarity
		}
	}

	// Rerank if requested
	if rerank && len(ranked) > 1 {
		if err := embed.EnsureRerankerModel(); err == nil {
			if err := embed.InitRerankerTokenizer(); err == nil {
				docs := make([]string, len(ranked))
				for i, r := range ranked {
					docs[i] = r.Heading + "\n" + r.Content
				}

				scores, err := embed.Rerank(query, docs)
				if err == nil {
					for i := range ranked {
						ranked[i].Score = scores[i]
					}
					sort.Slice(ranked, func(i, j int) bool {
						return ranked[i].Score > ranked[j].Score
					})
				}
			}
		}
	}

	// Trim to limit
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	return ranked, nil
}
```

- [ ] **Step 2: Create cmd/help/search.go**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"wonderinstruments.com/help/config"
	"wonderinstruments.com/help/embed"
	"wonderinstruments.com/help/index"
)

var (
	searchTag    string
	searchLimit  int
	searchJSON   bool
	searchRerank bool
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Semantic search over help documents",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Default()
		query := strings.Join(args, " ")

		// Ensure runtime is ready
		ready, err := embed.RuntimeReady()
		if err != nil {
			return err
		}
		if !ready {
			return fmt.Errorf("runtime not ready — run 'help index' first")
		}

		results, err := index.Search(cfg.DBPath, query, searchTag, searchLimit, searchRerank)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		if searchJSON {
			return json.NewEncoder(os.Stdout).Encode(results)
		}

		for i, r := range results {
			fmt.Printf("\n%d. %s", i+1, r.DocTitle)
			if r.Heading != "" {
				fmt.Printf(" > %s", r.Heading)
			}
			fmt.Printf(" (%.2f)\n", r.Score)
			fmt.Printf("   [%s] %s\n", r.Tags, r.DocPath)
			// Show snippet (first 2 lines)
			lines := strings.SplitN(r.Content, "\n", 3)
			for _, l := range lines[:min(len(lines), 2)] {
				l = strings.TrimSpace(l)
				if l != "" {
					fmt.Printf("   %s\n", l)
				}
			}
		}
		fmt.Println()
		return nil
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchTag, "tag", "", "Filter by tag")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10, "Max results")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "JSON output")
	searchCmd.Flags().BoolVar(&searchRerank, "rerank", true, "Rerank results")
	rootCmd.AddCommand(searchCmd)
}
```

- [ ] **Step 3: Verify build**

```bash
go build ./cmd/help
```

- [ ] **Step 4: Commit**

```bash
git add index/search.go cmd/help/search.go
git commit -m "feat: add search command with reranking"
```

---

## Task 6: List, Tags, and Read Commands

**Files:**
- Create: `cmd/help/list.go`
- Create: `cmd/help/tags.go`
- Create: `cmd/help/read.go`

- [ ] **Step 1: Create cmd/help/list.go**

```go
package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"wonderinstruments.com/help/config"
	"wonderinstruments.com/help/index"
)

var listTag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List indexed documents",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Default()
		db, err := index.Open(cfg.DBPath)
		if err != nil {
			return err
		}
		defer db.Close()

		docs, err := db.ListDocuments(listTag)
		if err != nil {
			return err
		}

		for _, d := range docs {
			tags := ""
			if len(d.Tags) > 0 {
				tags = " [" + strings.Join(d.Tags, ", ") + "]"
			}
			fmt.Printf("  %s%s\n    %s\n", d.Title, tags, d.Path)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listTag, "tag", "", "Filter by tag")
	rootCmd.AddCommand(listCmd)
}
```

- [ ] **Step 2: Create cmd/help/tags.go**

```go
package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"wonderinstruments.com/help/config"
	"wonderinstruments.com/help/index"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Default()
		db, err := index.Open(cfg.DBPath)
		if err != nil {
			return err
		}
		defer db.Close()

		tags, err := db.ListTags()
		if err != nil {
			return err
		}

		sort.Strings(tags)
		for _, t := range tags {
			fmt.Println(t)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagsCmd)
}
```

- [ ] **Step 3: Create cmd/help/read.go**

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
	"wonderinstruments.com/help/config"
)

var readCmd = &cobra.Command{
	Use:   "read [doc-path]",
	Short: "Read a help document in the terminal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Default()
		docPath := filepath.Join(cfg.DocsDir, args[0])

		data, err := os.ReadFile(docPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", args[0], err)
		}

		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(100),
		)
		if err != nil {
			return err
		}

		out, err := renderer.Render(string(data))
		if err != nil {
			return err
		}

		fmt.Print(out)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(readCmd)
}
```

- [ ] **Step 4: Add glamour dependency**

```bash
go get github.com/charmbracelet/glamour
```

- [ ] **Step 5: Verify build**

```bash
go build ./cmd/help
```

- [ ] **Step 6: Commit**

```bash
git add cmd/help/list.go cmd/help/tags.go cmd/help/read.go
git commit -m "feat: add list, tags, and read commands"
```

---

## Task 7: Wails UI — Project Setup

**Files:**
- Create: `ui/main.go`
- Create: `ui/app.go`
- Create: `ui/wails.json`
- Create: `ui/frontend/package.json`
- Create: `ui/frontend/index.html`
- Create: `ui/frontend/app.js`
- Create: `ui/frontend/style.css`

- [ ] **Step 1: Create ui/wails.json**

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "help-ui",
  "outputfilename": "help-ui",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "http://localhost:34116",
  "wailsjsdir": "./frontend",
  "author": {
    "name": "Edmund Mills"
  }
}
```

- [ ] **Step 2: Create ui/frontend/package.json**

```json
{
  "name": "help-ui-frontend",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "build": "node build.mjs",
    "dev": "node dev.mjs"
  },
  "devDependencies": {
    "esbuild": "^0.20.0"
  }
}
```

- [ ] **Step 3: Create ui/frontend/build.mjs**

```javascript
import * as esbuild from 'esbuild';
import { cpSync, mkdirSync, writeFileSync, readFileSync } from 'fs';

mkdirSync('dist/assets', { recursive: true });

await esbuild.build({
  entryPoints: ['app.js'],
  bundle: true,
  minify: true,
  outfile: 'dist/assets/app.js',
  format: 'esm',
});

cpSync('style.css', 'dist/assets/style.css');
cpSync('index.html', 'dist/index.html');
```

- [ ] **Step 4: Create ui/frontend/dev.mjs**

```javascript
import * as esbuild from 'esbuild';
import { cpSync, mkdirSync } from 'fs';

mkdirSync('dist/assets', { recursive: true });
cpSync('style.css', 'dist/assets/style.css');
cpSync('index.html', 'dist/index.html');

const ctx = await esbuild.context({
  entryPoints: ['app.js'],
  bundle: true,
  outfile: 'dist/assets/app.js',
  format: 'esm',
  sourcemap: true,
});

await ctx.serve({ port: 34116, servedir: 'dist' });
console.log('Dev server running on http://localhost:34116');
```

- [ ] **Step 5: Create ui/frontend/index.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Help</title>
  <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
  <div id="app">
    <header id="header">
      <input type="text" id="search-input" placeholder="Search help docs..." autofocus>
    </header>
    <main id="main">
      <nav id="sidebar"></nav>
      <article id="content"></article>
    </main>
  </div>
  <script type="module" src="/assets/app.js"></script>
</body>
</html>
```

- [ ] **Step 6: Create ui/frontend/style.css with base16 theming**

```css
:root {
  /* Base16 Default Dark — override with a different scheme */
  --base00: #181818;
  --base01: #282828;
  --base02: #383838;
  --base03: #585858;
  --base04: #b8b8b8;
  --base05: #d8d8d8;
  --base06: #e8e8e8;
  --base07: #f8f8f8;
  --base08: #ab4642;
  --base09: #dc9656;
  --base0A: #f7ca88;
  --base0B: #a1b56c;
  --base0C: #86c1b9;
  --base0D: #7cafc2;
  --base0E: #ba8baf;
  --base0F: #a16946;

  --font-body: system-ui, -apple-system, sans-serif;
  --font-mono: ui-monospace, "Cascadia Code", "JetBrains Mono", monospace;
}

* { margin: 0; padding: 0; box-sizing: border-box; }

body {
  font-family: var(--font-body);
  background: var(--base00);
  color: var(--base05);
  height: 100vh;
  overflow: hidden;
}

#app {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

#header {
  padding: 0.75rem 1rem;
  background: var(--base01);
  border-bottom: 1px solid var(--base02);
}

#search-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  background: var(--base00);
  border: 1px solid var(--base02);
  border-radius: 4px;
  color: var(--base05);
  font-family: var(--font-body);
  font-size: 0.9rem;
  outline: none;
}

#search-input:focus {
  border-color: var(--base0D);
}

#main {
  display: flex;
  flex: 1;
  overflow: hidden;
}

#sidebar {
  width: 240px;
  padding: 0.75rem;
  background: var(--base01);
  border-right: 1px solid var(--base02);
  overflow-y: auto;
}

#sidebar .doc-item {
  padding: 0.4rem 0.5rem;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.85rem;
  color: var(--base04);
}

#sidebar .doc-item:hover {
  background: var(--base02);
  color: var(--base05);
}

#sidebar .doc-item.active {
  background: var(--base02);
  color: var(--base0D);
}

#sidebar .tag {
  display: inline-block;
  padding: 0.1rem 0.4rem;
  margin: 0.1rem;
  border-radius: 3px;
  background: var(--base02);
  color: var(--base0C);
  font-size: 0.7rem;
}

#content {
  flex: 1;
  padding: 1.5rem 2rem;
  overflow-y: auto;
}

/* Rendered HTML content styles */
#content h1 { color: var(--base0D); margin-bottom: 1rem; font-size: 1.5rem; }
#content h2 { color: var(--base0E); margin: 1.5rem 0 0.5rem; font-size: 1.2rem; }
#content h3 { color: var(--base0A); margin: 1rem 0 0.4rem; font-size: 1rem; }
#content p { margin-bottom: 0.75rem; line-height: 1.6; }
#content code {
  font-family: var(--font-mono);
  background: var(--base01);
  padding: 0.15rem 0.35rem;
  border-radius: 3px;
  font-size: 0.85em;
}
#content pre {
  background: var(--base01);
  border: 1px solid var(--base02);
  border-radius: 4px;
  padding: 0.75rem 1rem;
  margin: 0.75rem 0;
  overflow-x: auto;
}
#content pre code {
  background: none;
  padding: 0;
}
#content a { color: var(--base0D); }
#content ul, #content ol { margin: 0.5rem 0 0.75rem 1.5rem; }
#content li { margin-bottom: 0.3rem; line-height: 1.5; }
#content blockquote {
  border-left: 3px solid var(--base03);
  padding-left: 1rem;
  color: var(--base04);
  margin: 0.75rem 0;
}

/* Search results mode */
.search-result {
  padding: 0.75rem;
  margin-bottom: 0.5rem;
  border-radius: 4px;
  background: var(--base01);
  cursor: pointer;
}

.search-result:hover {
  background: var(--base02);
}

.search-result .title {
  color: var(--base0D);
  font-weight: 600;
  font-size: 0.9rem;
}

.search-result .heading {
  color: var(--base0E);
  font-size: 0.8rem;
}

.search-result .snippet {
  color: var(--base04);
  font-size: 0.8rem;
  margin-top: 0.25rem;
}

.search-result .score {
  color: var(--base03);
  font-size: 0.7rem;
  float: right;
}
```

- [ ] **Step 7: Create ui/frontend/app.js**

```javascript
const searchInput = document.getElementById('search-input');
const sidebar = document.getElementById('sidebar');
const content = document.getElementById('content');

let searchTimeout = null;
let currentView = 'list'; // 'list', 'search', 'read'

async function init() {
  const docs = await window.go.main.App.ListDocuments('');
  renderSidebar(docs);
}

function renderSidebar(docs) {
  sidebar.innerHTML = docs.map(d => `
    <div class="doc-item" data-path="${d.Path}">
      ${d.Title}
      ${d.Tags.map(t => `<span class="tag">${t}</span>`).join('')}
    </div>
  `).join('');

  sidebar.querySelectorAll('.doc-item').forEach(el => {
    el.addEventListener('click', () => readDoc(el.dataset.path));
  });
}

async function readDoc(path) {
  const html = await window.go.main.App.GetDocument(path);
  content.innerHTML = html;
  currentView = 'read';

  sidebar.querySelectorAll('.doc-item').forEach(el => {
    el.classList.toggle('active', el.dataset.path === path);
  });
}

async function doSearch(query) {
  if (!query.trim()) {
    init();
    content.innerHTML = '';
    currentView = 'list';
    return;
  }

  const results = await window.go.main.App.Search(query, '', 10);
  currentView = 'search';

  content.innerHTML = results.map(r => `
    <div class="search-result" data-path="${r.DocPath}">
      <span class="score">${r.Score.toFixed(2)}</span>
      <div class="title">${r.DocTitle}</div>
      ${r.Heading ? `<div class="heading">${r.Heading}</div>` : ''}
      <div class="snippet">${r.Content.slice(0, 150)}...</div>
    </div>
  `).join('');

  content.querySelectorAll('.search-result').forEach(el => {
    el.addEventListener('click', () => readDoc(el.dataset.path));
  });
}

searchInput.addEventListener('input', (e) => {
  clearTimeout(searchTimeout);
  searchTimeout = setTimeout(() => doSearch(e.target.value), 300);
});

searchInput.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    searchInput.value = '';
    doSearch('');
  }
});

init();
```

- [ ] **Step 8: Commit frontend**

```bash
git add ui/frontend/ ui/wails.json
git commit -m "feat: add Wails UI frontend with base16 theming"
```

---

## Task 8: Wails UI — Go Backend

**Files:**
- Create: `ui/main.go`
- Create: `ui/app.go`

- [ ] **Step 1: Create ui/app.go**

```go
package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"wonderinstruments.com/help/config"
	"wonderinstruments.com/help/embed"
	"wonderinstruments.com/help/index"
)

type App struct {
	ctx context.Context
	cfg config.Config
}

func NewApp() *App {
	return &App{cfg: config.Default()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Initialize embedding runtime (best effort — search won't work if not indexed)
	ready, _ := embed.RuntimeReady()
	if ready {
		embed.InitONNX()
		embed.InitTokenizer()
	}
}

func (a *App) shutdown(ctx context.Context) {
	embed.CleanupONNX()
}

type DocInfo struct {
	Path  string
	Title string
	Tags  []string
}

func (a *App) ListDocuments(tag string) ([]DocInfo, error) {
	db, err := index.Open(a.cfg.DBPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	docs, err := db.ListDocuments(tag)
	if err != nil {
		return nil, err
	}

	result := make([]DocInfo, len(docs))
	for i, d := range docs {
		result[i] = DocInfo{Path: d.Path, Title: d.Title, Tags: d.Tags}
	}
	return result, nil
}

func (a *App) GetDocument(path string) (string, error) {
	fullPath := filepath.Join(a.cfg.DocsDir, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	// Strip frontmatter
	content := stripFrontmatter(data)

	// Render markdown to HTML
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)

	var buf bytes.Buffer
	if err := md.Convert(content, &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func stripFrontmatter(data []byte) []byte {
	// Skip YAML frontmatter if present
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("---")) {
		return data
	}
	// Find closing ---
	rest := data[bytes.Index(data, []byte("---"))+3:]
	idx := bytes.Index(rest, []byte("---"))
	if idx < 0 {
		return data
	}
	return rest[idx+3:]
}

type UISearchResult struct {
	DocPath  string
	DocTitle string
	Heading  string
	Content  string
	Tags     string
	Score    float32
}

func (a *App) Search(query string, tag string, limit int) ([]UISearchResult, error) {
	results, err := index.Search(a.cfg.DBPath, query, tag, limit, true)
	if err != nil {
		return nil, err
	}

	uiResults := make([]UISearchResult, len(results))
	for i, r := range results {
		uiResults[i] = UISearchResult{
			DocPath:  r.DocPath,
			DocTitle: r.DocTitle,
			Heading:  r.Heading,
			Content:  r.Content,
			Tags:     r.Tags,
			Score:    r.Score,
		}
	}
	return uiResults, nil
}
```

- [ ] **Step 2: Create ui/main.go**

```go
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Help",
		Width:  1024,
		Height: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		panic(err)
	}
}
```

- [ ] **Step 3: Add goldmark and wails dependencies**

```bash
go get github.com/yuin/goldmark
go get github.com/wailsapp/wails/v2
```

- [ ] **Step 4: Verify build**

```bash
go build ./ui/
```

- [ ] **Step 5: Commit**

```bash
git add ui/main.go ui/app.go
git commit -m "feat: add Wails backend with search and goldmark rendering"
```

---

## Task 9: Integration Test — End to End

**Files:**
- Create: `index/search_test.go`

- [ ] **Step 1: Create integration test**

```go
package index

import (
	"os"
	"path/filepath"
	"testing"

	"wonderinstruments.com/help/embed"
)

func TestBuildAndSearch(t *testing.T) {
	// Skip if runtime not available
	ready, err := embed.RuntimeReady()
	if err != nil || !ready {
		t.Skip("ONNX runtime not available")
	}

	// Setup temp docs dir
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	os.MkdirAll(docsDir, 0755)

	// Write test docs
	os.WriteFile(filepath.Join(docsDir, "git.md"), []byte(`---
title: "Git Basics"
tags: [cli, git]
---

# Git Basics

## Committing changes

Use git add to stage files, then git commit to save them.

## Branching

Create a new branch with git checkout -b branch-name.
`), 0644)

	os.WriteFile(filepath.Join(docsDir, "ssh.md"), []byte(`---
title: "SSH Configuration"
tags: [system, ssh]
---

# SSH Configuration

## Key generation

Run ssh-keygen -t ed25519 to generate a modern key pair.

## Config file

Edit ~/.ssh/config to set up host aliases and options.
`), 0644)

	// Build index
	dbPath := filepath.Join(tmpDir, "test.db")
	if err := embed.InitONNX(); err != nil {
		t.Fatal(err)
	}
	defer embed.CleanupONNX()
	if err := embed.InitTokenizer(); err != nil {
		t.Fatal(err)
	}

	if err := BuildIndex(docsDir, dbPath); err != nil {
		t.Fatal(err)
	}

	// Search
	results, err := Search(dbPath, "how to create git branches", "", 5, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Fatal("no results")
	}

	// Top result should be from git.md
	if results[0].DocPath != "git.md" {
		t.Errorf("expected git.md, got %s", results[0].DocPath)
	}
}
```

- [ ] **Step 2: Run the test (requires runtime)**

```bash
go test ./index/ -run TestBuildAndSearch -v
```

Expected: PASS (if ONNX runtime is available; SKIP otherwise).

- [ ] **Step 3: Commit**

```bash
git add index/search_test.go
git commit -m "test: add integration test for build and search"
```

---

## Task 10: Final Wiring and Makefile

**Files:**
- Modify: `Makefile`
- Modify: `cmd/help/main.go` (add ui subcommand)

- [ ] **Step 1: Add ui command to CLI**

Create `cmd/help/ui.go`:
```go
package main

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Open the help viewer",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Launch the Wails app binary if built, otherwise suggest wails dev
		bin := "help-ui"
		path, err := exec.LookPath(bin)
		if err != nil {
			return fmt.Errorf("%s not found in PATH — build with 'cd ui && wails build'", bin)
		}
		proc := exec.Command(path)
		return proc.Start()
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
```

- [ ] **Step 2: Update Makefile**

```makefile
.PHONY: build build-ui run index search dev-ui

build:
	go build -o bin/help ./cmd/help

build-ui:
	cd ui && wails build

run:
	go run ./cmd/help

index:
	go run ./cmd/help index

search:
	go run ./cmd/help search $(QUERY)

dev-ui:
	cd ui && wails dev

test:
	go test ./...

clean:
	rm -rf bin/ ui/build/
	rm -f ~/.cache/help/index.db
```

- [ ] **Step 3: Verify full build**

```bash
make build
./bin/help --help
./bin/help index
./bin/help search "git branching"
./bin/help list
./bin/help tags
./bin/help read example.md
```

- [ ] **Step 4: Commit**

```bash
git add cmd/help/ui.go Makefile
git commit -m "feat: add ui command and finalize Makefile"
```

---

## Task 11: Theme Configuration

**Files:**
- Modify: `config/config.go`
- Create: `config/themes.go`

- [ ] **Step 1: Create config/themes.go**

```go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Base16Theme struct {
	Name   string `json:"name"`
	Base00 string `json:"base00"`
	Base01 string `json:"base01"`
	Base02 string `json:"base02"`
	Base03 string `json:"base03"`
	Base04 string `json:"base04"`
	Base05 string `json:"base05"`
	Base06 string `json:"base06"`
	Base07 string `json:"base07"`
	Base08 string `json:"base08"`
	Base09 string `json:"base09"`
	Base0A string `json:"base0A"`
	Base0B string `json:"base0B"`
	Base0C string `json:"base0C"`
	Base0D string `json:"base0D"`
	Base0E string `json:"base0E"`
	Base0F string `json:"base0F"`
}

var DefaultDarkTheme = Base16Theme{
	Name:   "default-dark",
	Base00: "#181818",
	Base01: "#282828",
	Base02: "#383838",
	Base03: "#585858",
	Base04: "#b8b8b8",
	Base05: "#d8d8d8",
	Base06: "#e8e8e8",
	Base07: "#f8f8f8",
	Base08: "#ab4642",
	Base09: "#dc9656",
	Base0A: "#f7ca88",
	Base0B: "#a1b56c",
	Base0C: "#86c1b9",
	Base0D: "#7cafc2",
	Base0E: "#ba8baf",
	Base0F: "#a16946",
}

func LoadTheme(cfg Config) Base16Theme {
	// Try loading from config dir
	themePath := filepath.Join(filepath.Dir(cfg.DBPath), "..", "help", "theme.json")
	data, err := os.ReadFile(themePath)
	if err != nil {
		return DefaultDarkTheme
	}
	var theme Base16Theme
	if err := json.Unmarshal(data, &theme); err != nil {
		return DefaultDarkTheme
	}
	return theme
}

func (t Base16Theme) ToCSS() string {
	return fmt.Sprintf(`:root {
  --base00: %s;
  --base01: %s;
  --base02: %s;
  --base03: %s;
  --base04: %s;
  --base05: %s;
  --base06: %s;
  --base07: %s;
  --base08: %s;
  --base09: %s;
  --base0A: %s;
  --base0B: %s;
  --base0C: %s;
  --base0D: %s;
  --base0E: %s;
  --base0F: %s;
}`, t.Base00, t.Base01, t.Base02, t.Base03, t.Base04, t.Base05,
		t.Base06, t.Base07, t.Base08, t.Base09, t.Base0A, t.Base0B,
		t.Base0C, t.Base0D, t.Base0E, t.Base0F)
}
```

- [ ] **Step 2: Add theme endpoint to Wails app**

Add to `ui/app.go`:
```go
func (a *App) GetThemeCSS() string {
	theme := config.LoadTheme(a.cfg)
	return theme.ToCSS()
}
```

- [ ] **Step 3: Load theme in frontend**

Add to the top of `ui/frontend/app.js` init function:
```javascript
async function init() {
  // Load theme
  const themeCSS = await window.go.main.App.GetThemeCSS();
  const style = document.createElement('style');
  style.textContent = themeCSS;
  document.head.appendChild(style);

  const docs = await window.go.main.App.ListDocuments('');
  renderSidebar(docs);
}
```

- [ ] **Step 4: Commit**

```bash
git add config/themes.go ui/app.go ui/frontend/app.js
git commit -m "feat: add configurable base16 theme support"
```

---

## Summary

After all tasks:
- `help index` builds the search index from `docs/`
- `help search <query>` performs semantic search with reranking
- `help list`, `help tags` for browsing
- `help read <path>` renders docs in terminal
- `help ui` launches the Wails desktop viewer
- Theme is configurable via `~/.cache/help/theme.json` (base16 format)
- System fonts used throughout
- JSON output mode enables future integrations (rofi, foot, etc.)
