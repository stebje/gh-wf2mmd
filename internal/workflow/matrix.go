package workflow

import (
	"maps"
	"reflect"
	"slices"

	"gopkg.in/yaml.v3"
)

// maxCombinations stops absurd matrices from exhausting memory. GitHub itself
// runs at most 256 jobs per matrix.
const maxCombinations = 10_000

type combination = map[string]any

// matrixSize returns how many jobs a strategy.matrix expands to, following
// GitHub's rules: `exclude` removes every combination it partially matches,
// then each `include` entry either extends the combinations it matches on the
// original matrix keys or, if it matches none, adds a combination of its own.
// It returns 0 when the size is only known at runtime, for example when the
// matrix is built with fromJSON.
func matrixSize(node *yaml.Node) int {
	var matrix map[string]any
	if node.Kind != yaml.MappingNode || node.Decode(&matrix) != nil {
		return 0
	}

	dims := map[string][]any{}
	var include, exclude []combination
	for key, value := range matrix {
		values, ok := value.([]any)
		if !ok {
			return 0 // an expression such as ${{ fromJSON(...) }}
		}
		switch key {
		case "include", "exclude":
			entries := make([]combination, 0, len(values))
			for _, v := range values {
				entry, ok := v.(combination)
				if !ok {
					return 0
				}
				entries = append(entries, entry)
			}
			if key == "include" {
				include = entries
			} else {
				exclude = entries
			}
		default:
			dims[key] = values
		}
	}

	var combos []combination
	if len(dims) > 0 {
		total := 1
		for _, values := range dims {
			if total *= len(values); total > maxCombinations {
				return 0
			}
		}
		combos = []combination{{}}
		for key, values := range dims {
			next := make([]combination, 0, len(combos)*len(values))
			for _, combo := range combos {
				for _, v := range values {
					c := maps.Clone(combo)
					c[key] = v
					next = append(next, c)
				}
			}
			combos = next
		}
	}

	kept := combos[:0]
	for _, combo := range combos {
		excluded := slices.ContainsFunc(exclude, func(e combination) bool { return matches(combo, e, nil) })
		if !excluded {
			kept = append(kept, combo)
		}
	}
	size := len(kept)
	for _, entry := range include {
		if !slices.ContainsFunc(kept, func(c combination) bool { return matches(c, entry, dims) }) {
			size++
		}
	}
	return size
}

// matches reports whether combo has entry's value for each of entry's keys.
// When dims is non-nil, only keys of the original matrix are compared.
func matches(combo, entry combination, dims map[string][]any) bool {
	for key, value := range entry {
		if _, isDim := dims[key]; dims != nil && !isDim {
			continue
		}
		if !reflect.DeepEqual(combo[key], value) {
			return false
		}
	}
	return true
}
