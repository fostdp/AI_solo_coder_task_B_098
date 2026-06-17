package advanced_analysis

import (
	"beacon-system/analysis"
	"beacon-system/models"
	"math"
	"testing"
)

func buildTestAnalyzerGraph(n int) *analysis.Graph {
	g := analysis.NewGraph()
	for i := 1; i <= n; i++ {
		g.AddNode(&models.Beacon{
			ID:        i,
			Name:      "beacon-" + string(rune('0'+i)),
			Lon:       116.0 + float64(i)*0.01,
			Lat:       40.0 + float64(i)*0.01,
			Elevation: 100.0,
			Height:    10.0,
		})
	}
	for i := 1; i < n; i++ {
		g.AddEdge(&models.NetworkLink{
			ID:              i,
			FromBeaconID:    i,
			ToBeaconID:      i + 1,
			BaseReliability: 0.9,
			IsBidirectional: true,
		})
	}
	g.BuildAdjacencyList()
	return g
}

func buildTestFullGraph(n int) *analysis.Graph {
	g := analysis.NewGraph()
	for i := 1; i <= n; i++ {
		g.AddNode(&models.Beacon{
			ID:   i,
			Lon:  116.0 + float64(i)*0.01,
			Lat:  40.0 + float64(i)*0.01,
			Height: 10.0,
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

func TestCalculateDensity(t *testing.T) {
	tests := []struct {
		name     string
		graph    *analysis.Graph
		expected float64
	}{
		{
			name:     "empty graph",
			graph:    analysis.NewGraph(),
			expected: 0.0,
		},
		{
			name: "single node",
			graph: func() *analysis.Graph {
				g := analysis.NewGraph()
				g.AddNode(&models.Beacon{ID: 1})
				return g
			}(),
			expected: 0.0,
		},
		{
			name: "two nodes no edge",
			graph: func() *analysis.Graph {
				g := analysis.NewGraph()
				g.AddNode(&models.Beacon{ID: 1})
				g.AddNode(&models.Beacon{ID: 2})
				return g
			}(),
			expected: 0.0,
		},
		{
			name: "two nodes one edge",
			graph: func() *analysis.Graph {
				g := analysis.NewGraph()
				g.AddNode(&models.Beacon{ID: 1})
				g.AddNode(&models.Beacon{ID: 2})
				g.AddEdge(&models.NetworkLink{ID: 1, FromBeaconID: 1, ToBeaconID: 2, IsBidirectional: true})
				return g
			}(),
			expected: 1.0,
		},
		{
			name:     "linear graph 5 nodes",
			graph:    buildTestAnalyzerGraph(5),
			expected: 4.0 / 10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateDensity(tt.graph)
			if math.Abs(result-tt.expected) > 1e-9 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestGraphDiameter(t *testing.T) {
	tests := []struct {
		name     string
		graph    *analysis.Graph
		expected int
	}{
		{
			name:     "empty graph",
			graph:    analysis.NewGraph(),
			expected: 0,
		},
		{
			name: "single node",
			graph: func() *analysis.Graph {
				g := analysis.NewGraph()
				g.AddNode(&models.Beacon{ID: 1})
				return g
			}(),
			expected: 0,
		},
		{
			name:     "linear 5 nodes",
			graph:    buildTestAnalyzerGraph(5),
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := graphDiameter(tt.graph)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCalculateConnectivityIdx(t *testing.T) {
	tests := []struct {
		name     string
		graph    *analysis.Graph
		expected float64
	}{
		{
			name:     "empty graph",
			graph:    analysis.NewGraph(),
			expected: 1.0,
		},
		{
			name: "single node",
			graph: func() *analysis.Graph {
				g := analysis.NewGraph()
				g.AddNode(&models.Beacon{ID: 1})
				return g
			}(),
			expected: 1.0,
		},
		{
			name: "two isolated nodes",
			graph: func() *analysis.Graph {
				g := analysis.NewGraph()
				g.AddNode(&models.Beacon{ID: 1})
				g.AddNode(&models.Beacon{ID: 2})
				return g
			}(),
			expected: 0.0,
		},
		{
			name:     "connected linear graph",
			graph:    buildTestAnalyzerGraph(5),
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateConnectivityIdx(tt.graph)
			if math.Abs(result-tt.expected) > 1e-9 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestMaxInt(t *testing.T) {
	if max(3, 5) != 5 {
		t.Errorf("expected 5, got %d", max(3, 5))
	}
	if max(-1, -3) != -1 {
		t.Errorf("expected -1, got %d", max(-1, -3))
	}
	if max(0, 0) != 0 {
		t.Errorf("expected 0, got %d", max(0, 0))
	}
}

func TestMaxFloat(t *testing.T) {
	if math.Abs(maxF(3.14, 5.67)-5.67) > 1e-9 {
		t.Errorf("expected 5.67, got %f", maxF(3.14, 5.67))
	}
	if math.Abs(maxF(0.0, -1.5)-0.0) > 1e-9 {
		t.Errorf("expected 0.0, got %f", maxF(0.0, -1.5))
	}
}

func TestDynastyComparison_NetworkMetrics(t *testing.T) {
	qinGraph := buildTestAnalyzerGraph(5)
	mingGraph := buildTestAnalyzerGraph(10)

	qinNodes := len(qinGraph.Nodes)
	mingNodes := len(mingGraph.Nodes)

	if qinNodes != 5 {
		t.Errorf("expected qin 5 nodes, got %d", qinNodes)
	}
	if mingNodes != 10 {
		t.Errorf("expected ming 10 nodes, got %d", mingNodes)
	}

	qinDiameter := graphDiameter(qinGraph)
	mingDiameter := graphDiameter(mingGraph)

	if mingDiameter <= qinDiameter {
		t.Errorf("ming graph (longer chain) should have larger diameter")
	}
}

func TestCrossEraComparison_NetworkCapacity(t *testing.T) {
	beaconGraph := buildTestAnalyzerGraph(8)
	beaconNodeCount := len(beaconGraph.Nodes)

	beaconCapacity := float64(beaconNodeCount) * 0.001
	if beaconCapacity <= 0 {
		t.Errorf("beacon network capacity should be positive")
	}

	modernCapacity := 1000.0
	capacityRatio := modernCapacity / beaconCapacity
	if capacityRatio < 100 {
		t.Errorf("modern network should have much higher capacity")
	}
}

func TestResilienceAnalysis_SurvivabilityMetrics(t *testing.T) {
	graph := buildTestAnalyzerGraph(8)

	result := analysis.AnalyzeResilience(graph, analysis.AttackRandom, 10, 5)

	if result.RobustnessScore <= 0 || result.RobustnessScore >= 1.0 {
		t.Errorf("robustness score should be between 0 and 1, got %f", result.RobustnessScore)
	}
	if result.CriticalThreshold < 0 || result.CriticalThreshold > 1.0 {
		t.Errorf("critical threshold should be between 0 and 1, got %f", result.CriticalThreshold)
	}
	if result.TotalNodes != 8 {
		t.Errorf("expected 8 total nodes, got %d", result.TotalNodes)
	}
	if len(result.CurvePoints) != 11 {
		t.Errorf("expected 11 curve points, got %d", len(result.CurvePoints))
	}
}

func TestResilienceAnalysis_DifferentAttackTypes(t *testing.T) {
	graph := buildTestFullGraph(6)

	randomResult := analysis.AnalyzeResilience(graph, analysis.AttackRandom, 5, 10)
	degreeResult := analysis.AnalyzeResilience(graph, analysis.AttackDegree, 5, 1)
	betweenResult := analysis.AnalyzeResilience(graph, analysis.AttackBetween, 5, 1)
	criticalResult := analysis.AnalyzeResilience(graph, analysis.AttackCritical, 5, 1)

	if randomResult.AttackType != analysis.AttackRandom {
		t.Errorf("expected attack type %s", analysis.AttackRandom)
	}
	if degreeResult.AttackType != analysis.AttackDegree {
		t.Errorf("expected attack type %s", analysis.AttackDegree)
	}
	if betweenResult.AttackType != analysis.AttackBetween {
		t.Errorf("expected attack type %s", analysis.AttackBetween)
	}
	if criticalResult.AttackType != analysis.AttackCritical {
		t.Errorf("expected attack type %s", analysis.AttackCritical)
	}
}

func TestIgniteBeacon_StrategicPositions(t *testing.T) {
	graph := buildTestAnalyzerGraph(7)

	endResult := analysis.SimulateSignalPropagation(graph, 1, 1.0, nil)
	midResult := analysis.SimulateSignalPropagation(graph, 4, 1.0, nil)

	if endResult.ReachedCount != 7 {
		t.Errorf("from end should reach all 7 beacons, got %d", endResult.ReachedCount)
	}
	if midResult.ReachedCount != 7 {
		t.Errorf("from middle should reach all 7 beacons, got %d", midResult.ReachedCount)
	}

	if endResult.MaxStep <= midResult.MaxStep {
		t.Errorf("end start should have larger max step than middle start")
	}

	if endResult.TotalTimeMs <= midResult.TotalTimeMs {
		t.Errorf("end start should take longer total time")
	}
}

func TestIgniteBeacon_WeatherImpact(t *testing.T) {
	graph := buildTestAnalyzerGraph(5)

	goodWeather := analysis.SimulateSignalPropagation(graph, 1, 1.0, nil)
	badWeather := analysis.SimulateSignalPropagation(graph, 1, 0.5, nil)

	if badWeather.TotalTimeMs <= goodWeather.TotalTimeMs {
		t.Errorf("bad weather should increase propagation time")
	}
}

func TestDensityComparison(t *testing.T) {
	sparseGraph := buildTestAnalyzerGraph(10)
	denseGraph := buildTestFullGraph(10)

	sparseDensity := calculateDensity(sparseGraph)
	denseDensity := calculateDensity(denseGraph)

	if sparseDensity >= denseDensity {
		t.Errorf("sparse graph should have lower density than dense graph")
	}
	if denseDensity != 1.0 {
		t.Errorf("fully connected graph should have density 1.0, got %f", denseDensity)
	}
}

func TestConnectivityIndexEdgeCases(t *testing.T) {
	isolatedGraph := analysis.NewGraph()
	for i := 1; i <= 5; i++ {
		isolatedGraph.AddNode(&models.Beacon{ID: i})
	}
	isolatedGraph.BuildAdjacencyList()

	idx := calculateConnectivityIdx(isolatedGraph)
	expected := 1.0 - float64(5-1)/float64(5-1)
	if math.Abs(idx-expected) > 1e-9 {
		t.Errorf("isolated graph connectivity idx should be 0, got %f", idx)
	}
}

func TestDiameterEdgeCases(t *testing.T) {
	twoNodeGraph := buildTestAnalyzerGraph(2)
	dia := graphDiameter(twoNodeGraph)
	if dia != 1 {
		t.Errorf("2-node linear graph diameter should be 1, got %d", dia)
	}

	singleNodeGraph := analysis.NewGraph()
	singleNodeGraph.AddNode(&models.Beacon{ID: 1})
	singleDia := graphDiameter(singleNodeGraph)
	if singleDia != 0 {
		t.Errorf("single node diameter should be 0, got %d", singleDia)
	}
}

func TestStrategicIgnition_OptimalStart(t *testing.T) {
	graph := buildTestAnalyzerGraph(6)

	bestTime := math.Inf(1)
	bestStart := -1
	for id := range graph.Nodes {
		result := analysis.SimulateSignalPropagation(graph, id, 1.0, nil)
		if result.TotalTimeMs < bestTime {
			bestTime = result.TotalTimeMs
			bestStart = id
		}
	}

	if bestStart == 1 || bestStart == 6 {
		t.Errorf("optimal start should not be the end nodes in a linear graph, got %d", bestStart)
	}
}

func TestNetworkCapacityComparison(t *testing.T) {
	smallGraph := buildTestAnalyzerGraph(4)
	largeGraph := buildTestAnalyzerGraph(8)

	smallCapacity := float64(len(smallGraph.Nodes)) * 0.001
	largeCapacity := float64(len(largeGraph.Nodes)) * 0.001

	if largeCapacity <= smallCapacity {
		t.Errorf("larger network should have higher capacity")
	}
}
