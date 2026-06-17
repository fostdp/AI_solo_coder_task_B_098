package analysis

import (
	"container/list"
)

type PropagationStep struct {
	BeaconID   int
	BeaconName string
	Step       int
	DelayMs    float64
	FromBeacon int
}

type PropagationResult struct {
	StartBeaconID int
	ReachedCount  int
	TotalTimeMs   float64
	Path          []PropagationStep
	Steps         [][]int
	MaxStep       int
}

const (
	BasePropagationDelayMs = 3000.0
	DistanceFactorPerKm    = 50.0
)

func SimulateSignalPropagation(graph *Graph, startID int, weatherFactor float64, beaconNames map[int]string) *PropagationResult {
	if _, ok := graph.Nodes[startID]; !ok {
		return &PropagationResult{
			StartBeaconID: startID,
			ReachedCount:  0,
			TotalTimeMs:   0,
			Path:          []PropagationStep{},
			Steps:         [][]int{},
			MaxStep:       0,
		}
	}

	steps := make(map[int]int)
	delays := make(map[int]float64)
	fromBeacon := make(map[int]int)
	for id := range graph.Nodes {
		steps[id] = -1
		delays[id] = 0
		fromBeacon[id] = -1
	}

	steps[startID] = 0
	delays[startID] = 0

	queue := list.New()
	queue.PushBack(startID)

	maxStep := 0
	order := []int{startID}

	for queue.Len() > 0 {
		front := queue.Front()
		queue.Remove(front)
		current := front.Value.(int)
		currentStep := steps[current]

		for _, neighbor := range graph.Adj[current] {
			if steps[neighbor] == -1 || steps[neighbor] > currentStep+1 {
				linkReliability := getLinkReliability(graph, current, neighbor)
				propagationProbability := linkReliability * weatherFactor

				if propagationProbability >= 0.3 || steps[neighbor] == -1 {
					steps[neighbor] = currentStep + 1

					distance := getDistanceBetween(graph, current, neighbor)
					delay := BasePropagationDelayMs + distance*DistanceFactorPerKm
					if weatherFactor > 0 && weatherFactor < 1 {
						delay *= 1.0 / weatherFactor
					}
					delays[neighbor] = delays[current] + delay
					fromBeacon[neighbor] = current

					if steps[neighbor] > maxStep {
						maxStep = steps[neighbor]
					}
					order = append(order, neighbor)
					queue.PushBack(neighbor)
				}
			}
		}
	}

	reached := 0
	totalTime := 0.0
	path := []PropagationStep{}
	stepGroups := make([][]int, maxStep+1)

	for _, id := range order {
		if steps[id] >= 0 {
			reached++
			if delays[id] > totalTime {
				totalTime = delays[id]
			}

			name := ""
			if beaconNames != nil {
				if n, ok := beaconNames[id]; ok {
					name = n
				}
			}

			path = append(path, PropagationStep{
				BeaconID:   id,
				BeaconName: name,
				Step:       steps[id],
				DelayMs:    delays[id],
				FromBeacon: fromBeacon[id],
			})

			if steps[id] <= maxStep {
				stepGroups[steps[id]] = append(stepGroups[steps[id]], id)
			}
		}
	}

	return &PropagationResult{
		StartBeaconID: startID,
		ReachedCount:  reached,
		TotalTimeMs:   totalTime,
		Path:          path,
		Steps:         stepGroups,
		MaxStep:       maxStep,
	}
}

func getLinkReliability(graph *Graph, from, to int) float64 {
	for _, edge := range graph.Edges {
		if (edge.From == from && edge.To == to) || (edge.IsBidirectional && edge.From == to && edge.To == from) {
			return edge.BaseReliability
		}
	}
	return 0.5
}

func getDistanceBetween(graph *Graph, from, to int) float64 {
	a := graph.Nodes[from]
	b := graph.Nodes[to]
	if a == nil || b == nil {
		return 50.0
	}
	return HaversineDistance(a.Lat, a.Lon, b.Lat, b.Lon)
}
