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
