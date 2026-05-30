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
				fmt.Printf(" > %s", r.Heading)
			}
			fmt.Printf(" (%.2f)\n", r.Score)
			fmt.Printf("   [%s] %s\n", r.Tags, r.DocPath)
			lines := strings.SplitN(r.Content, "\n", 4)
			for _, l := range lines[:min(len(lines), 3)] {
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
