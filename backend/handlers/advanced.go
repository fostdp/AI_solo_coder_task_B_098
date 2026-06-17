package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"beacon-system/modules/advanced_analysis"
)

type AdvancedHandler struct {
	analyzer *advanced_analysis.AdvancedAnalyzer
}

func NewAdvancedHandler(analyzer *advanced_analysis.AdvancedAnalyzer) *AdvancedHandler {
	return &AdvancedHandler{analyzer: analyzer}
}

func (h *AdvancedHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/dynasties", h.GetDynasties)
	router.GET("/dynasty-comparison", h.CompareDynasties)
	router.GET("/modern-stations", h.GetModernStations)
	router.GET("/cross-era-comparison", h.CrossEraComparison)
	router.POST("/resilience", h.AnalyzeResilience)
	router.POST("/ignite", h.IgniteBeacon)
	router.GET("/topology/:dynasty", h.GetTopologyByDynasty)
}

func (h *AdvancedHandler) GetDynasties(c *gin.Context) {
	dynasties, err := h.analyzer.GetDynasties()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dynasties": dynasties})
}

func (h *AdvancedHandler) CompareDynasties(c *gin.Context) {
	dynastyParam := c.Query("dynasties")
	var dynasties []string
	if dynastyParam != "" {
		dynasties = strings.Split(dynastyParam, ",")
	} else {
		dynasties = []string{"qin", "han", "ming"}
	}

	results, err := h.analyzer.CompareDynasties(dynasties)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"comparisons": results,
		"dynasties":   dynasties,
	})
}

func (h *AdvancedHandler) GetModernStations(c *gin.Context) {
	stations, err := h.analyzer.GetModernBaseStations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"stations": stations})
}

func (h *AdvancedHandler) CrossEraComparison(c *gin.Context) {
	topologyID, _ := strconv.Atoi(c.DefaultQuery("topology_id", "1"))

	result, err := h.analyzer.CrossEraComparison(topologyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *AdvancedHandler) AnalyzeResilience(c *gin.Context) {
	var req struct {
		TopologyID int    `json:"topology_id" binding:"required"`
		AttackType string `json:"attack_type"`
		Steps      int    `json:"steps"`
		Iterations int    `json:"iterations"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.AttackType == "" {
		req.AttackType = "random"
	}
	if req.Steps <= 0 {
		req.Steps = 10
	}
	if req.Iterations <= 0 {
		req.Iterations = 1
	}

	result, err := h.analyzer.AnalyzeResilience(req.TopologyID, req.AttackType, req.Steps, req.Iterations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AdvancedHandler) IgniteBeacon(c *gin.Context) {
	var req struct {
		BeaconID      int     `json:"beacon_id" binding:"required"`
		TopologyID    int     `json:"topology_id"`
		SessionID     string  `json:"session_id"`
		WeatherFactor float64 `json:"weather_factor"`
		UserNote      string  `json:"user_note"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TopologyID <= 0 {
		req.TopologyID = 1
	}
	if req.WeatherFactor <= 0 {
		req.WeatherFactor = 1.0
	}
	if req.WeatherFactor > 1.0 {
		req.WeatherFactor = 1.0
	}

	result, err := h.analyzer.IgniteBeacon(req.BeaconID, req.TopologyID, req.SessionID, req.WeatherFactor, req.UserNote)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AdvancedHandler) GetTopologyByDynasty(c *gin.Context) {
	dynastyCode := c.Param("dynasty")

	topo, err := h.analyzer.GetTopologyByDynasty(dynastyCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "topology not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"topology": topo})
}
