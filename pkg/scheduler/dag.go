// Package scheduler implements DAG-based pipeline execution.
package scheduler

import (
	"fmt"
	"sync"

	"github.com/EdgarOrtegaRamirez/runpipe/pkg/models"
)

// DAG represents a directed acyclic graph for pipeline steps.
type DAG struct {
	Steps    map[string]*models.Step
	Children map[string][]string // step -> steps that depend on it
	Parents  map[string][]string // step -> steps it depends on
}

// NewDAG creates a DAG from a pipeline's steps.
func NewDAG(steps []models.Step) (*DAG, error) {
	dag := &DAG{
		Steps:    make(map[string]*models.Step),
		Children: make(map[string][]string),
		Parents:  make(map[string][]string),
	}

	for i := range steps {
		step := &steps[i]
		dag.Steps[step.ID] = step
		if _, ok := dag.Children[step.ID]; !ok {
			dag.Children[step.ID] = []string{}
		}
		if _, ok := dag.Parents[step.ID]; !ok {
			dag.Parents[step.ID] = []string{}
		}
		for _, dep := range step.Depends {
			dag.Children[dep] = append(dag.Children[dep], step.ID)
			dag.Parents[step.ID] = append(dag.Parents[step.ID], dep)
		}
	}

	if err := dag.validate(); err != nil {
		return nil, err
	}

	return dag, nil
}

// validate checks the DAG has no cycles.
func (d *DAG) validate() error {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var dfs func(string) error
	dfs = func(id string) error {
		if inStack[id] {
			return fmt.Errorf("cycle detected involving step %s", id)
		}
		if visited[id] {
			return nil
		}
		visited[id] = true
		inStack[id] = true
		for _, child := range d.Children[id] {
			if err := dfs(child); err != nil {
				return err
			}
		}
		inStack[id] = false
		return nil
	}

	for id := range d.Steps {
		if err := dfs(id); err != nil {
			return err
		}
	}
	return nil
}

// ReadySteps returns steps whose dependencies are all satisfied.
func (d *DAG) ReadySteps(completed map[string]models.StepStatus) []string {
	var ready []string
	for id := range d.Steps {
		if status, ok := completed[id]; ok {
			if status == models.StatusSuccess || status == models.StatusSkipped {
				continue
			}
			if status == models.StatusFailed || status == models.StatusRunning {
				continue
			}
		}
		allDone := true
		for _, parent := range d.Parents[id] {
			s, ok := completed[parent]
			if !ok || (s != models.StatusSuccess && s != models.StatusSkipped) {
				allDone = false
				break
			}
		}
		if allDone {
			ready = append(ready, id)
		}
	}
	return ready
}

// ExecutionOrder returns a topological sort of all steps.
func (d *DAG) ExecutionOrder() ([]string, error) {
	inDegree := make(map[string]int)
	for id := range d.Steps {
		inDegree[id] = len(d.Parents[id])
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var order []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)
		for _, child := range d.Children[current] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	if len(order) != len(d.Steps) {
		return nil, fmt.Errorf("topological sort failed: cycle detected")
	}
	return order, nil
}

// GroupParallelGroups returns groups of steps that can run in parallel.
// Each group contains steps that have no dependencies within the group.
func (d *DAG) ParallelGroups() ([][]string, error) {
	order, err := d.ExecutionOrder()
	if err != nil {
		return nil, err
	}

	var groups [][]string
	completed := make(map[string]bool)

	for {
		var currentGroup []string
		for _, id := range order {
			if completed[id] {
				continue
			}
			allParentsDone := true
			for _, parent := range d.Parents[id] {
				if !completed[parent] {
					allParentsDone = false
					break
				}
			}
			if allParentsDone {
				currentGroup = append(currentGroup, id)
			}
		}
		if len(currentGroup) == 0 {
			break
		}
		groups = append(groups, currentGroup)
		for _, id := range currentGroup {
			completed[id] = true
		}
	}

	return groups, nil
}

// ConcurrentReadyGroups returns groups of steps that can run concurrently.
// Steps in the same group have no dependencies on each other.
func (d *DAG) ConcurrentReadyGroups(completed map[string]models.StepStatus) [][]string {
	var groups [][]string
	used := make(map[string]bool)

	for {
		ready := d.ReadySteps(completed)
		if len(ready) == 0 {
			break
		}
		// Filter out already-used steps
		var group []string
		for _, id := range ready {
			if !used[id] {
				group = append(group, id)
				used[id] = true
			}
		}
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

// MutexDAG is a thread-safe DAG wrapper.
type MutexDAG struct {
	dag  *DAG
	mu   sync.RWMutex
	done map[string]models.StepStatus
}

// NewMutexDAG creates a thread-safe DAG.
func NewMutexDAG(dag *DAG) *MutexDAG {
	return &MutexDAG{
		dag:  dag,
		done: make(map[string]models.StepStatus),
	}
}

// MarkDone marks a step as completed.
func (m *MutexDAG) MarkDone(id string, status models.StepStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.done[id] = status
}

// Ready returns steps ready to execute.
func (m *MutexDAG) Ready() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dag.ReadySteps(m.done)
}

// IsComplete returns true if all steps are done.
func (m *MutexDAG) IsComplete() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.done) == len(m.dag.Steps)
}

// AllDone returns a copy of the completed map.
func (m *MutexDAG) AllDone() map[string]models.StepStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := make(map[string]models.StepStatus, len(m.done))
	for k, v := range m.done {
		cp[k] = v
	}
	return cp
}
