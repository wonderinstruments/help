package main

import (
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
