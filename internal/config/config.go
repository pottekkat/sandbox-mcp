package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// SandboxParameters represents the parameters configuration
type SandboxParameters struct {
	AdditionalFiles bool `json:"additionalFiles"`
	Files           []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"files,omitempty"`
}

// SandboxSecurity represents the security configuration
type SandboxSecurity struct {
	ReadOnly    bool     `json:"readOnly"`
	CapDrop     []string `json:"capDrop"`
	SecurityOpt []string `json:"securityOpt"`
	Network     string   `json:"network"`
}

// SandboxResources represents the resource limits
type SandboxResources struct {
	CPU       int   `json:"cpu"`
	Memory    int64 `json:"memory"`
	Processes int64 `json:"processes"`
	Files     int64 `json:"files"`
}

// SandboxMount represents the mount configuration
type SandboxMount struct {
	WorkDir        string `json:"workdir"`
	TmpDirPrefix   string `json:"tmpdirPrefix"`
	ScriptPermsRaw string `json:"scriptPerms"`
	ReadOnly       bool   `json:"readOnly"`
}

// ScriptPerms returns the script permissions as os.FileMode
func (m *SandboxMount) ScriptPerms() os.FileMode {
	mode, err := parseFileMode(m.ScriptPermsRaw)
	if err != nil {
		return 0755
	}
	return mode
}

// SandboxConfig represents the complete configuration for a sandbox environment
type SandboxConfig struct {
	// Basic configuration
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Image       string            `json:"image"`
	User        string            `json:"user"`
	Entrypoint  string            `json:"entrypoint"`
	TimeoutRaw  int               `json:"timeout"`
	Before      []string          `json:"before"`
	Command     []string          `json:"command"`
	Parameters  SandboxParameters `json:"parameters"`
	Security    SandboxSecurity   `json:"security"`
	Resources   SandboxResources  `json:"resources"`
	Mount       SandboxMount      `json:"mount"`
}

// Timeout returns the timeout as a time.Duration
func (c *SandboxConfig) Timeout() time.Duration {
	return time.Duration(c.TimeoutRaw) * time.Second
}

// parseFileMode converts a string like "0755" into os.FileMode
func parseFileMode(mode string) (os.FileMode, error) {
	// Parse as base-8 (octal)
	parsed, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid file mode %q: %w", mode, err)
	}
	return os.FileMode(parsed), nil
}

// LoadSandboxConfigs loads all sandbox configurations from the sandboxes directory
func LoadSandboxConfigs(sandboxDir string) (map[string]*SandboxConfig, error) {
	configs := make(map[string]*SandboxConfig)

	entries, err := os.ReadDir(sandboxDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read sandbox directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		configPath := filepath.Join(sandboxDir, entry.Name(), "config.json")
		configData, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %v", configPath, err)
		}

		var config SandboxConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %v", configPath, err)
		}

		configs[config.Name] = &config
	}

	return configs, nil
}
