package models

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Workflow represents the GitHub Actions workflow structure
type Workflow struct {
	Name string         `yaml:"name"`
	Jobs map[string]Job `yaml:"jobs"`
}

// Job represents a single job in the workflow
type Job struct {
	RunsOn   string    `yaml:"runs-on"`
	NeedsRaw yaml.Node `yaml:"needs"`
	Strategy Strategy  `yaml:"strategy"`
	needs    []string  // parsed value
}

// Strategy represents the strategy configuration for a job
type Strategy struct {
	Matrix map[string]interface{} `yaml:"matrix"`
}

// HasMatrix returns true if the job uses a matrix strategy
func (j *Job) HasMatrix() bool {
	return len(j.Strategy.Matrix) > 0
}

// GetMatrixDimensions returns a string describing the matrix dimensions
// e.g., "3x2" for 3 node versions and 2 operating systems
func (j *Job) GetMatrixDimensions() string {
	if !j.HasMatrix() {
		return ""
	}

	var dimensions []int
	for _, value := range j.Strategy.Matrix {
		switch v := value.(type) {
		case []interface{}:
			dimensions = append(dimensions, len(v))
		}
	}

	if len(dimensions) == 0 {
		return ""
	}

	// Calculate total combinations
	total := 1
	for _, dim := range dimensions {
		total *= dim
	}

	return fmt.Sprintf("%d jobs", total)
}

// GetNeeds returns the job dependencies as a slice
// Handles both single string and array formats in YAML
func (j *Job) GetNeeds() []string {
	if j.needs != nil {
		return j.needs
	}

	// Parse the needs field (can be string or []string)
	switch j.NeedsRaw.Kind {
	case yaml.ScalarNode:
		j.needs = []string{j.NeedsRaw.Value}
	case yaml.SequenceNode:
		j.needs = make([]string, len(j.NeedsRaw.Content))
		for i, node := range j.NeedsRaw.Content {
			j.needs[i] = node.Value
		}
	default:
		j.needs = []string{}
	}

	return j.needs
}
