package main_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEndToEndBuildAndRun(t *testing.T) {
	// 1. Build the binary
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "bv")

	// Go up to root
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/bv/main.go")
	cmd.Dir = "../../" // Run from project root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Build failed: %v\n%s", err, out)
	}

	// 2. Prepare a fake environment with .beads/beads.jsonl (canonical filename)
	envDir := filepath.Join(tempDir, "env")
	if err := os.MkdirAll(filepath.Join(envDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	jsonlContent := `{"id": "bd-1", "title": "E2E Test Issue", "status": "open", "priority": 0, "issue_type": "bug"}`
	if err := os.WriteFile(filepath.Join(envDir, ".beads", "beads.jsonl"), []byte(jsonlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Run bv --version to verify it runs
	runCmd := exec.Command(binPath, "--version")
	runCmd.Dir = envDir
	if out, err := runCmd.CombinedOutput(); err != nil {
		t.Fatalf("Execution failed: %v\n%s", err, out)
	}
}

func TestEndToEndRobotPlan(t *testing.T) {
	// 1. Build the binary
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "bv")

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/bv/main.go")
	cmd.Dir = "../../"
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Build failed: %v\n%s", err, out)
	}

	// 2. Create environment with dependency chain
	envDir := filepath.Join(tempDir, "env")
	if err := os.MkdirAll(filepath.Join(envDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create issues: epic -> task -> subtask (dependency chain)
	jsonlContent := `{"id": "epic-1", "title": "Epic", "status": "open", "priority": 0, "issue_type": "epic"}
{"id": "task-1", "title": "Task", "status": "open", "priority": 1, "issue_type": "task", "dependencies": [{"target_id": "epic-1", "type": "child_of"}]}
{"id": "subtask-1", "title": "Subtask", "status": "open", "priority": 2, "issue_type": "task", "dependencies": [{"target_id": "task-1", "type": "blocks"}]}`

	if err := os.WriteFile(filepath.Join(envDir, ".beads", "beads.jsonl"), []byte(jsonlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Run bv --robot-plan
	runCmd := exec.Command(binPath, "--robot-plan")
	runCmd.Dir = envDir
	out, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--robot-plan failed: %v\n%s", err, out)
	}

	// 4. Verify output is valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("--robot-plan output is not valid JSON: %v\nOutput: %s", err, out)
	}

	// 5. Verify expected top-level structure
	if _, ok := result["generated_at"]; !ok {
		t.Error("--robot-plan output missing 'generated_at' field")
	}
	plan, ok := result["plan"].(map[string]interface{})
	if !ok {
		t.Fatalf("'plan' is not an object: %T", result["plan"])
	}

	// 6. Verify plan structure
	if _, ok := plan["tracks"]; !ok {
		t.Error("--robot-plan output missing 'plan.tracks' field")
	}
	if _, ok := plan["summary"]; !ok {
		t.Error("--robot-plan output missing 'plan.summary' field")
	}

	// 7. Verify tracks is an array
	tracks, ok := plan["tracks"].([]interface{})
	if !ok {
		t.Fatalf("'plan.tracks' is not an array: %T", plan["tracks"])
	}

	// Should have at least one track with actionable items
	if len(tracks) == 0 {
		t.Error("Expected at least one track in execution plan")
	}
}
