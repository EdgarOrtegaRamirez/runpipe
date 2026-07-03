# RunPipe

A declarative YAML pipeline orchestrator for running tasks, scripts, and API calls.

## Features

- **DAG-based execution** — Steps with dependencies are automatically scheduled in topological order
- **Multiple step types** — Shell commands, HTTP requests, and script execution
- **Parallel execution** — Independent steps run concurrently with configurable worker pools
- **Retry with backoff** — Failed steps can retry with exponential backoff
- **Conditional execution** — Steps run only when conditions are met
- **Variable substitution** — `${var}` and `$var` references in step definitions
- **Multiple output formats** — Text (human-readable), compact, and JSON
- **Dry-run mode** — Validate and preview pipelines without executing

## Quick Start

```bash
# Install
go install github.com/EdgarOrtegaRamirez/runpipe/cmd/runpipe@latest

# Generate a sample pipeline
runpipe init

# Run it
runpipe run pipeline.yaml

# Validate a pipeline
runpipe validate pipeline.yaml

# Dry-run (validate without executing)
runpipe run pipeline.yaml --dry-run
```

## Pipeline Format

```yaml
name: my-pipeline
description: A sample pipeline
version: "1.0"

# Variables for substitution
vars:
  greeting: "Hello"

# Environment variables passed to all steps
env:
  MY_ENV: production

steps:
  - id: build
    name: Build project
    type: shell
    command: 'make build'
    env:
      GOOS: linux

  - id: test
    name: Run tests
    type: shell
    command: 'make test'
    depends:
      - build
    retry:
      max_attempts: 3
      backoff: 2.0
      max_backoff: 30.0

  - id: deploy
    name: Deploy to staging
    type: http
    url: https://api.example.com/deploy
    method: POST
    headers:
      Content-Type: application/json
    body: '{"env": "staging"}'
    depends:
      - test
    when: $MY_ENV==production

  - id: notify
    name: Send notification
    type: shell
    command: 'echo "Pipeline complete!"'
    depends:
      - deploy
    parallel: true
```

## Step Types

### Shell
Executes a shell command:
```yaml
- id: step1
  type: shell
  command: echo "Hello World"
  working_dir: /path/to/dir
```

### HTTP
Makes an HTTP request:
```yaml
- id: api-call
  type: http
  url: https://api.example.com/data
  method: GET
  headers:
    Authorization: Bearer $TOKEN
```

### Script
Runs a multi-line script:
```yaml
- id: process
  type: script
  script: |
    echo "Processing..."
    echo "Done!"
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `runpipe run <file>` | Execute a pipeline |
| `runpipe validate <file>` | Validate a pipeline file |
| `runpipe init` | Generate a sample pipeline |
| `runpipe version` | Print version info |

## Options

| Flag | Description |
|------|-------------|
| `-f, --format` | Output format: text, compact, json (default: text) |
| `-w, --workers` | Maximum parallel workers (default: 4) |
| `--dry-run` | Parse and validate without executing |
| `-v, --verbose` | Verbose output |
| `--set key=value` | Set a variable |
| `-d, --work-dir` | Working directory for shell steps |

## Variable Substitution

Use `${var}` or `$var` in step fields:

```yaml
vars:
  version: "1.0"

steps:
  - id: build
    type: shell
    command: 'echo "Building version ${version}"'
```

Variables are resolved from:
1. Pipeline `vars` section
2. Environment variables

## Conditionals

Use the `when` field for conditional execution:

```yaml
- id: deploy
  type: shell
  command: ./deploy.sh
  when: $ENVIRONMENT==production
```

Supported conditions:
- `$VAR` — Check if variable is set and non-empty
- `$VAR==value` — Check if variable equals value
- `true` / `false` — Always run / never run

## Retry Configuration

```yaml
- id: flaky-step
  type: shell
  command: ./unreliable-command.sh
  retry:
    max_attempts: 3      # Maximum retry attempts
    backoff: 2.0         # Initial backoff in seconds
    max_backoff: 30.0    # Maximum backoff in seconds
```

## License

MIT
