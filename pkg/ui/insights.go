package ui

import (
	"fmt"
	"strings"

	"beads_viewer/pkg/analysis"
	"github.com/charmbracelet/lipgloss"
)

type InsightsModel struct {
	insights analysis.Insights
	ready    bool
	width    int
	height   int
}

func NewInsightsModel(ins analysis.Insights) InsightsModel {
	return InsightsModel{
		insights: ins,
	}
}

func (i *InsightsModel) SetSize(w, h int) {
	i.width = w
	i.height = h
	i.ready = true
}

func (i InsightsModel) View() string {
	if !i.ready {
		return ""
	}

	// Layout:
	// [ Top Bottlenecks ] [ Top Keystones ]
	// [     Cycles      ] [    Stats      ]
	
	halfWidth := (i.width / 2) - 4
	halfHeight := (i.height / 2) - 2
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(0, 1).
		Width(halfWidth).
		Height(halfHeight)
		
	titleStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	
	// Bottlenecks
	var bnSb strings.Builder
	bnSb.WriteString(titleStyle.Render("🚧 Top Bottlenecks (Betweenness)"))
	bnSb.WriteString("\n\n")
	for _, id := range i.insights.Bottlenecks {
		bnSb.WriteString(fmt.Sprintf("• %s\n", id))
	}
	bnBox := boxStyle.Render(bnSb.String())
	
	// Keystones
	var ksSb strings.Builder
	ksSb.WriteString(titleStyle.Render("🏛️  Keystones (High Impact)"))
	ksSb.WriteString("\n\n")
	for _, id := range i.insights.Keystones {
		ksSb.WriteString(fmt.Sprintf("• %s\n", id))
	}
	ksBox := boxStyle.Render(ksSb.String())
	
	// Cycles
	var cySb strings.Builder
	cySb.WriteString(titleStyle.Render("🔄 Structural Risks (Cycles)"))
	cySb.WriteString("\n\n")
	if len(i.insights.Cycles) == 0 {
		cySb.WriteString(lipgloss.NewStyle().Foreground(ColorStatusOpen).Render("No cycles detected. Graph is healthy."))
	} else {
		for _, cycle := range i.insights.Cycles {
			cySb.WriteString(fmt.Sprintf("• %s\n", strings.Join(cycle, " -> ")))
		}
	}
	cyBox := boxStyle.Render(cySb.String())
	
	// Stats
	var stSb strings.Builder
	stSb.WriteString(titleStyle.Render("📊 Network Health"))
	stSb.WriteString("\n\n")
	stSb.WriteString(fmt.Sprintf("Density: %.4f\n", i.insights.ClusterDensity))
	stBox := boxStyle.Render(stSb.String())
	
topRow := lipgloss.JoinHorizontal(lipgloss.Top, bnBox, ksBox)
btmRow := lipgloss.JoinHorizontal(lipgloss.Top, cyBox, stBox)
	
	return lipgloss.JoinVertical(lipgloss.Left, topRow, btmRow)
}
