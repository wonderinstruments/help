package embed

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const ONNXRuntimeVersion = "1.24.1"

func GetONNXRuntimeURL() (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var platform string
	switch {
	case goos == "linux" && goarch == "amd64":
		platform = "linux-x64"
	case goos == "linux" && goarch == "arm64":
		platform = "linux-aarch64"
	case goos == "darwin" && goarch == "amd64":
		platform = "osx-x86_64"
	case goos == "darwin" && goarch == "arm64":
		platform = "osx-arm64"
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", goos, goarch)
	}

	return fmt.Sprintf(
		"https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-%s-%s.tgz",
		ONNXRuntimeVersion, platform, ONNXRuntimeVersion,
	), nil
}

func GetONNXRuntimeLibPath() (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}

	var libName string
	switch runtime.GOOS {
	case "linux":
		libName = "libonnxruntime.so"
	case "darwin":
		libName = "libonnxruntime.dylib"
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	return filepath.Join(cacheDir, libName), nil
}

func DownloadONNXRuntime() error {
	libPath, err := GetONNXRuntimeLibPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(libPath); err == nil {
		return nil
	}

	url, err := GetONNXRuntimeURL()
	if err != nil {
		return err
	}

	fmt.Println("Downloading ONNX Runtime...")
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		isVersionedLib := (strings.Contains(header.Name, "libonnxruntime.so.") ||
			(strings.Contains(header.Name, "libonnxruntime.") && strings.HasSuffix(header.Name, ".dylib"))) &&
			header.Typeflag == tar.TypeReg && header.Size > 0

		if isVersionedLib {
			out, err := os.Create(libPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			break
		}
	}

	return nil
}

func SetupRuntime() error {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	if err := DownloadONNXRuntime(); err != nil {
		return fmt.Errorf("failed to download ONNX runtime: %w", err)
	}
	if err := EnsureModel(); err != nil {
		return fmt.Errorf("failed to extract model: %w", err)
	}
	return nil
}
