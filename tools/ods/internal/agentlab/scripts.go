package agentlab

import (
	"embed"
	"fmt"
)

//go:embed scripts/*.py
var pythonScripts embed.FS

func loadPythonScript(name string) (string, error) {
	data, err := pythonScripts.ReadFile("scripts/" + name)
	if err != nil {
		return "", fmt.Errorf("load python script %s: %w", name, err)
	}
	return string(data), nil
}
