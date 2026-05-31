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
				fmt.Printf(" > %s", cleanHeading(r.Heading))
			}
			fmt.Printf("  (%.2f)\n", r.Score)
			fmt.Printf("   %s  %s\n", formatTags(r.Tags), r.DocPath)
			snippet := cleanSnippet(r.Content)
			if snippet != "" {
				fmt.Printf("   %s\n", snippet)
			}
		}
		fmt.Println()
		return nil
	},
}

func cleanHeading(h string) string {
	// Remove pandoc attribute syntax {#id} {.class}
	if idx := strings.Index(h, " {"); idx > 0 {
		h = h[:idx]
	}
	return h
}

func cleanSnippet(content string) string {
	var cleaned []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		// Skip heading underlines
		if len(line) >= 3 && (strings.Trim(line, "=") == "" || strings.Trim(line, "-") == "") {
			continue
		}
		// Skip latex index commands
		if strings.HasPrefix(line, "\\index{") {
			continue
		}
		// Skip empty lines
		if line == "" {
			continue
		}
		// Strip pandoc role syntax
		line = stripRoles(line)
		// Strip pandoc heading attributes {#id}
		if idx := strings.Index(line, " {#"); idx > 0 {
			line = line[:idx]
		}
		// Skip lines that are just markdown headings (already shown in result header)
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			continue
		}
		cleaned = append(cleaned, line)
		if len(cleaned) >= 2 {
			break
		}
	}
	return strings.Join(cleaned, "\n   ")
}

func stripRoles(s string) string {
	// Remove `text`{.interpreted-text role="mod"} → `text`
	for {
		idx := strings.Index(s, "{.interpreted-text")
		if idx < 0 {
			break
		}
		end := strings.Index(s[idx:], "}")
		if end < 0 {
			break
		}
		s = s[:idx] + s[idx+end+1:]
	}
	return s
}

func formatTags(tagsJSON string) string {
	tagsJSON = strings.TrimSpace(tagsJSON)
	if tagsJSON == "[]" || tagsJSON == "" {
		return ""
	}
	// Parse ["tag1","tag2"] → [tag1, tag2]
	tagsJSON = strings.TrimPrefix(tagsJSON, "[")
	tagsJSON = strings.TrimSuffix(tagsJSON, "]")
	parts := strings.Split(tagsJSON, ",")
	var tags []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return "[" + strings.Join(tags, ", ") + "]"
}

func init() {
	searchCmd.Flags().StringVar(&searchTag, "tag", "", "Filter by tag")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10, "Max results")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "JSON output")
	searchCmd.Flags().BoolVar(&searchRerank, "rerank", true, "Rerank results")
	rootCmd.AddCommand(searchCmd)
}
