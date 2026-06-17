package analysis

import (
	"beacon-system/models"
	"math"
	"testing"
)

func buildTestLinearGraph(n int) *Graph {
	g := NewGraph()
	for i := 1; i <= n; i++ {
		g.AddNode(&models.Beacon{
			ID:   i,
			Name: "beacon-" + string(rune('0'+i)),
			Lon:  116.0 + float64(i)*0.01,
			Lat:  40.0 + float64(i)*0.01,
		})
	}
	for i := 1; i < n; i++ {
		g.AddEdge(&models.NetworkLink{
			ID:              i,
			FromBeaconID:    i,
			ToBeaconID:      i + 1,
			BaseReliability: 0.9,
			IsBidirectional: true,
			IsCritical:      i == n/2,
		})
	}
	g.BuildAdjacencyList()
	return g
}

func buildTestFullyConnectedGraph(n int) *Graph {
	g := NewGraph()
	for i := 1; i <= n; i++ {
		g.AddNode(&models.Beacon{
			ID:  i,
			Lon: 116.0 + float64(i)*0.01,
			Lat: 40.0 + float64(i)*0.01,
		})
	}
	edgeID := 1
	for i := 1; i <= n; i++ {
		for j := i + 1; j <= n; j++ {
			g.AddEdge(&models.NetworkLink{
				ID:              edgeID,
				FromBeaconID:    i,
				ToBeaconID:      j,
				BaseReliability: 0.95,
				IsBidirectional: true,
			})
			edgeID++
		}
	}
	g.BuildAdjacencyList()
	return g
}

func buildTestStarGraph(center int, leaves int) *Graph {
	g := NewGraph()
	g.AddNode(&models.Beacon{ID: center, Lon: 116.0, Lat: 40.0})
	for i := 1; i <= leaves; i++ {
		g.AddNode(&models.Beacon{
			ID:  center + i,
			Lon: 116.0 + float64(i)*0.02,
			Lat: 40.0 + float64(i)*0.02,
		})
		g.AddEdge(&models.NetworkLink{
			ID:              i,
			FromBeaconID:    center,
			ToBeaconID:      center + i,
			BaseReliability: 0.9,
			IsBidirectional: true,
			IsCritical:      true,
		})
	}
	g.BuildAdjacencyList()
	return g
}

func TestAnalyzeResilience_RandomAttack_LinearGraph(t *testing.T) {
	graph := buildTestLinearGraph(10)
	result := AnalyzeResilience(graph, AttackRandom, 10, 5)

	if result.AttackType != AttackRandom {
		t.Errorf("expected attack type %s, got %s", AttackRandom, result.AttackType)
	}
	if result.TotalNodes != 10 {
		t.Errorf("expected 10 nodes, got %d", result.TotalNodes)
	}
	if len(result.CurvePoints) != 11 {
		t.Errorf("expected 11 curve points, got %d", len(result.CurvePoints))
	}
	if result.CurvePoints[0].RemovalRatio != 0.0 {
		t.Errorf("expected first point ratio 0.0, got %f", result.CurvePoints[0].RemovalRatio)
	}
	if result.CurvePoints[0].ConnectivityIndex != 1.0 {
		t.Errorf("expected initial connectivity 1.0, got %f", result.CurvePoints[0].ConnectivityIndex)
	}
	if result.RobustnessScore <= 0 || result.RobustnessScore >= 1.0 {
		t.Errorf("robustness score should be between 0 and 1, got %f", result.RobustnessScore)
	}
	if result.CriticalThreshold < 0 || result.CriticalThreshold > 1.0 {
		t.Errorf("critical threshold should be between 0 and 1, got %f", result.CriticalThreshold)
	}
}

func TestAnalyzeResilience_DegreeAttack_StarGraph(t *testing.T) {
	graph := buildTestStarGraph(1, 5)
	result := AnalyzeResilience(graph, AttackDegree, 10, 1)

	if result.AttackType != AttackDegree {
		t.Errorf("expected attack type %s, got %s", AttackDegree, result.AttackType)
	}
	if result.TotalNodes != 6 {
		t.Errorf("expected 6 nodes, got %d", result.TotalNodes)
	}
	initialConn := result.CurvePoints[0].ConnectivityIndex
	if initialConn != 1.0 {
		t.Errorf("expected initial connectivity 1.0, got %f", initialConn)
	}
	afterRemovingCenter := result.CurvePoints[2]
	if afterRemovingCenter.ConnectivityIndex >= initialConn {
		t.Errorf("connectivity should drop after removing center node in star graph")
	}
}

func TestAnalyzeResilience_BetweennessAttack(t *testing.T) {
	graph := buildTestLinearGraph(7)
	result := AnalyzeResilience(graph, AttackBetween, 7, 1)

	if result.AttackType != AttackBetween {
		t.Errorf("expected attack type %s, got %s", AttackBetween, result.AttackType)
	}
	if len(result.CurvePoints) != 8 {
		t.Errorf("expected 8 curve points, got %d", len(result.CurvePoints))
	}
}

func TestAnalyzeResilience_CriticalAttack(t *testing.T) {
	graph := buildTestLinearGraph(6)
	result := AnalyzeResilience(graph, AttackCritical, 6, 1)

	if result.AttackType != AttackCritical {
		t.Errorf("expected attack type %s, got %s", AttackCritical, result.AttackType)
	}
}

func TestAnalyzeResilience_EmptyGraph(t *testing.T) {
	graph := NewGraph()
	result := AnalyzeResilience(graph, AttackRandom, 10, 1)

	if result.TotalNodes != 0 {
		t.Errorf("expected 0 nodes, got %d", result.TotalNodes)
	}
	if result.RobustnessScore != 0.0 {
		t.Errorf("expected robustness score 0.0 for empty graph, got %f", result.RobustnessScore)
	}
}

func TestAnalyzeResilience_SingleNode(t *testing.T) {
	graph := NewGraph()
	graph.AddNode(&models.Beacon{ID: 1, Lon: 116.0, Lat: 40.0})
	graph.BuildAdjacencyList()

	result := AnalyzeResilience(graph, AttackRandom, 5, 1)

	if result.TotalNodes != 1 {
		t.Errorf("expected 1 node, got %d", result.TotalNodes)
	}
	if result.CurvePoints[0].ConnectivityIndex != 1.0 {
		t.Errorf("expected connectivity 1.0 for single node, got %f", result.CurvePoints[0].ConnectivityIndex)
	}
}

func TestAnalyzeResilience_DefaultParams(t *testing.T) {
	graph := buildTestLinearGraph(5)
	result := AnalyzeResilience(graph, AttackRandom, -1, -1)

	if len(result.CurvePoints) != 11 {
		t.Errorf("expected default 10 steps (11 points), got %d", len(result.CurvePoints))
	}
	if result.Iterations != 1 {
		t.Errorf("expected default 1 iteration, got %d", result.Iterations)
	}
}

func TestAnalyzeResilience_FullyConnectedRobustness(t *testing.T) {
	fullGraph := buildTestFullyConnectedGraph(6)
	linearGraph := buildTestLinearGraph(6)

	fullResult := AnalyzeResilience(fullGraph, AttackRandom, 5, 3)
	linearResult := AnalyzeResilience(linearGraph, AttackRandom, 5, 3)

	if fullResult.RobustnessScore <= linearResult.RobustnessScore {
		t.Errorf("fully connected graph should have higher robustness score than linear graph")
	}
}

func TestCalculateConnectivityIndex(t *testing.T) {
	tests := []struct {
		name     string
		graph    *Graph
		expected float64
	}{
		{
			name:     "empty graph",
			graph:    NewGraph(),
			expected: 1.0,
		},
		{
			name: "single node",
			graph: func() *Graph {
				g := NewGraph()
				g.AddNode(&models.Beacon{ID: 1})
				return g
			}(),
			expected: 1.0,
		},
		{
			name: "two isolated nodes",
			graph: func() *Graph {
				g := NewGraph()
				g.AddNode(&models.Beacon{ID: 1})
				g.AddNode(&models.Beacon{ID: 2})
				return g
			}(),
			expected: 0.0,
		},
		{
			name: "two connected nodes",
			graph: func() *Graph {
				g := NewGraph()
				g.AddNode(&models.Beacon{ID: 1})
				g.AddNode(&models.Beacon{ID: 2})
				g.AddEdge(&models.NetworkLink{ID: 1, FromBeaconID: 1, ToBeaconID: 2, IsBidirectional: true})
				return g
			}(),
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateConnectivityIndex(tt.graph)
			if math.Abs(result-tt.expected) > 1e-9 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestGetGiantComponentSize(t *testing.T) {
	tests := []struct {
		name     string
		graph    *Graph
		expected int
	}{
		{
			name:     "empty graph",
			graph:    NewGraph(),
			expected: 0,
		},
		{
			name: "linear graph",
			graph: func() *Graph {
				g := buildTestLinearGraph(5)
				return g
			}(),
			expected: 5,
		},
		{
			name: "three isolated components",
			graph: func() *Graph {
				g := NewGraph()
				for i := 1; i <= 6; i++ {
					g.AddNode(&models.Beacon{ID: i})
				}
				g.AddEdge(&models.NetworkLink{ID: 1, FromBeaconID: 1, ToBeaconID: 2, IsBidirectional: true})
				g.AddEdge(&models.NetworkLink{ID: 2, FromBeaconID: 3, ToBeaconID: 4, IsBidirectional: true})
				g.AddEdge(&models.NetworkLink{ID: 3, FromBeaconID: 5, ToBeaconID: 6, IsBidirectional: true})
				g.BuildAdjacencyList()
				return g
			}(),
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getGiantComponentSize(tt.graph)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCalculateRobustnessScore(t *testing.T) {
	tests := []struct {
		name     string
		curve    []ResilienceCurvePoint
		expected float64
	}{
		{
			name:     "empty curve",
			curve:    []ResilienceCurvePoint{},
			expected: 0.0,
		},
		{
			name:     "single point",
			curve:    []ResilienceCurvePoint{{RemovalRatio: 0.0, ConnectivityIndex: 1.0}},
			expected: 0.0,
		},
		{
			name: "perfect robustness",
			curve: []ResilienceCurvePoint{
				{RemovalRatio: 0.0, ConnectivityIndex: 1.0},
				{RemovalRatio: 1.0, ConnectivityIndex: 1.0},
			},
			expected: 1.0,
		},
		{
			name: "linear decline",
			curve: []ResilienceCurvePoint{
				{RemovalRatio: 0.0, ConnectivityIndex: 1.0},
				{RemovalRatio: 1.0, ConnectivityIndex: 0.0},
			},
			expected: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateRobustnessScore(tt.curve)
			if math.Abs(result-tt.expected) > 1e-9 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestFindCriticalThreshold(t *testing.T) {
	tests := []struct {
		name     string
		curve    []ResilienceCurvePoint
		expected float64
	}{
		{
			name:     "empty curve",
			curve:    []ResilienceCurvePoint{},
			expected: 1.0,
		},
		{
			name: "all above 50%",
			curve: []ResilienceCurvePoint{
				{RemovalRatio: 0.0, GiantComponentPct: 1.0},
				{RemovalRatio: 0.5, GiantComponentPct: 0.8},
				{RemovalRatio: 1.0, GiantComponentPct: 0.6},
			},
			expected: 1.0,
		},
		{
			name: "starts below 50%",
			curve: []ResilienceCurvePoint{
				{RemovalRatio: 0.0, GiantComponentPct: 0.3},
			},
			expected: 0.0,
		},
		{
			name: "crosses at 0.4",
			curve: []ResilienceCurvePoint{
				{RemovalRatio: 0.0, GiantComponentPct: 1.0},
				{RemovalRatio: 0.3, GiantComponentPct: 0.7},
				{RemovalRatio: 0.5, GiantComponentPct: 0.3},
			},
			expected: 0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findCriticalThreshold(tt.curve)
			if math.Abs(result-tt.expected) > 1e-9 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestRemoveNodes(t *testing.T) {
	graph := buildTestLinearGraph(5)
	removed := removeNodes(graph, []int{3})

	if len(removed.Nodes) != 4 {
		t.Errorf("expected 4 nodes after removal, got %d", len(removed.Nodes))
	}
	if _, exists := removed.Nodes[3]; exists {
		t.Errorf("node 3 should be removed")
	}
}

func TestBuildAdjacencyList(t *testing.T) {
	graph := NewGraph()
	graph.AddNode(&models.Beacon{ID: 1})
	graph.AddNode(&models.Beacon{ID: 2})
	graph.AddNode(&models.Beacon{ID: 3})
	graph.AddEdge(&models.NetworkLink{ID: 1, FromBeaconID: 1, ToBeaconID: 2, IsBidirectional: true})
	graph.AddEdge(&models.NetworkLink{ID: 2, FromBeaconID: 2, ToBeaconID: 3, IsBidirectional: false})

	graph.BuildAdjacencyList()

	if len(graph.Adj[1]) != 1 {
		t.Errorf("node 1 should have 1 neighbor, got %d", len(graph.Adj[1]))
	}
	if len(graph.Adj[2]) != 2 {
		t.Errorf("node 2 should have 2 neighbors (bidirectional), got %d", len(graph.Adj[2]))
	}
	if len(graph.Adj[3]) != 0 {
		t.Errorf("node 3 should have 0 neighbors (unidirectional edge), got %d", len(graph.Adj[3]))
	}
}

func TestIsCriticalNode(t *testing.T) {
	graph := buildTestLinearGraph(5)

	if isCriticalNode(graph, 1) {
		t.Errorf("node 1 should not be critical")
	}
}

func TestPickRandomNodes(t *testing.T) {
	graph := buildTestLinearGraph(10)

	result := pickRandomNodes(graph, 3, 42)
	if len(result) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(result))
	}

	result2 := pickRandomNodes(graph, 3, 42)
	if len(result) != len(result2) {
		t.Errorf("same seed should produce same result")
	}

	allResult := pickRandomNodes(graph, 100, 1)
	if len(allResult) != 10 {
		t.Errorf("requesting more than available should return all, got %d", len(allResult))
	}
}

func TestComputeBetweenness(t *testing.T) {
	graph := buildTestLinearGraph(5)
	betweenness := computeBetweenness(graph)

	if len(betweenness) != 5 {
		t.Errorf("expected 5 betweenness values, got %d", len(betweenness))
	}
	middleNodeBw := betweenness[3]
	endNodeBw := betweenness[1]
	if middleNodeBw <= endNodeBw {
		t.Errorf("middle node should have higher betweenness than end node")
	}
}

func TestAttackTypesComparison(t *testing.T) {
	graph := buildTestStarGraph(1, 8)

	randomResult := AnalyzeResilience(graph, AttackRandom, 5, 10)
	degreeResult := AnalyzeResilience(graph, AttackDegree, 5, 1)

	if degreeResult.RobustnessScore >= randomResult.RobustnessScore {
		t.Errorf("degree attack should result in lower robustness than random attack on star graph")
	}
}
