package workflow

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMatrixSize(t *testing.T) {
	tests := []struct {
		name   string
		matrix string
		want   int
	}{
		{"one dimension", `os: [ubuntu, windows, macos]`, 3},
		{"all combinations", `{os: [ubuntu, windows], node: [18, 20, 22]}`, 6},
		{
			// From GitHub's docs: 12 combinations, three of them excluded.
			"exclude removes partial matches", `
os: [macos-latest, windows-latest]
version: [12, 14, 16]
environment: [staging, production]
exclude:
  - {os: macos-latest, version: 12, environment: production}
  - {os: windows-latest, version: 16}
`, 9,
		},
		{
			// From GitHub's docs: entries that match on the original keys extend
			// existing combinations; the two banana entries add new ones.
			"include extends or adds", `
fruit: [apple, pear]
animal: [cat, dog]
include:
  - color: green
  - {color: pink, animal: cat}
  - {fruit: apple, shape: circle}
  - fruit: banana
  - {fruit: banana, animal: cat}
`, 6,
		},
		{
			"include can restore an excluded combination", `
os: [ubuntu, windows]
node: [18, 20]
exclude: [{os: windows}]
include: [{os: windows, node: 20}]
`, 3,
		},
		{"include only", `include: [{site: production}, {site: staging}]`, 2},
		{"dynamic matrix", `${{ fromJSON(needs.setup.outputs.matrix) }}`, 0},
		{"dynamic dimension", `{os: "${{ fromJSON(inputs.os) }}", node: [20]}`, 0},
		{"dynamic include", `{os: [ubuntu], include: "${{ fromJSON(inputs.extra) }}"}`, 0},
		{
			"too many combinations", `{a: [1,2,3,4,5,6,7,8,9,10], b: [1,2,3,4,5,6,7,8,9,10],
c: [1,2,3,4,5,6,7,8,9,10], d: [1,2,3,4,5,6,7,8,9,10], e: [1,2,3,4,5,6,7,8,9,10]}`, 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc yaml.Node
			if err := yaml.Unmarshal([]byte(tt.matrix), &doc); err != nil {
				t.Fatal(err)
			}
			if got := matrixSize(doc.Content[0]); got != tt.want {
				t.Errorf("matrixSize() = %d, want %d", got, tt.want)
			}
		})
	}
}
