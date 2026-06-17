package network_reliability_analyzer

import (
	"beacon-system/analysis"
	"beacon-system/config"
	"beacon-system/database"
	"beacon-system/models"
	"beacon-system/modules/eventbus"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type NetworkReliabilityAnalyzer struct {
	cfg *config.Config
	bus *eventbus.EventBus
}

func New(cfg *config.Config) *NetworkReliabilityAnalyzer {
	return &NetworkReliabilityAnalyzer{
		cfg: cfg,
		bus: eventbus.Get(),
	}
}

func (n *NetworkReliabilityAnalyzer) BuildGraph() (*analysis.Graph, error) {
	graph := analysis.NewGraph()

	var beacons []struct {
		models.Beacon
		Lon float64 `db:"lon"`
		Lat float64 `db:"lat"`
	}
	beaconQuery := `
		SELECT id, name, code, dynasty,
			ST_X(location::geometry) as lon,
			ST_Y(location::geometry) as lat,
			elevation, height, description, status, created_at, updated_at
		FROM beacons WHERE status = 'active'
	`
	if err := database.DB.Select(&beacons, beaconQuery); err != nil {
		return nil, err
	}

	for _, b := range beacons {
		beacon := b.Beacon
		beacon.Lon = b.Lon
		beacon.Lat = b.Lat
		graph.AddNode(&beacon)
	}

	var links []models.NetworkLink
	linkQuery := `
		SELECT l.* FROM network_links l
		JOIN network_topology t ON l.topology_id = t.id
		WHERE t.is_active = true
	`
	if err := database.DB.Select(&links, linkQuery); err != nil {
		return nil, err
	}

	for _, link := range links {
		graph.AddEdge(&link)
	}

	return graph, nil
}

func (n *NetworkReliabilityAnalyzer) RunMonteCarlo(iterations int, weatherFactor float64) (
	*models.MonteCarloResult, map[string]float64, error) {

	if iterations <= 0 {
		iterations = n.cfg.Params.Reliability.DefaultMCIterations
	}
	if iterations > n.cfg.Params.Reliability.MaxMCIterations {
		iterations = n.cfg.Params.Reliability.MaxMCIterations
	}

	if weatherFactor < 0 {
		weatherFactor = 0
	}
	if weatherFactor > 1 {
		weatherFactor = 1
	}

	graph, err := n.BuildGraph()
	if err != nil {
		return nil, nil, fmt.Errorf("build graph failed: %w", err)
	}

	config := analysis.MonteCarloConfig{
		Iterations:            iterations,
		WeatherFactor:         weatherFactor,
		UseImportanceSampling: len(graph.Edges) >= n.cfg.Params.Reliability.ISEdgeThreshold,
	}

	result := analysis.MonteCarloReliability(graph, config)
	metrics := analysis.CalculateNetworkMetrics(graph, weatherFactor)

	analysisRecord := n.saveAnalysisRecord(&result, metrics, iterations, weatherFactor)

	n.bus.Publish(eventbus.Event{
		Type: eventbus.EventReliabilityAnalyzed,
		Payload: eventbus.ReliabilityPayload{
			Result:          result,
			Metrics:         metrics,
			ConnectivityIdx: metrics["connectivity_index"],
		},
		Time: time.Now().UnixNano(),
	})

	log.Printf("[Reliability] MC done: iterations=%d success=%.4f connectivity=%.4f paths=%.2f",
		iterations, result.SuccessRate, metrics["connectivity_index"], metrics["avg_path_length"])

	_ = analysisRecord

	return &result, metrics, nil
}

func (n *NetworkReliabilityAnalyzer) saveAnalysisRecord(
	result *models.MonteCarloResult, metrics map[string]float64,
	iterations int, weatherFactor float64) *models.ReliabilityAnalysis {

	record := &models.ReliabilityAnalysis{
		TopologyID:           1,
		AnalysisType:         "monte_carlo",
		Timestamp:            time.Now(),
		OverallReliability:   result.SuccessRate,
		ConnectivityIndex:    metrics["connectivity_index"],
		AveragePathLength:    metrics["avg_path_length"],
		NodeCount:            int(metrics["node_count"]),
		LinkCount:            int(metrics["link_count"]),
		MonteCarloIterations: iterations,
		WeatherCondition:     n.getWeatherCondition(weatherFactor),
	}

	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"confidence_interval": result.ConfidenceInterval,
		"metrics":             metrics,
	})
	record.Details = string(detailsJSON)

	query := `
		INSERT INTO reliability_analysis (
			topology_id, analysis_type, timestamp, overall_reliability,
			connectivity_index, average_path_length, node_count, link_count,
			monte_carlo_iterations, weather_condition, details
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	var id int
	err := database.DB.Get(&id, query,
		record.TopologyID, record.AnalysisType, record.Timestamp,
		record.OverallReliability, record.ConnectivityIndex,
		record.AveragePathLength, record.NodeCount, record.LinkCount,
		record.MonteCarloIterations, record.WeatherCondition, record.Details,
	)
	if err == nil {
		record.ID = id
	}

	return record
}

func (n *NetworkReliabilityAnalyzer) CheckConnectivity() (
	isConnected bool, connectivityIdx float64, avgPathLen float64, err error) {

	graph, err := n.BuildGraph()
	if err != nil {
		return false, 0, 0, err
	}

	isConnected = graph.IsConnected()
	connectivityIdx = graph.ConnectivityIndex()
	avgPathLen = graph.AveragePathLength()

	n.bus.Publish(eventbus.Event{
		Type: eventbus.EventConnectivityCheck,
		Payload: eventbus.ReliabilityPayload{
			ConnectivityIdx: connectivityIdx,
			Metrics: map[string]float64{
				"is_connected":       boolToFloat(isConnected),
				"connectivity_index": connectivityIdx,
				"avg_path_length":    avgPathLen,
			},
		},
		Time: time.Now().UnixNano(),
	})

	log.Printf("[Reliability] Connectivity check: connected=%v index=%.4f avg_path=%.2f",
		isConnected, connectivityIdx, avgPathLen)

	return
}

func (n *NetworkReliabilityAnalyzer) GetTopology() ([]models.NetworkLink, error) {
	var links []models.NetworkLink
	query := `
		SELECT l.* FROM network_links l
		JOIN network_topology t ON l.topology_id = t.id
		WHERE t.is_active = true
		ORDER BY l.id
	`

	err := database.DB.Select(&links, query)
	return links, err
}

func (n *NetworkReliabilityAnalyzer) GetCriticalLinks() ([]models.NetworkLink, int, error) {
	graph, err := n.BuildGraph()
	if err != nil {
		return nil, 0, err
	}

	var criticalLinks []models.NetworkLink
	query := `
		SELECT l.* FROM network_links l
		JOIN network_topology t ON l.topology_id = t.id
		WHERE t.is_active = true AND l.is_critical = true
		ORDER BY l.id
	`

	err = database.DB.Select(&criticalLinks, query)
	return criticalLinks, len(graph.Edges), err
}

func (n *NetworkReliabilityAnalyzer) GetHistory(limit int) ([]models.ReliabilityAnalysis, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var history []models.ReliabilityAnalysis
	query := `
		SELECT * FROM reliability_analysis
		ORDER BY timestamp DESC
		LIMIT $1
	`

	err := database.DB.Select(&history, query, limit)
	return history, err
}

func (n *NetworkReliabilityAnalyzer) getWeatherCondition(factor float64) string {
	if factor >= 0.9 {
		return "clear"
	} else if factor >= 0.7 {
		return "light_haze"
	} else if factor >= 0.5 {
		return "foggy"
	} else if factor >= 0.3 {
		return "heavy_fog"
	}
	return "storm"
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func (n *NetworkReliabilityAnalyzer) Start() {
	log.Println("[Reliability] Analyzer module started")
}
