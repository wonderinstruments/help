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
		t.Fatalf("got %d chunks, want 3", len(doc.Chunks))
	}
	if doc.Chunks[0].Heading != "SSH Keys" {
		t.Errorf("chunk 0 heading: %q", doc.Chunks[0].Heading)
	}
	if doc.Chunks[1].Heading != "Generating a key" {
		t.Errorf("chunk 1 heading: %q", doc.Chunks[1].Heading)
	}
	if doc.Chunks[2].Heading != "Adding to agent" {
		t.Errorf("chunk 2 heading: %q", doc.Chunks[2].Heading)
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
