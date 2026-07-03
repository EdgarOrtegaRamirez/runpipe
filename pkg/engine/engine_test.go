package engine

import (
	"bytes"
	"testing"

	"github.com/EdgarOrtegaRamirez/runpipe/pkg/executor"
	"github.com/EdgarOrtegaRamirez/runpipe/pkg/parser"
	"github.com/EdgarOrtegaRamirez/runpipe/pkg/reporter"
)

func TestEngineRun(t *testing.T) {
	yaml := `
name: test-pipeline
steps:
  - id: step1
    type: shell
    command: echo hello
  - id: step2
    type: shell
    command: echo world
    depends:
      - step1
`

	p, err := parser.ParseBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	// Expand variables
	for i := range p.Steps {
		parser.ExpandStep(&p.Steps[i], p.Vars, p.Templates)
	}

	var buf bytes.Buffer
	exec := executor.New("/tmp")
	rep := reporter.New(reporter.FormatText, &buf)

	eng, err := NewEngine(p, exec, rep, 4)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	result := eng.Run()

	if result.Status != "success" {
		t.Errorf("expected success, got %s", result.Status)
	}
	if result.PassedSteps != 2 {
		t.Errorf("expected 2 passed steps, got %d", result.PassedSteps)
	}
	if result.TotalSteps != 2 {
		t.Errorf("expected 2 total steps, got %d", result.TotalSteps)
	}
}

func TestEngineRunWithFailure(t *testing.T) {
	yaml := `
name: test-failure
steps:
  - id: step1
    type: shell
    command: exit 1
  - id: step2
    type: shell
    command: echo should-not-run
    depends:
      - step1
`

	p, err := parser.ParseBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	for i := range p.Steps {
		parser.ExpandStep(&p.Steps[i], p.Vars, p.Templates)
	}

	var buf bytes.Buffer
	exec := executor.New("/tmp")
	rep := reporter.New(reporter.FormatText, &buf)

	eng, err := NewEngine(p, exec, rep, 4)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	result := eng.Run()

	if result.Status != "failed" {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if result.FailedSteps != 1 {
		t.Errorf("expected 1 failed step, got %d", result.FailedSteps)
	}
	if result.SkippedSteps != 1 {
		t.Errorf("expected 1 skipped step, got %d", result.SkippedSteps)
	}
}

func TestEngineParallelSteps(t *testing.T) {
	yaml := `
name: test-parallel
steps:
  - id: parallel1
    type: shell
    command: echo p1
    parallel: true
  - id: parallel2
    type: shell
    command: echo p2
    parallel: true
  - id: sequential
    type: shell
    command: echo seq
    depends:
      - parallel1
      - parallel2
`

	p, err := parser.ParseBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	for i := range p.Steps {
		parser.ExpandStep(&p.Steps[i], p.Vars, p.Templates)
	}

	var buf bytes.Buffer
	exec := executor.New("/tmp")
	rep := reporter.New(reporter.FormatText, &buf)

	eng, err := NewEngine(p, exec, rep, 4)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	result := eng.Run()

	if result.Status != "success" {
		t.Errorf("expected success, got %s", result.Status)
	}
	if result.PassedSteps != 3 {
		t.Errorf("expected 3 passed steps, got %d", result.PassedSteps)
	}
}
