package analysis

import (
	"sort"
)

// Insights is a high-level summary of graph analysis
type Insights struct {
	Bottlenecks    []string // Top betweenness nodes
	Keystones      []string // Top impact nodes
	Orphans        []string // No dependencies (and not blocked?) - Leaf nodes
	Cycles         [][]string
	ClusterDensity float64
}

// GenerateInsights translates raw stats into actionable data
func (s GraphStats) GenerateInsights(limit int) Insights {
	return Insights{
		Bottlenecks:    getTopKeys(s.Betweenness, limit),
		Keystones:      getTopKeys(s.CriticalPathScore, limit),
		Cycles:         s.Cycles,
		ClusterDensity: s.Density,
	}
}

func getTopKeys(m map[string]float64, limit int) []string {
	type kv struct {
		Key   string
		Value float64
	}
	var ss []kv
	for k, v := range m {
		ss = append(ss, kv{k, v})
	}
	
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})
	
	var result []string
	for i := 0; i < len(ss) && i < limit; i++ {
		result = append(result, ss[i].Key)
	}
	return result
}
