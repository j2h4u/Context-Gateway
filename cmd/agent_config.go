package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/compresr/context-gateway/internal/config"
	"gopkg.in/yaml.v3"
)

// AgentConfig is the top-level agent YAML structure.
type AgentConfig struct {
	Agent AgentSpec `yaml:"agent"`
}

// AgentSpec defines an agent's properties.
type AgentSpec struct {
	Name         string        `yaml:"name"`
	DisplayName  string        `yaml:"display_name"`
	Description  string        `yaml:"description"`
	Models       []AgentModel  `yaml:"models"`
	DefaultModel string        `yaml:"default_model"`
	Environment  []AgentEnvVar `yaml:"environment"`
	Command      AgentCommand  `yaml:"command"`
}

// AgentModel defines a selectable model for agents like OpenClaw.
type AgentModel struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Provider string `yaml:"provider"`
}

// AgentEnvVar defines an environment variable to export.
type AgentEnvVar struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// AgentCommand defines how to check, run, and install the agent.
type AgentCommand struct {
	Check           string   `yaml:"check"`
	Run             string   `yaml:"run"`
	Args            []string `yaml:"args"`
	Install         string   `yaml:"install"`
	FallbackMessage string   `yaml:"fallback_message"`
}

// parseAgentConfig parses agent YAML bytes into an AgentConfig.
// Environment variable references in values are expanded.
func parseAgentConfig(data []byte) (*AgentConfig, error) {
	// Expand env vars in the YAML before parsing
	expanded := config.ExpandEnvWithDefaults(string(data))

	var ac AgentConfig
	if err := yaml.Unmarshal([]byte(expanded), &ac); err != nil {
		return nil, fmt.Errorf("failed to parse agent config: %w", err)
	}

	if ac.Agent.Name == "" {
		return nil, fmt.Errorf("agent.name is required")
	}

	return &ac, nil
}

// loadAgentConfig loads an agent config by name.
// It checks filesystem locations in order of priority, then falls back to embedded.
func loadAgentConfig(name string) (*AgentConfig, []byte, error) {
	// Ensure no extension in name for lookup
	name = strings.TrimSuffix(name, ".yaml")

	// Check filesystem override locations
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		overridePath := filepath.Join(homeDir, ".config", "context-gateway", "agents", name+".yaml")
		if data, err := os.ReadFile(overridePath); err == nil {
			ac, err := parseAgentConfig(data)
			return ac, data, err
		}
	}

	// Check local agents directory
	localPath := filepath.Join("agents", name+".yaml")
	if data, err := os.ReadFile(localPath); err == nil {
		ac, err := parseAgentConfig(data)
		return ac, data, err
	}

	// Fall back to embedded agent
	if data, err := getEmbeddedAgent(name); err == nil {
		ac, err := parseAgentConfig(data)
		return ac, data, err
	}

	return nil, nil, fmt.Errorf("agent '%s' not found", name)
}
