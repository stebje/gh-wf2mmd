package workflow

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	src := `
name: CI
on:
  push:
    branches: [main]
  workflow_dispatch:
jobs:
  build:
    name: Build app
    runs-on: [self-hosted, linux]
    steps:
      - name: Check out
        uses: actions/checkout@v4
      - run: make build
  test:
    needs: build
    runs-on:
      group: large-runners
      labels: [linux, x64]
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest]
        node: [20, 22]
  deploy:
    needs: [Build, test]
    if: ${{ github.ref == 'refs/heads/main' }}
    environment:
      name: production
      url: https://example.com
    runs-on: ubuntu-latest
  release:
    needs: deploy
    uses: ./.github/workflows/release.yml
`
	want := &Workflow{
		Name:     "CI",
		Triggers: []string{"push", "workflow_dispatch"},
		Jobs: []Job{
			{
				ID: "build", Name: "Build app", RunsOn: "self-hosted, linux",
				Steps: []Step{{Name: "Check out", Uses: "actions/checkout@v4"}, {Run: "make build"}},
			},
			{ID: "test", Needs: []string{"build"}, RunsOn: "large-runners, linux, x64", Matrix: true, MatrixSize: 4},
			{
				ID: "deploy", Needs: []string{"build", "test"}, If: "github.ref == 'refs/heads/main'",
				Environment: "production", RunsOn: "ubuntu-latest",
			},
			{ID: "release", Needs: []string{"deploy"}, Uses: "./.github/workflows/release.yml"},
		},
	}

	got, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestParseFieldForms(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		check func(*testing.T, *Workflow)
	}{
		{
			name: "single trigger",
			src:  "on: push\njobs:\n  a:\n    runs-on: x\n",
			check: func(t *testing.T, w *Workflow) {
				assertEqual(t, w.Triggers, []string{"push"})
			},
		},
		{
			name: "trigger list",
			src:  "on: [push, pull_request]\njobs:\n  a:\n    runs-on: x\n",
			check: func(t *testing.T, w *Workflow) {
				assertEqual(t, w.Triggers, []string{"push", "pull_request"})
			},
		},
		{
			name: "jobs keep file order",
			src:  "jobs:\n  zeta: {}\n  alpha: {}\n  mid: {}\n",
			check: func(t *testing.T, w *Workflow) {
				var ids []string
				for _, job := range w.Jobs {
					ids = append(ids, job.ID)
				}
				assertEqual(t, ids, []string{"zeta", "alpha", "mid"})
			},
		},
		{
			name: "non-string scalars and plain environment",
			src:  "jobs:\n  a:\n    name: 123\n    if: false\n    environment: staging\n",
			check: func(t *testing.T, w *Workflow) {
				job := w.Jobs[0]
				assertEqual(t, []string{job.Name, job.If, job.Environment}, []string{"123", "false", "staging"})
			},
		},
		{
			name: "condition with several expressions is kept as is",
			src:  "jobs:\n  a:\n    if: ${{ a }} && ${{ b }}\n",
			check: func(t *testing.T, w *Workflow) {
				assertEqual(t, w.Jobs[0].If, "${{ a }} && ${{ b }}")
			},
		},
		{
			name: "dynamic matrix",
			src:  "jobs:\n  a:\n    strategy:\n      matrix: ${{ fromJSON(needs.setup.outputs.matrix) }}\n",
			check: func(t *testing.T, w *Workflow) {
				assertEqual(t, []any{w.Jobs[0].Matrix, w.Jobs[0].MatrixSize}, []any{true, 0})
			},
		},
		{
			name: "empty matrix is no matrix",
			src:  "jobs:\n  a:\n    strategy:\n      matrix:\n",
			check: func(t *testing.T, w *Workflow) {
				assertEqual(t, w.Jobs[0].Matrix, false)
			},
		},
		{
			name: "anchors and aliases",
			src: `jobs:
  build:
    runs-on: &runner ubuntu-latest
  test:
    runs-on: *runner
    needs: &deps [build]
  lint:
    runs-on: *runner
    needs: *deps
`,
			check: func(t *testing.T, w *Workflow) {
				for _, job := range w.Jobs {
					assertEqual(t, job.RunsOn, "ubuntu-latest")
				}
				assertEqual(t, w.Jobs[2].Needs, []string{"build"})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := Parse([]byte(tt.src))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			tt.check(t, w)
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{"invalid YAML", "jobs: [", "yaml:"},
		{"empty file", "", "no jobs found"},
		{"not a workflow", "runs:\n  using: node20\n", "no jobs found"},
		{"empty jobs", "jobs: {}\n", "no jobs found"},
		{"jobs is a list", "jobs: [build]\n", "no jobs found"},
		{"unknown need", "jobs:\n  build: {}\n  test:\n    needs: buld\n", `job "test" needs unknown job "buld"`},
		{"malformed job", "jobs:\n  build:\n    steps: oops\n", `job "build"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.src))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Parse() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}
