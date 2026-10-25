// Package workflow reads the parts of a GitHub Actions workflow file that are
// needed to draw it.
package workflow

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Workflow is the subset of a GitHub Actions workflow that gets drawn.
type Workflow struct {
	Name     string
	Triggers []string // event names under `on`
	Jobs     []Job    // in file order
}

// Job is an entry under `jobs`.
type Job struct {
	ID          string
	Name        string // empty when the job has no display name
	Needs       []string
	If          string // without the optional ${{ }} wrapper
	RunsOn      string
	Environment string
	Uses        string // set when the job calls a reusable workflow
	Matrix      bool
	MatrixSize  int // jobs the matrix expands to; 0 if only known at runtime
	Steps       []Step
}

// Step is an entry under a job's `steps`.
type Step struct {
	Name string `yaml:"name"`
	Uses string `yaml:"uses"`
	Run  string `yaml:"run"`
}

// DisplayName returns the name GitHub shows for the job.
func (j Job) DisplayName() string {
	if j.Name != "" {
		return j.Name
	}
	return j.ID
}

// The raw types mirror the YAML layout. Fields that GitHub allows in several
// forms are kept as nodes and normalized when building a Job.
type rawWorkflow struct {
	Name string    `yaml:"name"`
	On   yaml.Node `yaml:"on"`
	Jobs yaml.Node `yaml:"jobs"`
}

type rawJob struct {
	Name        string      `yaml:"name"`
	Needs       yaml.Node   `yaml:"needs"`
	If          string      `yaml:"if"`
	RunsOn      yaml.Node   `yaml:"runs-on"`
	Environment yaml.Node   `yaml:"environment"`
	Uses        string      `yaml:"uses"`
	Strategy    rawStrategy `yaml:"strategy"`
	Steps       []Step      `yaml:"steps"`
}

type rawStrategy struct {
	Matrix yaml.Node `yaml:"matrix"`
}

// Parse decodes a workflow and checks that it can be drawn: it must have at
// least one job, and every `needs` entry must refer to one of them.
func Parse(data []byte) (*Workflow, error) {
	var doc rawWorkflow
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	jobs := resolve(&doc.Jobs)
	if jobs.Kind != yaml.MappingNode || len(jobs.Content) == 0 {
		return nil, errors.New("no jobs found; is this a GitHub Actions workflow?")
	}

	w := &Workflow{Name: doc.Name, Triggers: triggers(&doc.On)}
	for i := 0; i+1 < len(jobs.Content); i += 2 {
		job, err := parseJob(jobs.Content[i].Value, jobs.Content[i+1])
		if err != nil {
			return nil, err
		}
		w.Jobs = append(w.Jobs, job)
	}
	if err := resolveNeeds(w.Jobs); err != nil {
		return nil, err
	}
	return w, nil
}

func parseJob(id string, node *yaml.Node) (Job, error) {
	var raw rawJob
	if err := node.Decode(&raw); err != nil {
		return Job{}, fmt.Errorf("job %q: %w", id, err)
	}

	job := Job{
		ID:          id,
		Name:        raw.Name,
		Needs:       scalars(&raw.Needs),
		If:          condition(raw.If),
		RunsOn:      strings.Join(scalars(&raw.RunsOn), ", "),
		Environment: environment(&raw.Environment),
		Uses:        raw.Uses,
		Steps:       raw.Steps,
	}
	if matrix := resolve(&raw.Strategy.Matrix); matrix.Kind != 0 && matrix.ShortTag() != "!!null" {
		job.Matrix = true
		job.MatrixSize = matrixSize(matrix)
	}
	return job, nil
}

// resolveNeeds checks that every `needs` entry names a job and rewrites it to
// that job's ID. Like GitHub, it matches job IDs case-insensitively.
func resolveNeeds(jobs []Job) error {
	ids := make(map[string]string, len(jobs))
	for _, job := range jobs {
		ids[strings.ToLower(job.ID)] = job.ID
	}
	for j := range jobs {
		for i, need := range jobs[j].Needs {
			id, ok := ids[strings.ToLower(need)]
			if !ok {
				return fmt.Errorf("job %q needs unknown job %q", jobs[j].ID, need)
			}
			jobs[j].Needs[i] = id
		}
	}
	return nil
}

// triggers returns the event names from `on`, which can be a single event, a
// list of events, or a mapping from events to their settings.
func triggers(n *yaml.Node) []string {
	n = resolve(n)
	if n.Kind != yaml.MappingNode {
		return scalars(n)
	}
	events := make([]string, 0, len(n.Content)/2)
	for i := 0; i < len(n.Content); i += 2 {
		events = append(events, n.Content[i].Value)
	}
	return events
}

// environment returns the name from `environment: prod` or
// `environment: {name: prod, url: ...}`.
func environment(n *yaml.Node) string {
	n = resolve(n)
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == "name" {
				return scalar(n.Content[i+1])
			}
		}
	}
	return scalar(n)
}

// condition strips the optional ${{ }} around an `if` expression.
func condition(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "${{") && strings.HasSuffix(s, "}}") && strings.Count(s, "${{") == 1 {
		s = strings.TrimSpace(s[3 : len(s)-2])
	}
	return s
}

// scalars flattens a node into its scalar values in document order, so that
// `needs: a`, `needs: [a, b]` and `runs-on: {group: g, labels: [l]}` can all
// be read the same way.
func scalars(n *yaml.Node) []string {
	n = resolve(n)
	var values []string
	switch n.Kind {
	case yaml.ScalarNode:
		if s := scalar(n); s != "" {
			values = append(values, s)
		}
	case yaml.SequenceNode:
		for _, child := range n.Content {
			values = append(values, scalars(child)...)
		}
	case yaml.MappingNode:
		for i := 1; i < len(n.Content); i += 2 {
			values = append(values, scalars(n.Content[i])...)
		}
	}
	return values
}

// scalar returns the value of a scalar node, or "" for null and non-scalars.
func scalar(n *yaml.Node) string {
	n = resolve(n)
	if n.Kind != yaml.ScalarNode || n.ShortTag() == "!!null" {
		return ""
	}
	return n.Value
}

// resolve follows YAML aliases (*name) to the anchored node.
func resolve(n *yaml.Node) *yaml.Node {
	for n.Kind == yaml.AliasNode && n.Alias != nil {
		n = n.Alias
	}
	return n
}
