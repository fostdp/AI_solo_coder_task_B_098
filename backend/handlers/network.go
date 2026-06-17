package handlers

import (
	"beacon-system/analysis"
	"beacon-system/database"
	"beacon-system/models"
	"beacon-system/mqtt"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var mqttClient *mqtt.Client
var connectivityThreshold float64 = 0.7

func InitHandlers(client *mqtt.Client, threshold float64) {
	mqttClient = client
	connectivityThreshold = threshold
}

func GetNetworkTopology(c *gin.Context) {
	var links []models.NetworkLink
	query := `
		SELECT l.* FROM network_links l
		JOIN network_topology t ON l.topology_id = t.id
		WHERE t.is_active = true
		ORDER BY l.id
	`

	err := database.DB.Select(&links, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, links)
}

func AnalyzeReliability(c *gin.Context) {
	iterationsStr := c.DefaultQuery("iterations", "1000")
	weatherFactorStr := c.DefaultQuery("weather_factor", "1.0")

	iterations, err := strconv.Atoi(iterationsStr)
	if err != nil || iterations <= 0 || iterations > 100000 {
		iterations = 1000
	}

	weatherFactor, err := strconv.ParseFloat(weatherFactorStr, 64)
	if err != nil || weatherFactor < 0 || weatherFactor > 1 {
		weatherFactor = 1.0
	}

	graph, err := buildNetworkGraph()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	config := analysis.MonteCarloConfig{
		Iterations:    iterations,
		WeatherFactor: weatherFactor,
	}

	result := analysis.MonteCarloReliability(graph, config)
	metrics := analysis.CalculateNetworkMetrics(graph, weatherFactor)

	analysisRecord := models.ReliabilityAnalysis{
		TopologyID:           1,
		AnalysisType:         "monte_carlo",
		Timestamp:            time.Now(),
		OverallReliability:   result.SuccessRate,
		ConnectivityIndex:    metrics["connectivity_index"],
		AveragePathLength:    metrics["avg_path_length"],
		NodeCount:            int(metrics["node_count"]),
		LinkCount:            int(metrics["link_count"]),
		MonteCarloIterations: iterations,
		WeatherCondition:     getWeatherCondition(weatherFactor),
	}

	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"confidence_interval": result.ConfidenceInterval,
		"metrics":             metrics,
	})
	analysisRecord.Details = string(detailsJSON)

	saveAnalysisRecord(&analysisRecord)

	c.JSON(http.StatusOK, gin.H{
		"monte_carlo":     result,
		"network_metrics": metrics,
		"analysis_record": analysisRecord,
	})
}

func GetReliabilityHistory(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
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
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

func CheckConnectivity(c *gin.Context) {
	graph, err := buildNetworkGraph()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	isConnected := graph.IsConnected()
	connectivityIdx := graph.ConnectivityIndex()
	avgPathLen := graph.AveragePathLength()

	belowThreshold := connectivityIdx < connectivityThreshold

	alertTriggered := false
	if belowThreshold && mqttClient != nil {
		alert := &models.Alert{
			AlertType:   "connectivity_low",
			Severity:    "high",
			Title:       "网络连通度低于阈值",
			Description: "烽火台通信网络连通度已低于安全阈值",
		}
		alertData, _ := json.Marshal(map[string]float64{
			"connectivity_index": connectivityIdx,
			"threshold":          connectivityThreshold,
		})
		alert.RelatedData = string(alertData)
		saveAlert(alert)
		mqttClient.PublishAlert(alert)
		alertTriggered = true
	}

	c.JSON(http.StatusOK, gin.H{
		"is_connected":        isConnected,
		"connectivity_index":  connectivityIdx,
		"average_path_length": avgPathLen,
		"threshold":           connectivityThreshold,
		"below_threshold":     belowThreshold,
		"alert_triggered":     alertTriggered,
	})
}

func GetCriticalLinks(c *gin.Context) {
	graph, err := buildNetworkGraph()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var criticalLinks []models.NetworkLink
	query := `
		SELECT l.* FROM network_links l
		JOIN network_topology t ON l.topology_id = t.id
		WHERE t.is_active = true AND l.is_critical = true
		ORDER BY l.id
	`

	err = database.DB.Select(&criticalLinks, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"critical_links": criticalLinks,
		"total_links":    len(graph.Edges),
	})
}

func buildNetworkGraph() (*analysis.Graph, error) {
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

func saveAnalysisRecord(record *models.ReliabilityAnalysis) error {
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
	return err
}

func saveAlert(alert *models.Alert) error {
	query := `
		INSERT INTO alerts (
			alert_type, severity, title, description, beacon_id, link_id, related_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	var id int64
	var createdAt time.Time
	err := database.DB.QueryRow(query,
		alert.AlertType, alert.Severity, alert.Title,
		alert.Description, alert.BeaconID, alert.LinkID, alert.RelatedData,
	).Scan(&id, &createdAt)
	if err == nil {
		alert.ID = id
		alert.CreatedAt = createdAt
	}
	return err
}

func GetAlerts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	resolvedStr := c.Query("resolved")

	limit, _ := strconv.Atoi(limitStr)
	if limit > 200 {
		limit = 50
	}

	var alerts []models.Alert
	var query string
	var args []interface{}

	if resolvedStr != "" {
		resolved, _ := strconv.ParseBool(resolvedStr)
		query = `
			SELECT * FROM alerts
			WHERE is_resolved = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		args = append(args, resolved, limit)
	} else {
		query = `
			SELECT * FROM alerts
			ORDER BY created_at DESC
			LIMIT $1
		`
		args = append(args, limit)
	}

	err := database.DB.Select(&alerts, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

func ResolveAlert(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	query := `
		UPDATE alerts
		SET is_resolved = true, resolved_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id, is_resolved, resolved_at
	`

	var alertID int64
	var isResolved bool
	var resolvedAt time.Time

	err = database.DB.QueryRow(query, id).Scan(&alertID, &isResolved, &resolvedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          alertID,
		"is_resolved": isResolved,
		"resolved_at": resolvedAt,
	})
}

func getWeatherCondition(factor float64) string {
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
