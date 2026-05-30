package embed

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

const EmbeddingDim = 256

//go:embed models/model.onnx
var embeddedModel []byte

func GetCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".cache", "help", "models"), nil
}

func GetModelPath() (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "model.onnx"), nil
}

func EnsureModel() error {
	modelPath, err := GetModelPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(modelPath); err == nil {
		return nil
	}
	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	if err := os.WriteFile(modelPath, embeddedModel, 0644); err != nil {
		return fmt.Errorf("failed to write model file: %w", err)
	}
	return nil
}

func RuntimeReady() (bool, error) {
	libPath, err := GetONNXRuntimeLibPath()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return false, nil
	}
	modelPath, err := GetModelPath()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return false, nil
	}
	return true, nil
}
