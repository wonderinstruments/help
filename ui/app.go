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
	ctx        context.Context
	cfg        config.Config
	embedReady chan struct{}
}

func NewApp() *App {
	return &App{
		cfg:        config.Default(),
		embedReady: make(chan struct{}),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go func() {
		embed.InitONNX()
		embed.InitTokenizer()
		close(a.embedReady)
	}()
}

func (a *App) waitEmbed() {
	<-a.embedReady
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

	content := stripFrontmatter(data)

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
	trimmed := bytes.TrimSpace(data)
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return data
	}
	rest := trimmed[3:]
	idx := bytes.Index(rest, []byte("\n---"))
	if idx < 0 {
		return data
	}
	return rest[idx+4:]
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
	a.waitEmbed()
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

func (a *App) GetThemeCSS() string {
	theme := config.LoadTheme(a.cfg)
	return theme.ToCSS()
}
