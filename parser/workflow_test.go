package parser

import (
	"os"
	"testing"

	"github.com/stebje/gh-wf-to-mmd/models"
)

func TestParseWorkflowFile(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		validate    func(*testing.T, *models.Workflow)
	}{
		{
			name: "simple workflow with sequential jobs",
			yamlContent: `name: Test Workflow
jobs:
  build:
    runs-on: ubuntu-latest
  test:
    runs-on: ubuntu-latest
    needs: build
  deploy:
    runs-on: ubuntu-latest
    needs: test
`,
			wantErr: false,
			validate: func(t *testing.T, w *models.Workflow) {
				if w.Name != "Test Workflow" {
					t.Errorf("expected name 'Test Workflow', got '%s'", w.Name)
				}
				if len(w.Jobs) != 3 {
					t.Errorf("expected 3 jobs, got %d", len(w.Jobs))
				}

				// Check build job has no dependencies
				build := w.Jobs["build"]
				buildNeeds := build.GetNeeds()
				if len(buildNeeds) != 0 {
					t.Errorf("expected build to have 0 dependencies, got %d", len(buildNeeds))
				}

				// Check test job depends on build
				test := w.Jobs["test"]
				testNeeds := test.GetNeeds()
				if len(testNeeds) != 1 || testNeeds[0] != "build" {
					t.Errorf("expected test to depend on [build], got %v", testNeeds)
				}

				// Check deploy job depends on test
				deploy := w.Jobs["deploy"]
				deployNeeds := deploy.GetNeeds()
				if len(deployNeeds) != 1 || deployNeeds[0] != "test" {
					t.Errorf("expected deploy to depend on [test], got %v", deployNeeds)
				}
			},
		},
		{
			name: "workflow with multiple dependencies (array syntax)",
			yamlContent: `name: Complex Workflow
jobs:
  lint:
    runs-on: ubuntu-latest
  test:
    runs-on: ubuntu-latest
  build:
    runs-on: ubuntu-latest
    needs: [lint, test]
`,
			wantErr: false,
			validate: func(t *testing.T, w *models.Workflow) {
				build := w.Jobs["build"]
				buildNeeds := build.GetNeeds()
				if len(buildNeeds) != 2 {
					t.Errorf("expected build to have 2 dependencies, got %d", len(buildNeeds))
				}
				// Check both dependencies exist (order may vary in maps)
				needsMap := make(map[string]bool)
				for _, need := range buildNeeds {
					needsMap[need] = true
				}
				if !needsMap["lint"] || !needsMap["test"] {
					t.Errorf("expected build to depend on lint and test, got %v", buildNeeds)
				}
			},
		},
		{
			name: "workflow with single dependency (string syntax)",
			yamlContent: `name: String Need
jobs:
  build:
    runs-on: ubuntu-latest
  test:
    runs-on: ubuntu-latest
    needs: build
`,
			wantErr: false,
			validate: func(t *testing.T, w *models.Workflow) {
				test := w.Jobs["test"]
				testNeeds := test.GetNeeds()
				if len(testNeeds) != 1 || testNeeds[0] != "build" {
					t.Errorf("expected test to depend on [build], got %v", testNeeds)
				}
			},
		},
		{
			name: "workflow with no dependencies",
			yamlContent: `name: Parallel Jobs
jobs:
  job1:
    runs-on: ubuntu-latest
  job2:
    runs-on: ubuntu-latest
  job3:
    runs-on: ubuntu-latest
`,
			wantErr: false,
			validate: func(t *testing.T, w *models.Workflow) {
				for jobName, job := range w.Jobs {
					needs := job.GetNeeds()
					if len(needs) != 0 {
						t.Errorf("expected %s to have no dependencies, got %v", jobName, needs)
					}
				}
			},
		},
		{
			name:        "invalid YAML",
			yamlContent: `invalid: yaml: content: [`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test YAML to temp file
			tmpFile := t.TempDir() + "/workflow.yml"
			if err := writeFile(tmpFile, tt.yamlContent); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Parse the workflow
			workflow, err := ParseWorkflowFile(tmpFile)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseWorkflowFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Run validation if provided and no error expected
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, workflow)
			}
		})
	}
}

// Helper function to write test files
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
