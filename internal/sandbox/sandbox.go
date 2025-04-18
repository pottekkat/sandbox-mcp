package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/pottekkat/sandbox-mcp/internal/config"
)

// NewSandboxTool creates a sandbox tool from a config
func NewSandboxTool(sandboxConfig *config.SandboxConfig) mcp.Tool {
	options := []mcp.ToolOption{
		// All tools have a description and an entrypoint
		mcp.WithDescription(sandboxConfig.Description),
		withEntrypoint(sandboxConfig.Entrypoint, fmt.Sprintf("%s code to execute in the sandbox", sandboxConfig.Name)),
	}

	// Add any specific additional files if provided in the config
	for _, file := range sandboxConfig.Parameters.Files {
		options = append(options, withFile(file.Name, file.Description, true))
	}

	// Allow adding more files if enabled
	if sandboxConfig.Parameters.AdditionalFiles {
		options = append(options, withAdditionalFiles())
	}

	// Return a new tool with the tool name and provided options
	return mcp.NewTool(sandboxConfig.Name, options...)
}

// NewSandboxToolHandler creates a handler function for a sandbox tool
func NewSandboxToolHandler(sandboxConfig *config.SandboxConfig) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Return the handler function that will be run when the tool is called
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// withEntrypoint ToolOption
		// Get the contents of the entrypoint file from the request
		entrypointContent, ok := request.Params.Arguments[sandboxConfig.Entrypoint].(string)
		if !ok || entrypointContent == "" {
			return nil, fmt.Errorf("%s file is required", sandboxConfig.Entrypoint)
		}

		// Create a temporary directory for the entrypoint file
		dir, err := os.MkdirTemp("", sandboxConfig.Mount.TmpDirPrefix)
		if err != nil {
			return nil, fmt.Errorf("failed to create a temporary directory: %v", err)
		}
		defer os.RemoveAll(dir)

		// Write the entrypoint to script file in the temp directory
		cmdFile := filepath.Join(dir, sandboxConfig.Entrypoint)
		if err := os.WriteFile(cmdFile, []byte(entrypointContent), sandboxConfig.Mount.ScriptPerms()); err != nil {
			return nil, fmt.Errorf("failed to write command file: %v", err)
		}

		// withFile ToolOption
		// Get the contents of the required files from the request
		for _, file := range sandboxConfig.Parameters.Files {
			content, ok := request.Params.Arguments[file.Name].(string)
			if !ok || content == "" {
				return nil, fmt.Errorf("%s file is required", file.Name)
			}

			filePath := filepath.Join(dir, file.Name)
			if err := os.WriteFile(filePath, []byte(content), sandboxConfig.Mount.ScriptPerms()); err != nil {
				return nil, fmt.Errorf("failed to write file %s: %v", file.Name, err)
			}
		}

		// withAdditionalFiles ToolOption
		// Handle additional files if provided
		if files, ok := request.Params.Arguments["files"].([]any); ok {
			for _, file := range files {
				if fileMap, ok := file.(map[string]any); ok {
					filename := fileMap["filename"].(string)
					content := fileMap["content"].(string)

					filePath := filepath.Join(dir, filename)
					if err := os.WriteFile(filePath, []byte(content), sandboxConfig.Mount.ScriptPerms()); err != nil {
						return nil, fmt.Errorf("failed to write file %s: %v", filename, err)
					}
				}
			}
		}

		// Initialize Docker client
		cli, err := client.NewClientWithOpts(
			// Let the client be configured through environment variables
			client.FromEnv,
			// Try to support whatever version of the daemon is available
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Docker client: %v", err)
		}
		defer cli.Close()

		// Create container config
		containerConfig := &container.Config{
			Image:      sandboxConfig.Image,
			Cmd:        sandboxConfig.Command,
			WorkingDir: sandboxConfig.Mount.WorkDir,
			User:       sandboxConfig.User,
			Tty:        false,
		}

		// Create host config
		hostConfig := &container.HostConfig{
			Resources: container.Resources{
				Memory:    sandboxConfig.Resources.Memory * 1024 * 1024,
				NanoCPUs:  int64(sandboxConfig.Resources.CPU * 1e9),
				PidsLimit: &sandboxConfig.Resources.Processes,
				Ulimits: []*container.Ulimit{
					{
						Name: "nofile",
						Soft: sandboxConfig.Resources.Files,
						Hard: sandboxConfig.Resources.Files,
					},
				},
			},
			NetworkMode:    container.NetworkMode(sandboxConfig.Security.Network),
			ReadonlyRootfs: sandboxConfig.Security.ReadOnly,
			Mounts: []mount.Mount{
				{
					Type:     mount.TypeBind,
					Source:   dir,
					Target:   sandboxConfig.Mount.WorkDir,
					ReadOnly: sandboxConfig.Mount.ReadOnly,
				},
			},
			CapDrop:     sandboxConfig.Security.CapDrop,
			SecurityOpt: sandboxConfig.Security.SecurityOpt,
		}

		// Create execution context with timeout
		execCtx, cancel := context.WithTimeout(ctx, sandboxConfig.Timeout())
		defer cancel()

		// Create container
		resp, err := cli.ContainerCreate(execCtx, containerConfig, hostConfig, nil, nil, "")
		if err != nil {
			return nil, fmt.Errorf("failed to create container: %v", err)
		}

		// Ensure container cleanup
		defer func() {
			killCtx, killCancel := context.WithTimeout(context.Background(), sandboxConfig.Timeout())
			defer killCancel()

			_ = cli.ContainerRemove(killCtx, resp.ID, container.RemoveOptions{
				Force:         true,
				RemoveVolumes: true,
			})
		}()

		// Start the container
		if err := cli.ContainerStart(execCtx, resp.ID, container.StartOptions{}); err != nil {
			return nil, fmt.Errorf("failed to start container: %v", err)
		}

		// Wait for execution to finish
		statusCh, errCh := cli.ContainerWait(execCtx, resp.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				return nil, fmt.Errorf("error waiting for container: %v", err)
			}
		case status := <-statusCh:
			// Get container logs
			logs, err := cli.ContainerLogs(execCtx, resp.ID, container.LogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Timestamps: false,
				Follow:     false,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get logs: %v", err)
			}
			defer logs.Close()

			// Read stdout and stderr
			stdout := new(bytes.Buffer)
			stderr := new(bytes.Buffer)
			if _, err := stdcopy.StdCopy(stdout, stderr, logs); err != nil {
				return nil, fmt.Errorf("failed to read logs: %v", err)
			}

			// Return error if command failed
			if status.StatusCode != 0 {
				return mcp.NewToolResultError(stderr.String()), nil
			}

			// Include stderr in stdout if present
			if stderr.Len() > 0 {
				stdout.WriteString("\nStderr:\n")
				stdout.Write(stderr.Bytes())
			}

			return mcp.NewToolResultText(stdout.String()), nil
		case <-execCtx.Done():
			return nil, fmt.Errorf("execution timeout after %d seconds", int(sandboxConfig.Timeout().Seconds()))
		}

		return nil, fmt.Errorf("unexpected error: container wait returned no result")
	}
}
