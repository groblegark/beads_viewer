package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"beads_viewer/pkg/analysis"
	"beads_viewer/pkg/export"
	"beads_viewer/pkg/loader"
	"beads_viewer/pkg/ui"
	"beads_viewer/pkg/version"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	help := flag.Bool("help", false, "Show help")
	versionFlag := flag.Bool("version", false, "Show version")
	exportFile := flag.String("export-md", "", "Export issues to a Markdown file (e.g., report.md)")
	robotHelp := flag.Bool("robot-help", false, "Show AI agent help")
	robotInsights := flag.Bool("robot-insights", false, "Output graph analysis and insights as JSON for AI agents")
	flag.Parse()

	if *help {
		fmt.Println("Usage: bv [options]")
		fmt.Println("\nA TUI viewer for beads issue tracker.")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if *robotHelp {
		fmt.Println("bv (Beads Viewer) AI Agent Interface")
		fmt.Println("====================================")
		fmt.Println("This tool provides structural analysis of the issue tracker graph (DAG).")
		fmt.Println("Use these commands to understand project state without parsing raw JSONL.")
		fmt.Println("")
		fmt.Println("Commands:")
		fmt.Println("  --robot-insights")
		fmt.Println("      Outputs a JSON object containing deep graph analysis.")
		fmt.Println("      Key metrics explained:")
		fmt.Println("      - PageRank: Measures 'blocking power'. High score = Fundamental dependency.")
		fmt.Println("      - Betweenness: Measures 'bottleneck status'. High score = Connects disparate clusters.")
		fmt.Println("      - CriticalPathScore: Heuristic for depth. High score = Blocking a long chain of work.")
		fmt.Println("      - Hubs/Authorities: HITS algorithm scores for dependency relationships.")
		fmt.Println("      - Cycles: Lists of circular dependencies (unhealthy state).")
		fmt.Println("")
		fmt.Println("  --export-md <file>")
		fmt.Println("      Generates a readable status report with Mermaid.js visualizations.")
		os.Exit(0)
	}

	if *versionFlag {
		fmt.Printf("bv %s\n", version.Version)
		os.Exit(0)
	}

	// Load issues from current directory
	issues, err := loader.LoadIssues("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading beads: %v\n", err)
		fmt.Fprintln(os.Stderr, "Make sure you are in a project initialized with 'bd init'.")
		os.Exit(1)
	}

	if *robotInsights {
		analyzer := analysis.NewAnalyzer(issues)
		stats := analyzer.Analyze()
		// Generate top 50 lists for summary, but full stats are included in the struct
		insights := stats.GenerateInsights(50)

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(insights); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding insights: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *exportFile != "" {
		fmt.Printf("Exporting to %s...\n", *exportFile)
		if err := export.SaveMarkdownToFile(issues, *exportFile); err != nil {
			fmt.Printf("Error exporting: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Done!")
		os.Exit(0)
	}

	if len(issues) == 0 {
		fmt.Println("No issues found. Create some with 'bd create'!")
		os.Exit(0)
	}

	// Initial Model
	m := ui.NewModel(issues)

	// Run Program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running beads viewer: %v\n", err)
		os.Exit(1)
	}
}
