package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the example diagrams in examples/")

const testWorkflow = `name: CI
jobs:
  build:
    runs-on: ubuntu-latest
  test:
    needs: build
    runs-on: ubuntu-latest
`

const testDiagram = `---
title: "CI"
---
flowchart LR
    build["build"]
    test["test"]
    build --> test
`

// TestExamples checks the committed example diagrams against the current
// output. Run `go test . -update` to regenerate them.
func TestExamples(t *testing.T) {
	inputs, err := filepath.Glob("examples/*.yml")
	if err != nil || len(inputs) == 0 {
		t.Fatalf("no example workflows found: %v", err)
	}
	for _, input := range inputs {
		for _, level := range []string{"simple", "detailed"} {
			golden := strings.TrimSuffix(input, ".yml") + ".md"
			if level != "simple" {
				golden = strings.TrimSuffix(input, ".yml") + "-" + level + ".md"
			}
			t.Run(golden, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				if err := run([]string{input, "--level", level, "-o", golden, "--dry-run"}, nil, &stdout, &stderr); err != nil {
					t.Fatal(err)
				}
				if *update {
					if err := os.WriteFile(golden, stdout.Bytes(), 0o644); err != nil {
						t.Fatal(err)
					}
					return
				}
				want, err := os.ReadFile(golden)
				if err != nil {
					t.Fatalf("%v (run `go test . -update` to create it)", err)
				}
				if got := stdout.String(); got != string(want) {
					t.Errorf("output differs from %s (run `go test . -update` if intended):\n%s", golden, got)
				}
			})
		}
	}
}

func TestRunOutput(t *testing.T) {
	dir := t.TempDir()
	input := writeFile(t, dir, "ci.yml", testWorkflow)
	fenced := "```mermaid\n" + testDiagram + "```\n"

	tests := []struct {
		name       string
		args       []string
		stdin      string
		wantStdout string
		wantFile   string // path of the file that should be written
		wantFileIn string // and its content
	}{
		{
			name:       "flags after the workflow file",
			args:       []string{input, "--direction", "lr"},
			wantStdout: testDiagram,
		},
		{
			name:       "workflow from stdin",
			args:       []string{"-d", "LR", "-"},
			stdin:      testWorkflow,
			wantStdout: testDiagram,
		},
		{
			name:       "explicit stdout",
			args:       []string{"-d", "LR", "-o", "-", input},
			wantStdout: testDiagram,
		},
		{
			name:       "mermaid file",
			args:       []string{"-d", "LR", "-o", filepath.Join(dir, "ci.mmd"), input},
			wantFile:   filepath.Join(dir, "ci.mmd"),
			wantFileIn: testDiagram,
		},
		{
			name:       "markdown file gets a code block",
			args:       []string{"-d", "LR", "--output", filepath.Join(dir, "ci.md"), input},
			wantFile:   filepath.Join(dir, "ci.md"),
			wantFileIn: fenced,
		},
		{
			name:       "dry run prints instead of writing",
			args:       []string{"-d", "LR", "-o", filepath.Join(dir, "dry.md"), "--dry-run", input},
			wantStdout: fenced,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if err := run(tt.args, strings.NewReader(tt.stdin), &stdout, &stderr); err != nil {
				t.Fatalf("run() error = %v", err)
			}
			if got := stdout.String(); got != tt.wantStdout {
				t.Errorf("stdout =\n%s\nwant\n%s", got, tt.wantStdout)
			}
			if tt.wantFile != "" {
				got, err := os.ReadFile(tt.wantFile)
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != tt.wantFileIn {
					t.Errorf("%s =\n%s\nwant\n%s", tt.wantFile, got, tt.wantFileIn)
				}
			}
		})
	}
	if _, err := os.Stat(filepath.Join(dir, "dry.md")); !os.IsNotExist(err) {
		t.Errorf("dry run wrote dry.md (stat error: %v)", err)
	}
}

func TestRunErrors(t *testing.T) {
	dir := t.TempDir()
	input := writeFile(t, dir, "ci.yml", testWorkflow)

	tests := []struct {
		name    string
		args    []string
		stdin   string
		wantErr string
	}{
		{"no workflow file", nil, "", "expected exactly one workflow file"},
		{"two workflow files", []string{input, input}, "", "expected exactly one workflow file"},
		{"unknown flag", []string{"--orientation", "LR", input}, "", "flag provided but not defined"},
		{"invalid level", []string{"--level", "verbose", input}, "", `invalid level "verbose"`},
		{"invalid direction", []string{"-d", "BT", input}, "", `invalid direction "BT"`},
		{"missing file", []string{filepath.Join(dir, "missing.yml")}, "", "no such file"},
		{"invalid workflow", []string{"-"}, "foo: bar\n", "stdin: no jobs found"},
		{"overwriting the workflow", []string{input, "-o", input}, "", "refusing to overwrite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(tt.args, strings.NewReader(tt.stdin), &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("run() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
	if got, _ := os.ReadFile(input); string(got) != testWorkflow {
		t.Errorf("workflow file was modified:\n%s", got)
	}
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--help"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage:\n  gh wf2mmd [flags] <workflow-file>") {
		t.Errorf("help output missing usage line:\n%s", stdout.String())
	}
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
