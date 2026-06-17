package advanced_analysis

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"beacon-system/analysis"
	"beacon-system/models"
	"beacon-system/modules/eventbus"

	"github.com/jmoiron/sqlx"
)

type AdvancedAnalyzer struct {
	db       *sqlx.DB
	eventBus *eventbus.EventBus
}

func NewAdvancedAnalyzer(db *sqlx.DB, eb *eventbus.EventBus) *AdvancedAnalyzer {
	return &AdvancedAnalyzer{
		db:       db,
		eventBus: eb,
	}
}

func (a *AdvancedAnalyzer) GetDynasties() ([]models.Dynasty, error) {
	var dynasties []models.Dynasty
	err := a.db.Select(&dynasties, `
		SELECT code, name, period, description, color, sort_order
		FROM dynasties ORDER BY sort_order ASC
	`)
	return dynasties, err
}

func (a *AdvancedAnalyzer) GetTopologyByDynasty(dynastyCode string) (*models.NetworkTopology, error) {
	var topo models.NetworkTopology
	err := a.db.Get(&topo, `
		SELECT id, version, name, description, is_active, dynasty_code, created_at
		FROM network_topology
		WHERE dynasty_code = $1
		ORDER BY id LIMIT 1
	`, dynastyCode)
	if err != nil {
		return nil, err
	}
	return &topo, nil
}

func (a *AdvancedAnalyzer) CompareDynasties(dynastyCodes []string) ([]models.DynastyComparison, error) {
	results := []models.DynastyComparison{}

	for _, code := range dynastyCodes {
		comp, err := a.getDynastyMetrics(code)
		if err != nil {
			log.Printf("获取朝代 %s 指标失败: %v", code, err)
			continue
		}
		results = append(results, *comp)
	}

	return results, nil
}

func (a *AdvancedAnalyzer) getDynastyMetrics(dynastyCode string) (*models.DynastyComparison, error) {
	var dynasty models.Dynasty
	err := a.db.Get(&dynasty, `SELECT code, name, period, description, color FROM dynasties WHERE code = $1`, dynastyCode)
	if err != nil {
		return nil, err
	}

	var topoID int
	var topoName string
	err = a.db.Get(&topoID, `SELECT id FROM network_topology WHERE dynasty_code = $1 ORDER BY id LIMIT 1`, dynastyCode)
	if err != nil {
		return nil, err
	}
	a.db.Get(&topoName, `SELECT name FROM network_topology WHERE id = $1`, topoID)

	var nodeCount int
	a.db.Get(&nodeCount, `
		SELECT COUNT(DISTINCT CASE WHEN nl.from_beacon_id = b.id OR nl.to_beacon_id = b.id THEN b.id END)
		FROM beacons b
		LEFT JOIN network_links nl ON nl.topology_id = $1 AND (nl.from_beacon_id = b.id OR nl.to_beacon_id = b.id)
	`, topoID)

	var linkCount int
	a.db.Get(&linkCount, `SELECT COUNT(*) FROM network_links WHERE topology_id = $1`, topoID)

	var avgReliability float64
	a.db.Get(&avgReliability, `SELECT COALESCE(AVG(base_reliability), 0) FROM network_links WHERE topology_id = $1`, topoID)

	graph := a.buildGraphFromTopology(topoID)
	if graph == nil || len(graph.Nodes) == 0 {
		return &models.DynastyComparison{
			DynastyCode:    dynastyCode,
			DynastyName:    dynasty.Name,
			Color:          dynasty.Color,
			NodeCount:      nodeCount,
			LinkCount:      linkCount,
			AvgReliability: avgReliability,
			TopologyID:     topoID,
		}, nil
	}

	connected := graph.IsConnected()
	avgPath := graph.AveragePathLength()
	diameter := graphDiameter(graph)
	density := calculateDensity(graph)

	mcConfig := analysis.MonteCarloConfig{
		Iterations:            500,
		WeatherFactor:         1.0,
		UseImportanceSampling: false,
	}
	mcResult := analysis.MonteCarloReliability(graph, mcConfig)
	reliability := mcResult.SuccessRate

	var connIdx float64
	if connected {
		connIdx = 1.0
	} else {
		connIdx = calculateConnectivityIdx(graph)
	}

	return &models.DynastyComparison{
		DynastyCode:     dynastyCode,
		DynastyName:     dynasty.Name,
		Color:           dynasty.Color,
		NodeCount:       len(graph.Nodes),
		LinkCount:       len(graph.Edges),
		ConnectivityIdx: connIdx,
		AvgPathLength:   avgPath,
		Diameter:        diameter,
		Density:         density,
		Reliability:     reliability,
		AvgReliability:  avgReliability,
		TopologyID:      topoID,
	}, nil
}

func (a *AdvancedAnalyzer) GetModernBaseStations() ([]models.ModernBaseStation, error) {
	var stations []models.ModernBaseStation
	err := a.db.Select(&stations, `
		SELECT s.id, s.name, s.station_type, t.type_name,
		       ST_X(s.location::geometry) as lon, ST_Y(s.location::geometry) as lat,
		       s.height, s.coverage_radius_km, s.capacity_mbps, s.latency_ms,
		       s.frequency_ghz, s.power_kw, s.is_standard_compliant, s.standard_version,
		       s.status, s.created_at
		FROM modern_base_stations s
		LEFT JOIN base_station_types t ON t.type_code = s.station_type
		WHERE s.status = 'active'
		ORDER BY s.id
	`)
	return stations, err
}

func (a *AdvancedAnalyzer) GetBaseStationTypes() ([]models.BaseStationType, error) {
	var types []models.BaseStationType
	err := a.db.Select(&types, `
		SELECT type_code, type_name, standard_version, description,
		       min_coverage_radius_km, max_coverage_radius_km, standard_coverage_radius_km,
		       min_capacity_mbps, max_capacity_mbps, standard_capacity_mbps,
		       min_latency_ms, max_latency_ms, standard_latency_ms,
		       frequency_band, typical_height_m, typical_power_kw,
		       technology_generation, sort_order
		FROM base_station_types
		ORDER BY sort_order ASC
	`)
	return types, err
}

func (a *AdvancedAnalyzer) CrossEraComparison(topologyID int) (*models.CrossEraComparison, error) {
	beaconStats := a.getBeaconNetworkStats(topologyID)
	modernStats := a.getModernNetworkStats()

	comparison := map[string]interface{}{
		"node_count_ratio": float64(modernStats["node_count"].(int)) / float64(max(1, beaconStats["node_count"].(int))),
		"coverage_ratio":   modernStats["total_coverage_km2"].(float64) / maxF(1.0, beaconStats["total_coverage_km2"].(float64)),
		"capacity_ratio":   modernStats["total_capacity_mbps"].(float64) / maxF(1.0, beaconStats["total_capacity_mbps"].(float64)),
		"latency_ratio_ms": beaconStats["avg_latency_ms"].(float64) / maxF(1.0, modernStats["avg_latency_ms"].(float64)),
		"power_ratio_kw":   modernStats["total_power_kw"].(float64) / maxF(1.0, beaconStats["total_power_kw"].(float64)),
		"era_gap_years":    2200,
		"tech_paradigm":    "visual-signal-vs-electromagnetic",
	}

	return &models.CrossEraComparison{
		BeaconNetwork: beaconStats,
		ModernNetwork: modernStats,
		Comparison:    comparison,
	}, nil
}

func (a *AdvancedAnalyzer) getBeaconNetworkStats(topologyID int) map[string]interface{} {
	graph := a.buildGraphFromTopology(topologyID)
	nodeCount := len(graph.Nodes)
	linkCount := len(graph.Edges)

	totalCoverage := 0.0
	for _, node := range graph.Nodes {
		horizonDist := analysis.ITURRefractedHorizonDistance(node.Height+node.Elevation) * 1000
		totalCoverage += 3.14159 * horizonDist * horizonDist / 1e6
	}

	avgLatency := 0.0
	if nodeCount > 0 {
		totalDist := 0.0
		count := 0
		for _, edge := range graph.Edges {
			if from, ok := graph.Nodes[edge.From]; ok {
				if to, ok2 := graph.Nodes[edge.To]; ok2 {
					dist := analysis.HaversineDistance(from.Lat, from.Lon, to.Lat, to.Lon)
					totalDist += dist
					count++
				}
			}
		}
		if count > 0 {
			avgDist := totalDist / float64(count)
			avgLatency = avgDist * 1000 / 343.0
		}
	}

	return map[string]interface{}{
		"type":                "beacon",
		"node_count":          nodeCount,
		"link_count":          linkCount,
		"total_coverage_km2":  totalCoverage,
		"total_capacity_mbps": float64(nodeCount) * 0.001,
		"avg_latency_ms":      avgLatency,
		"total_power_kw":      float64(nodeCount) * 0.0,
		"transmission_medium": "visible_light_smoke",
		"typical_speed_kmh":   0,
		"era":                 "ancient",
	}
}

func (a *AdvancedAnalyzer) getModernNetworkStats() map[string]interface{} {
	stations, _ := a.GetModernBaseStations()
	nodeCount := len(stations)

	totalCoverage := 0.0
	totalCapacity := 0.0
	totalPower := 0.0
	avgLatency := 0.0

	for _, s := range stations {
		totalCoverage += 3.14159 * s.CoverageRadiusKm * s.CoverageRadiusKm
		totalCapacity += s.CapacityMbps
		totalPower += s.PowerKw
		avgLatency += s.LatencyMs
	}
	if nodeCount > 0 {
		avgLatency /= float64(nodeCount)
	}

	return map[string]interface{}{
		"type":                "modern",
		"node_count":          nodeCount,
		"total_coverage_km2":  totalCoverage,
		"total_capacity_mbps": totalCapacity,
		"avg_latency_ms":      avgLatency,
		"total_power_kw":      totalPower,
		"transmission_medium": "radio_wave_optical",
		"typical_speed_kmh":   1080000000,
		"era":                 "modern",
	}
}

func (a *AdvancedAnalyzer) AnalyzeResilience(topologyID int, attackType string, steps int, iterations int) (*models.ResilienceResult, error) {
	graph := a.buildGraphFromTopology(topologyID)
	if graph == nil || len(graph.Nodes) == 0 {
		return nil, errors.New("topology not found or empty")
	}

	strategy := analysis.AttackStrategy{
		AttackType:      attackType,
		Steps:           steps,
		Iterations:      iterations,
		CascadeAlpha:    0.5,
		CascadeMaxDepth: 5,
	}

	var result *analysis.ResilienceResult

	isLinkAttack := attackType == "link_random" || attackType == "link_critical" ||
		attackType == "link_betweenness" || attackType == "link_reliability"

	if isLinkAttack || attackType == "cascading" || attackType == "coordinated" {
		result = analysis.AnalyzeResilienceWithStrategy(graph, strategy)
	} else {
		result = analysis.AnalyzeResilience(graph, attackType, steps, iterations)
	}

	resp := &models.ResilienceResult{
		AttackType:        result.AttackType,
		RobustnessScore:   result.RobustnessScore,
		CriticalThreshold: result.CriticalThreshold,
		TotalNodes:        result.TotalNodes,
		Iterations:        result.Iterations,
	}
	resp.CurvePoints = make([]models.ResilienceCurvePoint, len(result.CurvePoints))
	for i, p := range result.CurvePoints {
		resp.CurvePoints[i] = models.ResilienceCurvePoint{
			RemovalRatio:      p.RemovalRatio,
			ConnectivityIndex: p.ConnectivityIndex,
			GiantComponentPct: p.GiantComponentPct,
		}
	}

	go a.saveResilienceResult(topologyID, "node", attackType, result)

	return resp, nil
}

func (a *AdvancedAnalyzer) saveResilienceResult(topologyID int, analysisType string, attackType string, result *analysis.ResilienceResult) {
	for _, p := range result.CurvePoints {
		details, _ := json.Marshal(map[string]interface{}{
			"giant_component_pct": p.GiantComponentPct,
		})
		_, err := a.db.Exec(`
			INSERT INTO resilience_analysis
			(topology_id, analysis_type, attack_type, node_removal_ratio,
			 connectivity_index, giant_component_size, robustness_score, iterations, details)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, topologyID, analysisType, attackType, p.RemovalRatio,
			p.ConnectivityIndex, int(p.GiantComponentPct*float64(result.TotalNodes)),
			result.RobustnessScore, result.Iterations, string(details))
		if err != nil {
			log.Printf("保存抗毁性分析结果失败: %v", err)
		}
	}
}

func (a *AdvancedAnalyzer) IgniteBeacon(beaconID int, topologyID int, sessionID string, weatherFactor float64, userNote string) (*models.IgnitionResult, error) {
	graph := a.buildGraphFromTopology(topologyID)
	if graph == nil || len(graph.Nodes) == 0 {
		return nil, errors.New("topology not found or empty")
	}

	if _, ok := graph.Nodes[beaconID]; !ok {
		return nil, errors.New("beacon not found in topology")
	}

	beaconNames := make(map[int]string)
	for id, node := range graph.Nodes {
		beaconNames[id] = node.Name
	}

	propResult := analysis.SimulateSignalPropagation(graph, beaconID, weatherFactor, beaconNames)

	result := &models.IgnitionResult{
		StartBeaconID: beaconID,
		ReachedCount:  propResult.ReachedCount,
		TotalTimeMs:   propResult.TotalTimeMs,
		TopologyID:    topologyID,
		WeatherFactor: weatherFactor,
	}
	result.Path = make([]models.IgnitionPropagationStep, len(propResult.Path))
	for i, s := range propResult.Path {
		result.Path[i] = models.IgnitionPropagationStep{
			BeaconID:   s.BeaconID,
			BeaconName: s.BeaconName,
			Step:       s.Step,
			DelayMs:    s.DelayMs,
		}
	}

	go a.saveIgnitionRecord(result, sessionID, userNote)

	if a.eventBus != nil {
		a.eventBus.Publish(eventbus.Event{
			Type: eventbus.EventBeaconIgnited,
			Payload: map[string]interface{}{
				"beacon_id":     beaconID,
				"topology_id":   topologyID,
				"session_id":    sessionID,
				"reached_count": propResult.ReachedCount,
				"total_time_ms": propResult.TotalTimeMs,
			},
		})
	}

	return result, nil
}

func (a *AdvancedAnalyzer) saveIgnitionRecord(result *models.IgnitionResult, sessionID string, userNote string) {
	pathJSON, _ := json.Marshal(result.Path)

	var id int64
	err := a.db.QueryRow(`
		INSERT INTO user_beacon_ignitions
		(session_id, beacon_id, topology_id, user_note, weather_factor,
		 reached_count, total_propagation_time_ms, propagation_path)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, sessionID, result.StartBeaconID, result.TopologyID, userNote, result.WeatherFactor,
		result.ReachedCount, result.TotalTimeMs, string(pathJSON)).Scan(&id)

	if err != nil {
		log.Printf("保存点燃记录失败: %v", err)
		return
	}
	result.IgnitionID = id
}

func (a *AdvancedAnalyzer) buildGraphFromTopology(topologyID int) *analysis.Graph {
	graph := analysis.NewGraph()

	var beacons []models.Beacon
	err := a.db.Select(&beacons, `
		SELECT b.id, b.name, b.code, b.dynasty,
		       ST_X(b.location::geometry) as lon, ST_Y(b.location::geometry) as lat,
		       b.elevation, b.height, b.description, b.status, b.created_at
		FROM beacons b
		INNER JOIN network_links nl ON nl.topology_id = $1 AND (nl.from_beacon_id = b.id OR nl.to_beacon_id = b.id)
		WHERE b.status = 'active'
		GROUP BY b.id
	`, topologyID)
	if err != nil {
		log.Printf("查询烽火台失败: %v", err)
		return graph
	}

	for _, b := range beacons {
		beacon := b
		graph.AddNode(&beacon)
	}

	var links []models.NetworkLink
	err = a.db.Select(&links, `
		SELECT id, topology_id, from_beacon_id, to_beacon_id,
		       link_type, base_reliability, is_bidirectional, is_critical, created_at
		FROM network_links WHERE topology_id = $1
	`, topologyID)
	if err != nil {
		log.Printf("查询链路失败: %v", err)
		return graph
	}

	for _, l := range links {
		graph.AddEdge(&l)
	}

	graph.BuildAdjacencyList()
	return graph
}

func graphDiameter(graph *analysis.Graph) int {
	maxDist := 0
	for id := range graph.Nodes {
		levels := graph.BFSLevels(id)
		for _, d := range levels {
			if d > maxDist {
				maxDist = d
			}
		}
	}
	return maxDist
}

func calculateDensity(graph *analysis.Graph) float64 {
	n := len(graph.Nodes)
	if n <= 1 {
		return 0
	}
	e := len(graph.Edges)
	maxEdges := n * (n - 1) / 2
	if maxEdges == 0 {
		return 0
	}
	return float64(e) / float64(maxEdges)
}

func calculateConnectivityIdx(graph *analysis.Graph) float64 {
	if len(graph.Nodes) <= 1 {
		return 1.0
	}
	visited := make(map[int]bool)
	componentCount := 0
	for id := range graph.Nodes {
		if !visited[id] {
			componentCount++
			dfsVisit(graph, id, visited)
		}
	}
	return 1.0 - float64(componentCount-1)/float64(len(graph.Nodes)-1)
}

func dfsVisit(graph *analysis.Graph, start int, visited map[int]bool) {
	stack := []int{start}
	visited[start] = true
	for len(stack) > 0 {
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, n := range graph.Adj[v] {
			if !visited[n] {
				visited[n] = true
				stack = append(stack, n)
			}
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

var _ = sql.ErrNoRows
