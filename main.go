// Command gh-wf2mmd is a GitHub CLI extension that draws GitHub Actions
// workflows as Mermaid flowcharts.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/stebje/gh-wf2mmd/internal/mermaid"
	"github.com/stebje/gh-wf2mmd/internal/workflow"
)

const usage = `Draw a GitHub Actions workflow as a Mermaid flowchart.

Usage:
  gh wf2mmd [flags] <workflow-file>

Use - as <workflow-file> to read the workflow from stdin.

Flags:
  -o, --output <file>    Write to <file> instead of stdout. A .md or .markdown
                         file gets the diagram inside a mermaid code block.
  -l, --level <level>    Level of detail: simple or detailed (default "simple")
  -d, --direction <dir>  Flowchart direction: TB or LR (default "TB")
      --dry-run          Print what --output would contain without writing it
  -h, --help             Show this help

Examples:
  gh wf2mmd .github/workflows/ci.yml
  gh wf2mmd .github/workflows/ci.yml -o docs/ci.md
  gh wf2mmd .github/workflows/ci.yml --level detailed --direction LR
  gh api repos/OWNER/REPO/contents/.github/workflows/ci.yml \
    -H "Accept: application/vnd.github.raw" | gh wf2mmd -
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	var output, level, direction string
	var dryRun bool
	flags := flag.NewFlagSet("gh wf2mmd", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&output, "o", "", "")
	flags.StringVar(&output, "output", "", "")
	flags.StringVar(&level, "l", string(mermaid.Simple), "")
	flags.StringVar(&level, "level", string(mermaid.Simple), "")
	flags.StringVar(&direction, "d", "TB", "")
	flags.StringVar(&direction, "direction", "TB", "")
	flags.BoolVar(&dryRun, "dry-run", false, "")

	files, err := parseArgs(flags, args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		_, err = io.WriteString(stdout, usage)
		return err
	case err != nil:
		return usageError(err.Error())
	case len(files) != 1:
		return usageError("expected exactly one workflow file")
	}

	opts := mermaid.Options{
		Level:     mermaid.Level(strings.ToLower(level)),
		Direction: strings.ToUpper(direction),
	}
	if opts.Level != mermaid.Simple && opts.Level != mermaid.Detailed {
		return usageError(fmt.Sprintf("invalid level %q: use simple or detailed", level))
	}
	if opts.Direction != "TB" && opts.Direction != "LR" {
		return usageError(fmt.Sprintf("invalid direction %q: use TB or LR", direction))
	}

	input := files[0]
	var data []byte
	if input == "-" {
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(input)
	}
	if err != nil {
		return err
	}
	wf, err := workflow.Parse(data)
	if err != nil {
		if input == "-" {
			input = "stdin"
		}
		return fmt.Errorf("%s: %w", input, err)
	}
	diagram := mermaid.Flowchart(wf, opts)

	if output == "" || output == "-" {
		_, err = io.WriteString(stdout, diagram)
		return err
	}
	if sameFile(input, output) {
		return fmt.Errorf("refusing to overwrite the workflow file %s", output)
	}
	if isMarkdown(output) {
		diagram = "```mermaid\n" + diagram + "```\n"
	}
	if dryRun {
		if _, err := io.WriteString(stdout, diagram); err != nil {
			return err
		}
		fmt.Fprintf(stderr, "Dry run: %s was not written\n", output)
		return nil
	}
	if err := os.WriteFile(output, []byte(diagram), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "Wrote %s\n", output)
	return nil
}

// parseArgs allows flags both before and after the workflow file; the flag
// package on its own stops parsing at the first non-flag argument.
func parseArgs(flags *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := flags.Parse(args); err != nil {
			return nil, err
		}
		if flags.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, flags.Arg(0))
		args = flags.Args()[1:]
	}
}

func usageError(msg string) error {
	return fmt.Errorf("%s\nRun 'gh wf2mmd --help' for usage.", msg)
}

func isMarkdown(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

// sameFile reports whether a and b are the same existing file.
func sameFile(a, b string) bool {
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(infoA, infoB)
}
