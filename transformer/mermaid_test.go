package transformer

import (
	"strings"
	"testing"

	"github.com/stebje/gh-wf-to-mmd/models"
	"gopkg.in/yaml.v3"
)

func TestToMermaid(t *testing.T) {
	tests := []struct {
		name        string
		workflow    *models.Workflow
		level       DetailLevel
		orientation Orientation
		want        string
	}{
		{
			name: "simple linear workflow - TB",
			workflow: &models.Workflow{
				Name: "Simple",
				Jobs: map[string]models.Job{
					"build":  {RunsOn: "ubuntu-latest"},
					"test":   {RunsOn: "ubuntu-latest", NeedsRaw: stringToYAMLNode("build")},
					"deploy": {RunsOn: "ubuntu-latest", NeedsRaw: stringToYAMLNode("test")},
				},
			},
			level:       Simple,
			orientation: TopToBottom,
			want: `flowchart TB
    build
    build --> test
    test --> deploy
`,
		},
		{
			name: "simple linear workflow - LR",
			workflow: &models.Workflow{
				Name: "Simple",
				Jobs: map[string]models.Job{
					"build":  {RunsOn: "ubuntu-latest"},
					"test":   {RunsOn: "ubuntu-latest", NeedsRaw: stringToYAMLNode("build")},
					"deploy": {RunsOn: "ubuntu-latest", NeedsRaw: stringToYAMLNode("test")},
				},
			},
			level:       Simple,
			orientation: LeftToRight,
			want: `flowchart LR
    build
    build --> test
    test --> deploy
`,
		},
		{
			name: "parallel jobs with fan-in",
			workflow: &models.Workflow{
				Name: "Parallel",
				Jobs: map[string]models.Job{
					"lint":     {RunsOn: "ubuntu-latest"},
					"test":     {RunsOn: "ubuntu-latest"},
					"security": {RunsOn: "ubuntu-latest"},
					"build":    {RunsOn: "ubuntu-latest", NeedsRaw: arrayToYAMLNode([]string{"lint", "test", "security"})},
				},
			},
			level:       Simple,
			orientation: TopToBottom,
			want: `flowchart TB
    lint
    test
    security
    lint --> build
    test --> build
    security --> build
`,
		},
		{
			name: "single job no dependencies",
			workflow: &models.Workflow{
				Name: "Single",
				Jobs: map[string]models.Job{
					"build": {RunsOn: "ubuntu-latest"},
				},
			},
			level:       Simple,
			orientation: TopToBottom,
			want: `flowchart TB
    build
`,
		},
		{
			name: "multiple independent jobs",
			workflow: &models.Workflow{
				Name: "Independent",
				Jobs: map[string]models.Job{
					"job1": {RunsOn: "ubuntu-latest"},
					"job2": {RunsOn: "ubuntu-latest"},
					"job3": {RunsOn: "ubuntu-latest"},
				},
			},
			level:       Simple,
			orientation: TopToBottom,
			want: `flowchart TB
    job1
    job2
    job3
`,
		},
		{
			name: "matrix job with dependencies",
			workflow: &models.Workflow{
				Name: "Matrix",
				Jobs: map[string]models.Job{
					"test": {
						RunsOn: "ubuntu-latest",
						Strategy: models.Strategy{
							Matrix: map[string]interface{}{
								"node-version": []interface{}{18, 20, 22},
								"os":           []interface{}{"ubuntu-latest", "windows-latest"},
							},
						},
					},
					"build": {
						RunsOn:   "ubuntu-latest",
						NeedsRaw: stringToYAMLNode("test"),
					},
				},
			},
			level:       Simple,
			orientation: TopToBottom,
			want: `flowchart TB
    test@{ shape: procs, label: "test\n(6 jobs)"}
    test --> build
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToMermaid(tt.workflow, tt.level, tt.orientation)

			// Since map iteration order is non-deterministic, we need to check content, not exact order
			// We'll verify that all expected lines exist
			gotLines := strings.Split(strings.TrimSpace(got), "\n")
			wantLines := strings.Split(strings.TrimSpace(tt.want), "\n")

			if len(gotLines) != len(wantLines) {
				t.Errorf("ToMermaid() line count mismatch:\nGot %d lines:\n%s\n\nWant %d lines:\n%s",
					len(gotLines), got, len(wantLines), tt.want)
				return
			}

			// Check header is correct
			if strings.TrimSpace(gotLines[0]) != strings.TrimSpace(wantLines[0]) {
				t.Errorf("ToMermaid() header mismatch:\nGot:  %s\nWant: %s", gotLines[0], wantLines[0])
			}

			// Check all expected lines exist (ignoring order)
			gotSet := make(map[string]bool)
			for _, line := range gotLines[1:] {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					gotSet[trimmed] = true
				}
			}

			for _, line := range wantLines[1:] {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" && !gotSet[trimmed] {
					t.Errorf("ToMermaid() missing line: %s\nGot:\n%s", trimmed, got)
				}
			}
		})
	}
}

// Helper function to create a YAML node from a string
func stringToYAMLNode(s string) yaml.Node {
	var node yaml.Node
	node.Kind = yaml.ScalarNode
	node.Value = s
	return node
}

// Helper function to create a YAML node from a string array
func arrayToYAMLNode(arr []string) yaml.Node {
	var node yaml.Node
	node.Kind = yaml.SequenceNode
	node.Content = make([]*yaml.Node, len(arr))
	for i, s := range arr {
		node.Content[i] = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: s,
		}
	}
	return node
}
