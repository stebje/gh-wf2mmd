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
		// First pass: declare all nodes (jobs with no dependencies or matrix jobs need explicit declaration)
		for jobID := range workflow.Jobs {
			job := workflow.Jobs[jobID]
			needs := job.GetNeeds()

			if len(needs) == 0 {
				// Job with no dependencies - declare it with appropriate shape
				builder.WriteString(formatJobNode(jobID, job))
			}
		}

		// Second pass: create arrows for dependencies
		for jobID := range workflow.Jobs {
			job := workflow.Jobs[jobID]
			needs := job.GetNeeds()

			if len(needs) > 0 {
				// Job with dependencies - create arrows
				for _, dependency := range needs {
					builder.WriteString(fmt.Sprintf("    %s --> %s\n", dependency, formatJobID(jobID, workflow.Jobs[jobID])))
				}
			}
		}
	}

	return builder.String()
}

// formatJobNode formats a job node declaration with appropriate shape
func formatJobNode(jobID string, job models.Job) string {
	if job.HasMatrix() {
		matrixInfo := job.GetMatrixDimensions()
		// Use procs shape for matrix jobs (layered box)
		return fmt.Sprintf("    %s@{ shape: procs, label: \"%s\\n(%s)\"}\n", jobID, jobID, matrixInfo)
	}
	// Regular job - simple node
	return fmt.Sprintf("    %s\n", jobID)
}

// formatJobID formats a job ID for use in arrows
func formatJobID(jobID string, job models.Job) string {
	// Matrix jobs need special handling in node references
	if job.HasMatrix() {
		return jobID
	}
	return jobID
}
