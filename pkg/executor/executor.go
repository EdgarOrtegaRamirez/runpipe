// Package executor handles the actual execution of pipeline steps.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/EdgarOrtegaRamirez/runpipe/pkg/models"
)

// Executor runs individual pipeline steps.
type Executor struct {
	WorkDir string
	Timeout time.Duration
	Stdout  io.Writer
	Stderr  io.Writer
}

// New creates a new Executor with the given working directory.
func New(workDir string) *Executor {
	return &Executor{
		WorkDir: workDir,
		Timeout: 5 * time.Minute,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}

// Execute runs a step and returns the result.
func (e *Executor) Execute(step *models.Step, env map[string]string) *models.StepResult {
	result := &models.StepResult{
		StepID:    step.ID,
		Status:    models.StatusRunning,
		StartedAt: time.Now(),
	}

	// Parse timeout
	timeout := e.Timeout
	if step.Timeout != "" {
		if d, err := time.ParseDuration(step.Timeout); err == nil {
			timeout = d
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Merge environment variables
	fullEnv := make(map[string]string)
	for k, v := range env {
		fullEnv[k] = v
	}
	for k, v := range step.Env {
		fullEnv[k] = v
	}

	// Convert to slice format
	envSlice := make([]string, 0, len(fullEnv))
	for k, v := range fullEnv {
		envSlice = append(envSlice, k+"="+v)
	}

	var output []byte
	var err error

	switch step.Type {
	case "shell":
		output, err = e.executeShell(ctx, step, envSlice)
	case "http":
		output, err = e.executeHTTP(ctx, step, envSlice)
	case "script":
		output, err = e.executeScript(ctx, step, envSlice)
	default:
		err = fmt.Errorf("unknown step type: %s", step.Type)
	}

	result.EndedAt = time.Now()
	result.Duration = result.EndedAt.Sub(result.StartedAt).Seconds()

	if err != nil {
		result.Status = models.StatusFailed
		result.Error = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	} else {
		result.Status = models.StatusSuccess
		result.ExitCode = 0
	}

	result.Output = string(output)
	return result
}

// executeShell runs a shell command.
func (e *Executor) executeShell(ctx context.Context, step *models.Step, env []string) ([]byte, error) {
	workDir := e.WorkDir
	if step.WorkingDir != "" {
		workDir = step.WorkingDir
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", step.Command)
	cmd.Env = env
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	combined := stdout.Bytes()
	if stderr.Len() > 0 {
		combined = append(combined, stderr.Bytes()...)
	}

	// Write to configured output
	if e.Stdout != nil {
		e.Stdout.Write(stdout.Bytes())
	}
	if e.Stderr != nil && stderr.Len() > 0 {
		e.Stderr.Write(stderr.Bytes())
	}

	return combined, err
}

// executeHTTP makes an HTTP request.
func (e *Executor) executeHTTP(ctx context.Context, step *models.Step, env []string) ([]byte, error) {
	method := step.Method
	if method == "" {
		method = "GET"
	}

	var body io.Reader
	if step.Body != "" {
		// Substitute env vars in body
		bodyStr := step.Body
		for _, envVar := range env {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) == 2 {
				bodyStr = strings.ReplaceAll(bodyStr, "${"+parts[0]+"}", parts[1])
				bodyStr = strings.ReplaceAll(bodyStr, "$"+parts[0], parts[1])
			}
		}
		body = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequestWithContext(ctx, method, step.URL, body)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	// Set headers
	for k, v := range step.Headers {
		req.Header.Set(k, v)
	}
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	output, err := io.ReadAll(resp.Body)
	if err != nil {
		return output, fmt.Errorf("reading HTTP response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return output, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(output))
	}

	return output, nil
}

// executeScript runs a script string directly.
func (e *Executor) executeScript(ctx context.Context, step *models.Step, env []string) ([]byte, error) {
	workDir := e.WorkDir
	if step.WorkingDir != "" {
		workDir = step.WorkingDir
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", step.Script)
	cmd.Env = env
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	combined := stdout.Bytes()
	if stderr.Len() > 0 {
		combined = append(combined, stderr.Bytes()...)
	}

	if e.Stdout != nil {
		e.Stdout.Write(stdout.Bytes())
	}
	if e.Stderr != nil && stderr.Len() > 0 {
		e.Stderr.Write(stderr.Bytes())
	}

	return combined, err
}

// EvaluateCondition evaluates a simple condition string.
// Supports: true, false, $VAR (checks if set and non-empty), $VAR==value
func EvaluateCondition(cond string, env map[string]string) bool {
	if cond == "" || cond == "true" {
		return true
	}
	if cond == "false" {
		return false
	}

	// Check for equality: $VAR==value
	if strings.Contains(cond, "==") {
		parts := strings.SplitN(cond, "==", 2)
		if len(parts) == 2 {
			varName := strings.TrimPrefix(parts[0], "$")
			expected := parts[1]
			val, ok := env[varName]
			if !ok {
				// Check environment
				val = os.Getenv(varName)
			}
			return val == expected
		}
	}

	// Check if variable is set and non-empty
	varName := strings.TrimPrefix(cond, "$")
	if val, ok := env[varName]; ok {
		return val != ""
	}
	val := os.Getenv(varName)
	return val != ""
}
