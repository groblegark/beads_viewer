package loader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	json "github.com/goccy/go-json"

	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
)

func TestMapProtoStatus(t *testing.T) {
	tests := []struct {
		input string
		want  model.Status
	}{
		{"ISSUE_STATUS_OPEN", model.StatusOpen},
		{"ISSUE_STATUS_IN_PROGRESS", model.StatusInProgress},
		{"ISSUE_STATUS_BLOCKED", model.StatusBlocked},
		{"ISSUE_STATUS_DEFERRED", model.StatusDeferred},
		{"ISSUE_STATUS_PINNED", model.StatusPinned},
		{"ISSUE_STATUS_HOOKED", model.StatusHooked},
		{"ISSUE_STATUS_CLOSED", model.StatusClosed},
		{"ISSUE_STATUS_TOMBSTONE", model.StatusTombstone},
		// Fallback: unknown status
		{"ISSUE_STATUS_CUSTOM", model.Status("custom")},
		// No prefix at all
		{"WHATEVER", model.Status("whatever")},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapProtoStatus(tt.input)
			if got != tt.want {
				t.Errorf("mapProtoStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapProtoType(t *testing.T) {
	tests := []struct {
		input string
		want  model.IssueType
	}{
		{"ISSUE_TYPE_TASK", model.TypeTask},
		{"ISSUE_TYPE_BUG", model.TypeBug},
		{"ISSUE_TYPE_FEATURE", model.TypeFeature},
		{"ISSUE_TYPE_EPIC", model.TypeEpic},
		{"ISSUE_TYPE_CHORE", model.TypeChore},
		// Extensible fallback
		{"ISSUE_TYPE_MOLECULE", model.IssueType("molecule")},
		{"ISSUE_TYPE_AGENT", model.IssueType("agent")},
		// No prefix
		{"role", model.IssueType("role")},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapProtoType(tt.input)
			if got != tt.want {
				t.Errorf("mapProtoType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToModelIssueDependencyFlattening(t *testing.T) {
	p := &protoIssue{
		ID:        "bv-1",
		Title:     "Test issue",
		Status:    "ISSUE_STATUS_OPEN",
		Type:      "ISSUE_TYPE_TASK",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-02T00:00:00Z",
		DependsOn: []string{"bv-2", "bv-3"},
		BlockedBy: []string{"bv-3", "bv-4"}, // bv-3 overlaps with dependsOn
		Blocks:    []string{"bv-5"},          // should be skipped
		Children:  []string{"bv-6"},          // should be skipped
		Parent:    "bv-7",
	}

	issue, err := toModelIssue(p)
	if err != nil {
		t.Fatalf("toModelIssue() error: %v", err)
	}

	// Expected deps: bv-2 (dependsOn), bv-3 (dependsOn), bv-4 (blockedBy, deduped bv-3), bv-7 (parent)
	if len(issue.Dependencies) != 4 {
		t.Fatalf("expected 4 dependencies, got %d: %+v", len(issue.Dependencies), issue.Dependencies)
	}

	// Verify dependency contents
	depMap := make(map[string]model.DependencyType)
	for _, d := range issue.Dependencies {
		if d.IssueID != "bv-1" {
			t.Errorf("dependency IssueID = %q, want %q", d.IssueID, "bv-1")
		}
		depMap[d.DependsOnID] = d.Type
	}

	// Check dependsOn entries
	if depMap["bv-2"] != model.DepBlocks {
		t.Errorf("bv-2 dep type = %q, want %q", depMap["bv-2"], model.DepBlocks)
	}
	if depMap["bv-3"] != model.DepBlocks {
		t.Errorf("bv-3 dep type = %q, want %q", depMap["bv-3"], model.DepBlocks)
	}
	if depMap["bv-4"] != model.DepBlocks {
		t.Errorf("bv-4 dep type = %q, want %q", depMap["bv-4"], model.DepBlocks)
	}
	// Parent
	if depMap["bv-7"] != model.DepParentChild {
		t.Errorf("bv-7 dep type = %q, want %q", depMap["bv-7"], model.DepParentChild)
	}
	// blocks and children should NOT appear
	if _, ok := depMap["bv-5"]; ok {
		t.Error("bv-5 (blocks) should not appear in dependencies")
	}
	if _, ok := depMap["bv-6"]; ok {
		t.Error("bv-6 (children) should not appear in dependencies")
	}
}

func TestLoadIssuesFromURLEndToEnd(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	closedAt := now.Add(-time.Hour)

	response := listIssuesResponse{
		Issues: []protoIssue{
			{
				ID:        "bv-10",
				Title:     "Open task",
				Status:    "ISSUE_STATUS_OPEN",
				Type:      "ISSUE_TYPE_TASK",
				Priority:  2,
				CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
				UpdatedAt: now.Format(time.RFC3339),
				Assignee:  "alice",
				Labels:    []string{"backend", "urgent"},
				DependsOn: []string{"bv-11"},
			},
			{
				ID:        "bv-11",
				Title:     "Closed bug",
				Status:    "ISSUE_STATUS_CLOSED",
				Type:      "ISSUE_TYPE_BUG",
				Priority:  1,
				CreatedAt: now.Add(-48 * time.Hour).Format(time.RFC3339),
				UpdatedAt: now.Add(-time.Hour).Format(time.RFC3339),
				ClosedAt:  closedAt.Format(time.RFC3339),
			},
		},
		Total: 2,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/bd.v1.BeadsService/List" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json, got %s", ct)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	issues, err := loadIssuesFromURL(context.Background(), srv.URL, "", ParseOptions{}, srv.Client())
	if err != nil {
		t.Fatalf("LoadIssuesFromURL() error: %v", err)
	}

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}

	// Verify first issue
	i0 := issues[0]
	if i0.ID != "bv-10" {
		t.Errorf("issue[0].ID = %q, want %q", i0.ID, "bv-10")
	}
	if i0.Status != model.StatusOpen {
		t.Errorf("issue[0].Status = %q, want %q", i0.Status, model.StatusOpen)
	}
	if i0.IssueType != model.TypeTask {
		t.Errorf("issue[0].IssueType = %q, want %q", i0.IssueType, model.TypeTask)
	}
	if i0.Priority != 2 {
		t.Errorf("issue[0].Priority = %d, want %d", i0.Priority, 2)
	}
	if i0.Assignee != "alice" {
		t.Errorf("issue[0].Assignee = %q, want %q", i0.Assignee, "alice")
	}
	if len(i0.Labels) != 2 {
		t.Errorf("issue[0].Labels = %v, want 2 labels", i0.Labels)
	}
	if len(i0.Dependencies) != 1 || i0.Dependencies[0].DependsOnID != "bv-11" {
		t.Errorf("issue[0].Dependencies unexpected: %+v", i0.Dependencies)
	}

	// Verify second issue
	i1 := issues[1]
	if i1.ID != "bv-11" {
		t.Errorf("issue[1].ID = %q, want %q", i1.ID, "bv-11")
	}
	if i1.Status != model.StatusClosed {
		t.Errorf("issue[1].Status = %q, want %q", i1.Status, model.StatusClosed)
	}
	if i1.ClosedAt == nil {
		t.Fatal("issue[1].ClosedAt should not be nil")
	}
	if !i1.ClosedAt.Equal(closedAt) {
		t.Errorf("issue[1].ClosedAt = %v, want %v", *i1.ClosedAt, closedAt)
	}
}

func TestLoadIssuesFromURLHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := loadIssuesFromURL(context.Background(), srv.URL, "", ParseOptions{}, srv.Client())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if got := err.Error(); !contains(got, "HTTP 500") {
		t.Errorf("error = %q, want to contain 'HTTP 500'", got)
	}
}

func TestLoadIssuesFromURLConnectionRefused(t *testing.T) {
	_, err := loadIssuesFromURL(context.Background(), "http://127.0.0.1:1", "", ParseOptions{}, &http.Client{Timeout: time.Second})
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestLoadIssuesFromURLInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := loadIssuesFromURL(context.Background(), srv.URL, "", ParseOptions{}, srv.Client())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if got := err.Error(); !contains(got, "parse response JSON") {
		t.Errorf("error = %q, want to contain 'parse response JSON'", got)
	}
}

func TestLoadIssuesFromURLFilter(t *testing.T) {
	response := listIssuesResponse{
		Issues: []protoIssue{
			{
				ID:        "bv-20",
				Title:     "Keep me",
				Status:    "ISSUE_STATUS_OPEN",
				Type:      "ISSUE_TYPE_TASK",
				CreatedAt: "2024-01-01T00:00:00Z",
				UpdatedAt: "2024-01-02T00:00:00Z",
			},
			{
				ID:        "bv-21",
				Title:     "Filter me",
				Status:    "ISSUE_STATUS_CLOSED",
				Type:      "ISSUE_TYPE_BUG",
				CreatedAt: "2024-01-01T00:00:00Z",
				UpdatedAt: "2024-01-02T00:00:00Z",
			},
		},
		Total: 2,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	opts := ParseOptions{
		IssueFilter: func(i *model.Issue) bool {
			return i.Status == model.StatusOpen
		},
	}
	issues, err := loadIssuesFromURL(context.Background(), srv.URL, "", opts, srv.Client())
	if err != nil {
		t.Fatalf("LoadIssuesFromURL() error: %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue after filter, got %d", len(issues))
	}
	if issues[0].ID != "bv-20" {
		t.Errorf("filtered issue ID = %q, want %q", issues[0].ID, "bv-20")
	}
}

func TestLoadIssuesFromURLTrailingSlash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bd.v1.BeadsService/List" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"issues":[],"total":0}`))
	}))
	defer srv.Close()

	// URL with trailing slash should still work
	issues, err := loadIssuesFromURL(context.Background(), srv.URL+"/", "", ParseOptions{}, srv.Client())
	if err != nil {
		t.Fatalf("LoadIssuesFromURL() error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestLoadIssuesFromURLAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-secret" {
			http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"issues":[{"id":"bv-99","title":"Authed issue","status":"ISSUE_STATUS_OPEN","type":"ISSUE_TYPE_TASK","createdAt":"2024-01-01T00:00:00Z","updatedAt":"2024-01-02T00:00:00Z"}],"total":1}`))
	}))
	defer srv.Close()

	// Without key → should fail
	_, err := loadIssuesFromURL(context.Background(), srv.URL, "", ParseOptions{}, srv.Client())
	if err == nil {
		t.Fatal("expected error without API key")
	}

	// With key → should succeed
	issues, err := loadIssuesFromURL(context.Background(), srv.URL, "test-secret", ParseOptions{}, srv.Client())
	if err != nil {
		t.Fatalf("expected success with API key, got: %v", err)
	}
	if len(issues) != 1 || issues[0].ID != "bv-99" {
		t.Errorf("unexpected issues: %+v", issues)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
