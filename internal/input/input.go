// Package input handles reading markdown content from files or stdin.
package input

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	maxInputBytes = 10 * 1024 * 1024 // 10 MB
	maxLineBytes  = 1 * 1024 * 1024  // 1 MB per line
)

// Input holds the resolved markdown content.
type Input struct {
	Content  string
	FilePath string // empty for stdin
}

// ReadFile reads markdown from a file path.
func ReadFile(path string) (*Input, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %q: %w", path, err)
	}
	if len(data) > maxInputBytes {
		return nil, fmt.Errorf("file too large (max %d bytes)", maxInputBytes)
	}
	return &Input{
		Content:  string(data),
		FilePath: path,
	}, nil
}

// ReadStdin reads markdown from stdin with DoS-safe limits.
func ReadStdin() (*Input, error) {
	var sb strings.Builder
	reader := bufio.NewReader(os.Stdin)
	total := 0

	for {
		line, err := reader.ReadString('\n')
		total += len(line)
		if total > maxInputBytes {
			return nil, fmt.Errorf("input too large (max %d bytes)", maxInputBytes)
		}
		if len(line) > maxLineBytes {
			return nil, fmt.Errorf("line too long (max %d bytes per line)", maxLineBytes)
		}
		sb.WriteString(line)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
	}

	return &Input{Content: sb.String()}, nil
}

// IsStdinPiped returns true if stdin is being piped (not a terminal).
func IsStdinPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == 0
}
