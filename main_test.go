package main_test

import (
	"os/exec"
	"testing"
)

func TestVersion(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run --help: %v\n%s", err, out)
	}
}