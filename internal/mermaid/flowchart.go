// Package mermaid draws workflows as Mermaid flowcharts.
package mermaid

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/stebje/gh-wf2mmd/internal/workflow"
)

// Level controls how much of a workflow the flowchart shows.
type Level string

const (
	// Simple shows jobs and the dependencies between them.
	Simple Level = "simple"
	// Detailed also shows triggers, conditions, runners, environments and steps.
	Detailed Level = "detailed"
)

// Options controls how a flowchart is drawn.
type Options struct {
	Level     Level
	Direction string // TB (top to bottom) or LR (left to right)
}

// Flowchart returns w as Mermaid flowchart source. The output is stable:
// nodes, edges and steps follow the order of the workflow file.
func Flowchart(w *workflow.Workflow, opts Options) string {
	var b strings.Builder
	if w.Name != "" {
		fmt.Fprintf(&b, "---\ntitle: %s\n---\n", strconv.Quote(w.Name))
	}
	fmt.Fprintf(&b, "flowchart %s\n", opts.Direction)

	detailed := opts.Level == Detailed
	ids := nodeIDs{}
	jobIDs := make(map[string]string, len(w.Jobs))
	for _, job := range w.Jobs {
		jobIDs[job.ID] = ids.add(job.ID)
	}

	var trigger string
	if detailed && len(w.Triggers) > 0 {
		trigger = ids.add("on")
		fmt.Fprintf(&b, "    %s([\"on: %s\"])\n", trigger, escape(strings.Join(w.Triggers, ", ")))
	}
	for _, job := range w.Jobs {
		fmt.Fprintf(&b, "    %s\n", jobNode(jobIDs[job.ID], job, detailed))
	}

	for _, job := range w.Jobs {
		// In detailed mode, dotted arrows lead to jobs that only run if their
		// condition holds.
		arrow := "-->"
		if detailed && job.If != "" {
			arrow = "-.->"
		}
		var from []string
		if trigger != "" && len(job.Needs) == 0 {
			from = append(from, trigger)
		}
		for _, need := range job.Needs {
			if id, ok := jobIDs[need]; ok {
				from = append(from, id)
			}
		}
		for _, id := range from {
			fmt.Fprintf(&b, "    %s %s %s\n", id, arrow, jobIDs[job.ID])
		}
	}
	return b.String()
}

// jobNode declares a job. Matrix jobs are drawn as stacked boxes and jobs that
// call a reusable workflow as subroutines.
func jobNode(id string, job workflow.Job, detailed bool) string {
	lines := []string{escape(job.DisplayName())}
	if job.Matrix {
		lines = append(lines, matrixText(job.MatrixSize))
	}
	if detailed {
		lines = append(lines, details(job)...)
	}
	label := strings.Join(lines, "<br>")

	switch {
	case job.Matrix:
		return fmt.Sprintf(`%s@{ shape: procs, label: "%s" }`, id, label)
	case job.Uses != "":
		return fmt.Sprintf(`%s[["%s"]]`, id, label)
	default:
		return fmt.Sprintf(`%s["%s"]`, id, label)
	}
}

func matrixText(size int) string {
	switch size {
	case 0:
		return "matrix"
	case 1:
		return "1 job"
	default:
		return fmt.Sprintf("%d jobs", size)
	}
}

// details lists what a job calls or runs on, when it runs, and its steps.
func details(job workflow.Job) []string {
	var lines []string
	add := func(key, value string) {
		if value != "" {
			lines = append(lines, key+": "+escape(clip(value)))
		}
	}
	add("uses", job.Uses)
	add("runs-on", job.RunsOn)
	add("environment", job.Environment)
	add("if", job.If)
	for i, step := range job.Steps {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, escape(clip(stepText(step)))))
	}
	return lines
}

// stepText describes a step by its name, the action it uses, or the first
// line of its script, in that order of preference.
func stepText(step workflow.Step) string {
	switch {
	case step.Name != "":
		return step.Name
	case step.Uses != "":
		return step.Uses
	}
	first, rest, multiline := strings.Cut(strings.TrimSpace(step.Run), "\n")
	if multiline && strings.TrimSpace(rest) != "" {
		return first + " …"
	}
	return first
}

// maxText keeps long conditions and scripts from making nodes very wide.
const maxText = 50

func clip(s string) string {
	if utf8.RuneCountInString(s) <= maxText {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:maxText-1])) + "…"
}

// labelEscaper replaces characters that could end a quoted label or be read
// as markup with Mermaid entity codes, which render as the original text.
var labelEscaper = strings.NewReplacer(
	"#", "#35;",
	`"`, "#quot;",
	"<", "#lt;",
	">", "#gt;",
	`\`, "#92;",
	"`", "#96;",
)

func escape(s string) string {
	return labelEscaper.Replace(strings.Join(strings.Fields(s), " "))
}

// reserved holds words Mermaid can't parse as node IDs.
var reserved = map[string]bool{
	"end": true, "graph": true, "flowchart": true, "subgraph": true,
	"style": true, "linkstyle": true, "classdef": true, "class": true,
	"click": true, "call": true, "href": true, "interpolate": true,
	"_self": true, "_blank": true, "_parent": true, "_top": true,
}

// nodeIDs hands out node IDs that are unique and safe to use in Mermaid.
type nodeIDs map[string]bool

// add derives a node ID from name. Characters other than ASCII letters, digits
// and underscores become underscores (otherwise "end-to-end" would be read as
// the keyword "end"), and a numeric suffix avoids keywords and duplicates.
func (ids nodeIDs) add(name string) string {
	base := strings.Map(func(r rune) rune {
		if r == '_' || 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' {
			return r
		}
		return '_'
	}, name)
	id := base
	for n := 2; id == "" || ids[id] || reserved[strings.ToLower(id)]; n++ {
		id = fmt.Sprintf("%s_%d", base, n)
	}
	ids[id] = true
	return id
}
