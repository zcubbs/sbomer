package syft

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Generator struct {
	format      string
	syftBinPath string
}

func New(format, syftBinPath string) *Generator {
	return &Generator{
		format:      format,
		syftBinPath: syftBinPath,
	}
}

// findSyftBinary attempts to find the syft binary in PATH or uses the configured path
func (g *Generator) findSyftBinary() (string, error) {
	// If syftBinPath is just the binary name without any path
	if filepath.Base(g.syftBinPath) == g.syftBinPath {
		// Try to find it in PATH
		binPath, err := exec.LookPath(g.syftBinPath)
		if err != nil {
			return "", fmt.Errorf("syft binary not found in PATH: %w", err)
		}
		return binPath, nil
	}

	// Convert relative path to absolute
	absPath, err := filepath.Abs(g.syftBinPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Verify the binary exists
	if _, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("syft binary not found at path %s: %w", absPath, err)
	}

	return absPath, nil
}

// containsPathSeparator checks if a string contains a path separator
func containsPathSeparator(path string) bool {
	return path != filepath.Base(path)
}

func (g *Generator) GenerateSBOM(projectPath string, outputPath string) error {
	syftPath, err := g.findSyftBinary()
	if err != nil {
		return err
	}

	// Convert paths to use correct separators for the platform
	projectPath = filepath.Clean(projectPath)
	outputPath = filepath.Clean(outputPath)

	cmd := exec.Command(syftPath, "scan", projectPath,
		fmt.Sprintf("-o=%s=%s", g.format, outputPath))

	// Set up environment
	cmd.Env = os.Environ()

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate SBOM: %w, output: %s", err, string(output))
	}

	return nil
}
