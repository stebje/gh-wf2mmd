package renderer

import (
	"fmt"
	"os"
)

// Output writes the Mermaid diagram to the specified destination
// filepath can be "-" for stdout or a file path
func Output(mermaidDiagram string, filepath string) error {
	if filepath == "-" || filepath == "" {
		// Output to stdout
		fmt.Print(mermaidDiagram)
		return nil
	}

	// Write to file
	if err := os.WriteFile(filepath, []byte(mermaidDiagram), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
