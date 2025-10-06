package models

import "gopkg.in/yaml.v3"

// Workflow represents the GitHub Actions workflow structure
type Workflow struct {
	Name string         `yaml:"name"`
	Jobs map[string]Job `yaml:"jobs"`
}

// Job represents a single job in the workflow
type Job struct {
	RunsOn   string    `yaml:"runs-on"`
	NeedsRaw yaml.Node `yaml:"needs"`
	needs    []string  // parsed value
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
