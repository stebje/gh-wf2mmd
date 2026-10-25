package mermaid

import (
	"strings"
	"testing"

	"github.com/stebje/gh-wf2mmd/internal/workflow"
)

func TestFlowchart(t *testing.T) {
	detailedWorkflow := &workflow.Workflow{
		Triggers: []string{"push", "workflow_dispatch"},
		Jobs: []workflow.Job{
			{ID: "build", RunsOn: "ubuntu-latest", Steps: []workflow.Step{
				{Uses: "actions/checkout@v4"},
				{Name: "Build", Run: "make"},
				{Run: "npm ci\nnpm test\n"},
				{Run: "echo " + strings.Repeat("x", 60)},
			}},
			{
				ID: "deploy", Needs: []string{"build"}, If: "github.ref == 'refs/heads/main'",
				Environment: "production", RunsOn: "ubuntu-latest",
			},
			{ID: "notify", If: "always()", Uses: "./.github/workflows/notify.yml"},
		},
	}

	tests := []struct {
		name string
		wf   *workflow.Workflow
		opts Options
		want []string
	}{
		{
			name: "jobs in file order with title",
			wf: &workflow.Workflow{Name: "CI", Jobs: []workflow.Job{
				{ID: "build"},
				{ID: "test", Needs: []string{"build"}},
				{ID: "deploy", Needs: []string{"test"}},
			}},
			opts: Options{Level: Simple, Direction: "TB"},
			want: []string{
				`---`,
				`title: "CI"`,
				`---`,
				`flowchart TB`,
				`    build["build"]`,
				`    test["test"]`,
				`    deploy["deploy"]`,
				`    build --> test`,
				`    test --> deploy`,
			},
		},
		{
			name: "fan-in left to right without title",
			wf: &workflow.Workflow{Jobs: []workflow.Job{
				{ID: "lint"},
				{ID: "test"},
				{ID: "build", Needs: []string{"lint", "test"}},
			}},
			opts: Options{Level: Simple, Direction: "LR"},
			want: []string{
				`flowchart LR`,
				`    lint["lint"]`,
				`    test["test"]`,
				`    build["build"]`,
				`    lint --> build`,
				`    test --> build`,
			},
		},
		{
			name: "shapes for matrix and reusable workflow jobs",
			wf: &workflow.Workflow{Jobs: []workflow.Job{
				{ID: "test", Name: "Test", Matrix: true, MatrixSize: 6},
				{ID: "one", Matrix: true, MatrixSize: 1},
				{ID: "dynamic", Matrix: true},
				{ID: "release", Uses: "./.github/workflows/release.yml"},
			}},
			opts: Options{Level: Simple, Direction: "TB"},
			want: []string{
				`flowchart TB`,
				`    test@{ shape: procs, label: "Test<br>6 jobs" }`,
				`    one@{ shape: procs, label: "one<br>1 job" }`,
				`    dynamic@{ shape: procs, label: "dynamic<br>matrix" }`,
				`    release[["release"]]`,
			},
		},
		{
			name: "node IDs avoid keywords and collisions",
			wf: &workflow.Workflow{Jobs: []workflow.Job{
				{ID: "end-to-end"},
				{ID: "end"},
				{ID: "a-b"},
				{ID: "a_b"},
				{ID: "style", Needs: []string{"end-to-end", "end", "a-b", "a_b"}},
			}},
			opts: Options{Level: Simple, Direction: "TB"},
			want: []string{
				`flowchart TB`,
				`    end_to_end["end-to-end"]`,
				`    end_2["end"]`,
				`    a_b["a-b"]`,
				`    a_b_2["a_b"]`,
				`    style_2["style"]`,
				`    end_to_end --> style_2`,
				`    end_2 --> style_2`,
				`    a_b --> style_2`,
				`    a_b_2 --> style_2`,
			},
		},
		{
			name: "labels are escaped",
			wf: &workflow.Workflow{Name: `Say "hi"`, Jobs: []workflow.Job{
				{ID: "a", Name: "Say \"hi\" <now>\n#1 \\ `x`"},
			}},
			opts: Options{Level: Simple, Direction: "TB"},
			want: []string{
				`---`,
				`title: "Say \"hi\""`,
				`---`,
				`flowchart TB`,
				`    a["Say #quot;hi#quot; #lt;now#gt; #35;1 #92; #96;x#96;"]`,
			},
		},
		{
			name: "simple level leaves out details",
			wf:   detailedWorkflow,
			opts: Options{Level: Simple, Direction: "TB"},
			want: []string{
				`flowchart TB`,
				`    build["build"]`,
				`    deploy["deploy"]`,
				`    notify[["notify"]]`,
				`    build --> deploy`,
			},
		},
		{
			name: "detailed level",
			wf:   detailedWorkflow,
			opts: Options{Level: Detailed, Direction: "TB"},
			want: []string{
				`flowchart TB`,
				`    on(["on: push, workflow_dispatch"])`,
				`    build["build<br>runs-on: ubuntu-latest<br>1. actions/checkout@v4<br>2. Build<br>3. npm ci …<br>4. echo ` + strings.Repeat("x", 44) + `…"]`,
				`    deploy["deploy<br>runs-on: ubuntu-latest<br>environment: production<br>if: github.ref == 'refs/heads/main'"]`,
				`    notify[["notify<br>uses: ./.github/workflows/notify.yml<br>if: always()"]]`,
				`    on --> build`,
				`    build -.-> deploy`,
				`    on -.-> notify`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := strings.Join(tt.want, "\n") + "\n"
			if got := Flowchart(tt.wf, tt.opts); got != want {
				t.Errorf("Flowchart() =\n%s\nwant\n%s", got, want)
			}
		})
	}
}
