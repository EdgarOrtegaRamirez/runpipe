# RunPipe

RunPipe is a declarative YAML pipeline orchestrator for running tasks, scripts, and API calls.

## Building

```bash
go build -o runpipe .
```

## Testing

```bash
go test ./...
```

## Project Structure

```
runpipe/
├── cmd/
│   └── cmd.go         # CLI commands (Cobra)
├── pkg/
│   ├── models/        # Data types
│   ├── parser/        # YAML parsing & variable substitution
│   ├── scheduler/     # DAG execution
│   ├── executor/      # Step execution
│   ├── engine/        # Pipeline orchestration
│   └── reporter/      # Output formatting
├── tests/             # Integration tests
├── main.go            # Entry point
├── go.mod             # Go module
└── README.md          # Documentation
```

## Architecture

1. **Parser** reads YAML and validates pipeline structure
2. **Scheduler** builds a DAG and determines execution order
3. **Engine** orchestrates step execution with retry and parallelism
4. **Executor** runs individual steps (shell, HTTP, script)
5. **Reporter** formats output in text, compact, or JSON

## Key Algorithms

- **Topological Sort** (Kahn's algorithm) for execution ordering
- **Cycle Detection** (DFS) to validate DAGs
- **Parallel Grouping** for concurrent step execution
- **Exponential Backoff** for retry logic

## Dependencies

- `gopkg.in/yaml.v3` — YAML parsing
- `github.com/spf13/cobra` — CLI framework
