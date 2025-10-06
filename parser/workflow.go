package parser

import (
	"fmt"
	"os"

	"github.com/stebje/gh-wf-to-mmd/models"
	"gopkg.in/yaml.v3"
)

// ParseWorkflowFile reads and parses a GitHub Actions workflow YAML file
func ParseWorkflowFile(filepath string) (*models.Workflow, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Declare workflow var with zero values
	var workflow models.Workflow

	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &workflow, nil
}
