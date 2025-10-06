package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/stebje/gh-wf-to-mmd/parser"
	"github.com/stebje/gh-wf-to-mmd/renderer"
	"github.com/stebje/gh-wf-to-mmd/transformer"
)

func main() {
	// Define flags
	level := flag.String("level", "simple", "Detail level (simple)")
	output := flag.String("output", "-", "Output file path (- for stdout)")
	orientation := flag.String("orientation", "TB", "Flowchart orientation (TB or LR)")

	flag.Parse()

	// Get the workflow file path from arguments
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: gh wf-to-mmd [flags] <workflow-file.yml>")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	workflowFile := args[0]

	// Validate orientation
	var orient transformer.Orientation
	switch *orientation {
	case "TB":
		orient = transformer.TopToBottom
	case "LR":
		orient = transformer.LeftToRight
	default:
		fmt.Fprintf(os.Stderr, "Invalid orientation: %s (must be TB or LR)\n", *orientation)
		os.Exit(1)
	}

	// Validate detail level (currently only 'simple' is supported)
	if *level != "simple" {
		fmt.Fprintf(os.Stderr, "Invalid level: %s (currently only 'simple' is supported)\n", *level)
		os.Exit(1)
	}

	// Parse the workflow
	workflow, err := parser.ParseWorkflowFile(workflowFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing workflow: %v\n", err)
		os.Exit(1)
	}

	// Transform to Mermaid
	mermaidDiagram := transformer.ToMermaid(workflow, transformer.Simple, orient)

	// Output the result
	if err := renderer.Output(mermaidDiagram, *output); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	// Success message to stderr if writing to file
	if *output != "-" && *output != "" {
		fmt.Fprintf(os.Stderr, "Mermaid diagram written to: %s\n", *output)
	}
}
