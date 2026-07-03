package reporter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/EdgarOrtegaRamirez/runpipe/pkg/models"
)

func TestReporterText(t *testing.T) {
	var buf bytes.Buffer
	rep := New(FormatText, &buf)

	// Test pipeline started
	p := &models.Pipeline{
		Name:        "test-pipeline",
		Description: "A test",
		Steps:       []models.Step{{ID: "step1"}, {ID: "step2"}},
	}
	rep.PipelineStarted(p)

	if !bytes.Contains(buf.Bytes(), []byte("test-pipeline")) {
		t.Error("expected pipeline name in output")
	}
	if !bytes.Contains(buf.Bytes(), []byte("Steps: 2")) {
		t.Error("expected step count in output")
	}

	// Test step started
	buf.Reset()
	step := &models.Step{ID: "step1", Type: "shell", Command: "echo hello"}
	rep.StepStarted(step)
	if !bytes.Contains(buf.Bytes(), []byte("step1")) {
		t.Error("expected step id in output")
	}

	// Test step completed
	buf.Reset()
	result := &models.StepResult{
		StepID:   "step1",
		Status:   models.StatusSuccess,
		Duration: 1.5,
	}
	rep.StepCompleted(result)
	if !bytes.Contains(buf.Bytes(), []byte("success")) {
		t.Error("expected success status in output")
	}

	// Test pipeline completed
	buf.Reset()
	pResult := &models.PipelineResult{
		PipelineName: "test-pipeline",
		Status:       models.StatusSuccess,
		TotalSteps:   2,
		PassedSteps:  2,
		Duration:     5.0,
	}
	rep.PipelineCompleted(pResult)
	if !bytes.Contains(buf.Bytes(), []byte("success")) {
		t.Error("expected success in pipeline result")
	}
}

func TestReporterCompact(t *testing.T) {
	var buf bytes.Buffer
	rep := New(FormatCompact, &buf)

	// Test pipeline started
	p := &models.Pipeline{Name: "test", Steps: []models.Step{{ID: "s1"}}}
	rep.PipelineStarted(p)
	if !bytes.Contains(buf.Bytes(), []byte("Running test")) {
		t.Error("expected 'Running test' in compact output")
	}

	// Test step completed
	buf.Reset()
	result := &models.StepResult{StepID: "s1", Status: models.StatusSuccess, Duration: 0.1}
	rep.StepCompleted(result)
	if !bytes.Contains(buf.Bytes(), []byte("✅")) {
		t.Error("expected checkmark in compact output")
	}
}

func TestReporterJSON(t *testing.T) {
	var buf bytes.Buffer
	rep := New(FormatJSON, &buf)

	pResult := &models.PipelineResult{
		PipelineName: "test",
		Status:       models.StatusSuccess,
		TotalSteps:   1,
		PassedSteps:  1,
		Duration:     1.5,
	}
	rep.PipelineCompleted(pResult)

	var parsed models.PipelineResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if parsed.PipelineName != "test" {
		t.Errorf("expected pipeline name 'test', got '%s'", parsed.PipelineName)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name  string
		dur   float64
		want  string
	}{
		{"milliseconds", 0.001, "1ms"},
		{"seconds", 1.5, "1.5s"},
		{"minutes", 125, "2m5s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.dur)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.dur, got, tt.want)
			}
		})
	}
}
