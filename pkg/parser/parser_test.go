package parser

import (
	"testing"
)

func TestParseBytes(t *testing.T) {
	yaml := `
name: test-pipeline
description: A test pipeline
version: "1.0"

vars:
  greeting: "Hello"

steps:
  - id: step1
    type: shell
    command: 'echo "${greeting}"'
  - id: step2
    type: shell
    command: 'echo "done"'
    depends:
      - step1
`

	p, err := ParseBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if p.Name != "test-pipeline" {
		t.Errorf("expected name 'test-pipeline', got '%s'", p.Name)
	}
	if len(p.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(p.Steps))
	}
	if p.Steps[0].ID != "step1" {
		t.Errorf("expected step id 'step1', got '%s'", p.Steps[0].ID)
	}
	if p.Steps[1].ID != "step2" {
		t.Errorf("expected step id 'step2', got '%s'", p.Steps[1].ID)
	}
	if len(p.Steps[1].Depends) != 1 || p.Steps[1].Depends[0] != "step1" {
		t.Errorf("expected step2 to depend on step1, got %v", p.Steps[1].Depends)
	}
}

func TestExpandStep(t *testing.T) {
	yaml := `
name: test-pipeline
vars:
  greeting: "Hello"

steps:
  - id: step1
    type: shell
    command: 'echo "${greeting}"'
`

	p, err := ParseBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	// Expand with default vars
	ExpandStep(&p.Steps[0], p.Vars, p.Templates)
	if p.Steps[0].Command != `echo "Hello"` {
		t.Errorf("expected command 'echo \"Hello\"', got '%s'", p.Steps[0].Command)
	}

	// Re-expand with different vars
	p.Vars["greeting"] = "World"
	ExpandStep(&p.Steps[0], p.Vars, p.Templates)
	if p.Steps[0].Command != `echo "World"` {
		t.Errorf("expected command 'echo \"World\"', got '%s'", p.Steps[0].Command)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid pipeline",
			yaml: `
name: test
steps:
  - id: step1
    type: shell
    command: echo hello
`,
			wantErr: false,
		},
		{
			name: "missing name",
			yaml: `
steps:
  - id: step1
    type: shell
    command: echo hello
`,
			wantErr: true,
		},
		{
			name: "no steps",
			yaml: `
name: test
steps: []
`,
			wantErr: true,
		},
		{
			name: "missing step id",
			yaml: `
name: test
steps:
  - type: shell
    command: echo hello
`,
			wantErr: true,
		},
		{
			name: "duplicate step id",
			yaml: `
name: test
steps:
  - id: step1
    type: shell
    command: echo hello
  - id: step1
    type: shell
    command: echo world
`,
			wantErr: true,
		},
		{
			name: "missing type",
			yaml: `
name: test
steps:
  - id: step1
    command: echo hello
`,
			wantErr: true,
		},
		{
			name: "unknown type",
			yaml: `
name: test
steps:
  - id: step1
    type: unknown
    command: echo hello
`,
			wantErr: true,
		},
		{
			name: "shell without command",
			yaml: `
name: test
steps:
  - id: step1
    type: shell
`,
			wantErr: true,
		},
		{
			name: "http without url",
			yaml: `
name: test
steps:
  - id: step1
    type: http
`,
			wantErr: true,
		},
		{
			name: "script without script",
			yaml: `
name: test
steps:
  - id: step1
    type: script
`,
			wantErr: true,
		},
		{
			name: "depends on unknown step",
			yaml: `
name: test
steps:
  - id: step1
    type: shell
    command: echo hello
    depends:
      - unknown
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExpandString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		vars  map[string]string
		want  string
	}{
		{
			name:  "no variables",
			input: "hello world",
			vars:  map[string]string{},
			want:  "hello world",
		},
		{
			name:  "braced variable",
			input: "hello ${name}",
			vars:  map[string]string{"name": "world"},
			want:  "hello world",
		},
		{
			name:  "unbraced variable",
			input: "hello $name",
			vars:  map[string]string{"name": "world"},
			want:  "hello world",
		},
		{
			name:  "multiple variables",
			input: "${greeting} ${name}!",
			vars:  map[string]string{"greeting": "Hello", "name": "world"},
			want:  "Hello world!",
		},
		{
			name:  "unresolved variable",
			input: "hello ${unknown}",
			vars:  map[string]string{},
			want:  "hello ${unknown}",
		},
		{
			name:  "empty string",
			input: "",
			vars:  map[string]string{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandString(tt.input, tt.vars)
			if got != tt.want {
				t.Errorf("expandString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMultipleStepTypes(t *testing.T) {
	yaml := `
name: test-mixed
steps:
  - id: shell-step
    type: shell
    command: echo hello
  - id: http-step
    type: http
    url: https://example.com
    method: GET
  - id: script-step
    type: script
    script: echo script
`

	p, err := ParseBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if len(p.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(p.Steps))
	}

	if p.Steps[0].Type != "shell" {
		t.Errorf("expected step 0 type 'shell', got '%s'", p.Steps[0].Type)
	}
	if p.Steps[1].Type != "http" {
		t.Errorf("expected step 1 type 'http', got '%s'", p.Steps[1].Type)
	}
	if p.Steps[2].Type != "script" {
		t.Errorf("expected step 2 type 'script', got '%s'", p.Steps[2].Type)
	}

	// Check that http method defaults to GET
	if p.Steps[1].Method != "GET" {
		t.Errorf("expected http method 'GET', got '%s'", p.Steps[1].Method)
	}
}
