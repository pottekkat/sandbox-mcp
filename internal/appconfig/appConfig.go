package appconfig

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

const (
	appName               = "sandbox-mcp"
	defaultConfigFileName = "config.json"
)

// Config holds the core configuration for sandbox-mcp
type Config struct {
	// SandboxesPath is the path to the sandboxes directory
	SandboxesPath string `json:"sandboxesPath"`
}

// DefaultConfig creates a default configuration
func DefaultConfig() *Config {
	// Default config path plus the sandboxes directory
	defaultSandboxesPath := filepath.Join(xdg.ConfigHome, appName, "sandboxes")
	log.Printf("Creating default configuration with sandboxes path: %s", defaultSandboxesPath)
	return &Config{
		SandboxesPath: defaultSandboxesPath,
	}
}

// LoadConfig loads the configuration from the config.json file
// If the config file doesn't exist, it creates one with default values
func LoadConfig() (*Config, error) {
	configPath := filepath.Join(xdg.ConfigHome, appName, defaultConfigFileName)
	log.Printf("Looking for config file at: %s", configPath)

	// Check if config directory exists, create if not
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		log.Printf("Config directory does not exist, creating: %s", configDir)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}
		log.Printf("Created config directory: %s", configDir)
	} else {
		log.Printf("Config directory already exists: %s", configDir)
	}

	// Try to read existing config
	config := &Config{}
	if data, err := os.ReadFile(configPath); err == nil {
		log.Printf("Found existing config file, attempting to parse")
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
		log.Printf("Successfully loaded existing config with sandboxes path: %s", config.SandboxesPath)
		return config, nil
	} else {
		log.Printf("No existing config file found: %v", err)
	}

	// Config doesn't exist, create default config
	log.Println("Creating new configuration with default values")
	config = DefaultConfig()

	// Create sandboxes directory
	if _, err := os.Stat(config.SandboxesPath); os.IsNotExist(err) {
		log.Printf("Sandboxes directory does not exist, creating: %s", config.SandboxesPath)
		if err := os.MkdirAll(config.SandboxesPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create sandboxes directory: %w", err)
		}
		log.Printf("Created sandboxes directory: %s", config.SandboxesPath)
	} else {
		log.Printf("Sandboxes directory already exists: %s", config.SandboxesPath)
	}

	// Save default config
	log.Printf("Saving default configuration to: %s", configPath)
	if err := config.Save(); err != nil {
		return nil, fmt.Errorf("failed to save default config: %w", err)
	}
	log.Println("Successfully saved default configuration")

	return config, nil
}

// Save saves the configuration to the config file
func (c *Config) Save() error {
	configPath := filepath.Join(xdg.ConfigHome, appName, defaultConfigFileName)
	log.Printf("Preparing to save configuration to: %s", configPath)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	log.Printf("Configuration JSON prepared: %s", string(data))

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	log.Printf("Successfully wrote configuration to: %s", configPath)

	return nil
}
