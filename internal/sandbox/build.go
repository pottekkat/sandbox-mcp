package sandbox

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/pottekkat/sandbox-mcp/internal/config"
)

// BuildImage builds the Docker image of a sandbox
func BuildImage(ctx context.Context, sandboxConfig *config.SandboxConfig, basePath string) error {
	// Initialize Docker client
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %v", err)
	}
	defer cli.Close()

	// Get the sandbox directory which contains the Dockerfile
	sandboxDir := filepath.Join(basePath, sandboxConfig.Id)

	// Create build context tar
	buildCtx, err := archive.TarWithOptions(sandboxDir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("failed to create build context: %v", err)
	}

	// Build the image with the specified tag
	resp, err := cli.ImageBuild(ctx, buildCtx, types.ImageBuildOptions{
		Tags:       []string{sandboxConfig.Image},
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to build image: %v", err)
	}
	defer resp.Body.Close()

	// Stream build output to stdout
	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read build output: %v", err)
	}

	log.Printf("Successfully built image: %s", sandboxConfig.Image)
	return nil
}
