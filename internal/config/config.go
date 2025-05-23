package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SandboxHints stores hints provided to the MCP client
type SandboxHints struct {
	IsReadOnlyRaw            *bool `json:"isReadOnly,omitempty"`
	IsDestructiveRaw         *bool `json:"isDestructive,omitempty"`
	IsIdempotentRaw          *bool `json:"isIdempotent,omitempty"`
	IsExternalInteractionRaw *bool `json:"isExternalInteraction,omitempty"`
}

// IsReadOnly returns true if the sandbox is meant to be read-only
// If the raw value is not set, check the mount and security configurations
func (h *SandboxHints) IsReadOnly(mountReadOnly bool, securityReadOnly bool) bool {
	// If the raw value is set to either true or false, return that
	if h.IsReadOnlyRaw != nil {
		return *h.IsReadOnlyRaw
	}
	// If the raw value is not set, check the mount and security configurations
	return mountReadOnly || securityReadOnly
}

// IsDestructive returns true if the sandbox is meant to be destructive
// Defaults to false
func (h *SandboxHints) IsDestructive() bool {
	if h.IsDestructiveRaw != nil {
		return *h.IsDestructiveRaw
	}
	return false
}

// IsIdempotent returns true if the sandbox is meant to be idempotent
// Defaults to true
func (h *SandboxHints) IsIdempotent() bool {
	if h.IsIdempotentRaw != nil {
		return *h.IsIdempotentRaw
	}
	return true
}

// IsExternalInteraction returns true if the sandbox is meant to interact with external systems
// Defaults to the network configuration under security
func (h *SandboxHints) IsExternalInteraction(securityNetwork string) bool {
	if h.IsExternalInteractionRaw != nil {
		return *h.IsExternalInteractionRaw
	}
	// If the raw value is not set, check the network configuration
	return securityNetwork != "none"
}

// SandboxFile represents a file parameter
type SandboxFile struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ParamName returns a safe parameter name for a file (replaces '.' with '_')
func (f *SandboxFile) ParamName() string {
	return strings.ReplaceAll(f.Name, ".", "_")
}

// SandboxParameters represents the parameters configuration
type SandboxParameters struct {
	AdditionalFiles bool          `json:"additionalFiles"`
	Files           []SandboxFile `json:"files,omitempty"`
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
	Id          string            `json:"id"`
	NameRaw     string            `json:"name"`
	Description string            `json:"description"`
	Hints       SandboxHints      `json:"hints,omitempty"`
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

// Name returns the name if set, otherwise falls back to Id
func (c *SandboxConfig) Name() string {
	if c.NameRaw != "" {
		return c.NameRaw
	}
	return c.Id
}

// ParamEntrypoint returns the entrypoint file name with proper formatting
func (c *SandboxConfig) ParamEntrypoint() string {
	return strings.ReplaceAll(c.Entrypoint, ".", "_")
}

// Timeout returns the timeout as a time.Duration
func (c *SandboxConfig) Timeout() time.Duration {
	return time.Duration(c.TimeoutRaw) * time.Second
}

// RunCommand returns the initial command to run in the sandbox
func (c *SandboxConfig) RunCommand() []string {
	if len(c.Before) > 0 {
		return c.Before
	}
	return c.Command
}

// ExecCommand returns the command to execute in a running sandbox
func (c *SandboxConfig) ExecCommand() []string {
	if len(c.Before) > 0 {
		return c.Command
	}
	return nil
}

// Tty returns true if the sandbox should be run with a TTY
func (c *SandboxConfig) Tty() bool {
	return len(c.Before) > 0
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

		configs[config.Id] = &config
	}

	return configs, nil
}
