// Package cmd implements the RunPipe CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/EdgarOrtegaRamirez/runpipe/pkg/executor"
	"github.com/EdgarOrtegaRamirez/runpipe/pkg/engine"
	"github.com/EdgarOrtegaRamirez/runpipe/pkg/parser"
	"github.com/EdgarOrtegaRamirez/runpipe/pkg/reporter"
	"github.com/spf13/cobra"
)

var (
	flagFormat   string
	flagWorkers  int
	flagDryRun   bool
	flagVerbose  bool
	flagSetVar   []string
	flagWorkDir  string
)

// NewRootCmd creates the root CLI command.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "runpipe",
		Short: "Declarative YAML pipeline orchestrator",
		Long: `RunPipe executes pipelines defined in YAML files.

Features:
  - DAG-based step execution with dependency tracking
  - Shell, HTTP, and script step types
  - Conditional execution and retry with exponential backoff
  - Parallel execution with configurable workers
  - Variable substitution and environment management
  - Multiple output formats (text, compact, JSON)`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVarP(&flagFormat, "format", "f", "text", "Output format (text, compact, json)")
	rootCmd.PersistentFlags().IntVarP(&flagWorkers, "workers", "w", 4, "Maximum parallel workers")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Parse and validate pipeline without executing")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().StringArrayVar(&flagSetVar, "set", nil, "Set a variable (key=value)")
	rootCmd.PersistentFlags().StringVarP(&flagWorkDir, "work-dir", "d", ".", "Working directory for shell steps")

	rootCmd.AddCommand(NewRunCmd())
	rootCmd.AddCommand(NewValidateCmd())
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewInitCmd())

	return rootCmd
}

// NewRunCmd creates the run command.
func NewRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <pipeline.yaml>",
		Short: "Execute a pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			// Parse pipeline
			p, err := parser.ParseFile(path)
			if err != nil {
				return fmt.Errorf("parsing pipeline: %w", err)
			}

			// Apply --set variables
			for _, sv := range flagSetVar {
				key, val := parseSetVar(sv)
				p.Vars[key] = val
			}

			// Re-expand variables after --set
			for i := range p.Steps {
				parser.ExpandStep(&p.Steps[i], p.Vars, p.Templates)
			}

			if flagDryRun {
				fmt.Println("✓ Pipeline is valid")
				fmt.Printf("  Name: %s\n", p.Name)
				fmt.Printf("  Steps: %d\n", len(p.Steps))
				for _, s := range p.Steps {
					deps := ""
					if len(s.Depends) > 0 {
						deps = fmt.Sprintf(" (depends: %v)", s.Depends)
					}
					fmt.Printf("  - %s [%s]%s\n", s.ID, s.Type, deps)
				}
				return nil
			}

			// Create executor
			exec := executor.New(flagWorkDir)

			// Create reporter
			var rep *reporter.Reporter
			switch flagFormat {
			case "json":
				rep = reporter.New(reporter.FormatJSON, os.Stdout)
			case "compact":
				rep = reporter.New(reporter.FormatCompact, os.Stdout)
			default:
				rep = reporter.New(reporter.FormatText, os.Stdout)
			}

			// Create and run engine
			eng, err := engine.NewEngine(p, exec, rep, flagWorkers)
			if err != nil {
				return fmt.Errorf("creating engine: %w", err)
			}

			result := eng.Run()

			// Exit with appropriate code
			if result.Status != "success" {
				os.Exit(1)
			}
			return nil
		},
	}
}

// NewValidateCmd creates the validate command.
func NewValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <pipeline.yaml>",
		Short: "Validate a pipeline file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := parser.ParseFile(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Validation failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✓ Pipeline '%s' is valid\n", p.Name)
			fmt.Printf("  Steps: %d\n", len(p.Steps))
			return nil
		},
	}
}

// NewVersionCmd creates the version command.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("RunPipe v1.0.0")
			fmt.Println("Declarative YAML pipeline orchestrator")
		},
	}
}

// NewInitCmd creates the init command for generating a sample pipeline.
func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Generate a sample pipeline file",
		RunE: func(cmd *cobra.Command, args []string) error {
			sample := `# RunPipe Pipeline Example
name: sample-pipeline
description: A sample pipeline demonstrating RunPipe features
version: "1.0"

vars:
  greeting: "Hello from RunPipe"

env:
  MY_ENV: production

steps:
  - id: setup
    name: Setup environment
    type: shell
    command: echo "${greeting} - Environment: $MY_ENV"

  - id: fetch-data
    name: Fetch data from API
    type: http
    url: https://httpbin.org/get
    method: GET
    depends:
      - setup

  - id: process-data
    name: Process the data
    type: shell
    command: echo "Processing data..."
    depends:
      - fetch-data
    retry:
      max_attempts: 3
      backoff: 1.0
      max_backoff: 10.0

  - id: deploy
    name: Deploy results
    type: shell
    command: echo "Deploying..."
    depends:
      - process-data
`
			if err := os.WriteFile("pipeline.yaml", []byte(sample), 0644); err != nil {
				return fmt.Errorf("writing sample: %w", err)
			}
			fmt.Println("✓ Created pipeline.yaml")
			fmt.Println("  Run with: runpipe run pipeline.yaml")
			return nil
		},
	}
}

func parseSetVar(s string) (string, string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}
