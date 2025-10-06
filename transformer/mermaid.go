package transformer

import (
	"fmt"
	"strings"

	"github.com/stebje/gh-wf-to-mmd/models"
)

// DetailLevel represents the level of detail in the diagram
type DetailLevel string

const (
	Simple DetailLevel = "simple"
	// Normal and Detailed will be added later
)

// Orientation represents the flowchart direction
type Orientation string

const (
	TopToBottom Orientation = "TB"
	LeftToRight Orientation = "LR"
)

// ToMermaid converts a workflow to a Mermaid flowchart diagram
func ToMermaid(workflow *models.Workflow, level DetailLevel, orientation Orientation) string {
	var builder strings.Builder

	// Write flowchart header
	builder.WriteString(fmt.Sprintf("flowchart %s\n", orientation))

	// For simple mode: just job names and dependencies
	if level == Simple {
		// Build the graph
		for jobID := range workflow.Jobs {
			job := workflow.Jobs[jobID]
			needs := job.GetNeeds()

			if len(needs) == 0 {
				// Job with no dependencies - just declare it
				builder.WriteString(fmt.Sprintf("    %s\n", jobID))
			} else {
				// Job with dependencies - create arrows
				for _, dependency := range needs {
					builder.WriteString(fmt.Sprintf("    %s --> %s\n", dependency, jobID))
				}
			}
		}
	}

	return builder.String()
}
