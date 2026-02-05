package loader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	json "github.com/goccy/go-json"

	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
)

// DefaultHTTPTimeout is the default timeout for HTTP requests to the daemon.
const DefaultHTTPTimeout = 30 * time.Second

// protoIssue mirrors the ConnectRPC JSON response shape (camelCase).
type protoIssue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Type        string   `json:"type"`
	Priority    int      `json:"priority"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
	ClosedAt    string   `json:"closedAt"`
	Parent      string   `json:"parent"`
	Assignee    string   `json:"assignee"`
	CreatedBy   string   `json:"createdBy"`
	Labels      []string `json:"labels"`
	Children    []string `json:"children"`
	DependsOn   []string `json:"dependsOn"`
	Blocks      []string `json:"blocks"`
	BlockedBy   []string `json:"blockedBy"`
	HookBead    string   `json:"hookBead"`
	AgentState  string   `json:"agentState"`
}

type listIssuesResponse struct {
	Issues []protoIssue `json:"issues"`
	Total  int          `json:"total"`
}

// protoStatusMap maps ConnectRPC enum strings to model.Status values.
var protoStatusMap = map[string]model.Status{
	"ISSUE_STATUS_OPEN":        model.StatusOpen,
	"ISSUE_STATUS_IN_PROGRESS": model.StatusInProgress,
	"ISSUE_STATUS_BLOCKED":     model.StatusBlocked,
	"ISSUE_STATUS_DEFERRED":    model.StatusDeferred,
	"ISSUE_STATUS_PINNED":      model.StatusPinned,
	"ISSUE_STATUS_HOOKED":      model.StatusHooked,
	"ISSUE_STATUS_CLOSED":      model.StatusClosed,
	"ISSUE_STATUS_TOMBSTONE":   model.StatusTombstone,
}

// protoTypeMap maps ConnectRPC enum strings to model.IssueType values.
var protoTypeMap = map[string]model.IssueType{
	"ISSUE_TYPE_TASK":    model.TypeTask,
	"ISSUE_TYPE_BUG":     model.TypeBug,
	"ISSUE_TYPE_FEATURE": model.TypeFeature,
	"ISSUE_TYPE_EPIC":    model.TypeEpic,
	"ISSUE_TYPE_CHORE":   model.TypeChore,
}

// mapProtoStatus converts a ConnectRPC status enum string to model.Status.
// Falls back to stripping the prefix and lowercasing for unknown values.
func mapProtoStatus(s string) model.Status {
	if v, ok := protoStatusMap[s]; ok {
		return v
	}
	// Fallback: strip "ISSUE_STATUS_" prefix and lowercase
	s = strings.TrimPrefix(s, "ISSUE_STATUS_")
	return model.Status(strings.ToLower(s))
}

// mapProtoType converts a ConnectRPC type enum string to model.IssueType.
// Falls back to stripping the prefix and lowercasing for unknown values,
// supporting extensible types like "molecule", "agent".
func mapProtoType(s string) model.IssueType {
	if v, ok := protoTypeMap[s]; ok {
		return v
	}
	// Fallback: strip "ISSUE_TYPE_" prefix and lowercase
	s = strings.TrimPrefix(s, "ISSUE_TYPE_")
	return model.IssueType(strings.ToLower(s))
}

// parseProtoTime parses an RFC3339 timestamp string, returning zero time for empty strings.
func parseProtoTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

// toModelIssue converts a protoIssue from the ConnectRPC response to a model.Issue.
//
// Dependency flattening rules:
//   - dependsOn → Dep{IssueID: thisIssue, DependsOnID: dep, Type: DepBlocks}
//   - blockedBy → same direction, deduped against dependsOn
//   - parent → Dep{IssueID: thisIssue, DependsOnID: parent, Type: DepParentChild}
//   - blocks → SKIP (inverse direction; handled when processing the blocked issue)
//   - children → SKIP (inverse; handled when processing the child issue)
func toModelIssue(p *protoIssue) (model.Issue, error) {
	createdAt, err := parseProtoTime(p.CreatedAt)
	if err != nil {
		return model.Issue{}, fmt.Errorf("issue %s: invalid created_at %q: %w", p.ID, p.CreatedAt, err)
	}
	updatedAt, err := parseProtoTime(p.UpdatedAt)
	if err != nil {
		return model.Issue{}, fmt.Errorf("issue %s: invalid updated_at %q: %w", p.ID, p.UpdatedAt, err)
	}

	issue := model.Issue{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		Status:      mapProtoStatus(p.Status),
		IssueType:   mapProtoType(p.Type),
		Priority:    p.Priority,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		Assignee:    p.Assignee,
		Labels:      p.Labels,
	}

	// Parse optional closed_at
	if p.ClosedAt != "" {
		t, err := parseProtoTime(p.ClosedAt)
		if err != nil {
			return model.Issue{}, fmt.Errorf("issue %s: invalid closed_at %q: %w", p.ID, p.ClosedAt, err)
		}
		issue.ClosedAt = &t
	}

	// Flatten dependencies: dependsOn → blocking deps
	seen := make(map[string]bool, len(p.DependsOn)+len(p.BlockedBy)+1)
	for _, dep := range p.DependsOn {
		if dep == "" || seen[dep] {
			continue
		}
		seen[dep] = true
		issue.Dependencies = append(issue.Dependencies, &model.Dependency{
			IssueID:     p.ID,
			DependsOnID: dep,
			Type:        model.DepBlocks,
		})
	}

	// blockedBy → same direction as dependsOn, deduped
	for _, dep := range p.BlockedBy {
		if dep == "" || seen[dep] {
			continue
		}
		seen[dep] = true
		issue.Dependencies = append(issue.Dependencies, &model.Dependency{
			IssueID:     p.ID,
			DependsOnID: dep,
			Type:        model.DepBlocks,
		})
	}

	// parent → parent-child dep
	if p.Parent != "" && !seen[p.Parent] {
		issue.Dependencies = append(issue.Dependencies, &model.Dependency{
			IssueID:     p.ID,
			DependsOnID: p.Parent,
			Type:        model.DepParentChild,
		})
	}

	// blocks and children are intentionally skipped — they are inverse
	// relationships that will be captured when processing the other issue.

	return issue, nil
}

// LoadIssuesFromURL loads issues from a Gas Town daemon via ConnectRPC over HTTP.
// The baseURL should be the daemon address (e.g., "http://localhost:8443").
func LoadIssuesFromURL(ctx context.Context, baseURL string, opts ParseOptions) ([]model.Issue, error) {
	return loadIssuesFromURL(ctx, baseURL, opts, http.DefaultClient)
}

// loadIssuesFromURL is the internal implementation that accepts an *http.Client for testability.
func loadIssuesFromURL(ctx context.Context, baseURL string, opts ParseOptions, client *http.Client) ([]model.Issue, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	endpoint := baseURL + "/gastown.v1.BeadsService/ListIssues"

	body := []byte(`{"status":"","limit":0}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request to %s failed: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("daemon returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var listResp listIssuesResponse
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	warn := opts.WarningHandler
	if warn == nil {
		warn = func(msg string) {
			fmt.Fprintf(io.Discard, "%s", msg)
		}
	}

	issues := make([]model.Issue, 0, len(listResp.Issues))
	for i := range listResp.Issues {
		issue, err := toModelIssue(&listResp.Issues[i])
		if err != nil {
			warn(fmt.Sprintf("skipping issue: %v", err))
			continue
		}

		issue.Status = normalizeIssueStatus(issue.Status)

		if err := issue.Validate(); err != nil {
			warn(fmt.Sprintf("skipping invalid issue %s: %v", issue.ID, err))
			continue
		}

		if opts.IssueFilter != nil && !opts.IssueFilter(&issue) {
			continue
		}

		issues = append(issues, issue)
	}

	return issues, nil
}
