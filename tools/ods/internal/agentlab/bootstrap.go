package agentlab

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type BootstrapMode string

const (
	BootstrapModeAuto  BootstrapMode = "auto"
	BootstrapModeSkip  BootstrapMode = "skip"
	BootstrapModeLink  BootstrapMode = "link"
	BootstrapModeCopy  BootstrapMode = "copy"
	BootstrapModeClone BootstrapMode = "clone"
	BootstrapModeNPM   BootstrapMode = "npm"
)

type BootstrapOptions struct {
	EnvMode    BootstrapMode
	PythonMode BootstrapMode
	WebMode    BootstrapMode
}

type BootstrapResult struct {
	Actions []string
}

func Bootstrap(manifest Manifest, opts BootstrapOptions) (*BootstrapResult, error) {
	result := &BootstrapResult{}

	if err := bootstrapEnvFiles(manifest, opts.EnvMode, result); err != nil {
		return nil, err
	}
	if err := bootstrapPython(manifest, opts.PythonMode, result); err != nil {
		return nil, err
	}
	if err := bootstrapWeb(manifest, opts.WebMode, result); err != nil {
		return nil, err
	}

	return result, nil
}

func bootstrapEnvFiles(manifest Manifest, mode BootstrapMode, result *BootstrapResult) error {
	if mode == BootstrapModeSkip {
		return nil
	}

	vscodeDir := filepath.Join(manifest.CheckoutPath, ".vscode")
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		return fmt.Errorf("create .vscode dir: %w", err)
	}

	sources := []struct {
		source string
		target string
		label  string
	}{
		{
			source: filepath.Join(manifest.CreatedFromPath, ".vscode", ".env"),
			target: filepath.Join(manifest.CheckoutPath, ".vscode", ".env"),
			label:  ".vscode/.env",
		},
		{
			source: filepath.Join(manifest.CreatedFromPath, ".vscode", ".env.web"),
			target: filepath.Join(manifest.CheckoutPath, ".vscode", ".env.web"),
			label:  ".vscode/.env.web",
		},
	}

	for _, item := range sources {
		if _, err := os.Stat(item.source); err != nil {
			continue
		}
		if _, err := os.Lstat(item.target); err == nil {
			result.Actions = append(result.Actions, fmt.Sprintf("kept existing %s", item.label))
			continue
		}

		currentMode := mode
		if currentMode == BootstrapModeAuto {
			currentMode = BootstrapModeLink
		}
		switch currentMode {
		case BootstrapModeLink:
			if err := os.Symlink(item.source, item.target); err != nil {
				return fmt.Errorf("symlink %s: %w", item.label, err)
			}
			result.Actions = append(result.Actions, fmt.Sprintf("linked %s from source checkout", item.label))
		case BootstrapModeCopy, BootstrapModeClone:
			if err := copyFile(item.source, item.target); err != nil {
				return fmt.Errorf("copy %s: %w", item.label, err)
			}
			result.Actions = append(result.Actions, fmt.Sprintf("copied %s from source checkout", item.label))
		default:
			return fmt.Errorf("unsupported env bootstrap mode: %s", currentMode)
		}
	}

	return nil
}

func bootstrapPython(manifest Manifest, mode BootstrapMode, result *BootstrapResult) error {
	if mode == BootstrapModeSkip {
		return nil
	}

	sourceVenv := filepath.Join(manifest.CreatedFromPath, ".venv")
	targetVenv := filepath.Join(manifest.CheckoutPath, ".venv")
	if _, err := os.Stat(targetVenv); err == nil {
		result.Actions = append(result.Actions, "kept existing .venv")
		return nil
	}
	if _, err := os.Stat(sourceVenv); err != nil {
		result.Actions = append(result.Actions, "source .venv missing; backend bootstrap deferred")
		return nil
	}

	currentMode := mode
	if currentMode == BootstrapModeAuto {
		currentMode = BootstrapModeLink
	}

	switch currentMode {
	case BootstrapModeLink:
		if err := os.Symlink(sourceVenv, targetVenv); err != nil {
			return fmt.Errorf("symlink .venv: %w", err)
		}
		result.Actions = append(result.Actions, "linked shared .venv from source checkout")
	case BootstrapModeCopy, BootstrapModeClone:
		if err := cloneDirectory(sourceVenv, targetVenv); err != nil {
			return fmt.Errorf("clone .venv: %w", err)
		}
		result.Actions = append(result.Actions, "cloned .venv from source checkout")
	default:
		return fmt.Errorf("unsupported python bootstrap mode: %s", currentMode)
	}

	return nil
}

func bootstrapWeb(manifest Manifest, mode BootstrapMode, result *BootstrapResult) error {
	if mode == BootstrapModeSkip {
		return nil
	}

	sourceModules := filepath.Join(manifest.CreatedFromPath, "web", "node_modules")
	targetModules := filepath.Join(manifest.CheckoutPath, "web", "node_modules")
	if _, err := os.Lstat(targetModules); err == nil {
		result.Actions = append(result.Actions, "kept existing web/node_modules")
		return nil
	}

	currentMode := mode
	if currentMode == BootstrapModeAuto {
		if _, err := os.Stat(sourceModules); err == nil {
			currentMode = BootstrapModeClone
		} else {
			currentMode = BootstrapModeNPM
		}
	}

	switch currentMode {
	case BootstrapModeClone, BootstrapModeCopy:
		if _, err := os.Stat(sourceModules); err != nil {
			webDir := filepath.Join(manifest.CheckoutPath, "web")
			cmd := exec.Command("npm", "ci", "--prefer-offline", "--no-audit")
			cmd.Dir = webDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("npm ci: %w", err)
			}
			result.Actions = append(result.Actions, "installed web/node_modules with npm ci")
			return nil
		}
		if err := cloneDirectory(sourceModules, targetModules); err != nil {
			return fmt.Errorf("clone web/node_modules: %w", err)
		}
		result.Actions = append(result.Actions, "cloned local web/node_modules into worktree")
		return nil
	case BootstrapModeNPM:
		webDir := filepath.Join(manifest.CheckoutPath, "web")
		cmd := exec.Command("npm", "ci", "--prefer-offline", "--no-audit")
		cmd.Dir = webDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("npm ci: %w", err)
		}
		result.Actions = append(result.Actions, "installed web/node_modules with npm ci")
	default:
		return fmt.Errorf("unsupported web bootstrap mode: %s", currentMode)
	}

	return nil
}

func cloneDirectory(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("create parent dir for %s: %w", target, err)
	}

	if runtime.GOOS == "darwin" {
		cmd := exec.Command("cp", "-R", "-c", source, target)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	if runtime.GOOS != "windows" {
		cmd := exec.Command("cp", "-R", source, target)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no supported directory clone strategy succeeded for %s", source)
}

func copyFile(source, target string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, 0644)
}
