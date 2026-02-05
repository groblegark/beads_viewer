# Beads Viewer (bv) ↔ Gas Town Beads: Compatibility Report

**Date**: 2026-02-05
**Author**: obsidian (beads_viewer polecat)
**Bead**: gt-0w4bpr

---

## Executive Summary

beads_viewer (bv) is a well-architected Go TUI with powerful graph analysis capabilities that would be valuable for our beads ecosystem. However, there is a **significant schema and architecture gap** between what bv expects and what our system provides. The recommended approach is **Bridge (adapter layer)** — not fork, not direct adapt.

**Key finding**: bv's ~25-field Issue model is a strict subset of our ~160-field Issue type. The data flows one way: our system's data can be projected onto bv's schema, but bv cannot represent our full semantics. This makes an adapter/bridge the natural integration pattern.

---

## 1. Data Format Compatibility

### Schema Comparison

| Field | bv (`model.Issue`) | Our System (`types.Issue`) | Compatible? |
|-------|-------------------|---------------------------|-------------|
| `id` | string | string | Yes |
| `title` | string | string | Yes |
| `description` | string | string | Yes |
| `status` | 8 values | 8 values (same set) | Yes |
| `priority` | int (0-4) | int (0-4) | Yes |
| `issue_type` | 5 built-in + custom | 10+ built-in + custom | Partial |
| `created_at` | time.Time | time.Time | Yes |
| `updated_at` | time.Time | time.Time | Yes |
| `assignee` | string | string | Yes |
| `labels` | []string | []string | Yes |
| `dependencies` | []*Dependency | []*Dependency | Partial |
| `comments` | []*Comment | []*Comment | Yes |
| `closed_at` | *time.Time | *time.Time | Yes |
| `estimated_minutes` | *int | *int | Yes |
| `due_date` | *time.Time | `due_at` | Rename needed |
| `design` | string | string | Yes |
| `acceptance_criteria` | string | string | Yes |
| `notes` | string | string | Yes |
| `external_ref` | *string | *string | Yes |
| `source_repo` | string | string | Yes |
| `compaction_level` | int | int | Yes |

### Fields Our System Has That bv Lacks

These represent the **schema gap** — our system's extensions that bv cannot display:

| Category | Fields | Count |
|----------|--------|-------|
| **Agent Coordination** | `hook_bead`, `role_bead`, `agent_state`, `role_type`, `rig`, `last_activity` | 6 |
| **Molecules/Wisps** | `mol_type`, `work_type`, `ephemeral`, `is_template`, `bonded_from` | 5 |
| **Async Gates** | `await_type`, `await_id`, `timeout`, `waiters`, `holder` | 5 |
| **Decision Points** | `decision_point` (full DecisionPoint struct) | 1 |
| **HOP/Governance** | `creator` (EntityRef), `validations`, `quality_score`, `crystallizes` | 4 |
| **Events** | `event_kind`, `actor`, `target`, `payload` | 4 |
| **Skills** | `skill_*` fields (7+) | 7+ |
| **Advice/Hooks** | `advice_hook_*`, `advice_subscriptions` | 5+ |
| **Extended Metadata** | `metadata` (json.RawMessage), `semantic_slug`, `content_hash` | 3 |
| **Soft Delete** | `deleted_at`, `deleted_by`, `delete_reason`, `original_type` | 4 |
| **Scheduling** | `defer_until`, `close_reason`, `auto_close`, `pinned` | 4 |
| **Total** | | **~50+ fields** |

### Dependency Type Gap

| bv Dependency Types | Our Dependency Types |
|--------------------|--------------------|
| `blocks` | `blocks` |
| `related` | `relates-to` |
| `parent-child` | `parent-child` |
| `discovered-from` | `discovered-from` |
| — | `conditional-blocks` |
| — | `waits-for` |
| — | `replies-to` |
| — | `duplicates` |
| — | `supersedes` |
| — | `attests` |
| — | 10+ more |

### Issue Type Gap

| bv Issue Types | Our Issue Types |
|---------------|----------------|
| bug, feature, task, epic, chore | bug, feature, task, epic, chore |
| (custom extensible) | advice, formula, config, role, message, gate, merge-request, decision, etc. |

**bv handles custom types gracefully** — it stores them as strings and displays them. The gap is in semantics, not parsing.

### File Path Compatibility

| Aspect | bv | Our System |
|--------|-----|-----------|
| Primary file | `.beads/issues.jsonl` | `.beads/issues.jsonl` | ✅ Match |
| Fallback | `.beads/beads.jsonl` | — |
| Environment | `BEADS_DIR` | — |
| Storage backend | Flat JSONL only | Dolt + SQLite + JSONL export |

**Verdict**: bv can already read our JSONL exports. The core data path is compatible.

---

## 2. Daemon Integration

### Current Architecture Gap

| Aspect | bv | Our System |
|--------|-----|-----------|
| Data source | Reads `.beads/issues.jsonl` file | Dolt-backed daemon (bd-daemon) via RPC |
| Write path | None (read-only viewer) | Full CRUD via `bd` CLI and daemon |
| Live reload | Watches JSONL file via fsnotify | Daemon events (not file-based) |
| Multi-repo | `.bv/workspace.yaml` | Routing via `.beads/routes.jsonl` |

### Integration Options

**Option A: JSONL Export Bridge** (Recommended for Phase 1)
- Run `bd export` to generate `.beads/issues.jsonl` from daemon
- bv reads the exported JSONL (already works)
- Add a periodic export or daemon-triggered export
- **Effort**: Minimal — just a cron/hook to run `bd export`
- **Latency**: Seconds (export cycle time)

**Option B: Daemon RPC Adapter** (Phase 2)
- Add a new loader backend to bv that calls bd-daemon RPC
- Replace flat-file reader with RPC client
- **Effort**: Medium — needs new loader implementation
- **Benefit**: Real-time data, no export step

**Option C: Shared SQLite** (Not recommended)
- Both tools read same SQLite cache
- **Risk**: Lock contention, schema coupling

---

## 3. Feature Gap Analysis

### Features bv Has That We Lack

| Feature | Value for Us | Effort to Integrate |
|---------|-------------|-------------------|
| **TUI Dashboard** | High — visual issue browsing | Works today with JSONL export |
| **PageRank/Betweenness/HITS** | High — identifies bottleneck beads | Works today |
| **Critical Path Analysis** | High — convoy/epic tracking | Works today |
| **Kanban Board** | Medium — visual flow | Works today |
| **Robot Protocol** | High — polecats/crew could use `--robot-triage` | Works today with JSONL |
| **Graph Visualization** | High — dependency DAG navigation | Works today |
| **Semantic Search** | Medium — hybrid text+graph search | Works today |
| **Time-Travel** | Medium — compare against git history | Works with git-tracked JSONL |
| **Export to Markdown/Mermaid** | Medium — documentation | Works today |
| **WASM Graph Viewer** | Low — browser-based visualization | Self-contained |
| **Sprint/Burndown** | Low — we don't use sprints | N/A |
| **Self-Update** | Low — we'd vendor it | N/A |

### Features We Have That bv Doesn't Understand

| Feature | Impact on bv | Mitigation |
|---------|-------------|-----------|
| **Molecules/Wisps** | Treats as regular issues | Add `mol_type` to labels for filtering |
| **Config Beads** | Clutters issue list | Filter out `type=config` in recipe |
| **Agent Beads** | Clutters issue list | Filter out `type=role` in recipe |
| **Decision Points** | Not displayed | Could add to description field |
| **Async Gates** | Not displayed | Could show as blocker annotation |
| **HOP Validations** | Not displayed | Could add to comments |
| **20+ Dep Types** | Collapses to blocks/related | Acceptable loss for visualization |

---

## 4. Architecture Assessment

### Modularity Rating: **Excellent**

The codebase is cleanly separated:

```
loader/     → Data ingestion (swappable)
model/      → Domain types (clean, minimal)
analysis/   → Graph algorithms (gonum-based, reusable)
correlation/→ Git history mapping
search/     → Hybrid text+graph search
export/     → Multiple output formats
ui/         → Bubble Tea TUI (self-contained)
hooks/      → Pre/post-export automation
recipe/     → YAML-based view configurations
```

**Key interfaces**:
- `loader` package reads JSONL → can be extended to read from daemon
- `model.Issue` struct → maps cleanly to our core fields
- `analysis.Analyzer` → operates on `[]model.Issue` → reusable with any data source
- Recipe system → perfect for filtering out Gas Town-specific beads

### Code Quality

| Metric | Status |
|--------|--------|
| Build | Clean (zero warnings) |
| Tests | 25 packages, all passing |
| E2E Tests | Comprehensive (147s test suite) |
| Test Coverage | Property-based tests, golden tests, benchmarks |
| Dependencies | Well-chosen (charmbracelet, gonum, goccy/go-json) |
| Go Version | 1.25 (current) |

### Integration Points

1. **Loader** (`pkg/loader/loader.go`): Add daemon backend here
2. **Model** (`pkg/model/types.go`): Extend with optional Gas Town fields
3. **Recipe** (`pkg/recipe/types.go`): Add Gas Town presets (filter molecules, agents, etc.)
4. **Analysis** (`pkg/analysis/`): No changes needed — works on any `[]model.Issue`
5. **Export** (`pkg/export/`): Add Gas Town-specific export formats

---

## 5. Graph Capabilities Assessment

### Algorithms Available

| Algorithm | Implementation | Reusability | Value for Convoys/Epics |
|-----------|---------------|------------|------------------------|
| PageRank | Power iteration (Go) + WASM | High | Identify critical blocking beads |
| Betweenness | Brandes + approx (Go) + WASM | High | Find bottleneck beads |
| HITS | gonum network.HITS (Go) + WASM | High | Identify hub epics vs leaf tasks |
| Eigenvector | Power method (Go) + WASM | High | Find influential beads |
| Critical Path | Topological sort + height (Go) + WASM | **Very High** | Direct convoy tracking |
| K-Core | Decomposition (Go) + WASM | Medium | Find tightly coupled work clusters |
| Articulation | Cut vertices (Go) + WASM | High | Find single-point-of-failure beads |
| Slack | Longest-path slack (Go) + WASM | High | Identify parallelizable work |
| Cycle Detection | Tarjan SCC (Go) + WASM | **Very High** | Find circular dependency bugs |

### Extraction Feasibility

The `analysis` package depends only on:
- `model.Issue` (our core subset)
- `gonum.org/v1/gonum/graph` (standard graph library)

**It can be extracted as a standalone library** with minimal effort. The main coupling is through the `model.Issue` struct, which can be adapted via an interface or mapper.

---

## 6. Robot Protocol Assessment

### Usefulness for Polecats/Crew

| Robot Command | Use Case | Priority |
|--------------|---------|----------|
| `--robot-triage` | Polecat picks next work | **Critical** |
| `--robot-plan` | Parallel execution tracks for swarms | **Critical** |
| `--robot-insights` | Convoy health monitoring | High |
| `--robot-priority` | Priority misalignment detection | High |
| `--robot-next` | Minimal "what should I work on" | High |
| `--robot-graph` | Dependency visualization for reports | Medium |
| `--robot-alerts` | Proactive issue detection | Medium |
| `--robot-suggest` | Hygiene (duplicates, missing deps) | Medium |
| `--robot-forecast` | ETA prediction | Low (needs estimated_minutes) |
| `--robot-burndown` | Sprint tracking | Low (we don't use sprints) |

### Integration Pattern

```bash
# Export from daemon
bd export --format=jsonl > .beads/issues.jsonl

# Run robot analysis
bv --robot-triage | jq '.recommendations[0]'
bv --robot-plan | jq '.plan.tracks'
bv --robot-insights | jq '.Cycles'
```

This works **today** with zero code changes to bv.

---

## 7. Build/Test Status

| Check | Result |
|-------|--------|
| `go build ./...` | **PASS** (clean, zero warnings) |
| `go test ./...` | **PASS** (25 packages) |
| Longest test | `tests/e2e` (148s) |
| Test approaches | Unit, integration, e2e, property-based, golden, benchmarks |
| Go version | 1.25 |
| Binary | Single binary (`bv`) |

---

## 8. Recommendation: Bridge (Adapter Layer)

### Why Not Fork?

- bv is actively maintained (v0.13.0), forking creates maintenance burden
- Our extensions are in the data model, not the viewer logic
- bv's architecture already supports extension via loader/recipe/workspace

### Why Not Direct Adapt?

- Our schema is 6x larger than bv's model
- Adding 50+ fields to bv's model bloats a clean codebase
- Many Gas Town concepts (molecules, wisps, HOP) have no visual representation in a TUI

### Why Bridge?

1. **Zero bv changes needed for Phase 1**: `bd export` → bv reads JSONL → works today
2. **Recipe presets handle filtering**: Hide molecules, agents, config beads via recipes
3. **Loader extension for Phase 2**: Add daemon RPC backend to bv's loader
4. **Graph algorithms work as-is**: `analysis.Analyzer` operates on `[]model.Issue` — our fields map cleanly
5. **Robot protocol usable immediately**: Polecats can call `bv --robot-triage` today

### Proposed Integration Plan

#### Phase 1: JSONL Bridge (1-2 days)

1. Create `bd export-bv` command that exports JSONL in bv-compatible format
   - Maps our fields → bv fields
   - Filters out non-issue beads (config, agent, ephemeral)
   - Adds Gas Town metadata as labels (e.g., `mol:work`, `rig:beads_viewer`)
2. Create `.bv/recipes.yaml` with Gas Town presets:
   - `gt-work`: Filter to actionable work beads only
   - `gt-epics`: Filter to epics/convoys
   - `gt-blocked`: Show blocked work with dependency analysis
3. Wire `bv --robot-triage` into polecat startup

#### Phase 2: Daemon Integration (3-5 days)

1. Add `loader.DaemonLoader` that calls bd-daemon RPC
2. Implement live-reload via daemon event subscription
3. Add Gas Town-specific analysis views (molecule progress, swarm status)

#### Phase 3: Gas Town Extensions (5-10 days)

1. Add molecule/wisp awareness to bv's UI (special icons, grouping)
2. Add decision point display in issue details
3. Add agent state visualization
4. Extract `analysis` package as standalone library for `bd` CLI integration

### Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|-----------|
| bv upstream changes break bridge | Low | Pin version, vendor |
| Schema drift between bv and our model | Medium | Adapter pattern isolates changes |
| Performance with large issue counts | Low | bv handles thousands of issues well |
| Daemon RPC adds complexity | Medium | Phase 1 (JSONL) works without it |

---

## Appendix A: Quick Start (Works Today)

```bash
# From any rig's worktree:
cd /home/ubuntu/gt11/beads_viewer/polecats/obsidian/beads_viewer

# Export beads to JSONL (from any rig with beads)
BD_DAEMON_HOST="" bd export > /tmp/test-beads/issues.jsonl

# View with bv (read-only, zero changes needed)
go run ./cmd/bv --robot-triage  # AI-friendly triage
go run ./cmd/bv --robot-plan    # Execution tracks
go run ./cmd/bv --robot-insights # Graph metrics
```

## Appendix B: File Reference

| Component | Location |
|-----------|---------|
| bv Issue model | `pkg/model/types.go` |
| bv JSONL loader | `pkg/loader/loader.go` |
| bv Graph analysis | `pkg/analysis/graph.go` |
| bv Robot protocol | `cmd/bv/main.go` (6534 lines) |
| bv Recipes | `pkg/recipe/types.go` |
| bv WASM module | `bv-graph-wasm/` |
| Our Issue model | `beads/internal/types/types.go` (~1460 lines) |
| Our bd CLI | `beads/cmd/bd/` (398 files) |
