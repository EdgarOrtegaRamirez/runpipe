// Package engine implements the core pipeline execution engine.
package engine

import (
	"fmt"
	"sync"
	"time"

	"github.com/EdgarOrtegaRamirez/runpipe/pkg/executor"
	"github.com/EdgarOrtegaRamirez/runpipe/pkg/models"
	"github.com/EdgarOrtegaRamirez/runpipe/pkg/reporter"
	"github.com/EdgarOrtegaRamirez/runpipe/pkg/scheduler"
)

// Engine orchestrates pipeline execution.
type Engine struct {
	pipeline *models.Pipeline
	dag      *scheduler.DAG
	executor *executor.Executor
	reporter *reporter.Reporter
	workers  int
	mu       sync.Mutex // protects result counters and slices
}

// NewEngine creates a new pipeline engine.
func NewEngine(p *models.Pipeline, exec *executor.Executor, rep *reporter.Reporter, workers int) (*Engine, error) {
	dag, err := scheduler.NewDAG(p.Steps)
	if err != nil {
		return nil, fmt.Errorf("building DAG: %w", err)
	}
	return &Engine{
		pipeline: p,
		dag:      dag,
		executor: exec,
		reporter: rep,
		workers:  workers,
	}, nil
}

// Run executes the pipeline and returns the result.
func (e *Engine) Run() *models.PipelineResult {
	startedAt := time.Now()
	result := &models.PipelineResult{
		PipelineName: e.pipeline.Name,
		Status:       models.StatusRunning,
		StartedAt:    startedAt,
		TotalSteps:   len(e.pipeline.Steps),
	}

	e.reporter.PipelineStarted(e.pipeline)

	// Thread-safe DAG tracking
	md := scheduler.NewMutexDAG(e.dag)

	// Merge global env vars
	globalEnv := make(map[string]string)
	for k, v := range e.pipeline.Env {
		globalEnv[k] = v
	}
	for k, v := range e.pipeline.Vars {
		globalEnv[k] = v
	}

	// Execute steps in parallel groups
	for !md.IsComplete() {
		readyIDs := md.Ready()
		if len(readyIDs) == 0 {
			// Check if any step failed
			allDone := md.AllDone()
			hasFailed := false
			for _, status := range allDone {
				if status == models.StatusFailed {
					hasFailed = true
					break
				}
			}
			if hasFailed {
				// Skip remaining steps
				for id := range e.dag.Steps {
					if _, ok := allDone[id]; !ok {
						md.MarkDone(id, models.StatusSkipped)
						e.mu.Lock()
						result.SkippedSteps++
						result.Results = append(result.Results, models.StepResult{
							StepID: id,
							Status: models.StatusSkipped,
						})
						e.mu.Unlock()
					}
				}
				break
			}
			break
		}

		// Determine which steps to run in parallel
		// Group steps: run those with parallel=true in parallel, others one at a time
		parallelBatch := []string{}
		sequentialBatch := []string{}

		for _, id := range readyIDs {
			step := e.dag.Steps[id]
			if step.Parallel {
				parallelBatch = append(parallelBatch, id)
			} else {
				sequentialBatch = append(sequentialBatch, id)
			}
		}

		// Run parallel batch
		if len(parallelBatch) > 0 {
			e.runParallelBatch(parallelBatch, md, globalEnv, result)
		}

		// Run sequential batch (one at a time, but we can use multiple workers for steps within)
		for _, id := range sequentialBatch {
			e.runStep(id, md, globalEnv, result)
		}
	}

	result.EndedAt = time.Now()
	result.Duration = result.EndedAt.Sub(result.StartedAt).Seconds()

	if result.FailedSteps > 0 {
		result.Status = models.StatusFailed
	} else {
		result.Status = models.StatusSuccess
	}

	e.reporter.PipelineCompleted(result)
	return result
}

// runParallelBatch executes a batch of steps concurrently.
func (e *Engine) runParallelBatch(ids []string, md *scheduler.MutexDAG, env map[string]string, result *models.PipelineResult) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, e.workers)

	for _, id := range ids {
		wg.Add(1)
		go func(stepID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			e.runStep(stepID, md, env, result)
		}(id)
	}
	wg.Wait()
}

// runStep executes a single step with optional retry.
func (e *Engine) runStep(id string, md *scheduler.MutexDAG, env map[string]string, result *models.PipelineResult) {
	step := e.dag.Steps[id]

	// Check condition
	if step.When != "" {
		if !executor.EvaluateCondition(step.When, env) {
			md.MarkDone(id, models.StatusSkipped)
			e.mu.Lock()
			result.SkippedSteps++
			e.mu.Unlock()
			return
		}
	}

	e.reporter.StepStarted(step)

	maxAttempts := 1
	backoff := 0.0
	maxBackoff := 30.0
	if step.Retry != nil {
		maxAttempts = step.Retry.MaxAttempts
		if maxAttempts < 1 {
			maxAttempts = 1
		}
		backoff = step.Retry.Backoff
		if step.Retry.MaxBackoff > 0 {
			maxBackoff = step.Retry.MaxBackoff
		}
	}

	var lastResult *models.StepResult
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		md.MarkDone(id, models.StatusRunning)
		lastResult = e.executor.Execute(step, env)
		lastResult.Attempts = attempt

		if lastResult.Status == models.StatusSuccess {
			break
		}

		if attempt < maxAttempts {
			md.MarkDone(id, models.StatusRetry)
			waitTime := backoff * float64(attempt)
			if waitTime > maxBackoff {
				waitTime = maxBackoff
			}
			if waitTime > 0 {
				time.Sleep(time.Duration(waitTime * float64(time.Second)))
			}
		}
	}

	e.reporter.StepCompleted(lastResult)

	e.mu.Lock()
	result.Results = append(result.Results, *lastResult)

	if lastResult.Status == models.StatusSuccess {
		md.MarkDone(id, models.StatusSuccess)
		result.PassedSteps++
	} else {
		md.MarkDone(id, models.StatusFailed)
		result.FailedSteps++
	}
	e.mu.Unlock()
}
