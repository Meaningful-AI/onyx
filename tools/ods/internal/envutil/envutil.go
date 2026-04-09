package envutil

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
)

// LoadFile parses a .env-style file into KEY=VALUE entries suitable for
// appending to os.Environ(). Blank lines and comments are skipped.
func LoadFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open env file %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	var envVars []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			value = strings.Trim(value, `"'`)
			envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read env file %s: %w", path, err)
	}

	return envVars, nil
}

// Merge combines shell environment with file-based defaults. Shell values take
// precedence, so file entries are only added for keys not already present.
func Merge(shellEnv, fileVars []string) []string {
	existing := make(map[string]bool, len(shellEnv))
	for _, entry := range shellEnv {
		if idx := strings.Index(entry, "="); idx > 0 {
			existing[entry[:idx]] = true
		}
	}

	merged := make([]string, len(shellEnv))
	copy(merged, shellEnv)
	for _, entry := range fileVars {
		if idx := strings.Index(entry, "="); idx > 0 {
			key := entry[:idx]
			if !existing[key] {
				merged = append(merged, entry)
			}
		}
	}

	return merged
}

// ApplyOverrides replaces or appends KEY=VALUE entries in env with the provided
// overrides. The returned slice contains at most one entry per overridden key.
func ApplyOverrides(env []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return env
	}

	overrideKeys := make(map[string]bool, len(overrides))
	for key := range overrides {
		overrideKeys[key] = true
	}

	filtered := make([]string, 0, len(env)+len(overrides))
	for _, entry := range env {
		if idx := strings.Index(entry, "="); idx > 0 {
			if overrideKeys[entry[:idx]] {
				continue
			}
		}
		filtered = append(filtered, entry)
	}

	filtered = append(filtered, MapToEnvEntries(overrides)...)
	return filtered
}

// MapToEnvEntries converts a string map into KEY=VALUE entries in stable order.
func MapToEnvEntries(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	entries := make([]string, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return entries
}
