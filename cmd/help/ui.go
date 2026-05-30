package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Open the help viewer",
	RunE: func(cmd *cobra.Command, args []string) error {
		binPath, err := findUIBinary()
		if err != nil {
			return err
		}
		proc := exec.Command(binPath)
		proc.Stdout = os.Stdout
		proc.Stderr = os.Stderr
		return proc.Run()
	},
}

func findUIBinary() (string, error) {
	// Check PATH first
	if path, err := exec.LookPath("help-ui"); err == nil {
		return path, nil
	}

	// Check next to the current executable
	exe, err := os.Executable()
	if err == nil {
		sibling := filepath.Join(filepath.Dir(exe), "help-ui")
		if _, err := os.Stat(sibling); err == nil {
			return sibling, nil
		}
	}

	return "", fmt.Errorf("help-ui not found — build with 'make build-ui'")
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
