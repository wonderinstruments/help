package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
	"wonderinstruments.com/help/config"
)

var readCmd = &cobra.Command{
	Use:   "read [doc-path]",
	Short: "Read a help document in the terminal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Default()
		docPath := filepath.Join(cfg.DocsDir, args[0])

		data, err := os.ReadFile(docPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", args[0], err)
		}

		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(100),
		)
		if err != nil {
			return err
		}

		out, err := renderer.Render(string(data))
		if err != nil {
			return err
		}

		fmt.Print(out)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(readCmd)
}
