# Beads Viewer (bv)

A polished, high-performance TUI for managing and exploring [Beads](https://github.com/steveyegge/beads) issue trackers.

## Features

### 🧠 Advanced Analytics & Insights
*   **Project Health Dashboard**: Press `i` to view high-level insights.
*   **Bottleneck Detection**: Identifies tasks blocking critical paths via Betweenness Centrality.
*   **Impact Analysis**: Scores tasks based on downstream dependencies.
*   **Cycle Alert**: Detects and lists circular dependencies.

### 🖥️ Visual Dashboard
*   **Kanban Board**: Press `b` to toggle a 4-column Kanban board.
*   **Sparklines**: Visual bars for Impact scores in Ultra-Wide mode.
*   **Adaptive Layouts**: Responsive design for all terminal sizes.

### ⚡ Workflow
*   **Instant Filtering**: `o` (Open), `r` (Ready), `c` (Closed), `a` (All).
*   **Mermaid Export**: `bv --export-md report.md`.

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/Dicklesworthstone/beads_viewer/main/install.sh | bash
```

## Usage

```bash
bv
```

### Controls

| Key | Context | Action |
| :--- | :--- | :--- |
| `b` | Global | Toggle **Kanban Board** |
| `i` | Global | Toggle **Insights Dashboard** |
| `Tab` | Split View | Switch focus |
| `h`/`j`/`k`/`l`| Board | Navigate |
| `o` / `r` / `c` | Global | Filter status |
| `q` | Global | Quit |

## CI/CD

*   **CI**: Runs tests on every push.
*   **Release**: Builds binaries for all platforms.

## License

MIT