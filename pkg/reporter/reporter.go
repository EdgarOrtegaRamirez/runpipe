// Package reporter provides output formatting for pipeline results.
package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/EdgarOrtegaRamirez/runpipe/pkg/models"
)

// Format represents the output format.
type Format string

const (
	FormatText    Format = "text"
	FormatJSON    Format = "json"
	FormatCompact Format = "compact"
)

// Reporter outputs pipeline execution results.
type Reporter struct {
	format Format
	writer io.Writer
	mu     sync.Mutex
}

// New creates a new Reporter.
func New(format Format, writer io.Writer) *Reporter {
	return &Reporter{
		format: format,
		writer: writer,
	}
}

// StepStarted is called when a step begins execution.
func (r *Reporter) StepStarted(step *models.Step) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch r.format {
	case FormatCompact:
		_, _ = fmt.Fprintf(r.writer, "  ▶ %s ", step.ID)
	case FormatText:
		_, _ = fmt.Fprintf(r.writer, "\n━━━ Step: %s (%s) ━━━\n", step.ID, step.Type)
		if step.Name != "" {
			_, _ = fmt.Fprintf(r.writer, "  Name: %s\n", step.Name)
		}
		if step.Command != "" {
			_, _ = fmt.Fprintf(r.writer, "  Command: %s\n", step.Command)
		} else if step.URL != "" {
			_, _ = fmt.Fprintf(r.writer, "  URL: %s %s\n", step.Method, step.URL)
		}
	}
}

// StepCompleted is called when a step finishes.
func (r *Reporter) StepCompleted(result *models.StepResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch r.format {
	case FormatCompact:
		icon := statusIcon(result.Status)
		dur := formatDuration(result.Duration)
		_, _ = fmt.Fprintf(r.writer, "%s (%s)\n", icon, dur)
	case FormatText:
		icon := statusIcon(result.Status)
		_, _ = fmt.Fprintf(r.writer, "  %s Status: %s\n", icon, result.Status)
		_, _ = fmt.Fprintf(r.writer, "  ⏱  Duration: %s\n", formatDuration(result.Duration))
		if result.Attempts > 1 {
			_, _ = fmt.Fprintf(r.writer, "  🔄 Attempts: %d\n", result.Attempts)
		}
		if result.Error != "" {
			_, _ = fmt.Fprintf(r.writer, "  ❌ Error: %s\n", result.Error)
		}
		if result.Output != "" && result.Status != models.StatusSuccess {
			lines := strings.Split(result.Output, "\n")
			if len(lines) > 10 {
				lines = lines[:10]
				lines = append(lines, "... (truncated)")
			}
			_, _ = fmt.Fprintf(r.writer, "  Output:\n")
			for _, line := range lines {
				_, _ = fmt.Fprintf(r.writer, "    %s\n", line)
			}
		}
	}
}

// PipelineStarted is called when the pipeline begins.
func (r *Reporter) PipelineStarted(p *models.Pipeline) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch r.format {
	case FormatCompact:
		_, _ = fmt.Fprintf(r.writer, "▶ Running %s (%d steps)\n", p.Name, len(p.Steps))
	case FormatText:
		_, _ = fmt.Fprintf(r.writer, "╔══════════════════════════════════════╗\n")
		_, _ = fmt.Fprintf(r.writer, "║  RunPipe: %s\n", p.Name)
		if p.Description != "" {
			_, _ = fmt.Fprintf(r.writer, "║  %s\n", p.Description)
		}
		_, _ = fmt.Fprintf(r.writer, "║  Steps: %d\n", len(p.Steps))
		_, _ = fmt.Fprintf(r.writer, "╚══════════════════════════════════════╝\n")
	}
}

// PipelineCompleted is called when the pipeline finishes.
func (r *Reporter) PipelineCompleted(result *models.PipelineResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch r.format {
	case FormatCompact:
		icon := statusIcon(result.Status)
		dur := formatDuration(result.Duration)
		_, _ = fmt.Fprintf(r.writer, "%s Completed in %s (%d/%d passed)\n",
			icon, dur, result.PassedSteps, result.TotalSteps)
	case FormatText:
		_, _ = fmt.Fprintf(r.writer, "\n╔══════════════════════════════════════╗\n")
		_, _ = fmt.Fprintf(r.writer, "║  Pipeline Result: %s\n", result.Status)
		_, _ = fmt.Fprintf(r.writer, "║  Duration: %s\n", formatDuration(result.Duration))
		_, _ = fmt.Fprintf(r.writer, "║  Steps: %d total, %d passed, %d failed, %d skipped\n",
			result.TotalSteps, result.PassedSteps, result.FailedSteps, result.SkippedSteps)
		_, _ = fmt.Fprintf(r.writer, "╚══════════════════════════════════════╝\n")
	case FormatJSON:
		enc := json.NewEncoder(r.writer)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	}
}

func statusIcon(status models.StepStatus) string {
	switch status {
	case models.StatusSuccess:
		return "✅"
	case models.StatusFailed:
		return "❌"
	case models.StatusSkipped:
		return "⏭️"
	case models.StatusRunning:
		return "🔄"
	case models.StatusPending:
		return "⏳"
	case models.StatusRetry:
		return "🔁"
	default:
		return "❓"
	}
}

func formatDuration(d float64) string {
	if d < 0.001 {
		return "<1ms"
	}
	if d < 1 {
		return fmt.Sprintf("%.0fms", d*1000)
	}
	if d < 60 {
		return fmt.Sprintf("%.1fs", d)
	}
	return time.Duration(d*float64(time.Second)).Round(time.Second).String()
}