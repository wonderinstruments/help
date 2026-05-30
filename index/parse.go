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

	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return fm, string(data), nil
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

		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") {
			if len(currentLines) > 0 {
				current.Content = strings.TrimSpace(strings.Join(currentLines, "\n"))
				current.StartLine = startLine
				current.EndLine = lineNum - 1
				if current.Content != "" {
					chunks = append(chunks, current)
				}
			}
			heading := trimmed
			for strings.HasPrefix(heading, "#") {
				heading = heading[1:]
			}
			heading = strings.TrimSpace(heading)
			current = Chunk{Heading: heading}
			currentLines = nil
			startLine = lineNum
		}

		currentLines = append(currentLines, line)
		lineNum++
	}

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
