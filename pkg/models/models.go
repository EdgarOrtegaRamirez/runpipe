// Package models defines the core data types for RunPipe pipelines.
package models

import (
	"time"
)

// Pipeline represents a complete pipeline definition loaded from YAML.
type Pipeline struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description" json:"description"`
	Version     string            `yaml:"version" json:"version"`
	Vars        map[string]string `yaml:"vars" json:"vars"`
	Env         map[string]string `yaml:"env" json:"env"`
	Steps       []Step            `yaml:"steps" json:"steps"`

	// Internal: original templates before variable expansion (step ID -> field -> template)
	Templates map[string]map[string]string `json:"-"`
}

// Step represents a single step in the pipeline.
type Step struct {
	ID          string            `yaml:"id" json:"id"`
	Name        string            `yaml:"name" json:"name"`
	Type        string            `yaml:"type" json:"type"` // shell, http, script
	Command     string            `yaml:"command,omitempty" json:"command,omitempty"`
	URL         string            `yaml:"url,omitempty" json:"url,omitempty"`
	Method      string            `yaml:"method,omitempty" json:"method,omitempty"`
	Headers     map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body        string            `yaml:"body,omitempty" json:"body,omitempty"`
	Script      string            `yaml:"script,omitempty" json:"script,omitempty"`
	Depends     []string          `yaml:"depends,omitempty" json:"depends,omitempty"`
	When        string            `yaml:"when,omitempty" json:"when,omitempty"`
	Retry       *RetryConfig      `yaml:"retry,omitempty" json:"retry,omitempty"`
	Timeout     string            `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Artifacts   []string          `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty" json:"working_dir,omitempty"`
	Parallel    bool              `yaml:"parallel,omitempty" json:"parallel,omitempty"`
}

// RetryConfig specifies retry behavior for a step.
type RetryConfig struct {
	MaxAttempts int     `yaml:"max_attempts" json:"max_attempts"`
	Backoff     float64 `yaml:"backoff" json:"backoff"` // seconds
	MaxBackoff  float64 `yaml:"max_backoff" json:"max_backoff"`
}

// StepStatus represents the execution status of a step.
type StepStatus string

const (
	StatusPending  StepStatus = "pending"
	StatusRunning  StepStatus = "running"
	StatusSuccess  StepStatus = "success"
	StatusFailed   StepStatus = "failed"
	StatusSkipped  StepStatus = "skipped"
	StatusRetry    StepStatus = "retry"
)

// StepResult holds the result of executing a step.
type StepResult struct {
	StepID    string     `json:"step_id"`
	Status    StepStatus `json:"status"`
	Output    string     `json:"output,omitempty"`
	Error     string     `json:"error,omitempty"`
	ExitCode  int        `json:"exit_code,omitempty"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   time.Time  `json:"ended_at"`
	Duration  float64    `json:"duration"`
	Attempts  int        `json:"attempts"`
}

// PipelineResult holds the overall result of running a pipeline.
type PipelineResult struct {
	PipelineName string      `json:"pipeline_name"`
	Status       StepStatus  `json:"status"`
	Results      []StepResult `json:"results"`
	StartedAt    time.Time   `json:"started_at"`
	EndedAt      time.Time   `json:"ended_at"`
	Duration     float64     `json:"duration"`
	TotalSteps   int         `json:"total_steps"`
	PassedSteps  int         `json:"passed_steps"`
	FailedSteps  int         `json:"failed_steps"`
	SkippedSteps int         `json:"skipped_steps"`
}
