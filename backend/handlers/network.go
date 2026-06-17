package handlers

import (
	alarm "beacon-system/modules/alarm_mqtt"
	netrel "beacon-system/modules/network_reliability_analyzer"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

var networkAnalyzer *netrel.NetworkReliabilityAnalyzer
var alarmModule *alarm.AlarmMQTT

func InitNetworkModules(n *netrel.NetworkReliabilityAnalyzer, a *alarm.AlarmMQTT) {
	networkAnalyzer = n
	alarmModule = a
}

func InitHandlers() {}

func GetNetworkTopology(c *gin.Context) {
	if networkAnalyzer == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Network analyzer not initialized"})
		return
	}

	links, err := networkAnalyzer.GetTopology()
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
	if err != nil {
		iterations = 1000
	}

	weatherFactor, err := strconv.ParseFloat(weatherFactorStr, 64)
	if err != nil {
		weatherFactor = 1.0
	}

	if networkAnalyzer == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Network analyzer not initialized"})
		return
	}

	result, metrics, err := networkAnalyzer.RunMonteCarlo(iterations, weatherFactor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"monte_carlo":     result,
		"network_metrics": metrics,
	})
}

func GetReliabilityHistory(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)

	if networkAnalyzer == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Network analyzer not initialized"})
		return
	}

	history, err := networkAnalyzer.GetHistory(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

func CheckConnectivity(c *gin.Context) {
	if networkAnalyzer == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Network analyzer not initialized"})
		return
	}

	isConnected, connectivityIdx, avgPathLen, err := networkAnalyzer.CheckConnectivity()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	threshold := 0.7
	if networkAnalyzer != nil {
		threshold = 0.7
	}
	belowThreshold := connectivityIdx < threshold

	alertTriggered := false

	c.JSON(http.StatusOK, gin.H{
		"is_connected":        isConnected,
		"connectivity_index":  connectivityIdx,
		"average_path_length": avgPathLen,
		"threshold":           threshold,
		"below_threshold":     belowThreshold,
		"alert_triggered":     alertTriggered,
	})
}

func GetCriticalLinks(c *gin.Context) {
	if networkAnalyzer == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Network analyzer not initialized"})
		return
	}

	criticalLinks, totalLinks, err := networkAnalyzer.GetCriticalLinks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"critical_links": criticalLinks,
		"total_links":    totalLinks,
	})
}

func GetAlerts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	resolvedStr := c.Query("resolved")

	limit, _ := strconv.Atoi(limitStr)
	var resolved *bool
	if resolvedStr != "" {
		r, err := strconv.ParseBool(resolvedStr)
		if err == nil {
			resolved = &r
		}
	}

	if alarmModule == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Alarm module not initialized"})
		return
	}

	alerts, err := alarmModule.GetAlerts(limit, resolved)
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

	if alarmModule == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Alarm module not initialized"})
		return
	}

	if err := alarmModule.ResolveAlert(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "is_resolved": true})
}
