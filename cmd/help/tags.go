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
