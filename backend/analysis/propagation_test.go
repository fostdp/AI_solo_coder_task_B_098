package analysis

import (
	"beacon-system/models"
	"math"
	"testing"
)

func buildTestPropagationGraph() *Graph {
	g := NewGraph()
	g.AddNode(&models.Beacon{ID: 1, Name: "beacon-1", Lon: 116.4, Lat: 40.0})
	g.AddNode(&models.Beacon{ID: 2, Name: "beacon-2", Lon: 116.42, Lat: 40.0})
	g.AddNode(&models.Beacon{ID: 3, Name: "beacon-3", Lon: 116.44, Lat: 40.0})
	g.AddNode(&models.Beacon{ID: 4, Name: "beacon-4", Lon: 116.46, Lat: 40.0})
	g.AddEdge(&models.NetworkLink{
		ID:              1,
		FromBeaconID:    1,
		ToBeaconID:      2,
		BaseReliability: 0.9,
		IsBidirectional: true,
	})
	g.AddEdge(&models.NetworkLink{
		ID:              2,
		FromBeaconID:    2,
		ToBeaconID:      3,
		BaseReliability: 0.9,
		IsBidirectional: true,
	})
	g.AddEdge(&models.NetworkLink{
		ID:              3,
		FromBeaconID:    3,
		ToBeaconID:      4,
		BaseReliability: 0.9,
		IsBidirectional: true,
	})
	g.BuildAdjacencyList()
	return g
}

func TestSimulateSignalPropagation_NormalCase(t *testing.T) {
	graph := buildTestPropagationGraph()
	beaconNames := map[int]string{
		1: "beacon-1",
		2: "beacon-2",
		3: "beacon-3",
		4: "beacon-4",
	}
	result := SimulateSignalPropagation(graph, 1, 1.0, beaconNames)

	if result.StartBeaconID != 1 {
		t.Errorf("expected start beacon 1, got %d", result.StartBeaconID)
	}
	if result.ReachedCount != 4 {
		t.Errorf("expected 4 reached beacons, got %d", result.ReachedCount)
	}
	if result.MaxStep != 3 {
		t.Errorf("expected max step 3, got %d", result.MaxStep)
	}
	if result.TotalTimeMs <= 0 {
		t.Errorf("expected positive total time, got %f", result.TotalTimeMs)
	}
	if len(result.Steps) != 4 {
		t.Errorf("expected 4 step groups, got %d", len(result.Steps))
	}
}

func TestSimulateSignalPropagation_InvalidStart(t *testing.T) {
	graph := buildTestPropagationGraph()
	result := SimulateSignalPropagation(graph, 999, 1.0, nil)

	if result.ReachedCount != 0 {
		t.Errorf("expected 0 reached for invalid start, got %d", result.ReachedCount)
	}
	if result.TotalTimeMs != 0 {
		t.Errorf("expected 0 time for invalid start, got %f", result.TotalTimeMs)
	}
	if len(result.Path) != 0 {
		t.Errorf("expected empty path for invalid start, got %d", len(result.Path))
	}
}

func TestSimulateSignalPropagation_SingleNode(t *testing.T) {
	graph := NewGraph()
	graph.AddNode(&models.Beacon{ID: 1, Lon: 116.0, Lat: 40.0})
	graph.BuildAdjacencyList()

	result := SimulateSignalPropagation(graph, 1, 1.0, nil)

	if result.ReachedCount != 1 {
		t.Errorf("expected 1 reached, got %d", result.ReachedCount)
	}
	if result.MaxStep != 0 {
		t.Errorf("expected max step 0, got %d", result.MaxStep)
	}
	if result.TotalTimeMs != 0 {
		t.Errorf("expected 0 time for single node, got %f", result.TotalTimeMs)
	}
}

func TestSimulateSignalPropagation_EmptyGraph(t *testing.T) {
	graph := NewGraph()
	result := SimulateSignalPropagation(graph, 1, 1.0, nil)

	if result.ReachedCount != 0 {
		t.Errorf("expected 0 reached for empty graph, got %d", result.ReachedCount)
	}
}

func TestSimulateSignalPropagation_WeatherFactorEffect(t *testing.T) {
	graph := buildTestPropagationGraph()

	goodWeather := SimulateSignalPropagation(graph, 1, 1.0, nil)
	badWeather := SimulateSignalPropagation(graph, 1, 0.5, nil)

	if badWeather.TotalTimeMs <= goodWeather.TotalTimeMs {
		t.Errorf("bad weather should increase propagation time. Good: %f, Bad: %f",
			goodWeather.TotalTimeMs, badWeather.TotalTimeMs)
	}
}

func TestSimulateSignalPropagation_BeaconNames(t *testing.T) {
	graph := buildTestPropagationGraph()
	beaconNames := map[int]string{
		1: "一号烽火台",
		2: "二号烽火台",
		3: "三号烽火台",
		4: "四号烽火台",
	}
	result := SimulateSignalPropagation(graph, 1, 1.0, beaconNames)

	if len(result.Path) != 4 {
		t.Fatalf("expected 4 path steps, got %d", len(result.Path))
	}
	for _, step := range result.Path {
		if step.BeaconName == "" {
			t.Errorf("beacon %d should have a name", step.BeaconID)
		}
	}
}

func TestSimulateSignalPropagation_StepStructure(t *testing.T) {
	graph := buildTestPropagationGraph()
	result := SimulateSignalPropagation(graph, 1, 1.0, nil)

	if len(result.Steps) != result.MaxStep+1 {
		t.Errorf("steps length should be maxStep+1")
	}
	if len(result.Steps[0]) != 1 {
		t.Errorf("step 0 should have 1 beacon (the start), got %d", len(result.Steps[0]))
	}
	if result.Steps[0][0] != 1 {
		t.Errorf("step 0 should be beacon 1, got %d", result.Steps[0][0])
	}
}

func TestSimulateSignalPropagation_PathOrder(t *testing.T) {
	graph := buildTestPropagationGraph()
	result := SimulateSignalPropagation(graph, 1, 1.0, nil)

	if len(result.Path) == 0 {
		t.Fatal("path should not be empty")
	}
	if result.Path[0].BeaconID != 1 {
		t.Errorf("first path entry should be start beacon, got %d", result.Path[0].BeaconID)
	}
	if result.Path[0].Step != 0 {
		t.Errorf("first path entry should be step 0, got %d", result.Path[0].Step)
	}
	if result.Path[0].DelayMs != 0 {
		t.Errorf("first path entry should have 0 delay, got %f", result.Path[0].DelayMs)
	}
	if result.Path[0].FromBeacon != -1 {
		t.Errorf("first path entry should have no predecessor, got %d", result.Path[0].FromBeacon)
	}
}

func TestSimulateSignalPropagation_DistanceDelay(t *testing.T) {
	graph := buildTestPropagationGraph()
	result := SimulateSignalPropagation(graph, 1, 1.0, nil)

	step1Delay := float64(0)
	step2Delay := float64(0)
	for _, step := range result.Path {
		if step.BeaconID == 2 {
			step1Delay = step.DelayMs
		}
		if step.BeaconID == 3 {
			step2Delay = step.DelayMs
		}
	}

	if step1Delay <= 0 {
		t.Errorf("beacon 2 should have positive delay")
	}
	if step2Delay <= step1Delay {
		t.Errorf("beacon 3 should have larger delay than beacon 2")
	}
}

func TestSimulateSignalPropagation_WeatherFactorZero(t *testing.T) {
	graph := buildTestPropagationGraph()
	result := SimulateSignalPropagation(graph, 1, 0.0, nil)

	if result.ReachedCount != 4 {
		t.Errorf("with weatherFactor=0, still should traverse all nodes, got %d", result.ReachedCount)
	}
}

func TestSimulateSignalPropagation_WeatherFactorOne(t *testing.T) {
	graph := buildTestPropagationGraph()
	result := SimulateSignalPropagation(graph, 1, 1.0, nil)

	if result.ReachedCount != 4 {
		t.Errorf("with weatherFactor=1, should reach all 4 beacons, got %d", result.ReachedCount)
	}
}

func TestGetDistanceBetween(t *testing.T) {
	graph := buildTestPropagationGraph()

	dist := getDistanceBetween(graph, 1, 2)
	if dist <= 0 {
		t.Errorf("distance should be positive, got %f", dist)
	}

	distSame := getDistanceBetween(graph, 1, 1)
	if distSame != 0 {
		t.Errorf("distance to self should be 0, got %f", distSame)
	}
}

func TestGetLinkReliability(t *testing.T) {
	graph := buildTestPropagationGraph()

	rel := getLinkReliability(graph, 1, 2)
	if math.Abs(rel-0.9) > 1e-9 {
		t.Errorf("expected reliability 0.9, got %f", rel)
	}

	relReverse := getLinkReliability(graph, 2, 1)
	if math.Abs(relReverse-0.9) > 1e-9 {
		t.Errorf("bidirectional link should have same reliability reverse, got %f", relReverse)
	}

	relMissing := getLinkReliability(graph, 1, 999)
	if relMissing != 0.5 {
		t.Errorf("non-existent link should return default 0.5, got %f", relMissing)
	}
}

func TestSimulateSignalPropagation_ComplexNetwork(t *testing.T) {
	graph := NewGraph()
	for i := 1; i <= 6; i++ {
		graph.AddNode(&models.Beacon{
			ID:  i,
			Lon: 116.0 + float64(i)*0.01,
			Lat: 40.0 + float64(i)*0.005,
		})
	}
	edges := [][2]int{
		{1, 2}, {1, 3}, {2, 4}, {3, 4},
		{4, 5}, {3, 6}, {5, 6},
	}
	for i, e := range edges {
		graph.AddEdge(&models.NetworkLink{
			ID:              i + 1,
			FromBeaconID:    e[0],
			ToBeaconID:      e[1],
			BaseReliability: 0.85,
			IsBidirectional: true,
		})
	}
	graph.BuildAdjacencyList()

	result := SimulateSignalPropagation(graph, 1, 1.0, nil)

	if result.ReachedCount != 6 {
		t.Errorf("expected all 6 beacons reached, got %d", result.ReachedCount)
	}
	if result.MaxStep > 5 {
		t.Errorf("max step should not exceed 5 for 6 nodes, got %d", result.MaxStep)
	}
}

func TestSimulateSignalPropagation_StrategicStartPoint(t *testing.T) {
	graph := buildTestPropagationGraph()

	resultFromEnd := SimulateSignalPropagation(graph, 1, 1.0, nil)
	resultFromMiddle := SimulateSignalPropagation(graph, 2, 1.0, nil)

	if resultFromEnd.MaxStep <= resultFromMiddle.MaxStep {
		t.Errorf("starting from end should have larger max step than from middle")
	}
}

func TestBasePropagationDelayConstants(t *testing.T) {
	if BasePropagationDelayMs != 3000.0 {
		t.Errorf("expected base delay 3000ms, got %f", BasePropagationDelayMs)
	}
	if DistanceFactorPerKm != 50.0 {
		t.Errorf("expected distance factor 50ms/km, got %f", DistanceFactorPerKm)
	}
}
