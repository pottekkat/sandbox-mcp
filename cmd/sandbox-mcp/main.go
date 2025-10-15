package main

import (
	"context"
	"flag"
	"log"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	"github.com/pottekkat/sandbox-mcp/internal/appconfig"
	"github.com/pottekkat/sandbox-mcp/internal/config"
	"github.com/pottekkat/sandbox-mcp/internal/sandbox"
)

func main() {
	// Parse flags
	stdio := flag.Bool("stdio", false, "Start the MCP via stdio transport")
	build := flag.Bool("build", false, "Build Docker images for all sandboxes")
	pull := flag.Bool("pull", false, "Pull default sandboxes from GitHub")
	force := flag.Bool("force", false, "Force overwrite existing sandboxes when pulling")
	disableSandboxes := flag.String("disable-sandboxes", "", "List of disabling sandboxes")
	flag.Parse()

	// Configure logging
	// TODO: Improve logging as per MCP spec
	log.SetPrefix("[Sandbox MCP] ")
	log.SetFlags(log.Ldate | log.Ltime)

	// Load application configuration
	cfg, err := appconfig.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load sandbox-mcp configuration: %v", err)
	}

	// Pull sandboxes if pull flag is present
	if *pull {
		if err := sandbox.PullSandboxes(cfg.SandboxesPath, *force); err != nil {
			log.Fatalf("Failed to pull sandboxes: %v", err)
		}
		return
	}

	// Load sandbox configurations from the configured path
	configs, err := config.LoadSandboxConfigs(cfg.SandboxesPath)
	if err != nil {
		log.Fatalf("Failed to load sandbox configurations: %v", err)
	}

	// Build Docker images if build flag is present
	if *build {
		log.Println("Building Docker images for all sandboxes...")
		for _, sandboxCfg := range configs {
			if err := sandbox.BuildImage(context.Background(), sandboxCfg, cfg.SandboxesPath); err != nil {
				log.Printf("Failed to build image for sandbox %s: %v", sandboxCfg.Id, err)
				continue
			}
		}
		return
	}

	// Only start MCP server if the stdio flag is present
	if *stdio {
		var disableSandboxesList []string
		if *disableSandboxes != "" {
			disableSandboxesList = strings.Split(*disableSandboxes, ",")
		}

		// Create a new MCP server
		s := server.NewMCPServer(
			"Sandbox MCP",
			"0.1.0",
			// We don't notify when the list of tools changes
			// The list of tools never change for now
			server.WithToolCapabilities(false),
		)

		// Create and add tools for each sandbox configuration
		for _, cfg := range configs {
			// Skip sandboxes in disable-sandboxes list
			if slices.Contains(disableSandboxesList, cfg.Id) {
				continue
			}

			// Create a new tool from the config
			tool := sandbox.NewSandboxTool(cfg)

			// Create a handler using the sandbox config
			handler := sandbox.NewSandboxToolHandler(cfg)

			// Add the tool to the server
			s.AddTool(tool, handler)

			log.Printf("Added %s tool from config", cfg.Id)
		}

		log.Println("Starting Sandbox MCP server...")

		// Start the server
		if err := server.ServeStdio(s); err != nil {
			log.Printf("Error starting server: %v\n", err)
		}
	}
}
