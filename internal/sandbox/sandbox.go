package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/pottekkat/sandbox-mcp/internal/config"
)

// waitForContainer waits for a container to be in running state with a specified timeout
func waitForContainer(ctx context.Context, cli *client.Client, containerID string, timeout time.Duration) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for container to start")
		case <-timeoutCh:
			return fmt.Errorf("container did not reach running state within %v", timeout)
		case <-ticker.C:
			inspect, err := cli.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %v", err)
			}
			if inspect.State != nil && inspect.State.Running {
				return nil
			}
		}
	}
}

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
		user := sandboxConfig.User
		if os.Getenv("CI") == "true" {
			user = "root"
		}
		containerConfig := &container.Config{
			Image:      sandboxConfig.Image,
			Cmd:        sandboxConfig.RunCommand(),
			WorkingDir: sandboxConfig.Mount.WorkDir,
			User:       user,
			Tty:        sandboxConfig.Tty(),
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

		// Only exec Command if Before was used to start the container
		if sandboxConfig.ExecCommand() != nil {

			// Wait for container to be running
			if err := waitForContainer(execCtx, cli, resp.ID, 10*time.Second); err != nil {
				return nil, err
			}

			execConfig := container.ExecOptions{
				Cmd:          sandboxConfig.Command,
				AttachStdout: true,
				AttachStderr: true,
				User:         user,
			}

			execResp, err := cli.ContainerExecCreate(execCtx, resp.ID, execConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create exec: %v", err)
			}

			// Attach to the exec command to capture output
			response, err := cli.ContainerExecAttach(execCtx, execResp.ID, container.ExecStartOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to attach to exec: %v", err)
			}
			defer response.Close()

			// Read stdout and stderr from the exec command
			stdout := new(bytes.Buffer)
			stderr := new(bytes.Buffer)
			if _, err := stdcopy.StdCopy(stdout, stderr, response.Reader); err != nil {
				return nil, fmt.Errorf("failed to read exec output: %v", err)
			}

			// Wait for the exec command to complete
			for {
				inspectResp, err := cli.ContainerExecInspect(execCtx, execResp.ID)
				if err != nil {
					return nil, fmt.Errorf("failed to inspect exec: %v", err)
				}
				if !inspectResp.Running {
					// Return error if exec command failed
					if inspectResp.ExitCode != 0 {
						if stderr.Len() > 0 {
							return mcp.NewToolResultError(stderr.String()), nil
						}
						return mcp.NewToolResultError(fmt.Sprintf("Command failed with exit code %d", inspectResp.ExitCode)), nil
					}

					// Include stderr in stdout if present
					if stderr.Len() > 0 {
						stdout.WriteString("\nStderr:\n")
						stdout.Write(stderr.Bytes())
					}

					return mcp.NewToolResultText(stdout.String()), nil
				}
				time.Sleep(100 * time.Millisecond)
			}
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
