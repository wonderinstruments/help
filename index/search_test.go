package index

import (
	"os"
	"path/filepath"
	"testing"

	"wonderinstruments.com/help/embed"
)

func TestBuildAndSearch(t *testing.T) {
	ready, err := embed.RuntimeReady()
	if err != nil || !ready {
		t.Skip("ONNX runtime not available")
	}

	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	os.MkdirAll(docsDir, 0755)

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

	results, err := Search(dbPath, "how to create git branches", "", 5, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Fatal("no results")
	}

	if results[0].DocPath != "git.md" {
		t.Errorf("expected git.md as top result, got %s", results[0].DocPath)
	}
}
