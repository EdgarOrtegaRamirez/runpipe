// Package parser handles YAML pipeline definition parsing with variable substitution.
package parser

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/EdgarOrtegaRamirez/runpipe/pkg/models"
	"gopkg.in/yaml.v3"
)

// ParseFile reads a YAML pipeline file and returns a Pipeline.
func ParseFile(path string) (*models.Pipeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading pipeline file: %w", err)
	}
	return ParseBytes(data)
}

// ParseBytes parses YAML bytes into a Pipeline with variable substitution.
func ParseBytes(data []byte) (*models.Pipeline, error) {
	var p models.Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	// Normalize: set defaults
	if p.Vars == nil {
		p.Vars = make(map[string]string)
	}
	if p.Env == nil {
		p.Env = make(map[string]string)
	}

	// Store original templates for re-expansion
	p.Templates = make(map[string]map[string]string)
	for i := range p.Steps {
		step := &p.Steps[i]
		templates := make(map[string]string)
		if step.Command != "" {
			templates["command"] = step.Command
		}
		if step.URL != "" {
			templates["url"] = step.URL
		}
		if step.Body != "" {
			templates["body"] = step.Body
		}
		if step.Script != "" {
			templates["script"] = step.Script
		}
		if step.When != "" {
			templates["when"] = step.When
		}
		if step.WorkingDir != "" {
			templates["working_dir"] = step.WorkingDir
		}
		if len(templates) > 0 {
			p.Templates[step.ID] = templates
		}
	}

	// Validate pipeline
	if err := Validate(&p); err != nil {
		return nil, fmt.Errorf("validating pipeline: %w", err)
	}

	return &p, nil
}

// Validate checks a pipeline for structural errors.
func Validate(p *models.Pipeline) error {
	if p.Name == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if len(p.Steps) == 0 {
		return fmt.Errorf("pipeline must have at least one step")
	}

	ids := make(map[string]bool)
	for _, step := range p.Steps {
		if step.ID == "" {
			return fmt.Errorf("step id is required")
		}
		if ids[step.ID] {
			return fmt.Errorf("duplicate step id: %s", step.ID)
		}
		ids[step.ID] = true

		if step.Type == "" {
			return fmt.Errorf("step %s: type is required (shell, http, script)", step.ID)
		}
		switch step.Type {
		case "shell":
			if step.Command == "" {
				return fmt.Errorf("step %s: command is required for shell type", step.ID)
			}
		case "http":
			if step.URL == "" {
				return fmt.Errorf("step %s: url is required for http type", step.ID)
			}
		case "script":
			if step.Script == "" {
				return fmt.Errorf("step %s: script is required for script type", step.ID)
			}
		default:
			return fmt.Errorf("step %s: unknown type %q (supported: shell, http, script)", step.ID, step.Type)
		}

		// Validate depends reference existing steps
		for _, dep := range step.Depends {
			if !ids[dep] {
				// Will be validated after all steps are parsed
				// Check if it's defined later (allowed in YAML)
				found := false
				for _, other := range p.Steps {
					if other.ID == dep {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("step %s: depends on unknown step %s", step.ID, dep)
				}
			}
		}
	}

	return nil
}

var varPattern = regexp.MustCompile(`\$\{(\w+)\}|\$(\w+)`)

// ExpandStep substitutes ${var} and $var references in step fields using stored templates.
func ExpandStep(step *models.Step, vars map[string]string, templates map[string]map[string]string) {
	stepTemplates := templates[step.ID]
	if stepTemplates == nil {
		return
	}

	if t, ok := stepTemplates["command"]; ok {
		step.Command = expandString(t, vars)
	}
	if t, ok := stepTemplates["url"]; ok {
		step.URL = expandString(t, vars)
	}
	if t, ok := stepTemplates["body"]; ok {
		step.Body = expandString(t, vars)
	}
	if t, ok := stepTemplates["script"]; ok {
		step.Script = expandString(t, vars)
	}
	if t, ok := stepTemplates["when"]; ok {
		step.When = expandString(t, vars)
	}
	if t, ok := stepTemplates["working_dir"]; ok {
		step.WorkingDir = expandString(t, vars)
	}

	if step.Method == "" && step.Type == "http" {
		step.Method = "GET"
	}
	if step.Headers == nil {
		step.Headers = make(map[string]string)
	}
	if step.Env == nil {
		step.Env = make(map[string]string)
	}
}

// expandString replaces ${var} and $var with values from the vars map.
func expandString(s string, vars map[string]string) string {
	if s == "" {
		return s
	}
	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := varPattern.FindStringSubmatch(match)
		name := parts[1]
		if name == "" {
			name = parts[2]
		}
		if val, ok := vars[name]; ok {
			return val
		}
		// Also check environment variables
		if val, ok := os.LookupEnv(name); ok {
			return val
		}
		return match // leave unresolved
	})
}

// FormatPipelineError returns a human-readable error for pipeline issues.
func FormatPipelineError(err error) string {
	msg := err.Error()
	msg = strings.ReplaceAll(msg, "\n", "\n  ")
	return fmt.Sprintf("Pipeline error: %s", msg)
}
