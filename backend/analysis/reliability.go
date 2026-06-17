package analysis

import (
	"beacon-system/models"
	"math"
	"math/rand"
	"sort"
)

type Graph struct {
	Nodes map[int]*models.Beacon
	Edges []*NetworkEdge
	Adj   map[int][]int
}

type NetworkEdge struct {
	ID              int
	From            int
	To              int
	BaseReliability float64
	IsCritical      bool
	IsBidirectional bool
}

func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[int]*models.Beacon),
		Edges: make([]*NetworkEdge, 0),
		Adj:   make(map[int][]int),
	}
}

func (g *Graph) AddNode(beacon *models.Beacon) {
	g.Nodes[beacon.ID] = beacon
	if _, exists := g.Adj[beacon.ID]; !exists {
		g.Adj[beacon.ID] = make([]int, 0)
	}
}

func (g *Graph) AddEdge(link *models.NetworkLink) {
	edge := &NetworkEdge{
		ID:              link.ID,
		From:            link.FromBeaconID,
		To:              link.ToBeaconID,
		BaseReliability: link.BaseReliability,
		IsCritical:      link.IsCritical,
		IsBidirectional: link.IsBidirectional,
	}
	g.Edges = append(g.Edges, edge)

	g.Adj[edge.From] = append(g.Adj[edge.From], edge.To)
	if edge.IsBidirectional {
		g.Adj[edge.To] = append(g.Adj[edge.To], edge.From)
	}
}

func (g *Graph) IsConnected() bool {
	if len(g.Nodes) == 0 {
		return false
	}

	visited := make(map[int]bool)
	var startNode int
	for id := range g.Nodes {
		startNode = id
		break
	}

	g.dfs(startNode, visited)
	return len(visited) == len(g.Nodes)
}

func (g *Graph) dfs(node int, visited map[int]bool) {
	visited[node] = true
	for _, neighbor := range g.Adj[node] {
		if !visited[neighbor] {
			g.dfs(neighbor, visited)
		}
	}
}

func (g *Graph) BFSLevels(start int) map[int]int {
	levels := make(map[int]int)
	for id := range g.Nodes {
		levels[id] = -1
	}
	levels[start] = 0

	queue := []int{start}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		for _, neighbor := range g.Adj[node] {
			if levels[neighbor] == -1 {
				levels[neighbor] = levels[node] + 1
				queue = append(queue, neighbor)
			}
		}
	}

	return levels
}

func (g *Graph) AveragePathLength() float64 {
	if len(g.Nodes) <= 1 {
		return 0
	}

	totalDistance := 0
	pairCount := 0

	for startID := range g.Nodes {
		levels := g.BFSLevels(startID)
		for _, dist := range levels {
			if dist > 0 {
				totalDistance += dist
				pairCount++
			}
		}
	}

	if pairCount == 0 {
		return 0
	}

	return float64(totalDistance) / float64(pairCount)
}

func (g *Graph) ConnectivityIndex() float64 {
	if len(g.Nodes) == 0 {
		return 0
	}

	visited := make(map[int]bool)
	componentCount := 0

	for nodeID := range g.Nodes {
		if !visited[nodeID] {
			g.dfs(nodeID, visited)
			componentCount++
		}
	}

	if componentCount <= 1 {
		return 1.0
	}

	return 1.0 - float64(componentCount-1)/float64(len(g.Nodes))
}

type MonteCarloConfig struct {
	Iterations            int
	WeatherFactor         float64
	VisibilityMap         map[int]map[int]bool
	UseImportanceSampling bool
}

type edgeFailureProb struct {
	idx      int
	edgeID   int
	failProb float64
}

func MonteCarloReliability(g *Graph, config MonteCarloConfig) models.MonteCarloResult {
	if config.UseImportanceSampling && len(g.Edges) > 20 {
		return monteCarloImportanceSampling(g, config)
	}
	return monteCarloStandard(g, config)
}

func monteCarloStandard(g *Graph, config MonteCarloConfig) models.MonteCarloResult {
	successCount := 0

	for i := 0; i < config.Iterations; i++ {
		simGraph := copyGraph(g)
		simulateFailures(simGraph, config)

		if simGraph.IsConnected() {
			successCount++
		}
	}

	successRate := float64(successCount) / float64(config.Iterations)
	stdErr := math.Sqrt(successRate*(1-successRate)) / math.Sqrt(float64(config.Iterations))
	confidenceInterval := [2]float64{
		math.Max(0, successRate-1.96*stdErr),
		math.Min(1, successRate+1.96*stdErr),
	}

	return models.MonteCarloResult{
		Iterations:         config.Iterations,
		SuccessRate:        successRate,
		ConfidenceInterval: confidenceInterval,
	}
}

func monteCarloImportanceSampling(g *Graph, config MonteCarloConfig) models.MonteCarloResult {
	sortedEdges := make([]edgeFailureProb, len(g.Edges))
	for i, edge := range g.Edges {
		effectiveReliability := edge.BaseReliability * config.WeatherFactor
		if config.VisibilityMap != nil {
			if visMap, ok := config.VisibilityMap[edge.From]; ok {
				if visible, ok2 := visMap[edge.To]; ok2 && !visible {
					effectiveReliability *= 0.1
				}
			}
		}
		sortedEdges[i] = edgeFailureProb{
			idx:      i,
			edgeID:   edge.ID,
			failProb: 1.0 - effectiveReliability,
		}
	}

	sort.Slice(sortedEdges, func(i, j int) bool {
		return sortedEdges[i].failProb > sortedEdges[j].failProb
	})

	biasFactor := 2.0
	weightSums := make([]float64, config.Iterations)
	weightedSuccess := 0.0

	for i := 0; i < config.Iterations; i++ {
		simGraph := copyGraph(g)
		weight := 1.0

		for _, ep := range sortedEdges {
			edge := simGraph.Edges[ep.idx]
			originalFailProb := ep.failProb

			biasedFailProb := 1.0 - math.Pow(1.0-originalFailProb, biasFactor)
			if biasedFailProb > 1.0 {
				biasedFailProb = 1.0
			}

			if rand.Float64() < biasedFailProb {
				likelihoodRatio := originalFailProb / biasedFailProb
				weight *= likelihoodRatio
				removeEdgeByRef(simGraph, edge)
			} else {
				survivalOrig := 1.0 - originalFailProb
				survivalBiased := 1.0 - biasedFailProb
				if survivalBiased > 0 {
					likelihoodRatio := survivalOrig / survivalBiased
					weight *= likelihoodRatio
				}
			}
		}

		weightSums[i] = weight
		if simGraph.IsConnected() {
			weightedSuccess += weight
		}
	}

	totalWeight := 0.0
	for _, w := range weightSums {
		totalWeight += w
	}

	successRate := 0.0
	if totalWeight > 0 {
		successRate = weightedSuccess / totalWeight
	}

	weightedVariance := 0.0
	meanW := totalWeight / float64(config.Iterations)
	for _, w := range weightSums {
		diff := w - meanW
		weightedVariance += diff * diff
	}
	weightedVariance /= float64(config.Iterations)
	stdErr := math.Sqrt(weightedVariance) / math.Sqrt(float64(config.Iterations))
	confidenceInterval := [2]float64{
		math.Max(0, successRate-1.96*stdErr),
		math.Min(1, successRate+1.96*stdErr),
	}

	return models.MonteCarloResult{
		Iterations:         config.Iterations,
		SuccessRate:        successRate,
		ConfidenceInterval: confidenceInterval,
	}
}

func copyGraph(g *Graph) *Graph {
	newG := NewGraph()
	for id, beacon := range g.Nodes {
		newG.Nodes[id] = beacon
		newG.Adj[id] = make([]int, len(g.Adj[id]))
		copy(newG.Adj[id], g.Adj[id])
	}
	newG.Edges = make([]*NetworkEdge, len(g.Edges))
	for i, edge := range g.Edges {
		newEdge := *edge
		newG.Edges[i] = &newEdge
	}
	return newG
}

func simulateFailures(g *Graph, config MonteCarloConfig) []int {
	failedLinks := make([]int, 0)

	for idx, edge := range g.Edges {
		effectiveReliability := edge.BaseReliability * config.WeatherFactor

		if config.VisibilityMap != nil {
			if visMap, ok := config.VisibilityMap[edge.From]; ok {
				if visible, ok2 := visMap[edge.To]; ok2 && !visible {
					effectiveReliability *= 0.1
				}
			}
		}

		if rand.Float64() > effectiveReliability {
			failedLinks = append(failedLinks, edge.ID)
			removeEdgeByIndex(g, idx)
		}
	}

	return failedLinks
}

func removeEdgeByIndex(g *Graph, edgeIdx int) {
	if edgeIdx >= len(g.Edges) {
		return
	}

	edge := g.Edges[edgeIdx]

	for i, neighbor := range g.Adj[edge.From] {
		if neighbor == edge.To {
			g.Adj[edge.From] = append(g.Adj[edge.From][:i], g.Adj[edge.From][i+1:]...)
			break
		}
	}

	if edge.IsBidirectional {
		for i, neighbor := range g.Adj[edge.To] {
			if neighbor == edge.From {
				g.Adj[edge.To] = append(g.Adj[edge.To][:i], g.Adj[edge.To][i+1:]...)
				break
			}
		}
	}
}

func removeEdgeByRef(g *Graph, edge *NetworkEdge) {
	for i, neighbor := range g.Adj[edge.From] {
		if neighbor == edge.To {
			g.Adj[edge.From] = append(g.Adj[edge.From][:i], g.Adj[edge.From][i+1:]...)
			break
		}
	}

	if edge.IsBidirectional {
		for i, neighbor := range g.Adj[edge.To] {
			if neighbor == edge.From {
				g.Adj[edge.To] = append(g.Adj[edge.To][:i], g.Adj[edge.To][i+1:]...)
				break
			}
		}
	}
}

func WeatherFactor(visibilityKm, windSpeed float64) float64 {
	visibilityFactor := 1.0
	if visibilityKm < 20 {
		visibilityFactor = visibilityKm / 20.0
	}
	if visibilityKm < 2 {
		visibilityFactor = 0.1
	}

	windFactor := 1.0
	if windSpeed > 5 {
		windFactor = 1.0 - (windSpeed-5.0)/20.0
	}
	if windFactor < 0.5 {
		windFactor = 0.5
	}

	return visibilityFactor * windFactor
}

func FindCriticalLinks(g *Graph, config MonteCarloConfig) []int {
	criticalLinks := make([]int, 0)

	for idx, edge := range g.Edges {
		testGraph := copyGraph(g)
		removeEdgeByIndex(testGraph, idx)

		if !testGraph.IsConnected() {
			criticalLinks = append(criticalLinks, edge.ID)
			continue
		}

		testConfig := config
		testConfig.Iterations = config.Iterations / 5
		if testConfig.Iterations < 100 {
			testConfig.Iterations = 100
		}

		result := MonteCarloReliability(testGraph, testConfig)
		originalResult := MonteCarloReliability(g, testConfig)

		if originalResult.SuccessRate-result.SuccessRate > 0.1 {
			criticalLinks = append(criticalLinks, edge.ID)
		}
	}

	return criticalLinks
}

func CalculateNetworkMetrics(g *Graph, weatherFactor float64) map[string]float64 {
	metrics := make(map[string]float64)

	metrics["node_count"] = float64(len(g.Nodes))
	metrics["link_count"] = float64(len(g.Edges))
	metrics["connectivity_index"] = g.ConnectivityIndex()
	metrics["avg_path_length"] = g.AveragePathLength()

	isConnected := g.IsConnected()
	if isConnected {
		metrics["is_connected"] = 1.0
	} else {
		metrics["is_connected"] = 0.0
	}

	criticalCount := 0
	for _, edge := range g.Edges {
		if edge.IsCritical {
			criticalCount++
		}
	}
	metrics["critical_links"] = float64(criticalCount)

	avgReliability := 0.0
	for _, edge := range g.Edges {
		avgReliability += edge.BaseReliability * weatherFactor
	}
	if len(g.Edges) > 0 {
		avgReliability /= float64(len(g.Edges))
	}
	metrics["avg_link_reliability"] = avgReliability

	return metrics
}
