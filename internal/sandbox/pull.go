package sandbox

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pottekkat/sandbox-mcp/internal/version"
)

const (
	githubReleaseURL = "https://github.com/pottekkat/sandbox-mcp/releases/download/%s/sandboxes.tar.gz"
)

// PullSandboxes downloads and extracts sandboxes from GitHub releases
func PullSandboxes(destPath string, force bool) error {
	// Build the download URL using the current version
	url := fmt.Sprintf(githubReleaseURL, version.GetVersion())
	log.Printf("Downloading sandboxes from: %s", url)

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download sandboxes: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download sandboxes: HTTP %d", resp.StatusCode)
	}

	// Create a temporary file to store the download
	tmpFile, err := os.CreateTemp("", "sandboxes-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			log.Printf("Failed to remove temp file: %v", err)
		}
	}()
	defer func() {
		if err := tmpFile.Close(); err != nil {
			log.Printf("Failed to close temp file: %v", err)
		}
	}()

	// Copy the download to the temporary file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save download: %v", err)
	}

	// Extract the tar.gz file
	if err := extractTarGz(tmpFile.Name(), destPath, force); err != nil {
		return fmt.Errorf("failed to extract sandboxes: %v", err)
	}

	log.Printf("Successfully downloaded and extracted sandboxes to: %s", destPath)
	return nil
}

// extractTarGz unpacks a tar.gz archive to the specified path
func extractTarGz(srcPath, destPath string, force bool) error {
	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	// Set up gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer func() {
		if err := gzr.Close(); err != nil {
			log.Printf("Failed to close gzip reader: %v", err)
		}
	}()

	// Read tar archive
	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Only process files in the sandboxes directory
		if !strings.HasPrefix(header.Name, "sandboxes/") {
			continue
		}

		// Get path relative to sandboxes directory
		relPath := strings.TrimPrefix(header.Name, "sandboxes/")
		if relPath == "" {
			continue
		}

		targetPath := filepath.Join(destPath, relPath)

		// Skip existing sandboxes unless force is true
		if !force && header.Typeflag == tar.TypeDir {
			if _, err := os.Stat(targetPath); err == nil {
				log.Printf("Skipping existing sandbox: %s", relPath)
				continue
			}
		}

		// Handle directories and files
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				cerr := outFile.Close()
				if cerr != nil {
					log.Printf("Failed to close output file after copy error: %v", cerr)
				}
				return err
			}
		}
	}

	return nil
}
