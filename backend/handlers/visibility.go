package handlers

import (
	vis "beacon-system/modules/visibility_analyzer"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

var visibilityAnalyzer *vis.VisibilityAnalyzer

func InitVisibilityAnalyzer(v *vis.VisibilityAnalyzer) {
	visibilityAnalyzer = v
}

func GetVisibilityMatrix(c *gin.Context) {
	if visibilityAnalyzer == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Visibility analyzer not initialized"})
		return
	}

	results, err := visibilityAnalyzer.GetAllResults()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

func CalculateVisibility(c *gin.Context) {
	fromIDStr := c.Query("from_id")
	toIDStr := c.Query("to_id")

	if fromIDStr == "" || toIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from_id and to_id are required"})
		return
	}

	fromID, _ := strconv.Atoi(fromIDStr)
	toID, _ := strconv.Atoi(toIDStr)

	if visibilityAnalyzer == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Visibility analyzer not initialized"})
		return
	}

	result, err := visibilityAnalyzer.Calculate(fromID, toID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func CalculateVisibilityMatrix(c *gin.Context) {
	if visibilityAnalyzer == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Visibility analyzer not initialized"})
		return
	}

	results, err := visibilityAnalyzer.CalculateMatrix()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_pairs": len(results),
		"results":     results,
	})
}

func GetViewShed(c *gin.Context) {
	beaconIDStr := c.Param("id")
	beaconID, err := strconv.Atoi(beaconIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid beacon ID"})
		return
	}

	azimuthStartStr := c.DefaultQuery("azimuth_start", "0")
	azimuthEndStr := c.DefaultQuery("azimuth_end", "360")
	maxDistStr := c.DefaultQuery("max_distance", "20")

	azimuthStart, _ := strconv.ParseFloat(azimuthStartStr, 64)
	azimuthEnd, _ := strconv.ParseFloat(azimuthEndStr, 64)
	maxDistance, _ := strconv.ParseFloat(maxDistStr, 64)

	if visibilityAnalyzer == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Visibility analyzer not initialized"})
		return
	}

	sectorPoints, err := visibilityAnalyzer.GetViewShed(beaconID, azimuthStart, azimuthEnd, maxDistance)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"beacon_id":     beaconID,
		"azimuth_start": azimuthStart,
		"azimuth_end":   azimuthEnd,
		"max_distance":  maxDistance,
		"polygon":       sectorPoints,
	})
}
