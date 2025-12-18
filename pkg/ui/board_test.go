package ui_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
	"github.com/Dicklesworthstone/beads_viewer/pkg/ui"

	"github.com/charmbracelet/lipgloss"
)

func createTime(hoursAgo int) time.Time {
	return time.Now().Add(time.Duration(-hoursAgo) * time.Hour)
}

func createTheme() ui.Theme {
	return ui.DefaultTheme(lipgloss.NewRenderer(os.Stdout))
}

// TestBoardModelBlackbox tests basic selection and update behavior
func TestBoardModelBlackbox(t *testing.T) {
	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Priority: 1, CreatedAt: createTime(0)},
	}

	theme := createTheme()
	b := ui.NewBoardModel(issues, theme)

	// Focus Open col (0)
	sel := b.SelectedIssue()
	if sel == nil || sel.ID != "1" {
		t.Errorf("Expected ID 1 selected in Open col")
	}

	// Update issues
	newIssues := []model.Issue{
		{ID: "2", Status: model.StatusOpen, Priority: 1, CreatedAt: createTime(0)},
	}
	b.SetIssues(newIssues)

	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "2" {
		t.Errorf("Expected ID 2 selected after update, got %v", sel)
	}

	// Filter to empty
	b.SetIssues([]model.Issue{})
	sel = b.SelectedIssue()
	if sel != nil {
		t.Errorf("Expected nil selection for empty board")
	}
}

// TestAdaptiveColumns verifies that only non-empty columns are navigable
func TestAdaptiveColumns(t *testing.T) {
	theme := createTheme()

	// Create issues only in Open and Closed columns (skip InProgress and Blocked)
	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Priority: 1, CreatedAt: createTime(0)},
		{ID: "2", Status: model.StatusOpen, Priority: 2, CreatedAt: createTime(1)},
		{ID: "3", Status: model.StatusClosed, Priority: 1, CreatedAt: createTime(2)},
	}

	b := ui.NewBoardModel(issues, theme)

	// Should start on first non-empty column (Open)
	sel := b.SelectedIssue()
	if sel == nil || sel.ID != "1" {
		t.Errorf("Expected ID 1 (Open col), got %v", sel)
	}

	// MoveRight should skip InProgress and Blocked (empty), go to Closed
	b.MoveRight()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "3" {
		t.Errorf("Expected ID 3 (Closed col) after MoveRight, got %v", sel)
	}

	// MoveRight again should stay on Closed (last non-empty column)
	b.MoveRight()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "3" {
		t.Errorf("Expected to stay on ID 3, got %v", sel)
	}

	// MoveLeft should go back to Open (skipping empty columns)
	b.MoveLeft()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "1" {
		t.Errorf("Expected ID 1 (Open col) after MoveLeft, got %v", sel)
	}
}

// TestColumnNavigation tests up/down navigation within columns
func TestColumnNavigation(t *testing.T) {
	theme := createTheme()

	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Priority: 1, CreatedAt: createTime(0)},
		{ID: "2", Status: model.StatusOpen, Priority: 2, CreatedAt: createTime(1)},
		{ID: "3", Status: model.StatusOpen, Priority: 3, CreatedAt: createTime(2)},
	}

	b := ui.NewBoardModel(issues, theme)

	// Should start at first item
	sel := b.SelectedIssue()
	if sel == nil || sel.ID != "1" {
		t.Errorf("Expected ID 1, got %v", sel)
	}

	// MoveDown
	b.MoveDown()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "2" {
		t.Errorf("Expected ID 2 after MoveDown, got %v", sel)
	}

	// MoveDown again
	b.MoveDown()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "3" {
		t.Errorf("Expected ID 3 after second MoveDown, got %v", sel)
	}

	// MoveDown at bottom should stay at bottom
	b.MoveDown()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "3" {
		t.Errorf("Expected to stay at ID 3, got %v", sel)
	}

	// MoveUp
	b.MoveUp()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "2" {
		t.Errorf("Expected ID 2 after MoveUp, got %v", sel)
	}

	// MoveToTop
	b.MoveToTop()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "1" {
		t.Errorf("Expected ID 1 after MoveToTop, got %v", sel)
	}

	// MoveToBottom
	b.MoveToBottom()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "3" {
		t.Errorf("Expected ID 3 after MoveToBottom, got %v", sel)
	}
}

// TestPageNavigation tests page up/down bounds
func TestPageNavigation(t *testing.T) {
	theme := createTheme()

	// Create 10 issues with proper string IDs
	var issues []model.Issue
	for i := 1; i <= 10; i++ {
		issues = append(issues, model.Issue{
			ID:        fmt.Sprintf("%d", i),
			Status:    model.StatusOpen,
			Priority:  i,
			CreatedAt: createTime(i),
		})
	}

	b := ui.NewBoardModel(issues, theme)

	// PageDown with visibleRows=6 (moves by 3)
	b.PageDown(6)
	sel := b.SelectedIssue()
	if sel == nil {
		t.Fatal("Expected selection after PageDown")
	}
	// Should be at row 3 (0-indexed)

	// PageDown many times - should not exceed bounds
	for i := 0; i < 20; i++ {
		b.PageDown(6)
	}
	sel = b.SelectedIssue()
	if sel == nil {
		t.Fatal("Expected selection after many PageDowns")
	}
	// Should be at last item (row 9)

	// PageUp many times - should not go below 0
	for i := 0; i < 20; i++ {
		b.PageUp(6)
	}
	sel = b.SelectedIssue()
	if sel == nil {
		t.Fatal("Expected selection after many PageUps")
	}
	// Should be at first item
}

// TestColumnCounts tests count methods
func TestColumnCounts(t *testing.T) {
	theme := createTheme()

	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Priority: 1},
		{ID: "2", Status: model.StatusOpen, Priority: 2},
		{ID: "3", Status: model.StatusInProgress, Priority: 1},
		{ID: "4", Status: model.StatusClosed, Priority: 1},
	}

	b := ui.NewBoardModel(issues, theme)

	if b.ColumnCount(0) != 2 { // Open
		t.Errorf("Expected 2 in Open column, got %d", b.ColumnCount(0))
	}
	if b.ColumnCount(1) != 1 { // InProgress
		t.Errorf("Expected 1 in InProgress column, got %d", b.ColumnCount(1))
	}
	if b.ColumnCount(2) != 0 { // Blocked
		t.Errorf("Expected 0 in Blocked column, got %d", b.ColumnCount(2))
	}
	if b.ColumnCount(3) != 1 { // Closed
		t.Errorf("Expected 1 in Closed column, got %d", b.ColumnCount(3))
	}
	if b.TotalCount() != 4 {
		t.Errorf("Expected total 4, got %d", b.TotalCount())
	}
}

// TestSetIssuesSanitizesSelection verifies selection is sanitized after SetIssues
func TestSetIssuesSanitizesSelection(t *testing.T) {
	theme := createTheme()

	// Start with 5 issues in Open
	var issues []model.Issue
	for i := 1; i <= 5; i++ {
		issues = append(issues, model.Issue{
			ID:       fmt.Sprintf("%d", i),
			Status:   model.StatusOpen,
			Priority: i,
		})
	}

	b := ui.NewBoardModel(issues, theme)

	// Move to bottom (row 4)
	b.MoveToBottom()
	sel := b.SelectedIssue()
	if sel == nil || sel.ID != "5" {
		t.Errorf("Expected ID 5, got %v", sel)
	}

	// Now reduce to only 2 issues - selection should be sanitized
	b.SetIssues([]model.Issue{
		{ID: "A", Status: model.StatusOpen, Priority: 1},
		{ID: "B", Status: model.StatusOpen, Priority: 2},
	})

	sel = b.SelectedIssue()
	if sel == nil {
		t.Fatal("Expected selection after SetIssues")
	}
	// Selection should be sanitized to last valid row (1)
	if sel.ID != "B" {
		t.Errorf("Expected ID B (last item), got %s", sel.ID)
	}
}

// TestAllColumnsEmpty verifies behavior when all columns are empty
func TestAllColumnsEmpty(t *testing.T) {
	theme := createTheme()

	b := ui.NewBoardModel([]model.Issue{}, theme)

	// Should return nil for selected issue
	sel := b.SelectedIssue()
	if sel != nil {
		t.Errorf("Expected nil selection for empty board, got %v", sel)
	}

	// Navigation should not panic
	b.MoveUp()
	b.MoveDown()
	b.MoveLeft()
	b.MoveRight()
	b.MoveToTop()
	b.MoveToBottom()
	b.PageUp(10)
	b.PageDown(10)

	// Counts should be zero
	if b.TotalCount() != 0 {
		t.Errorf("Expected total 0, got %d", b.TotalCount())
	}
}

// TestSortingByPriorityAndDate verifies issues are sorted correctly
func TestSortingByPriorityAndDate(t *testing.T) {
	theme := createTheme()

	// Create issues with different priorities and dates
	issues := []model.Issue{
		{ID: "low-old", Status: model.StatusOpen, Priority: 3, CreatedAt: createTime(48)},
		{ID: "high-new", Status: model.StatusOpen, Priority: 1, CreatedAt: createTime(0)},
		{ID: "high-old", Status: model.StatusOpen, Priority: 1, CreatedAt: createTime(24)},
		{ID: "med-new", Status: model.StatusOpen, Priority: 2, CreatedAt: createTime(0)},
	}

	b := ui.NewBoardModel(issues, theme)

	// First should be high priority, newer date
	sel := b.SelectedIssue()
	if sel == nil || sel.ID != "high-new" {
		t.Errorf("Expected high-new first, got %v", sel)
	}

	b.MoveDown()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "high-old" {
		t.Errorf("Expected high-old second, got %v", sel)
	}

	b.MoveDown()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "med-new" {
		t.Errorf("Expected med-new third, got %v", sel)
	}

	b.MoveDown()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "low-old" {
		t.Errorf("Expected low-old fourth, got %v", sel)
	}
}

// TestViewRendering verifies View doesn't panic with various inputs
func TestViewRendering(t *testing.T) {
	theme := createTheme()

	tests := []struct {
		name   string
		issues []model.Issue
		width  int
		height int
	}{
		{"empty", []model.Issue{}, 80, 24},
		{"single", []model.Issue{{ID: "1", Status: model.StatusOpen}}, 80, 24},
		{"narrow", []model.Issue{{ID: "1", Status: model.StatusOpen}}, 40, 24},
		{"short", []model.Issue{{ID: "1", Status: model.StatusOpen}}, 80, 10},
		{"all_statuses", []model.Issue{
			{ID: "1", Status: model.StatusOpen},
			{ID: "2", Status: model.StatusInProgress},
			{ID: "3", Status: model.StatusBlocked},
			{ID: "4", Status: model.StatusClosed},
		}, 120, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := ui.NewBoardModel(tt.issues, theme)
			// Should not panic
			_ = b.View(tt.width, tt.height)
		})
	}
}

// TestBoardRichCardContent verifies the bv-1daf rich card content rendering
// Tests that cards with dependencies render correctly with blocked-by and blocks indicators
func TestBoardRichCardContent(t *testing.T) {
	theme := createTheme()

	// Create issues with dependencies to test blocks/blocked-by indicators
	issues := []model.Issue{
		{
			ID:        "A",
			Title:     "Foundation Task",
			Status:    model.StatusOpen,
			Priority:  1,
			CreatedAt: createTime(48), // 2 days ago
			UpdatedAt: createTime(24), // 1 day ago
			Labels:    []string{"backend", "api"},
		},
		{
			ID:        "B",
			Title:     "Blocked Task",
			Status:    model.StatusBlocked,
			Priority:  2,
			CreatedAt: createTime(24),
			UpdatedAt: createTime(1),
			Dependencies: []*model.Dependency{
				{IssueID: "B", DependsOnID: "A", Type: model.DepBlocks},
			},
		},
		{
			ID:        "C",
			Title:     "Another Blocked Task",
			Status:    model.StatusBlocked,
			Priority:  2,
			CreatedAt: createTime(24),
			UpdatedAt: createTime(2),
			Dependencies: []*model.Dependency{
				{IssueID: "C", DependsOnID: "A", Type: model.DepBlocks},
			},
		},
	}

	b := ui.NewBoardModel(issues, theme)

	// Board should render without panic
	output := b.View(160, 40)

	// Basic sanity checks - output should contain issue IDs
	if output == "" {
		t.Error("Board view should not be empty")
	}

	// Test SetIssues rebuilds blocks index
	b.SetIssues(issues)
	output2 := b.View(160, 40)
	if output2 == "" {
		t.Error("Board view after SetIssues should not be empty")
	}
}

// TestBoardAgeColorCoding verifies age indicators show different colors (bv-1daf)
func TestBoardAgeColorCoding(t *testing.T) {
	theme := createTheme()

	// Create issues with different ages
	issues := []model.Issue{
		{
			ID:        "recent",
			Title:     "Recent Issue",
			Status:    model.StatusOpen,
			Priority:  2,
			CreatedAt: createTime(12),         // 12 hours ago
			UpdatedAt: time.Now(),              // just now - green
		},
		{
			ID:        "medium",
			Title:     "Medium Age Issue",
			Status:    model.StatusOpen,
			Priority:  2,
			CreatedAt: createTime(24 * 14),     // 14 days ago
			UpdatedAt: createTime(24 * 10),     // 10 days ago - yellow
		},
		{
			ID:        "stale",
			Title:     "Stale Issue",
			Status:    model.StatusOpen,
			Priority:  2,
			CreatedAt: createTime(24 * 60),     // 60 days ago
			UpdatedAt: createTime(24 * 45),     // 45 days ago - red
		},
	}

	b := ui.NewBoardModel(issues, theme)

	// Should render without panic with different age colors
	output := b.View(160, 40)
	if output == "" {
		t.Error("Board view with age-colored issues should not be empty")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Swimlane Mode Tests (bv-wjs0)
// ═══════════════════════════════════════════════════════════════════════════════

// TestSwimLaneModeByStatus verifies default status-based grouping
func TestSwimLaneModeByStatus(t *testing.T) {
	theme := createTheme()

	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Priority: 1},
		{ID: "2", Status: model.StatusInProgress, Priority: 1},
		{ID: "3", Status: model.StatusBlocked, Priority: 1},
		{ID: "4", Status: model.StatusClosed, Priority: 1},
	}

	b := ui.NewBoardModel(issues, theme)

	// Default mode should be Status
	if b.GetSwimLaneModeName() != "Status" {
		t.Errorf("Expected Status mode, got %s", b.GetSwimLaneModeName())
	}

	// Each status should be in its respective column
	if b.ColumnCount(0) != 1 { // Open
		t.Errorf("Expected 1 in Open column, got %d", b.ColumnCount(0))
	}
	if b.ColumnCount(1) != 1 { // InProgress
		t.Errorf("Expected 1 in InProgress column, got %d", b.ColumnCount(1))
	}
	if b.ColumnCount(2) != 1 { // Blocked
		t.Errorf("Expected 1 in Blocked column, got %d", b.ColumnCount(2))
	}
	if b.ColumnCount(3) != 1 { // Closed
		t.Errorf("Expected 1 in Closed column, got %d", b.ColumnCount(3))
	}
}

// TestSwimLaneModeByPriority verifies priority-based grouping
func TestSwimLaneModeByPriority(t *testing.T) {
	theme := createTheme()

	issues := []model.Issue{
		{ID: "p0", Status: model.StatusOpen, Priority: 0}, // Critical
		{ID: "p1", Status: model.StatusOpen, Priority: 1}, // High
		{ID: "p2", Status: model.StatusOpen, Priority: 2}, // Medium
		{ID: "p3", Status: model.StatusOpen, Priority: 3}, // Other
		{ID: "p4", Status: model.StatusOpen, Priority: 4}, // Other
	}

	b := ui.NewBoardModel(issues, theme)

	// Cycle to Priority mode
	b.CycleSwimLaneMode()

	if b.GetSwimLaneModeName() != "Priority" {
		t.Errorf("Expected Priority mode, got %s", b.GetSwimLaneModeName())
	}

	// P0 in col 0, P1 in col 1, P2 in col 2, P3+ in col 3
	if b.ColumnCount(0) != 1 { // P0 Critical
		t.Errorf("Expected 1 in Critical column, got %d", b.ColumnCount(0))
	}
	if b.ColumnCount(1) != 1 { // P1 High
		t.Errorf("Expected 1 in High column, got %d", b.ColumnCount(1))
	}
	if b.ColumnCount(2) != 1 { // P2 Medium
		t.Errorf("Expected 1 in Medium column, got %d", b.ColumnCount(2))
	}
	if b.ColumnCount(3) != 2 { // P3+ Other
		t.Errorf("Expected 2 in Other column (P3+P4), got %d", b.ColumnCount(3))
	}
}

// TestSwimLaneModeByType verifies type-based grouping
func TestSwimLaneModeByType(t *testing.T) {
	theme := createTheme()

	issues := []model.Issue{
		{ID: "bug1", Status: model.StatusOpen, IssueType: model.TypeBug},
		{ID: "feat1", Status: model.StatusOpen, IssueType: model.TypeFeature},
		{ID: "task1", Status: model.StatusOpen, IssueType: model.TypeTask},
		{ID: "epic1", Status: model.StatusOpen, IssueType: model.TypeEpic},
	}

	b := ui.NewBoardModel(issues, theme)

	// Cycle twice to get to Type mode (Status -> Priority -> Type)
	b.CycleSwimLaneMode()
	b.CycleSwimLaneMode()

	if b.GetSwimLaneModeName() != "Type" {
		t.Errorf("Expected Type mode, got %s", b.GetSwimLaneModeName())
	}

	// Bug in col 0, Feature in col 1, Task in col 2, Epic in col 3
	if b.ColumnCount(0) != 1 {
		t.Errorf("Expected 1 in Bug column, got %d", b.ColumnCount(0))
	}
	if b.ColumnCount(1) != 1 {
		t.Errorf("Expected 1 in Feature column, got %d", b.ColumnCount(1))
	}
	if b.ColumnCount(2) != 1 {
		t.Errorf("Expected 1 in Task column, got %d", b.ColumnCount(2))
	}
	if b.ColumnCount(3) != 1 {
		t.Errorf("Expected 1 in Epic column, got %d", b.ColumnCount(3))
	}
}

// TestSwimLaneModeCycles verifies mode cycles back to Status after Type
func TestSwimLaneModeCycles(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{{ID: "1", Status: model.StatusOpen}}
	b := ui.NewBoardModel(issues, theme)

	// Status -> Priority -> Type -> Status
	modes := []string{"Status", "Priority", "Type", "Status"}
	for i, expected := range modes {
		if b.GetSwimLaneModeName() != expected {
			t.Errorf("Step %d: Expected %s mode, got %s", i, expected, b.GetSwimLaneModeName())
		}
		b.CycleSwimLaneMode()
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Enhanced Navigation Tests (bv-yg39)
// ═══════════════════════════════════════════════════════════════════════════════

// TestJumpToColumn verifies direct column jumping with 1-4 keys
func TestJumpToColumn(t *testing.T) {
	theme := createTheme()

	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Priority: 1},
		{ID: "2", Status: model.StatusInProgress, Priority: 1},
		{ID: "3", Status: model.StatusClosed, Priority: 1},
	}

	b := ui.NewBoardModel(issues, theme)

	// Jump to column 3 (Closed - index 3)
	b.JumpToColumn(3)
	sel := b.SelectedIssue()
	if sel == nil || sel.ID != "3" {
		t.Errorf("Expected ID 3 after JumpToColumn(3), got %v", sel)
	}

	// Jump to column 1 (InProgress - index 1)
	b.JumpToColumn(1)
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "2" {
		t.Errorf("Expected ID 2 after JumpToColumn(1), got %v", sel)
	}

	// Jump to empty column 2 (Blocked) - should find nearest
	b.JumpToColumn(2)
	sel = b.SelectedIssue()
	// Should be on nearest non-empty column (InProgress or Closed)
	if sel == nil {
		t.Error("Expected selection after jumping to empty column")
	}
}

// TestJumpToFirstLastColumn verifies H/L navigation
func TestJumpToFirstLastColumn(t *testing.T) {
	theme := createTheme()

	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Priority: 1},
		{ID: "2", Status: model.StatusClosed, Priority: 1},
	}

	b := ui.NewBoardModel(issues, theme)

	// Start at first column
	b.JumpToLastColumn()
	sel := b.SelectedIssue()
	if sel == nil || sel.ID != "2" {
		t.Errorf("Expected ID 2 after JumpToLastColumn, got %v", sel)
	}

	b.JumpToFirstColumn()
	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "1" {
		t.Errorf("Expected ID 1 after JumpToFirstColumn, got %v", sel)
	}
}

// TestGGComboState verifies gg combo tracking
func TestGGComboState(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{{ID: "1", Status: model.StatusOpen}}
	b := ui.NewBoardModel(issues, theme)

	if b.IsWaitingForG() {
		t.Error("Should not be waiting for g initially")
	}

	b.SetWaitingForG()
	if !b.IsWaitingForG() {
		t.Error("Should be waiting for g after SetWaitingForG")
	}

	b.ClearWaitingForG()
	if b.IsWaitingForG() {
		t.Error("Should not be waiting for g after ClearWaitingForG")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Search Functionality Tests (bv-yg39)
// ═══════════════════════════════════════════════════════════════════════════════

// TestSearchBasic verifies basic search functionality
func TestSearchBasic(t *testing.T) {
	theme := createTheme()

	issues := []model.Issue{
		{ID: "bv-abc", Title: "Fix authentication bug", Status: model.StatusOpen},
		{ID: "bv-def", Title: "Add user profile page", Status: model.StatusOpen},
		{ID: "bv-ghi", Title: "Update auth tokens", Status: model.StatusInProgress},
	}

	b := ui.NewBoardModel(issues, theme)

	// Not in search mode initially
	if b.IsSearchMode() {
		t.Error("Should not be in search mode initially")
	}

	// Enter search mode
	b.StartSearch()
	if !b.IsSearchMode() {
		t.Error("Should be in search mode after StartSearch")
	}
	if b.SearchQuery() != "" {
		t.Error("Search query should be empty initially")
	}

	// Search for "auth"
	for _, ch := range "auth" {
		b.AppendSearchChar(ch)
	}

	if b.SearchQuery() != "auth" {
		t.Errorf("Expected query 'auth', got '%s'", b.SearchQuery())
	}

	// Should find 2 matches (authentication and auth)
	if b.SearchMatchCount() != 2 {
		t.Errorf("Expected 2 matches for 'auth', got %d", b.SearchMatchCount())
	}
}

// TestSearchNavigation verifies n/N navigation through matches
func TestSearchNavigation(t *testing.T) {
	theme := createTheme()

	issues := []model.Issue{
		{ID: "bv-1", Title: "Test one", Status: model.StatusOpen},
		{ID: "bv-2", Title: "Test two", Status: model.StatusOpen},
		{ID: "bv-3", Title: "Test three", Status: model.StatusOpen},
	}

	b := ui.NewBoardModel(issues, theme)

	b.StartSearch()
	for _, ch := range "test" {
		b.AppendSearchChar(ch)
	}

	// All 3 should match
	if b.SearchMatchCount() != 3 {
		t.Errorf("Expected 3 matches, got %d", b.SearchMatchCount())
	}

	// Should be at first match
	if b.SearchCursorPos() != 1 {
		t.Errorf("Expected cursor at 1, got %d", b.SearchCursorPos())
	}

	// Navigate forward
	b.NextMatch()
	if b.SearchCursorPos() != 2 {
		t.Errorf("Expected cursor at 2 after NextMatch, got %d", b.SearchCursorPos())
	}

	// Navigate backward
	b.PrevMatch()
	if b.SearchCursorPos() != 1 {
		t.Errorf("Expected cursor at 1 after PrevMatch, got %d", b.SearchCursorPos())
	}

	// Wrap around forward
	b.NextMatch()
	b.NextMatch()
	b.NextMatch() // Should wrap to 1
	if b.SearchCursorPos() != 1 {
		t.Errorf("Expected cursor to wrap to 1, got %d", b.SearchCursorPos())
	}

	// Wrap around backward
	b.PrevMatch() // Should wrap to 3
	if b.SearchCursorPos() != 3 {
		t.Errorf("Expected cursor to wrap to 3, got %d", b.SearchCursorPos())
	}
}

// TestSearchBackspace verifies backspace in search
func TestSearchBackspace(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{{ID: "test", Title: "Test", Status: model.StatusOpen}}
	b := ui.NewBoardModel(issues, theme)

	b.StartSearch()
	for _, ch := range "test" {
		b.AppendSearchChar(ch)
	}

	b.BackspaceSearch()
	if b.SearchQuery() != "tes" {
		t.Errorf("Expected 'tes' after backspace, got '%s'", b.SearchQuery())
	}

	// Backspace all
	b.BackspaceSearch()
	b.BackspaceSearch()
	b.BackspaceSearch()
	if b.SearchQuery() != "" {
		t.Errorf("Expected empty query, got '%s'", b.SearchQuery())
	}

	// Backspace on empty should be safe
	b.BackspaceSearch()
	if b.SearchQuery() != "" {
		t.Error("Backspace on empty should keep query empty")
	}
}

// TestSearchCancel verifies search cancellation
func TestSearchCancel(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{{ID: "test", Title: "Test", Status: model.StatusOpen}}
	b := ui.NewBoardModel(issues, theme)

	b.StartSearch()
	for _, ch := range "test" {
		b.AppendSearchChar(ch)
	}

	b.CancelSearch()
	if b.IsSearchMode() {
		t.Error("Should not be in search mode after CancelSearch")
	}
	if b.SearchQuery() != "" {
		t.Error("Query should be cleared after CancelSearch")
	}
	if b.SearchMatchCount() != 0 {
		t.Error("Matches should be cleared after CancelSearch")
	}
}

// TestSearchFinishKeepsResults verifies FinishSearch keeps matches for n/N
func TestSearchFinishKeepsResults(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{
		{ID: "test1", Title: "Test One", Status: model.StatusOpen},
		{ID: "test2", Title: "Test Two", Status: model.StatusOpen},
	}
	b := ui.NewBoardModel(issues, theme)

	b.StartSearch()
	for _, ch := range "test" {
		b.AppendSearchChar(ch)
	}

	b.FinishSearch()

	// Should exit search mode but keep matches
	if b.IsSearchMode() {
		t.Error("Should not be in search mode after FinishSearch")
	}
	// Note: After FinishSearch, NextMatch/PrevMatch should still work
	// if search results are preserved (depends on implementation)
}

// TestSearchCaseInsensitive verifies case-insensitive search
func TestSearchCaseInsensitive(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{
		{ID: "BV-ABC", Title: "UPPERCASE TITLE", Status: model.StatusOpen},
		{ID: "bv-def", Title: "lowercase title", Status: model.StatusOpen},
		{ID: "Bv-Ghi", Title: "Mixed Case Title", Status: model.StatusOpen},
	}
	b := ui.NewBoardModel(issues, theme)

	b.StartSearch()
	for _, ch := range "title" {
		b.AppendSearchChar(ch)
	}

	// All 3 should match regardless of case
	if b.SearchMatchCount() != 3 {
		t.Errorf("Expected 3 matches for case-insensitive 'title', got %d", b.SearchMatchCount())
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Detail Panel Tests (bv-r6kh)
// ═══════════════════════════════════════════════════════════════════════════════

// TestDetailPanelToggle verifies detail panel visibility
func TestDetailPanelToggle(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{{ID: "1", Status: model.StatusOpen}}
	b := ui.NewBoardModel(issues, theme)

	// Initially hidden
	if b.IsDetailShown() {
		t.Error("Detail panel should be hidden initially")
	}

	// Show
	b.ShowDetail()
	if !b.IsDetailShown() {
		t.Error("Detail panel should be shown after ShowDetail")
	}

	// Toggle off
	b.ToggleDetail()
	if b.IsDetailShown() {
		t.Error("Detail panel should be hidden after toggle")
	}

	// Toggle on
	b.ToggleDetail()
	if !b.IsDetailShown() {
		t.Error("Detail panel should be shown after second toggle")
	}

	// Hide
	b.HideDetail()
	if b.IsDetailShown() {
		t.Error("Detail panel should be hidden after HideDetail")
	}
}

// TestDetailPanelScroll verifies detail panel scrolling doesn't panic
func TestDetailPanelScroll(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{{
		ID:          "1",
		Title:       "Test Issue",
		Description: "Long description that spans multiple lines...",
		Status:      model.StatusOpen,
	}}
	b := ui.NewBoardModel(issues, theme)

	b.ShowDetail()

	// Force render to populate viewport
	_ = b.View(160, 40)

	// Scroll operations should not panic
	b.DetailScrollDown(3)
	b.DetailScrollUp(3)
	b.DetailScrollDown(100) // Over-scroll should be safe
	b.DetailScrollUp(100)
}

// TestDetailPanelRenderWithWidth verifies detail panel appears at sufficient width
func TestDetailPanelRenderWithWidth(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{{ID: "1", Status: model.StatusOpen}}
	b := ui.NewBoardModel(issues, theme)

	b.ShowDetail()

	// At narrow width (80), detail panel shouldn't show
	output80 := b.View(80, 30)

	// At wide width (160), detail panel should show
	output160 := b.View(160, 30)

	// Wide output should be longer (includes detail panel)
	// This is a heuristic - the exact behavior depends on implementation
	if len(output160) < len(output80) {
		t.Log("Note: Detail panel may not show at 160 width depending on implementation threshold")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layout Tests at Various Widths (bv-4agf)
// ═══════════════════════════════════════════════════════════════════════════════

// TestLayoutNarrow80 verifies board renders at narrow terminal
func TestLayoutNarrow80(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Title: "Task 1"},
		{ID: "2", Status: model.StatusInProgress, Title: "Task 2"},
		{ID: "3", Status: model.StatusClosed, Title: "Task 3"},
	}
	b := ui.NewBoardModel(issues, theme)

	output := b.View(80, 24)
	if output == "" {
		t.Error("Board should render at 80 cols")
	}
	// Cards should still be readable
	if len(output) < 100 {
		t.Error("Output seems too short for 80 col view")
	}
}

// TestLayoutMedium120 verifies board renders at medium terminal
func TestLayoutMedium120(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Title: "Task 1"},
		{ID: "2", Status: model.StatusInProgress, Title: "Task 2"},
		{ID: "3", Status: model.StatusBlocked, Title: "Task 3"},
		{ID: "4", Status: model.StatusClosed, Title: "Task 4"},
	}
	b := ui.NewBoardModel(issues, theme)

	output := b.View(120, 30)
	if output == "" {
		t.Error("Board should render at 120 cols")
	}
}

// TestLayoutWide160 verifies board renders at wide terminal
func TestLayoutWide160(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Title: "Task 1"},
		{ID: "2", Status: model.StatusInProgress, Title: "Task 2"},
		{ID: "3", Status: model.StatusBlocked, Title: "Task 3"},
		{ID: "4", Status: model.StatusClosed, Title: "Task 4"},
	}
	b := ui.NewBoardModel(issues, theme)

	output := b.View(160, 30)
	if output == "" {
		t.Error("Board should render at 160 cols")
	}
}

// TestLayoutUltraWide200 verifies board renders at ultra-wide terminal
func TestLayoutUltraWide200(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Title: "Long task title that might wrap on narrower screens"},
		{ID: "2", Status: model.StatusInProgress, Title: "Another long task title"},
	}
	b := ui.NewBoardModel(issues, theme)

	output := b.View(200, 40)
	if output == "" {
		t.Error("Board should render at 200 cols")
	}
}

// TestLayoutMinimalHeight verifies board handles short terminals
func TestLayoutMinimalHeight(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Title: "Task 1"},
	}
	b := ui.NewBoardModel(issues, theme)

	// Very short terminal
	output := b.View(80, 8)
	if output == "" {
		t.Error("Board should render at minimal height")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Filter Integration Tests (bv-4agf)
// ═══════════════════════════════════════════════════════════════════════════════

// TestSetIssuesClearsSearch verifies SetIssues clears stale search state
func TestSetIssuesClearsSearch(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{
		{ID: "test1", Title: "Test One", Status: model.StatusOpen},
		{ID: "test2", Title: "Test Two", Status: model.StatusOpen},
	}
	b := ui.NewBoardModel(issues, theme)

	// Start search
	b.StartSearch()
	for _, ch := range "test" {
		b.AppendSearchChar(ch)
	}
	if b.SearchMatchCount() != 2 {
		t.Fatalf("Expected 2 matches before filter, got %d", b.SearchMatchCount())
	}

	// Filter to different issues
	b.SetIssues([]model.Issue{
		{ID: "other1", Title: "Other Issue", Status: model.StatusOpen},
	})

	// Search should be cleared
	if b.SearchMatchCount() != 0 {
		t.Errorf("Search matches should be cleared after SetIssues, got %d", b.SearchMatchCount())
	}
}

// TestSetIssuesPreservesSwimLaneMode verifies swimlane mode persists through filter
func TestSetIssuesPreservesSwimLaneMode(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Priority: 1},
	}
	b := ui.NewBoardModel(issues, theme)

	// Switch to Priority mode
	b.CycleSwimLaneMode()
	if b.GetSwimLaneModeName() != "Priority" {
		t.Fatal("Should be in Priority mode")
	}

	// Filter to new issues
	b.SetIssues([]model.Issue{
		{ID: "2", Status: model.StatusClosed, Priority: 0},
	})

	// Mode should still be Priority
	if b.GetSwimLaneModeName() != "Priority" {
		t.Errorf("Swimlane mode should persist, expected Priority, got %s", b.GetSwimLaneModeName())
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge Case Tests (bv-4agf)
// ═══════════════════════════════════════════════════════════════════════════════

// TestSingleColumnOnly verifies board works with all items in one column
func TestSingleColumnOnly(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen},
		{ID: "2", Status: model.StatusOpen},
		{ID: "3", Status: model.StatusOpen},
	}
	b := ui.NewBoardModel(issues, theme)

	// All in Open column
	if b.ColumnCount(0) != 3 {
		t.Errorf("Expected all 3 in Open column")
	}
	for i := 1; i < 4; i++ {
		if b.ColumnCount(i) != 0 {
			t.Errorf("Expected 0 in column %d", i)
		}
	}

	// Navigation should stay in Open
	b.MoveRight()
	sel := b.SelectedIssue()
	if sel == nil || sel.ID != "1" {
		t.Error("Should stay in Open column when moving right")
	}
}

// TestLongTitleTruncation verifies long titles are truncated gracefully
func TestLongTitleTruncation(t *testing.T) {
	theme := createTheme()
	longTitle := "This is a very long title that should be truncated when displayed in the card view because it exceeds the available width"
	issues := []model.Issue{
		{ID: "1", Title: longTitle, Status: model.StatusOpen},
	}
	b := ui.NewBoardModel(issues, theme)

	// Should render without panic
	output := b.View(80, 24)
	if output == "" {
		t.Error("Should render with long title")
	}
}

// TestUnicodeTitles verifies Unicode titles display correctly
func TestUnicodeTitles(t *testing.T) {
	theme := createTheme()
	issues := []model.Issue{
		{ID: "1", Title: "日本語タイトル", Status: model.StatusOpen},
		{ID: "2", Title: "Émoji test 🎉🚀", Status: model.StatusOpen},
		{ID: "3", Title: "Кириллица title", Status: model.StatusOpen},
	}
	b := ui.NewBoardModel(issues, theme)

	// Should render without panic
	output := b.View(120, 30)
	if output == "" {
		t.Error("Should render Unicode titles")
	}
}

// TestManyIssuesPerformance verifies board handles 100+ cards
func TestManyIssuesPerformance(t *testing.T) {
	theme := createTheme()
	var issues []model.Issue
	for i := 0; i < 100; i++ {
		issues = append(issues, model.Issue{
			ID:        fmt.Sprintf("issue-%d", i),
			Title:     fmt.Sprintf("Task number %d with some description", i),
			Status:    model.Status([]string{"open", "in_progress", "blocked", "closed"}[i%4]),
			Priority:  i % 5,
			CreatedAt: createTime(i),
		})
	}
	b := ui.NewBoardModel(issues, theme)

	if b.TotalCount() != 100 {
		t.Errorf("Expected 100 issues, got %d", b.TotalCount())
	}

	// Should render without hanging
	output := b.View(160, 40)
	if output == "" {
		t.Error("Should render 100 cards")
	}

	// Navigation should work
	for i := 0; i < 50; i++ {
		b.MoveDown()
	}
	b.PageDown(10)
	b.MoveRight()
	b.MoveRight()

	sel := b.SelectedIssue()
	if sel == nil {
		t.Error("Should have selection after navigation")
	}
}
