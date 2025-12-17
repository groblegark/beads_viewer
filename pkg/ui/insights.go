package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Dicklesworthstone/beads_viewer/pkg/analysis"
	"github.com/Dicklesworthstone/beads_viewer/pkg/model"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// MetricPanel represents each panel type in the insights view
type MetricPanel int

const (
	PanelBottlenecks MetricPanel = iota
	PanelKeystones
	PanelInfluencers
	PanelHubs
	PanelAuthorities
	PanelCores
	PanelArticulation
	PanelSlack
	PanelCycles
	PanelPriority // Agent-first priority recommendations
	PanelCount    // Sentinel for wrapping
)

// MetricInfo contains explanation for each metric
type MetricInfo struct {
	Icon        string
	Title       string
	ShortDesc   string
	WhatIs      string
	WhyUseful   string
	HowToUse    string
	FormulaHint string
}

var metricDescriptions = map[MetricPanel]MetricInfo{
	PanelBottlenecks: {
		Icon:        "🚧",
		Title:       "Bottlenecks",
		ShortDesc:   "Betweenness Centrality",
		WhatIs:      "Measures how often a bead lies on **shortest paths** between other beads in the dependency graph.",
		WhyUseful:   "High-scoring beads are *critical junctions*. Delays here ripple across the entire project.",
		HowToUse:    "**Prioritize** these to unblock parallel workstreams. Consider breaking them into smaller pieces.",
		FormulaHint: "`BW(v) = Σ (σst(v) / σst)` for all s≠v≠t",
	},
	PanelKeystones: {
		Icon:        "🏛️",
		Title:       "Keystones",
		ShortDesc:   "Impact Depth",
		WhatIs:      "Measures how **deep** in the dependency chain a bead sits (downstream chain length).",
		WhyUseful:   "Keystones are *foundational*. Everything above them depends on their completion.",
		HowToUse:    "**Complete these first.** Blocking a keystone blocks the entire chain above it.",
		FormulaHint: "`Impact(v) = 1 + max(Impact(u))` for all u depending on v",
	},
	PanelInfluencers: {
		Icon:        "🌐",
		Title:       "Influencers",
		ShortDesc:   "Eigenvector Centrality",
		WhatIs:      "Scores beads by their connections to other **well-connected** beads.",
		WhyUseful:   "Influencers are connected to *important* beads. Changes here have wide-reaching effects.",
		HowToUse:    "**Review carefully** before changes. They're central to project structure.",
		FormulaHint: "`EV(v) = (1/λ) × Σ A[v,u] × EV(u)`",
	},
	PanelHubs: {
		Icon:        "🛰️",
		Title:       "Hubs",
		ShortDesc:   "HITS Hub Score",
		WhatIs:      "Beads that **depend on** many important authorities (aggregators).",
		WhyUseful:   "Hubs collect dependencies. They often represent *high-level features* or epics.",
		HowToUse:    "**Track for milestones.** Their completion signals major project progress.",
		FormulaHint: "`Hub(v) = Σ Authority(u)` for all u where v→u",
	},
	PanelAuthorities: {
		Icon:        "📚",
		Title:       "Authorities",
		ShortDesc:   "HITS Authority Score",
		WhatIs:      "Beads that are **depended upon** by many important hubs (providers).",
		WhyUseful:   "Authorities are *foundational services/components* that many features need.",
		HowToUse:    "**Stabilize early.** Breaking an authority breaks many dependent hubs.",
		FormulaHint: "`Auth(v) = Σ Hub(u)` for all u where u→v",
	},
	PanelCores: {
		Icon:        "🧠",
		Title:       "Cores",
		ShortDesc:   "k-core Cohesion",
		WhatIs:      "Nodes with highest **k-core numbers** (embedded in dense subgraphs).",
		WhyUseful:   "High-core nodes sit in *tightly knit clusters*—changes can ripple locally.",
		HowToUse:    "Use for **resilience checks**; prioritize when breaking apart tightly coupled areas.",
		FormulaHint: "Max `k` such that node remains in k-core after peeling",
	},
	PanelArticulation: {
		Icon:        "🪢",
		Title:       "Cut Points",
		ShortDesc:   "Articulation Vertices",
		WhatIs:      "Nodes whose **removal disconnects** the undirected graph.",
		WhyUseful:   "*Single points of failure.* Instability here can isolate workstreams.",
		HowToUse:    "**Harden or split** these nodes; avoid piling more dependencies onto them.",
		FormulaHint: "Tarjan articulation detection on undirected view",
	},
	PanelSlack: {
		Icon:        "⏳",
		Title:       "Slack",
		ShortDesc:   "Longest-path slack",
		WhatIs:      "Distance from **critical chain** (`0` = critical path; higher = parallel-friendly).",
		WhyUseful:   "Zero-slack tasks are *schedule-critical*; high-slack can fill gaps without blocking.",
		HowToUse:    "**Schedule zero-slack early**; slot high-slack tasks when waiting on blockers.",
		FormulaHint: "`Slack(v) = max_path_len - dist_start(v) - dist_end(v)`",
	},
	PanelCycles: {
		Icon:        "🔄",
		Title:       "Cycles",
		ShortDesc:   "Circular Dependencies",
		WhatIs:      "Groups of beads forming **dependency loops** (A→B→C→A).",
		WhyUseful:   "Cycles indicate *structural problems*. They can't be resolved in sequence.",
		HowToUse:    "**Break cycles** by removing or reversing a dependency. Refactor to decouple.",
		FormulaHint: "Detected via Tarjan's SCC algorithm",
	},
	PanelPriority: {
		Icon:        "🎯",
		Title:       "Priority",
		ShortDesc:   "Agent-First Triage",
		WhatIs:      "AI-computed recommendations combining **multiple signals** into actionable picks.",
		WhyUseful:   "Provides the *single best answer* for 'what should I work on next?'",
		HowToUse:    "**Work top to bottom.** High scores = high impact. Check unblocks count.",
		FormulaHint: "`Score = Σ(PageRank + Betweenness + BlockerRatio + ...)`",
	},
}

// InsightsModel is an interactive insights dashboard
type InsightsModel struct {
	insights       analysis.Insights
	issueMap       map[string]*model.Issue
	theme          Theme
	extraText      string
	labelAttention []analysis.LabelAttentionScore
	labelFlow      *analysis.CrossLabelFlow

	// Priority triage data (bv-91)
	topPicks []analysis.TopPick

	// Priority radar data (bv-93) - full recommendations with breakdown
	recommendations    []analysis.Recommendation
	recommendationMap  map[string]*analysis.Recommendation // ID -> Recommendation for quick lookup
	triageDataHash     string                              // Hash of data used for triage

	// Navigation state
	focusedPanel  MetricPanel
	selectedIndex [PanelCount]int // Selection per panel
	scrollOffset  [PanelCount]int // Scroll offset per panel

	// View options
	showExplanations bool
	showCalculation  bool
	showDetailPanel  bool
	showHeatmap      bool // Toggle between list and heatmap view (bv-95)

	// Markdown rendering for detail panel (bv-ui-polish)
	mdRenderer    *MarkdownRenderer
	detailVP      viewport.Model
	detailContent string // cached markdown content

	// Dimensions
	width  int
	height int
	ready  bool
}

// NewInsightsModel creates a new interactive insights model
func NewInsightsModel(ins analysis.Insights, issueMap map[string]*model.Issue, theme Theme) InsightsModel {
	// Initialize markdown renderer with theme for consistent styling
	mdRenderer := NewMarkdownRendererWithTheme(50, theme)

	// Initialize viewport for detail panel scrolling
	vp := viewport.New(50, 20)
	vp.Style = theme.Renderer.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(0, 1)

	return InsightsModel{
		insights:         ins,
		issueMap:         issueMap,
		theme:            theme,
		showExplanations: true,  // Visible by default
		showCalculation:  true,  // Always show calculation details
		showDetailPanel:  true,
		mdRenderer:       mdRenderer,
		detailVP:         vp,
	}
}

func (m *InsightsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.ready = true

	// Update detail panel viewport and markdown renderer dimensions
	if m.showDetailPanel && w > 120 {
		detailWidth := min(60, w/3)
		m.detailVP.Width = detailWidth - 4 // Account for border/padding
		m.detailVP.Height = h - 4
		if m.mdRenderer != nil {
			m.mdRenderer.SetWidthWithTheme(detailWidth-6, m.theme)
		}
	}
}

func (m *InsightsModel) SetInsights(ins analysis.Insights) {
	m.insights = ins
}

// SetTopPicks sets the priority triage recommendations (bv-91)
func (m *InsightsModel) SetTopPicks(picks []analysis.TopPick) {
	m.topPicks = picks
}

// SetRecommendations sets the full recommendations with breakdown data (bv-93)
func (m *InsightsModel) SetRecommendations(recs []analysis.Recommendation, dataHash string) {
	m.recommendations = recs
	m.triageDataHash = dataHash
	// Build lookup map
	m.recommendationMap = make(map[string]*analysis.Recommendation, len(recs))
	for i := range recs {
		m.recommendationMap[recs[i].ID] = &recs[i]
	}
}

// isPanelSkipped returns true and a reason if the metric for this panel was skipped
func (m *InsightsModel) isPanelSkipped(panel MetricPanel) (bool, string) {
	if m.insights.Stats == nil {
		return false, ""
	}

	// Check runtime status first (covers timeouts and dynamic skips)
	status := m.insights.Stats.Status()
	switch panel {
	case PanelBottlenecks:
		if status.Betweenness.State == "skipped" || status.Betweenness.State == "timeout" {
			return true, status.Betweenness.Reason
		}
	case PanelHubs, PanelAuthorities:
		if status.HITS.State == "skipped" || status.HITS.State == "timeout" {
			return true, status.HITS.Reason
		}
	case PanelCycles:
		if status.Cycles.State == "skipped" || status.Cycles.State == "timeout" {
			return true, status.Cycles.Reason
		}
	case PanelKeystones, PanelSlack: // Critical Path / Slack
		if status.Critical.State == "skipped" || status.Critical.State == "timeout" {
			return true, status.Critical.Reason
		}
	case PanelInfluencers: // Eigenvector
		if status.Eigenvector.State == "skipped" || status.Eigenvector.State == "timeout" {
			return true, status.Eigenvector.Reason
		}
	}

	// Fallback to config check (should be covered by status, but safe to keep)
	config := m.insights.Stats.Config

	switch panel {
	case PanelBottlenecks:
		if !config.ComputeBetweenness {
			return true, config.BetweennessSkipReason
		}
	case PanelHubs, PanelAuthorities:
		if !config.ComputeHITS {
			return true, config.HITSSkipReason
		}
	case PanelCycles:
		if !config.ComputeCycles {
			return true, config.CyclesSkipReason
		}
	}
	return false, ""
}

// Navigation methods
func (m *InsightsModel) MoveUp() {
	count := m.currentPanelItemCount()
	if count == 0 {
		return
	}
	if m.selectedIndex[m.focusedPanel] > 0 {
		m.selectedIndex[m.focusedPanel]--
	}
}

func (m *InsightsModel) MoveDown() {
	count := m.currentPanelItemCount()
	if count == 0 {
		return
	}
	if m.selectedIndex[m.focusedPanel] < count-1 {
		m.selectedIndex[m.focusedPanel]++
	}
}

func (m *InsightsModel) NextPanel() {
	m.focusedPanel = (m.focusedPanel + 1) % PanelCount
}

func (m *InsightsModel) PrevPanel() {
	if m.focusedPanel == 0 {
		m.focusedPanel = PanelCount - 1
	} else {
		m.focusedPanel--
	}
}

func (m *InsightsModel) ToggleExplanations() {
	m.showExplanations = !m.showExplanations
}

func (m *InsightsModel) ToggleCalculation() {
	m.showCalculation = !m.showCalculation
}

// ToggleHeatmap toggles between priority list and heatmap view (bv-95)
func (m *InsightsModel) ToggleHeatmap() {
	m.showHeatmap = !m.showHeatmap
}

// currentPanelItemCount returns the number of items in the focused panel (including cycles)
func (m *InsightsModel) currentPanelItemCount() int {
	switch m.focusedPanel {
	case PanelBottlenecks:
		return len(m.insights.Bottlenecks)
	case PanelKeystones:
		return len(m.insights.Keystones)
	case PanelInfluencers:
		return len(m.insights.Influencers)
	case PanelHubs:
		return len(m.insights.Hubs)
	case PanelAuthorities:
		return len(m.insights.Authorities)
	case PanelCores:
		return len(m.insights.Cores)
	case PanelArticulation:
		return len(m.insights.Articulation)
	case PanelSlack:
		return len(m.insights.Slack)
	case PanelCycles:
		return len(m.insights.Cycles)
	case PanelPriority:
		return len(m.topPicks)
	default:
		return 0
	}
}

// getPanelItems returns the InsightItems for a given panel (nil for cycles)
func (m *InsightsModel) getPanelItems(panel MetricPanel) []analysis.InsightItem {
	switch panel {
	case PanelBottlenecks:
		return m.insights.Bottlenecks
	case PanelKeystones:
		return m.insights.Keystones
	case PanelInfluencers:
		return m.insights.Influencers
	case PanelHubs:
		return m.insights.Hubs
	case PanelAuthorities:
		return m.insights.Authorities
	case PanelCores:
		return m.insights.Cores
	case PanelArticulation:
		items := make([]analysis.InsightItem, 0, len(m.insights.Articulation))
		for _, id := range m.insights.Articulation {
			items = append(items, analysis.InsightItem{ID: id, Value: 0})
		}
		return items
	case PanelSlack:
		return m.insights.Slack
	default:
		return nil
	}
}

// SelectedIssueID returns the currently selected issue ID
func (m *InsightsModel) SelectedIssueID() string {
	// For cycles panel, return first item in selected cycle
	if m.focusedPanel == PanelCycles {
		idx := m.selectedIndex[PanelCycles]
		if idx >= 0 && idx < len(m.insights.Cycles) && len(m.insights.Cycles[idx]) > 0 {
			return m.insights.Cycles[idx][0]
		}
		return ""
	}

	// For priority panel, return selected TopPick's ID
	if m.focusedPanel == PanelPriority {
		idx := m.selectedIndex[PanelPriority]
		if idx >= 0 && idx < len(m.topPicks) {
			return m.topPicks[idx].ID
		}
		return ""
	}

	// For other panels, return selected item's ID
	items := m.getPanelItems(m.focusedPanel)
	idx := m.selectedIndex[m.focusedPanel]
	if idx >= 0 && idx < len(items) {
		return items[idx].ID
	}
	return ""
}

// View renders the insights dashboard (pointer receiver to persist scroll state)
func (m *InsightsModel) View() string {
	if !m.ready {
		return ""
	}

	if m.extraText != "" {
		return m.theme.Base.Render(m.extraText)
	}

	t := m.theme

	// Optional throughput summary
	velocityLine := ""
	if m.insights.Velocity != nil {
		v := m.insights.Velocity
		weekly := ""
		if len(v.Weekly) > 0 {
			limit := min(3, len(v.Weekly))
			parts := make([]string, 0, limit)
			for i := 0; i < limit; i++ {
				parts = append(parts, fmt.Sprintf("%d", v.Weekly[i]))
			}
			weekly = fmt.Sprintf(" • weekly: [%s]", strings.Join(parts, ","))
		}
		estimate := ""
		if v.Estimated {
			estimate = " (estimated)"
		}
		velocityLine = t.Base.Render(fmt.Sprintf("Velocity: 7d=%d, 30d=%d, avg=%.1fd%s%s",
			v.Closed7, v.Closed30, v.AvgDays, weekly, estimate))
	}

	// Calculate layout dimensions
	mainWidth := m.width
	detailWidth := 0
	if m.showDetailPanel && m.width > 120 {
		detailWidth = min(50, m.width/3)
		mainWidth = m.width - detailWidth - 1
	}

	// 3-column layout; 4 rows (3 metric rows + 1 priority row)
	colWidth := (mainWidth - 6) / 3
	if colWidth < 25 {
		colWidth = 25
	}

	// With 4 rows, reduce individual row height
	rowHeight := (m.height - 8) / 4
	if rowHeight < 6 {
		rowHeight = 6
	}

	panels := []string{
		m.renderMetricPanel(PanelBottlenecks, colWidth, rowHeight, t),
		m.renderMetricPanel(PanelKeystones, colWidth, rowHeight, t),
		m.renderMetricPanel(PanelInfluencers, colWidth, rowHeight, t),
		m.renderMetricPanel(PanelHubs, colWidth, rowHeight, t),
		m.renderMetricPanel(PanelAuthorities, colWidth, rowHeight, t),
		m.renderMetricPanel(PanelCores, colWidth, rowHeight, t),
		m.renderMetricPanel(PanelArticulation, colWidth, rowHeight, t),
		m.renderMetricPanel(PanelSlack, colWidth, rowHeight, t),
		m.renderCyclesPanel(colWidth, rowHeight, t),
	}

	row1 := lipgloss.JoinHorizontal(lipgloss.Top, panels[0], panels[1], panels[2])
	row2 := lipgloss.JoinHorizontal(lipgloss.Top, panels[3], panels[4], panels[5])
	row3 := lipgloss.JoinHorizontal(lipgloss.Top, panels[6], panels[7], panels[8])
	// Priority panel spans full width for prominence (bv-91)
	// Toggle between priority list and heatmap view (bv-95)
	var row4 string
	if m.showHeatmap {
		row4 = m.renderHeatmapPanel(mainWidth-2, rowHeight, t)
	} else {
		row4 = m.renderPriorityPanel(mainWidth-2, rowHeight, t)
	}

	mainContent := lipgloss.JoinVertical(lipgloss.Left, row1, row2, row3, row4)

	// Add detail panel if enabled
	if detailWidth > 0 {
		detailPanel := m.renderDetailPanel(detailWidth, m.height-2, t)
		view := lipgloss.JoinHorizontal(lipgloss.Top, mainContent, detailPanel)
		if velocityLine != "" {
			view = lipgloss.JoinVertical(lipgloss.Left, velocityLine, view)
		}
		return view
	}

	if velocityLine != "" {
		return lipgloss.JoinVertical(lipgloss.Left, velocityLine, mainContent)
	}
	return mainContent
}

func (m *InsightsModel) renderMetricPanel(panel MetricPanel, width, height int, t Theme) string {
	info := metricDescriptions[panel]
	items := m.getPanelItems(panel)
	isFocused := m.focusedPanel == panel
	selectedIdx := m.selectedIndex[panel]

	// Check if this metric was skipped
	skipped, skipReason := m.isPanelSkipped(panel)

	// Panel border style
	borderColor := t.Secondary
	if isFocused {
		borderColor = t.Primary
	}
	if skipped {
		borderColor = t.Subtext // Dimmed for skipped panels
	}

	panelStyle := t.Renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(height).
		Padding(0, 1)

	// Title with count and value range
	titleStyle := t.Renderer.NewStyle().Bold(true)
	if skipped {
		titleStyle = titleStyle.Foreground(t.Subtext)
	} else if isFocused {
		titleStyle = titleStyle.Foreground(t.Primary)
	} else {
		titleStyle = titleStyle.Foreground(t.Secondary)
	}

	var sb strings.Builder

	// Header line: Icon Title (count) or [Skipped]
	var headerLine string
	if skipped {
		headerLine = fmt.Sprintf("%s %s [Skipped]", info.Icon, info.Title)
	} else {
		headerLine = fmt.Sprintf("%s %s (%d)", info.Icon, info.Title, len(items))
	}
	sb.WriteString(titleStyle.Render(headerLine))
	sb.WriteString("\n")

	// Subtitle: metric name
	subtitleStyle := t.Renderer.NewStyle().Foreground(t.Subtext).Italic(true)
	if skipped {
		subtitleStyle = subtitleStyle.Foreground(t.Subtext)
	}
	sb.WriteString(subtitleStyle.Render(info.ShortDesc))
	sb.WriteString("\n")

	// Explanation (if enabled)
	if m.showExplanations {
		explainStyle := t.Renderer.NewStyle().
			Foreground(t.Secondary).
			Width(width - 4)
		sb.WriteString(explainStyle.Render(info.WhatIs))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// If metric was skipped, show skip reason instead of items
	if skipped {
		skipStyle := t.Renderer.NewStyle().
			Foreground(t.Subtext).
			Italic(true).
			Width(width - 4).
			Align(lipgloss.Center)

		reason := skipReason
		if reason == "" {
			reason = "Skipped for performance"
		}
		sb.WriteString("\n")
		sb.WriteString(skipStyle.Render(reason))
		sb.WriteString("\n\n")
		sb.WriteString(skipStyle.Render("Use --force-full-analysis to compute"))

		return panelStyle.Render(sb.String())
	}

	// Items list
	// Calculate visible rows more conservatively
	// Header(1) + Subtitle(1) + Explain(2-3 lines typically) + Spacer(1) + Scroll(1) = ~7 lines overhead
	visibleRows := height - 7
	if m.showExplanations {
		// Explanations can wrap, so give more buffer
		visibleRows -= 1
	}
	if visibleRows < 3 {
		visibleRows = 3
	}

	// Scrolling
	startIdx := m.scrollOffset[panel]
	if selectedIdx >= startIdx+visibleRows {
		startIdx = selectedIdx - visibleRows + 1
	}
	if selectedIdx < startIdx {
		startIdx = selectedIdx
	}
	m.scrollOffset[panel] = startIdx

	endIdx := startIdx + visibleRows
	if endIdx > len(items) {
		endIdx = len(items)
	}

	for i := startIdx; i < endIdx; i++ {
		item := items[i]
		isSelected := isFocused && i == selectedIdx

		row := m.renderInsightRow(item.ID, item.Value, width-4, isSelected, t)
		sb.WriteString(row)
		sb.WriteString("\n")
	}

	// Scroll indicator
	if len(items) > visibleRows {
		scrollInfo := fmt.Sprintf("↕ %d/%d", selectedIdx+1, len(items))
		scrollStyle := t.Renderer.NewStyle().
			Foreground(t.Subtext).
			Align(lipgloss.Center).
			Width(width - 4)
		sb.WriteString(scrollStyle.Render(scrollInfo))
	}

	return panelStyle.Render(sb.String())
}

func (m *InsightsModel) renderInsightRow(id string, value float64, width int, isSelected bool, t Theme) string {
	issue := m.issueMap[id]

	// Format value
	var valueStr string
	if value >= 1.0 {
		valueStr = fmt.Sprintf("%.1f", value)
	} else if value >= 0.01 {
		valueStr = fmt.Sprintf("%.3f", value)
	} else {
		valueStr = fmt.Sprintf("%.2e", value)
	}

	// Build row content
	var rowBuilder strings.Builder

	// Selection indicator
	if isSelected {
		rowBuilder.WriteString(t.Renderer.NewStyle().Foreground(t.Primary).Bold(true).Render("▸ "))
	} else {
		rowBuilder.WriteString("  ")
	}

	// Value badge
	valueStyle := t.Renderer.NewStyle().
		Background(lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#3D3D3D"}).
		Foreground(t.Primary).
		Bold(true).
		Padding(0, 1)
	rowBuilder.WriteString(valueStyle.Render(valueStr))
	rowBuilder.WriteString(" ")

	// Issue content
	if issue != nil {
		// Type icon - measure actual display width for proper alignment
		icon, iconColor := t.GetTypeIcon(string(issue.IssueType))
		iconRendered := t.Renderer.NewStyle().Foreground(iconColor).Render(icon)
		rowBuilder.WriteString(iconRendered)
		rowBuilder.WriteString(" ")

		// Status indicator
		statusColor := t.GetStatusColor(string(issue.Status))
		statusDot := t.Renderer.NewStyle().Foreground(statusColor).Render("●")
		rowBuilder.WriteString(statusDot)
		rowBuilder.WriteString(" ")

		// Title (truncated) - leave room for description preview
		// Calculate actual used width by measuring rendered content
		// Selection(2) + valueBadge(rendered) + space(1) + icon(measured) + space(1) + dot(1) + space(1)
		usedWidth := 2 + lipgloss.Width(valueStyle.Render(valueStr)) + 1 + lipgloss.Width(icon) + 1 + 1 + 1
		remainingWidth := width - usedWidth
		titleWidth := remainingWidth * 2 / 3         // Title gets 2/3 of remaining
		descWidth := remainingWidth - titleWidth - 3 // -3 for " - "

		if titleWidth < 10 {
			titleWidth = 10
		}
		if descWidth < 5 {
			descWidth = 0 // Don't show description if not enough space
		}

		title := truncateRunesHelper(issue.Title, titleWidth, "…")

		titleStyle := t.Renderer.NewStyle()
		if isSelected {
			titleStyle = titleStyle.Foreground(t.Primary).Bold(true)
		}
		rowBuilder.WriteString(titleStyle.Render(title))

		// Description preview (if space allows)
		if descWidth > 0 && issue.Description != "" {
			// Clean up description - remove newlines, trim whitespace
			desc := strings.Join(strings.Fields(issue.Description), " ")
			desc = truncateRunesHelper(desc, descWidth, "…")
			descStyle := t.Renderer.NewStyle().Foreground(t.Subtext).Italic(true)
			rowBuilder.WriteString(t.Renderer.NewStyle().Foreground(t.Secondary).Render(" - "))
			rowBuilder.WriteString(descStyle.Render(desc))
		}
	} else {
		// Fallback: just show ID
		idTrunc := truncateRunesHelper(id, width-12-len(valueStr), "…")
		idStyle := t.Renderer.NewStyle().Foreground(t.Secondary)
		if isSelected {
			idStyle = idStyle.Foreground(t.Primary).Bold(true)
		}
		rowBuilder.WriteString(idStyle.Render(idTrunc))
	}

	return rowBuilder.String()
}

func (m *InsightsModel) renderCyclesPanel(width, height int, t Theme) string {
	info := metricDescriptions[PanelCycles]
	isFocused := m.focusedPanel == PanelCycles
	cycles := m.insights.Cycles

	// Check if cycles detection was skipped
	skipped, skipReason := m.isPanelSkipped(PanelCycles)

	borderColor := t.Secondary
	if isFocused {
		borderColor = t.Primary
	}
	if skipped {
		borderColor = t.Subtext // Dimmed for skipped panels
	}

	panelStyle := t.Renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(height).
		Padding(0, 1)

	titleStyle := t.Renderer.NewStyle().Bold(true)
	if skipped {
		titleStyle = titleStyle.Foreground(t.Subtext)
	} else if isFocused {
		titleStyle = titleStyle.Foreground(t.Primary)
	} else {
		titleStyle = titleStyle.Foreground(t.Secondary)
	}

	var sb strings.Builder

	// Header
	var headerLine string
	if skipped {
		headerLine = fmt.Sprintf("%s %s [Skipped]", info.Icon, info.Title)
	} else {
		headerLine = fmt.Sprintf("%s %s (%d)", info.Icon, info.Title, len(cycles))
	}
	sb.WriteString(titleStyle.Render(headerLine))
	sb.WriteString("\n")

	subtitleStyle := t.Renderer.NewStyle().Foreground(t.Subtext).Italic(true)
	sb.WriteString(subtitleStyle.Render(info.ShortDesc))
	sb.WriteString("\n")

	if m.showExplanations {
		explainStyle := t.Renderer.NewStyle().
			Foreground(t.Secondary).
			Width(width - 4)
		sb.WriteString(explainStyle.Render(info.WhatIs))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// If skipped, show skip reason
	if skipped {
		skipStyle := t.Renderer.NewStyle().
			Foreground(t.Subtext).
			Italic(true).
			Width(width - 4).
			Align(lipgloss.Center)

		reason := skipReason
		if reason == "" {
			reason = "Skipped for performance"
		}
		sb.WriteString("\n")
		sb.WriteString(skipStyle.Render(reason))
		sb.WriteString("\n\n")
		sb.WriteString(skipStyle.Render("Use --force-full-analysis to compute"))

		return panelStyle.Render(sb.String())
	}

	if len(cycles) == 0 {
		healthyStyle := t.Renderer.NewStyle().
			Foreground(t.Open).
			Bold(true)
		sb.WriteString(healthyStyle.Render("✓ No cycles detected"))
		sb.WriteString("\n")
		sb.WriteString(t.Renderer.NewStyle().Foreground(t.Subtext).Render("Graph is acyclic (DAG)"))
	} else {
		selectedIdx := m.selectedIndex[PanelCycles]
		visibleRows := height - 6
		if m.showExplanations {
			visibleRows -= 2
		}
		if visibleRows < 3 {
			visibleRows = 3
		}

		// Scrolling support for cycles (same logic as metric panels)
		startIdx := m.scrollOffset[PanelCycles]
		if selectedIdx >= startIdx+visibleRows {
			startIdx = selectedIdx - visibleRows + 1
		}
		if selectedIdx < startIdx {
			startIdx = selectedIdx
		}
		m.scrollOffset[PanelCycles] = startIdx

		endIdx := startIdx + visibleRows
		if endIdx > len(cycles) {
			endIdx = len(cycles)
		}

		for i := startIdx; i < endIdx; i++ {
			cycle := cycles[i]
			isSelected := isFocused && i == selectedIdx
			prefix := "  "
			if isSelected {
				prefix = t.Renderer.NewStyle().Foreground(t.Primary).Bold(true).Render("▸ ")
			}

			// Render cycle as chain
			cycleStr := m.renderCycleChain(cycle, width-6, t)

			warningStyle := t.Renderer.NewStyle().Foreground(t.Blocked)
			if isSelected {
				warningStyle = warningStyle.Bold(true)
			}

			sb.WriteString(prefix)
			sb.WriteString(warningStyle.Render(cycleStr))
			sb.WriteString("\n")
		}

		// Scroll indicator
		if len(cycles) > visibleRows {
			scrollInfo := fmt.Sprintf("↕ %d/%d", selectedIdx+1, len(cycles))
			scrollStyle := t.Renderer.NewStyle().
				Foreground(t.Subtext).
				Align(lipgloss.Center).
				Width(width - 4)
			sb.WriteString(scrollStyle.Render(scrollInfo))
		}
	}

	return panelStyle.Render(sb.String())
}

// renderPriorityPanel renders the priority recommendations panel (bv-91)
func (m *InsightsModel) renderPriorityPanel(width, height int, t Theme) string {
	info := metricDescriptions[PanelPriority]
	isFocused := m.focusedPanel == PanelPriority
	picks := m.topPicks

	borderColor := t.Secondary
	if isFocused {
		borderColor = t.Primary
	}

	panelStyle := t.Renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(height).
		Padding(0, 1)

	titleStyle := t.Renderer.NewStyle().Bold(true)
	if isFocused {
		titleStyle = titleStyle.Foreground(t.Primary)
	} else {
		titleStyle = titleStyle.Foreground(t.Secondary)
	}

	var sb strings.Builder

	// Header
	headerLine := fmt.Sprintf("%s %s (%d)", info.Icon, info.Title, len(picks))
	sb.WriteString(titleStyle.Render(headerLine))
	sb.WriteString("  ")

	// Inline subtitle for horizontal layout
	subtitleStyle := t.Renderer.NewStyle().Foreground(t.Subtext).Italic(true)
	sb.WriteString(subtitleStyle.Render(info.ShortDesc))
	sb.WriteString("\n")

	if len(picks) == 0 {
		emptyStyle := t.Renderer.NewStyle().
			Foreground(t.Subtext).
			Italic(true)
		sb.WriteString(emptyStyle.Render("No priority recommendations available. Run 'bv --robot-triage' to generate."))
		return panelStyle.Render(sb.String())
	}

	selectedIdx := m.selectedIndex[PanelPriority]
	// For horizontal layout, show items side by side
	visibleItems := min(len(picks), 5) // Show up to 5 items horizontally

	// Calculate width per item
	itemWidth := (width - 4) / visibleItems
	if itemWidth < 30 {
		itemWidth = 30
	}

	// Scrolling for selection
	startIdx := m.scrollOffset[PanelPriority]
	if selectedIdx >= startIdx+visibleItems {
		startIdx = selectedIdx - visibleItems + 1
	}
	if selectedIdx < startIdx {
		startIdx = selectedIdx
	}
	m.scrollOffset[PanelPriority] = startIdx

	endIdx := startIdx + visibleItems
	if endIdx > len(picks) {
		endIdx = len(picks)
	}

	// Render picks horizontally
	var pickRenderings []string
	for i := startIdx; i < endIdx; i++ {
		pick := picks[i]
		isSelected := isFocused && i == selectedIdx
		pickRenderings = append(pickRenderings, m.renderPriorityItem(pick, itemWidth, height-3, isSelected, t))
	}

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, pickRenderings...))

	// Scroll indicator
	if len(picks) > visibleItems {
		sb.WriteString("\n")
		scrollInfo := fmt.Sprintf("◀ %d/%d ▶", selectedIdx+1, len(picks))
		scrollStyle := t.Renderer.NewStyle().
			Foreground(t.Subtext).
			Align(lipgloss.Center).
			Width(width - 4)
		sb.WriteString(scrollStyle.Render(scrollInfo))
	}

	// Data hash footer (bv-93)
	if m.triageDataHash != "" {
		sb.WriteString("\n")
		hashStyle := t.Renderer.NewStyle().
			Foreground(t.Subtext).
			Italic(true).
			Align(lipgloss.Right).
			Width(width - 4)
		sb.WriteString(hashStyle.Render("📊 " + m.triageDataHash))
	}

	return panelStyle.Render(sb.String())
}

// renderMiniBar renders a compact progress bar for metric visualization (bv-93)
// label: 2-char label (e.g., "PR", "BW", "TI")
// value: normalized 0.0-1.0
// width: total width for the bar (including label)
func (m *InsightsModel) renderMiniBar(label string, value float64, width int, t Theme) string {
	// Ensure value is in range
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}

	prefix := label + ":"
	prefixLen := len([]rune(prefix))

	// Bar width = total - prefix
	barWidth := width - prefixLen
	if barWidth < 1 {
		// Not enough space for any bar
		if width >= prefixLen {
			return t.Renderer.NewStyle().Foreground(t.Subtext).Render(prefix)
		}
		return ""
	}

	filled := int(float64(barWidth) * value)
	if filled > barWidth {
		filled = barWidth
	}

	// Color based on value intensity
	var barColor lipgloss.AdaptiveColor
	switch {
	case value >= 0.7:
		barColor = lipgloss.AdaptiveColor{Light: "#50FA7B", Dark: "#50FA7B"} // Green - high
	case value >= 0.4:
		barColor = lipgloss.AdaptiveColor{Light: "#FFB86C", Dark: "#FFB86C"} // Orange - medium
	default:
		barColor = lipgloss.AdaptiveColor{Light: "#6272A4", Dark: "#6272A4"} // Gray - low
	}

	labelStyle := t.Renderer.NewStyle().Foreground(t.Subtext)
	filledStyle := t.Renderer.NewStyle().Foreground(barColor)
	emptyStyle := t.Renderer.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#3D3D3D", Dark: "#3D3D3D"})

	filledBar := strings.Repeat("█", filled)
	emptyBar := strings.Repeat("░", barWidth-filled)

	return labelStyle.Render(prefix) + filledStyle.Render(filledBar) + emptyStyle.Render(emptyBar)
}


// renderPriorityItem renders a single priority recommendation item
func (m *InsightsModel) renderPriorityItem(pick analysis.TopPick, width, height int, isSelected bool, t Theme) string {
	itemStyle := t.Renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(width-2).
		Height(height-1).
		Padding(0, 1)

	if isSelected {
		itemStyle = itemStyle.BorderForeground(t.Primary)
	} else {
		itemStyle = itemStyle.BorderForeground(t.Secondary)
	}

	var sb strings.Builder

	// Selection indicator + Score badge
	if isSelected {
		sb.WriteString(t.Renderer.NewStyle().Foreground(t.Primary).Bold(true).Render("▸ "))
	} else {
		sb.WriteString("  ")
	}

	// Score badge
	scoreStr := fmt.Sprintf("%.2f", pick.Score)
	scoreStyle := t.Renderer.NewStyle().
		Background(lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#3D3D3D"}).
		Foreground(t.Primary).
		Bold(true).
		Padding(0, 1)
	sb.WriteString(scoreStyle.Render(scoreStr))
	sb.WriteString("\n")

	// Issue details
	issue := m.issueMap[pick.ID]
	if issue != nil {
		// Type icon + Status
		icon, iconColor := t.GetTypeIcon(string(issue.IssueType))
		statusColor := t.GetStatusColor(string(issue.Status))

		sb.WriteString(t.Renderer.NewStyle().Foreground(iconColor).Render(icon))
		sb.WriteString(" ")
		sb.WriteString(t.Renderer.NewStyle().Foreground(statusColor).Bold(true).Render(strings.ToUpper(string(issue.Status))))
		sb.WriteString(" ")
		sb.WriteString(GetPriorityIcon(issue.Priority))
		sb.WriteString(fmt.Sprintf("P%d", issue.Priority))
		sb.WriteString("\n")

		// Title (truncated)
		titleWidth := width - 6
		title := truncateRunesHelper(issue.Title, titleWidth, "…")
		titleStyle := t.Renderer.NewStyle()
		if isSelected {
			titleStyle = titleStyle.Foreground(t.Primary).Bold(true)
		}
		sb.WriteString(titleStyle.Render(title))
		sb.WriteString("\n")
	} else {
		// Fallback to ID + Title from pick
		idStyle := t.Renderer.NewStyle().Foreground(t.Secondary)
		sb.WriteString(idStyle.Render(pick.ID))
		sb.WriteString("\n")
		titleStyle := t.Renderer.NewStyle()
		if isSelected {
			titleStyle = titleStyle.Foreground(t.Primary).Bold(true)
		}
		sb.WriteString(titleStyle.Render(truncateRunesHelper(pick.Title, width-6, "…")))
		sb.WriteString("\n")
	}

	// PR/BW/Impact mini-bars (bv-93)
	rec := m.recommendationMap[pick.ID]
	if rec != nil {
		barWidth := width - 4
		if barWidth > 20 {
			barWidth = 20 // Cap bar width for readability
		}
		sb.WriteString(m.renderMiniBar("PR", rec.Breakdown.PageRankNorm, barWidth, t))
		sb.WriteString(" ")
		sb.WriteString(m.renderMiniBar("BW", rec.Breakdown.BetweennessNorm, barWidth, t))
		sb.WriteString("\n")
		sb.WriteString(m.renderMiniBar("TI", rec.Breakdown.TimeToImpactNorm, barWidth, t))
		sb.WriteString("\n")
	}

	// Unblocks indicator
	if pick.Unblocks > 0 {
		unblockStyle := t.Renderer.NewStyle().Foreground(t.Open).Bold(true)
		sb.WriteString(unblockStyle.Render(fmt.Sprintf("↳ Unblocks %d", pick.Unblocks)))
		sb.WriteString("\n")
	}

	// Reasons (compact) - reduced to 1 reason to save space for bars
	reasonStyle := t.Renderer.NewStyle().Foreground(t.Subtext).Italic(true)
	for i, reason := range pick.Reasons {
		if i >= 1 { // Show max 1 reason (reduced from 2 to fit bars)
			break
		}
		reasonTrunc := truncateRunesHelper(reason, width-8, "…")
		sb.WriteString(reasonStyle.Render("• " + reasonTrunc))
		sb.WriteString("\n")
	}

	return itemStyle.Render(sb.String())
}

// renderHeatmapPanel renders a priority/depth heatmap visualization (bv-95)
// Maps priority score (X) vs critical-path depth (Y) with color for urgency
func (m *InsightsModel) renderHeatmapPanel(width, height int, t Theme) string {
	isFocused := m.focusedPanel == PanelPriority

	borderColor := t.Secondary
	if isFocused {
		borderColor = t.Primary
	}

	panelStyle := t.Renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(height).
		Padding(0, 1)

	var sb strings.Builder

	// Header
	titleStyle := t.Renderer.NewStyle().Bold(true)
	if isFocused {
		titleStyle = titleStyle.Foreground(t.Primary)
	} else {
		titleStyle = titleStyle.Foreground(t.Secondary)
	}
	sb.WriteString(titleStyle.Render("📊 Priority Heatmap"))
	sb.WriteString("  ")
	subtitleStyle := t.Renderer.NewStyle().Foreground(t.Subtext).Italic(true)
	sb.WriteString(subtitleStyle.Render("Score vs Depth (H=toggle view)"))
	sb.WriteString("\n")

	if m.insights.Stats == nil || len(m.topPicks) == 0 {
		emptyStyle := t.Renderer.NewStyle().
			Foreground(t.Subtext).
			Italic(true)
		sb.WriteString(emptyStyle.Render("No data available. Run 'bv --robot-triage' to generate."))
		return panelStyle.Render(sb.String())
	}

	// Build heatmap data: count issues in each (depth, score) bucket
	// Depth buckets: 0, 1-2, 3-5, 6-10, 10+
	// Score buckets: 0-0.2, 0.2-0.4, 0.4-0.6, 0.6-0.8, 0.8-1.0
	depthLabels := []string{"D=0", "D1-2", "D3-5", "D6-10", "D10+"}
	scoreLabels := []string{"0-.2", ".2-.4", ".4-.6", ".6-.8", ".8-1"}

	// Grid: [depth_bucket][score_bucket] = count
	grid := make([][]int, len(depthLabels))
	for i := range grid {
		grid[i] = make([]int, len(scoreLabels))
	}
	// Track urgency per cell (average due-in days, lower = more urgent)
	urgencyGrid := make([][]float64, len(depthLabels))
	urgencyCount := make([][]int, len(depthLabels))
	for i := range urgencyGrid {
		urgencyGrid[i] = make([]float64, len(scoreLabels))
		urgencyCount[i] = make([]int, len(scoreLabels))
	}

	stats := m.insights.Stats
	critPath := stats.CriticalPathScore()

	for _, pick := range m.topPicks {
		// Determine depth bucket
		depth := critPath[pick.ID]
		depthBucket := 0
		if depth >= 1 && depth <= 2 {
			depthBucket = 1
		} else if depth >= 3 && depth <= 5 {
			depthBucket = 2
		} else if depth >= 6 && depth <= 10 {
			depthBucket = 3
		} else if depth > 10 {
			depthBucket = 4
		}

		// Determine score bucket (pick.Score is 0-1)
		scoreBucket := int(pick.Score * 5)
		if scoreBucket > 4 {
			scoreBucket = 4
		}

		grid[depthBucket][scoreBucket]++

		// Track urgency from due date if available
		if issue := m.issueMap[pick.ID]; issue != nil && issue.DueDate != nil {
			daysUntilDue := time.Until(*issue.DueDate).Hours() / 24
			urgencyGrid[depthBucket][scoreBucket] += daysUntilDue
			urgencyCount[depthBucket][scoreBucket]++
		}
	}

	// Calculate max for normalization
	maxCount := 1
	for _, row := range grid {
		for _, c := range row {
			if c > maxCount {
				maxCount = c
			}
		}
	}

	// Render header row (score labels)
	cellWidth := (width - 10) / len(scoreLabels)
	if cellWidth < 6 {
		cellWidth = 6
	}

	headerStyle := t.Renderer.NewStyle().Foreground(t.Secondary).Bold(true)
	sb.WriteString(fmt.Sprintf("%5s │", ""))
	for _, label := range scoreLabels {
		sb.WriteString(headerStyle.Render(fmt.Sprintf(" %*s", cellWidth-1, label)))
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString(fmt.Sprintf("%5s─┼", "─────"))
	for range scoreLabels {
		sb.WriteString(strings.Repeat("─", cellWidth))
	}
	sb.WriteString("\n")

	// Render each depth row
	for i, depthLabel := range depthLabels {
		labelStyle := t.Renderer.NewStyle().Foreground(t.Secondary)
		sb.WriteString(labelStyle.Render(fmt.Sprintf("%5s", depthLabel)))
		sb.WriteString(" │")

		for j := range scoreLabels {
			count := grid[i][j]
			if count == 0 {
				// Empty cell
				sb.WriteString(t.Renderer.NewStyle().Foreground(t.Secondary).Render(fmt.Sprintf(" %*s", cellWidth-1, "·")))
			} else {
				// Color based on count intensity and urgency
				intensity := float64(count) / float64(maxCount)

				// Adjust color based on urgency (if we have due date data)
				color := GetHeatmapColor(intensity, t)
				if urgencyCount[i][j] > 0 {
					avgDays := urgencyGrid[i][j] / float64(urgencyCount[i][j])
					if avgDays < 7 {
						color = t.Primary // Urgent - pink/purple
					} else if avgDays < 14 {
						color = t.Feature // Soon - orange
					}
				}

				cellStyle := t.Renderer.NewStyle().
					Foreground(color).
					Bold(count >= maxCount/2)
				// Show count with visual intensity
				block := "█"
				if count >= 3 {
					block = "███"
				} else if count >= 2 {
					block = "██"
				}
				cellContent := fmt.Sprintf("%s%d", block, count)
				sb.WriteString(cellStyle.Render(fmt.Sprintf(" %*s", cellWidth-1, cellContent)))
			}
		}
		sb.WriteString("\n")
	}

	// Legend
	sb.WriteString("\n")
	legendStyle := t.Renderer.NewStyle().Foreground(t.Subtext)
	sb.WriteString(legendStyle.Render("Legend: "))
	sb.WriteString(t.Renderer.NewStyle().Foreground(t.Secondary).Render("· "))
	sb.WriteString(legendStyle.Render("empty "))
	sb.WriteString(t.Renderer.NewStyle().Foreground(t.InProgress).Render("█ "))
	sb.WriteString(legendStyle.Render("few "))
	sb.WriteString(t.Renderer.NewStyle().Foreground(t.Feature).Render("██ "))
	sb.WriteString(legendStyle.Render("some "))
	sb.WriteString(t.Renderer.NewStyle().Foreground(t.Primary).Render("███ "))
	sb.WriteString(legendStyle.Render("many/urgent"))

	return panelStyle.Render(sb.String())
}

func (m *InsightsModel) renderCycleChain(cycle []string, maxWidth int, t Theme) string {
	if len(cycle) == 0 {
		return ""
	}

	// Build chain: A → B → C → A
	var parts []string
	for _, id := range cycle {
		// Try to get short title (check both key existence and nil value)
		if issue, ok := m.issueMap[id]; ok && issue != nil {
			shortTitle := truncateRunesHelper(issue.Title, 15, "…")
			parts = append(parts, shortTitle)
		} else {
			parts = append(parts, truncateRunesHelper(id, 12, "…"))
		}
	}
	// Close the cycle
	if len(parts) > 0 {
		parts = append(parts, parts[0])
	}

	chain := strings.Join(parts, " → ")
	if len([]rune(chain)) > maxWidth {
		chain = truncateRunesHelper(chain, maxWidth, "…")
	}
	return chain
}

// buildDetailMarkdown generates markdown content for the detail panel
func (m *InsightsModel) buildDetailMarkdown(selectedID string) string {
	issue := m.issueMap[selectedID]
	if issue == nil {
		return ""
	}

	var sb strings.Builder

	// === HEADER: Title with Type Icon ===
	sb.WriteString(fmt.Sprintf("# %s %s\n\n", GetTypeIconMD(string(issue.IssueType)), issue.Title))

	// === Meta Table ===
	sb.WriteString("| Field | Value |\n|---|---|\n")
	sb.WriteString(fmt.Sprintf("| **ID** | `%s` |\n", issue.ID))
	sb.WriteString(fmt.Sprintf("| **Status** | **%s** |\n", strings.ToUpper(string(issue.Status))))
	sb.WriteString(fmt.Sprintf("| **Priority** | %s P%d |\n", GetPriorityIcon(issue.Priority), issue.Priority))
	if issue.Assignee != "" {
		sb.WriteString(fmt.Sprintf("| **Assignee** | @%s |\n", issue.Assignee))
	}
	sb.WriteString(fmt.Sprintf("| **Created** | %s |\n", issue.CreatedAt.Format("2006-01-02")))
	sb.WriteString("\n")

	// === Labels ===
	if len(issue.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("**Labels:** `%s`\n\n", strings.Join(issue.Labels, "` `")))
	}

	// === Graph Metrics Section ===
	if m.insights.Stats != nil {
		stats := m.insights.Stats
		sb.WriteString("### 📊 Graph Analysis\n\n")

		// Core metrics in a compact format
		pr := stats.GetPageRankScore(selectedID)
		bt := stats.GetBetweennessScore(selectedID)
		ev := stats.GetEigenvectorScore(selectedID)
		imp := stats.GetCriticalPathScore(selectedID)
		hub := stats.GetHubScore(selectedID)
		auth := stats.GetAuthorityScore(selectedID)

		sb.WriteString(fmt.Sprintf("- **Impact Depth:** `%.0f` _(downstream chain length)_\n", imp))
		sb.WriteString(fmt.Sprintf("- **Centrality:** PR `%.4f` • BW `%.4f` • EV `%.4f`\n", pr, bt, ev))
		sb.WriteString(fmt.Sprintf("- **Flow Role:** Hub `%.4f` • Auth `%.4f`\n", hub, auth))
		sb.WriteString(fmt.Sprintf("- **Degree:** In `%d` ← → Out `%d`\n\n", stats.InDegree[selectedID], stats.OutDegree[selectedID]))
	}

	// === Description ===
	if issue.Description != "" {
		sb.WriteString("### Description\n\n")
		sb.WriteString(issue.Description + "\n\n")
	}

	// === Design ===
	if issue.Design != "" {
		sb.WriteString("### Design\n\n")
		sb.WriteString(issue.Design + "\n\n")
	}

	// === Acceptance Criteria ===
	if issue.AcceptanceCriteria != "" {
		sb.WriteString("### Acceptance Criteria\n\n")
		sb.WriteString(issue.AcceptanceCriteria + "\n\n")
	}

	// === Notes ===
	if issue.Notes != "" {
		sb.WriteString("### Notes\n\n")
		sb.WriteString("> " + strings.ReplaceAll(issue.Notes, "\n", "\n> ") + "\n\n")
	}

	// === Dependencies ===
	if len(issue.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("### Dependencies (%d)\n\n", len(issue.Dependencies)))
		for _, dep := range issue.Dependencies {
			depIssue := m.issueMap[dep.DependsOnID]
			if depIssue != nil {
				sb.WriteString(fmt.Sprintf("- **%s:** %s\n", dep.Type, depIssue.Title))
			} else {
				sb.WriteString(fmt.Sprintf("- **%s:** `%s`\n", dep.Type, dep.DependsOnID))
			}
		}
		sb.WriteString("\n")
	}

	// === Calculation Proof Section ===
	if m.showCalculation && m.insights.Stats != nil {
		sb.WriteString(m.renderCalculationProofMD(selectedID))
	}

	return sb.String()
}

// renderCalculationProofMD generates markdown for calculation proof
func (m *InsightsModel) renderCalculationProofMD(selectedID string) string {
	var sb strings.Builder
	stats := m.insights.Stats
	info := metricDescriptions[m.focusedPanel]

	sb.WriteString("---\n\n")
	sb.WriteString("### 🔬 Calculation Proof\n\n")
	sb.WriteString(fmt.Sprintf("**Formula:** %s\n\n", info.FormulaHint))

	switch m.focusedPanel {
	case PanelBottlenecks:
		bw := stats.GetBetweennessScore(selectedID)
		sb.WriteString(fmt.Sprintf("**Betweenness Score:** `%.4f`\n\n", bw))
		upstream := m.findDependents(selectedID)
		downstream := m.findDependencies(selectedID)
		if len(upstream) > 0 {
			sb.WriteString(fmt.Sprintf("**Beads depending on this (%d):**\n", len(upstream)))
			for i, id := range upstream {
				if i >= 5 {
					sb.WriteString(fmt.Sprintf("- _...+%d more_\n", len(upstream)-5))
					break
				}
				sb.WriteString(fmt.Sprintf("- ↓ %s\n", m.getBeadTitle(id, 40)))
			}
			sb.WriteString("\n")
		}
		if len(downstream) > 0 {
			sb.WriteString(fmt.Sprintf("**This depends on (%d):**\n", len(downstream)))
			for i, id := range downstream {
				if i >= 5 {
					sb.WriteString(fmt.Sprintf("- _...+%d more_\n", len(downstream)-5))
					break
				}
				sb.WriteString(fmt.Sprintf("- ↑ %s\n", m.getBeadTitle(id, 40)))
			}
		}
		sb.WriteString("\n> This bead lies on many shortest paths, making it a *critical junction*.\n\n")

	case PanelKeystones:
		impact := stats.GetCriticalPathScore(selectedID)
		sb.WriteString(fmt.Sprintf("**Impact Depth:** `%.0f` levels deep\n\n", impact))
		chain := m.buildImpactChain(selectedID, int(impact))
		if len(chain) > 0 {
			sb.WriteString("**Dependency chain:**\n```\n")
			for i, id := range chain {
				indent := strings.Repeat("  ", i)
				title := m.getBeadTitle(id, 35)
				sb.WriteString(fmt.Sprintf("%s└─ %s\n", indent, title))
				if i >= 6 {
					sb.WriteString(fmt.Sprintf("%s   ... chain continues\n", indent))
					break
				}
			}
			sb.WriteString("```\n\n")
		}

	case PanelHubs:
		hubScore := stats.GetHubScore(selectedID)
		sb.WriteString(fmt.Sprintf("**Hub Score:** `%.4f`\n\n", hubScore))
		deps := m.findDependenciesWithScores(selectedID, stats.Authorities())
		if len(deps) > 0 {
			sb.WriteString("**Depends on these authorities:**\n")
			sumAuth := 0.0
			for _, d := range deps {
				sumAuth += d.score
			}
			for i, d := range deps {
				if i >= 5 {
					sb.WriteString(fmt.Sprintf("- _...+%d more_\n", len(deps)-5))
					break
				}
				sb.WriteString(fmt.Sprintf("- → %s (Auth: `%.4f`)\n", m.getBeadTitle(d.id, 30), d.score))
			}
			sb.WriteString(fmt.Sprintf("\n> Sum of %d authority scores: `%.4f`\n\n", len(deps), sumAuth))
		}

	case PanelAuthorities:
		authScore := stats.GetAuthorityScore(selectedID)
		sb.WriteString(fmt.Sprintf("**Authority Score:** `%.4f`\n\n", authScore))
		dependents := m.findDependentsWithScores(selectedID, stats.Hubs())
		if len(dependents) > 0 {
			sb.WriteString("**Hubs that depend on this:**\n")
			sumHub := 0.0
			for _, d := range dependents {
				sumHub += d.score
			}
			for i, d := range dependents {
				if i >= 5 {
					sb.WriteString(fmt.Sprintf("- _...+%d more_\n", len(dependents)-5))
					break
				}
				sb.WriteString(fmt.Sprintf("- ← %s (Hub: `%.4f`)\n", m.getBeadTitle(d.id, 30), d.score))
			}
			sb.WriteString(fmt.Sprintf("\n> Sum of %d hub scores: `%.4f`\n\n", len(dependents), sumHub))
		}

	case PanelCycles:
		idx := m.selectedIndex[PanelCycles]
		if idx >= 0 && idx < len(m.insights.Cycles) {
			cycle := m.insights.Cycles[idx]
			sb.WriteString(fmt.Sprintf("**Cycle with %d beads:**\n```\n", len(cycle)))
			for i, id := range cycle {
				arrow := "→"
				if i == len(cycle)-1 {
					arrow = "↺"
				}
				sb.WriteString(fmt.Sprintf("%s %s\n", arrow, m.getBeadTitle(id, 35)))
			}
			sb.WriteString("```\n\n")
			sb.WriteString("> These beads form a circular dependency. *Break the cycle* by removing or reversing one edge.\n\n")
		}

	default:
		// For other panels, show generic info
		sb.WriteString(fmt.Sprintf("> %s\n\n", info.HowToUse))
	}

	return sb.String()
}

func (m *InsightsModel) renderDetailPanel(width, height int, t Theme) string {
	panelStyle := t.Renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Width(width).
		Height(height).
		Padding(0, 1)

	selectedID := m.SelectedIssueID()
	if selectedID == "" {
		emptyContent := `
## Select a Bead

Navigate to a metric panel and select an item to view its details here.

**Navigation:**
- ← → to switch panels
- ↑ ↓ to select items
- Enter to view in main view
`
		if m.mdRenderer != nil {
			rendered, err := m.mdRenderer.Render(emptyContent)
			if err == nil {
				return panelStyle.Render(rendered)
			}
		}
		emptyStyle := t.Renderer.NewStyle().
			Foreground(t.Subtext).
			Italic(true).
			Width(width - 4).
			Align(lipgloss.Center)
		return panelStyle.Render(emptyStyle.Render("\nSelect a bead to view details"))
	}

	issue := m.issueMap[selectedID]
	if issue == nil {
		return panelStyle.Render(t.Renderer.NewStyle().Foreground(t.Subtext).Render("Issue not found: " + selectedID))
	}

	// Build markdown content and render with glamour
	mdContent := m.buildDetailMarkdown(selectedID)

	if m.mdRenderer != nil {
		rendered, err := m.mdRenderer.Render(mdContent)
		if err == nil {
			// Truncate to fit panel height (approximate line count)
			lines := strings.Split(rendered, "\n")
			maxLines := height - 2
			if len(lines) > maxLines {
				lines = lines[:maxLines]
				lines = append(lines, t.Renderer.NewStyle().Foreground(t.Subtext).Italic(true).Render("... (scroll for more)"))
			}
			return panelStyle.Render(strings.Join(lines, "\n"))
		}
	}

	// Fallback to raw markdown if renderer fails
	return panelStyle.Render(wrapText(mdContent, width-4))
}

// formatMetricValue formats a metric value nicely
func formatMetricValue(v float64) string {
	if v >= 100 {
		return fmt.Sprintf("%.0f", v)
	} else if v >= 1.0 {
		return fmt.Sprintf("%.2f", v)
	} else if v >= 0.01 {
		return fmt.Sprintf("%.3f", v)
	} else if v > 0 {
		return fmt.Sprintf("%.2e", v)
	}
	return "0"
}

// Helper type for scored items
type scoredItem struct {
	id    string
	score float64
}

// getBeadTitle returns a truncated title for a bead ID
func (m *InsightsModel) getBeadTitle(id string, maxWidth int) string {
	if issue, ok := m.issueMap[id]; ok && issue != nil {
		return truncateRunesHelper(issue.Title, maxWidth, "…")
	}
	return truncateRunesHelper(id, maxWidth, "…")
}

// findDependents returns IDs of beads that depend on the given bead (sorted for consistent order)
func (m *InsightsModel) findDependents(targetID string) []string {
	var dependents []string
	for id, issue := range m.issueMap {
		if issue == nil {
			continue
		}
		for _, dep := range issue.Dependencies {
			if dep.DependsOnID == targetID {
				dependents = append(dependents, id)
				break
			}
		}
	}
	// Sort for consistent display order (map iteration is non-deterministic)
	for i := 0; i < len(dependents)-1; i++ {
		for j := i + 1; j < len(dependents); j++ {
			if dependents[j] < dependents[i] {
				dependents[i], dependents[j] = dependents[j], dependents[i]
			}
		}
	}
	return dependents
}

// findDependencies returns IDs of beads that the given bead depends on
func (m *InsightsModel) findDependencies(targetID string) []string {
	issue := m.issueMap[targetID]
	if issue == nil {
		return nil
	}
	var deps []string
	for _, dep := range issue.Dependencies {
		deps = append(deps, dep.DependsOnID)
	}
	return deps
}

// findNeighborsWithScores returns neighbors with their metric scores, sorted by score
func (m *InsightsModel) findNeighborsWithScores(targetID string, scores map[string]float64) []scoredItem {
	var items []scoredItem
	seen := make(map[string]bool)

	// Add dependents
	for _, id := range m.findDependents(targetID) {
		if !seen[id] {
			seen[id] = true
			items = append(items, scoredItem{id: id, score: scores[id]})
		}
	}
	// Add dependencies (avoid duplicates from cycles)
	for _, id := range m.findDependencies(targetID) {
		if !seen[id] {
			seen[id] = true
			items = append(items, scoredItem{id: id, score: scores[id]})
		}
	}

	// Sort by score descending
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].score > items[i].score {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	return items
}

// findDependenciesWithScores returns dependencies with their metric scores
func (m *InsightsModel) findDependenciesWithScores(targetID string, scores map[string]float64) []scoredItem {
	var items []scoredItem
	for _, id := range m.findDependencies(targetID) {
		items = append(items, scoredItem{id: id, score: scores[id]})
	}
	// Sort by score descending
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].score > items[i].score {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	return items
}

// findDependentsWithScores returns dependents with their metric scores
func (m *InsightsModel) findDependentsWithScores(targetID string, scores map[string]float64) []scoredItem {
	var items []scoredItem
	for _, id := range m.findDependents(targetID) {
		items = append(items, scoredItem{id: id, score: scores[id]})
	}
	// Sort by score descending
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].score > items[i].score {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	return items
}

// buildImpactChain builds the dependency chain from a bead to its deepest dependency
func (m *InsightsModel) buildImpactChain(startID string, maxDepth int) []string {
	var chain []string
	if maxDepth <= 0 || m.insights.Stats == nil {
		return chain
	}

	current := startID
	visited := make(map[string]bool)

	for len(chain) < maxDepth && !visited[current] {
		visited[current] = true
		chain = append(chain, current)

		// Find the dependency with highest impact score
		deps := m.findDependencies(current)
		if len(deps) == 0 {
			break
		}

		bestDep := ""
		bestScore := -1.0
		for _, dep := range deps {
			score := m.insights.Stats.GetCriticalPathScore(dep)
			if score > bestScore {
				bestScore = score
				bestDep = dep
			}
		}
		if bestDep == "" {
			break
		}
		current = bestDep
	}
	return chain
}

// wrapText wraps text to fit within maxWidth
func wrapText(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return s
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine strings.Builder
	currentLen := 0

	for _, word := range words {
		wordLen := len([]rune(word))
		if currentLen+wordLen+1 > maxWidth && currentLen > 0 {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLen = 0
		}
		if currentLen > 0 {
			currentLine.WriteString(" ")
			currentLen++
		}
		currentLine.WriteString(word)
		currentLen += wordLen
	}
	if currentLen > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}
