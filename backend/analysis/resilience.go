package analysis

import (
	"math"
	"math/rand"
	"sort"
	"sync"
)

const (
	AttackRandom   = "random"
	AttackDegree   = "degree"
	AttackBetween  = "betweenness"
	AttackCritical = "critical"
)

type ResilienceCurvePoint struct {
	RemovalRatio      float64
	ConnectivityIndex float64
	GiantComponentPct float64
}

type ResilienceResult struct {
	AttackType        string
	CurvePoints       []ResilienceCurvePoint
	RobustnessScore   float64
	CriticalThreshold float64
	TotalNodes        int
	Iterations        int
}

func AnalyzeResilience(graph *Graph, attackType string, steps int, iterations int) *ResilienceResult {
	if steps <= 0 {
		steps = 10
	}
	if iterations <= 0 {
		iterations = 1
	}

	totalNodes := len(graph.Nodes)
	if totalNodes == 0 {
		return &ResilienceResult{
			AttackType:        attackType,
			CurvePoints:       []ResilienceCurvePoint{{RemovalRatio: 0.0, ConnectivityIndex: 0.0, GiantComponentPct: 0.0}},
			RobustnessScore:   0.0,
			CriticalThreshold: 0.0,
			TotalNodes:        0,
			Iterations:        iterations,
		}
	}
	curve := make([]ResilienceCurvePoint, steps+1)
	curve[0] = ResilienceCurvePoint{
		RemovalRatio:      0.0,
		ConnectivityIndex: calculateConnectivityIndex(graph),
		GiantComponentPct: 1.0,
	}

	nodeOrder := getNodeRemovalOrder(graph, attackType)

	for s := 1; s <= steps; s++ {
		ratio := float64(s) / float64(steps)
		removeCount := int(math.Round(float64(totalNodes) * ratio))
		if removeCount >= totalNodes {
			curve[s] = ResilienceCurvePoint{
				RemovalRatio:      ratio,
				ConnectivityIndex: 0.0,
				GiantComponentPct: 0.0,
			}
			continue
		}

		avgConnIdx := 0.0
		avgGiantPct := 0.0

		for iter := 0; iter < iterations; iter++ {
			var removeIDs []int
			if attackType == AttackRandom {
				removeIDs = pickRandomNodes(graph, removeCount, iter)
			} else {
				removeIDs = nodeOrder[:removeCount]
			}

			subGraph := removeNodes(graph, removeIDs)
			connIdx := calculateConnectivityIndex(subGraph)
			giantSize := getGiantComponentSize(subGraph)
			giantPct := float64(giantSize) / float64(totalNodes)

			avgConnIdx += connIdx
			avgGiantPct += giantPct
		}

		curve[s] = ResilienceCurvePoint{
			RemovalRatio:      ratio,
			ConnectivityIndex: avgConnIdx / float64(iterations),
			GiantComponentPct: avgGiantPct / float64(iterations),
		}
	}

	robustnessScore := calculateRobustnessScore(curve)
	criticalThreshold := findCriticalThreshold(curve)

	return &ResilienceResult{
		AttackType:        attackType,
		CurvePoints:       curve,
		RobustnessScore:   robustnessScore,
		CriticalThreshold: criticalThreshold,
		TotalNodes:        totalNodes,
		Iterations:        iterations,
	}
}

func getNodeRemovalOrder(graph *Graph, attackType string) []int {
	nodeIDs := make([]int, 0, len(graph.Nodes))
	for id := range graph.Nodes {
		nodeIDs = append(nodeIDs, id)
	}

	switch attackType {
	case AttackDegree:
		sort.Slice(nodeIDs, func(i, j int) bool {
			return len(graph.Adj[nodeIDs[i]]) > len(graph.Adj[nodeIDs[j]])
		})
	case AttackCritical:
		sort.Slice(nodeIDs, func(i, j int) bool {
			return isCriticalNode(graph, nodeIDs[i]) && !isCriticalNode(graph, nodeIDs[j])
		})
	case AttackBetween:
		betweenness := computeBetweenness(graph)
		sort.Slice(nodeIDs, func(i, j int) bool {
			return betweenness[nodeIDs[i]] > betweenness[nodeIDs[j]]
		})
	default:
		shuffle(nodeIDs)
	}

	return nodeIDs
}

func isCriticalNode(graph *Graph, nodeID int) bool {
	for _, edge := range graph.Edges {
		if edge.IsCritical && (edge.From == nodeID || edge.To == nodeID) {
			return true
		}
	}
	return false
}

func computeBetweenness(graph *Graph) map[int]float64 {
	betweenness := make(map[int]float64)
	for id := range graph.Nodes {
		betweenness[id] = 0.0
	}

	var wg sync.WaitGroup
	mu := sync.Mutex{}

	for srcID := range graph.Nodes {
		wg.Add(1)
		go func(src int) {
			defer wg.Done()
			localBw := bfsBetweenness(graph, src)
			mu.Lock()
			for k, v := range localBw {
				betweenness[k] += v
			}
			mu.Unlock()
		}(srcID)
	}
	wg.Wait()

	return betweenness
}

func bfsBetweenness(graph *Graph, source int) map[int]float64 {
	sigma := make(map[int]float64)
	dist := make(map[int]int)
	pred := make(map[int][]int)
	stack := []int{}

	for id := range graph.Nodes {
		sigma[id] = 0.0
		dist[id] = -1
	}
	sigma[source] = 1.0
	dist[source] = 0

	queue := []int{source}
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		stack = append(stack, v)

		for _, w := range graph.Adj[v] {
			if dist[w] < 0 {
				queue = append(queue, w)
				dist[w] = dist[v] + 1
			}
			if dist[w] == dist[v]+1 {
				sigma[w] += sigma[v]
				pred[w] = append(pred[w], v)
			}
		}
	}

	delta := make(map[int]float64)
	for id := range graph.Nodes {
		delta[id] = 0.0
	}

	for i := len(stack) - 1; i >= 0; i-- {
		w := stack[i]
		for _, v := range pred[w] {
			delta[v] += (sigma[v] / sigma[w]) * (1.0 + delta[w])
		}
		if w != source {
			delta[w] += 0.0
		}
	}

	result := make(map[int]float64)
	for k, v := range delta {
		result[k] = v / 2.0
	}
	return result
}

func pickRandomNodes(graph *Graph, count int, seed int) []int {
	ids := make([]int, 0, len(graph.Nodes))
	for id := range graph.Nodes {
		ids = append(ids, id)
	}
	r := rand.New(rand.NewSource(int64(seed * 1000)))
	for i := len(ids) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		ids[i], ids[j] = ids[j], ids[i]
	}
	if count > len(ids) {
		count = len(ids)
	}
	return ids[:count]
}

func removeNodes(graph *Graph, removeIDs []int) *Graph {
	removeSet := make(map[int]bool)
	for _, id := range removeIDs {
		removeSet[id] = true
	}

	newGraph := NewGraph()
	for id, node := range graph.Nodes {
		if !removeSet[id] {
			newGraph.AddNode(node)
		}
	}

	for _, edge := range graph.Edges {
		if !removeSet[edge.From] && !removeSet[edge.To] {
			newGraph.Edges = append(newGraph.Edges, edge)
		}
	}

	newGraph.BuildAdjacencyList()
	return newGraph
}

func calculateConnectivityIndex(graph *Graph) float64 {
	if len(graph.Nodes) <= 1 {
		return 1.0
	}

	visited := make(map[int]bool)
	componentCount := 0

	for id := range graph.Nodes {
		if !visited[id] {
			componentCount++
			dfsComponent(graph, id, visited)
		}
	}

	if componentCount == 1 {
		return 1.0
	}
	return 1.0 - float64(componentCount-1)/float64(len(graph.Nodes)-1)
}

func dfsComponent(graph *Graph, start int, visited map[int]bool) {
	stack := []int{start}
	visited[start] = true

	for len(stack) > 0 {
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		for _, neighbor := range graph.Adj[v] {
			if !visited[neighbor] {
				visited[neighbor] = true
				stack = append(stack, neighbor)
			}
		}
	}
}

func getGiantComponentSize(graph *Graph) int {
	if len(graph.Nodes) == 0 {
		return 0
	}

	visited := make(map[int]bool)
	maxSize := 0

	for id := range graph.Nodes {
		if !visited[id] {
			size := 0
			stack := []int{id}
			visited[id] = true

			for len(stack) > 0 {
				v := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				size++

				for _, neighbor := range graph.Adj[v] {
					if !visited[neighbor] {
						visited[neighbor] = true
						stack = append(stack, neighbor)
					}
				}
			}

			if size > maxSize {
				maxSize = size
			}
		}
	}

	return maxSize
}

func calculateRobustnessScore(curve []ResilienceCurvePoint) float64 {
	if len(curve) < 2 {
		return 0.0
	}

	area := 0.0
	for i := 1; i < len(curve); i++ {
		dx := curve[i].RemovalRatio - curve[i-1].RemovalRatio
		avgY := (curve[i].ConnectivityIndex + curve[i-1].ConnectivityIndex) / 2.0
		area += dx * avgY
	}

	return area
}

func findCriticalThreshold(curve []ResilienceCurvePoint) float64 {
	for i := 0; i < len(curve); i++ {
		if curve[i].GiantComponentPct < 0.5 {
			if i == 0 {
				return 0.0
			}
			ratio0 := curve[i-1].RemovalRatio
			ratio1 := curve[i].RemovalRatio
			y0 := curve[i-1].GiantComponentPct
			y1 := curve[i].GiantComponentPct

			t := (0.5 - y0) / (y1 - y0)
			return ratio0 + t*(ratio1-ratio0)
		}
	}
	return 1.0
}

func (g *Graph) BuildAdjacencyList() {
	g.Adj = make(map[int][]int)
	for id := range g.Nodes {
		g.Adj[id] = make([]int, 0)
	}
	for _, edge := range g.Edges {
		g.Adj[edge.From] = append(g.Adj[edge.From], edge.To)
		if edge.IsBidirectional {
			g.Adj[edge.To] = append(g.Adj[edge.To], edge.From)
		}
	}
}

func shuffle(slice []int) {
	for i := len(slice) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}
