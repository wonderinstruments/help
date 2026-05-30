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

	var files []string
	err = filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			// Skip docs in superpowers directory
			rel, _ := filepath.Rel(docsDir, path)
			if !strings.HasPrefix(rel, "superpowers") {
				files = append(files, path)
			}
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

		if docID == 0 {
			continue // unchanged
		}

		for i := range doc.Chunks {
			chunk := &doc.Chunks[i]
			chunkID, err := db.InsertChunk(docID, chunk)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  chunk error: %v\n", err)
				continue
			}

			// FTS index
			if err := db.InsertChunkFTS(chunkID, chunk.Heading, chunk.Content); err != nil {
				fmt.Fprintf(os.Stderr, "  fts error: %v\n", err)
			}

			// Vector embedding
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
