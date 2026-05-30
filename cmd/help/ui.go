package main

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Open the help viewer",
	RunE: func(cmd *cobra.Command, args []string) error {
		bin := "help-ui"
		path, err := exec.LookPath(bin)
		if err != nil {
			return fmt.Errorf("%s not found in PATH — build with 'cd ui && wails build'", bin)
		}
		proc := exec.Command(path)
		return proc.Start()
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
