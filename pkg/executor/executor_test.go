package executor

import (
	"testing"

	"github.com/EdgarOrtegaRamirez/runpipe/pkg/models"
)

func TestExecuteShell(t *testing.T) {
	exec := New("/tmp")
	step := &models.Step{
		ID:      "test",
		Type:    "shell",
		Command: "echo hello",
	}

	result := exec.Execute(step, nil)

	if result.Status != models.StatusSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Output != "hello\n" {
		t.Errorf("expected output 'hello\\n', got '%s'", result.Output)
	}
}

func TestExecuteShellFailure(t *testing.T) {
	exec := New("/tmp")
	step := &models.Step{
		ID:      "test",
		Type:    "shell",
		Command: "exit 1",
	}

	result := exec.Execute(step, nil)

	if result.Status != models.StatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

func TestExecuteShellWithEnv(t *testing.T) {
	exec := New("/tmp")
	step := &models.Step{
		ID:      "test",
		Type:    "shell",
		Command: "echo $MY_VAR",
	}

	env := map[string]string{"MY_VAR": "hello"}
	result := exec.Execute(step, env)

	if result.Status != models.StatusSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
	if result.Output != "hello\n" {
		t.Errorf("expected output 'hello\\n', got '%s'", result.Output)
	}
}

func TestExecuteScript(t *testing.T) {
	exec := New("/tmp")
	step := &models.Step{
		ID:     "test",
		Type:   "script",
		Script: "echo script-output",
	}

	result := exec.Execute(step, nil)

	if result.Status != models.StatusSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
	if result.Output != "script-output\n" {
		t.Errorf("expected output 'script-output\\n', got '%s'", result.Output)
	}
}

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name string
		cond string
		env  map[string]string
		want bool
	}{
		{"empty condition", "", nil, true},
		{"true", "true", nil, true},
		{"false", "false", nil, false},
		{"var set", "$MY_VAR", map[string]string{"MY_VAR": "value"}, true},
		{"var not set", "$MY_VAR", map[string]string{}, false},
		{"var empty", "$MY_VAR", map[string]string{"MY_VAR": ""}, false},
		{"var equals", "$MY_VAR==value", map[string]string{"MY_VAR": "value"}, true},
		{"var not equals", "$MY_VAR==value", map[string]string{"MY_VAR": "other"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateCondition(tt.cond, tt.env)
			if got != tt.want {
				t.Errorf("EvaluateCondition(%q) = %v, want %v", tt.cond, got, tt.want)
			}
		})
	}
}

func TestExecuteHTTP(t *testing.T) {
	exec := New("/tmp")
	step := &models.Step{
		ID:     "test",
		Type:   "http",
		URL:    "https://httpbin.org/get",
		Method: "GET",
	}

	result := exec.Execute(step, nil)

	// Note: This test requires network access
	if result.Status != models.StatusSuccess {
		t.Logf("HTTP request failed (may be network issue): %s", result.Error)
	}
}
