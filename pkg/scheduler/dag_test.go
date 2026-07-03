package scheduler

import (
	"testing"

	"github.com/EdgarOrtegaRamirez/runpipe/pkg/models"
)

func TestNewDAG(t *testing.T) {
	steps := []models.Step{
		{ID: "step1", Type: "shell", Command: "echo 1"},
		{ID: "step2", Type: "shell", Command: "echo 2", Depends: []string{"step1"}},
		{ID: "step3", Type: "shell", Command: "echo 3", Depends: []string{"step2"}},
	}

	dag, err := NewDAG(steps)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}

	if len(dag.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(dag.Steps))
	}
	if len(dag.Parents["step2"]) != 1 || dag.Parents["step2"][0] != "step1" {
		t.Errorf("expected step2 to have parent step1, got %v", dag.Parents["step2"])
	}
	if len(dag.Children["step1"]) != 1 || dag.Children["step1"][0] != "step2" {
		t.Errorf("expected step1 to have child step2, got %v", dag.Children["step1"])
	}
}

func TestDAGCycleDetection(t *testing.T) {
	steps := []models.Step{
		{ID: "step1", Type: "shell", Command: "echo 1", Depends: []string{"step2"}},
		{ID: "step2", Type: "shell", Command: "echo 2", Depends: []string{"step1"}},
	}

	_, err := NewDAG(steps)
	if err == nil {
		t.Error("expected cycle detection error")
	}
}

func TestReadySteps(t *testing.T) {
	steps := []models.Step{
		{ID: "step1", Type: "shell", Command: "echo 1"},
		{ID: "step2", Type: "shell", Command: "echo 2", Depends: []string{"step1"}},
		{ID: "step3", Type: "shell", Command: "echo 3", Depends: []string{"step1"}},
	}

	dag, err := NewDAG(steps)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}

	// Initially only step1 is ready
	completed := map[string]models.StepStatus{}
	ready := dag.ReadySteps(completed)
	if len(ready) != 1 || ready[0] != "step1" {
		t.Errorf("expected only step1 ready, got %v", ready)
	}

	// After step1 completes, step2 and step3 are ready
	completed["step1"] = models.StatusSuccess
	ready = dag.ReadySteps(completed)
	if len(ready) != 2 {
		t.Errorf("expected 2 steps ready, got %d", len(ready))
	}
}

func TestExecutionOrder(t *testing.T) {
	steps := []models.Step{
		{ID: "step1", Type: "shell", Command: "echo 1"},
		{ID: "step2", Type: "shell", Command: "echo 2", Depends: []string{"step1"}},
		{ID: "step3", Type: "shell", Command: "echo 3", Depends: []string{"step2"}},
	}

	dag, err := NewDAG(steps)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}

	order, err := dag.ExecutionOrder()
	if err != nil {
		t.Fatalf("ExecutionOrder failed: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 steps in order, got %d", len(order))
	}

	// step1 must come before step2, step2 before step3
	if order[0] != "step1" {
		t.Errorf("expected step1 first, got %s", order[0])
	}
	if order[1] != "step2" {
		t.Errorf("expected step2 second, got %s", order[1])
	}
	if order[2] != "step3" {
		t.Errorf("expected step3 third, got %s", order[2])
	}
}

func TestParallelGroups(t *testing.T) {
	steps := []models.Step{
		{ID: "step1", Type: "shell", Command: "echo 1"},
		{ID: "step2", Type: "shell", Command: "echo 2"},
		{ID: "step3", Type: "shell", Command: "echo 3", Depends: []string{"step1", "step2"}},
	}

	dag, err := NewDAG(steps)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}

	groups, err := dag.ParallelGroups()
	if err != nil {
		t.Fatalf("ParallelGroups failed: %v", err)
	}

	// step1 and step2 can run in parallel, then step3
	if len(groups) < 2 {
		t.Fatalf("expected at least 2 groups, got %d", len(groups))
	}

	// First group should have step1 and step2
	if len(groups[0]) != 2 {
		t.Errorf("expected 2 steps in first group, got %d", len(groups[0]))
	}
}

func TestMutexDAG(t *testing.T) {
	steps := []models.Step{
		{ID: "step1", Type: "shell", Command: "echo 1"},
		{ID: "step2", Type: "shell", Command: "echo 2", Depends: []string{"step1"}},
	}

	dag, err := NewDAG(steps)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}

	md := NewMutexDAG(dag)

	// Initially only step1 is ready
	ready := md.Ready()
	if len(ready) != 1 || ready[0] != "step1" {
		t.Errorf("expected only step1 ready, got %v", ready)
	}

	// Mark step1 as done
	md.MarkDone("step1", models.StatusSuccess)
	if md.IsComplete() {
		t.Error("should not be complete yet")
	}

	// Now step2 should be ready
	ready = md.Ready()
	if len(ready) != 1 || ready[0] != "step2" {
		t.Errorf("expected step2 ready, got %v", ready)
	}

	// Mark step2 as done
	md.MarkDone("step2", models.StatusSuccess)
	if !md.IsComplete() {
		t.Error("should be complete")
	}
}
