package handlers

import (
	"beacon-system/database"
	"beacon-system/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func GetSensorData(c *gin.Context) {
	beaconIDStr := c.Query("beacon_id")
	limitStr := c.DefaultQuery("limit", "100")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit > 1000 {
		limit = 100
	}

	var data []models.SensorData
	var query string
	var args []interface{}

	if beaconIDStr != "" {
		beaconID, _ := strconv.Atoi(beaconIDStr)
		query = `
			SELECT * FROM sensor_data
			WHERE beacon_id = $1
			ORDER BY timestamp DESC
			LIMIT $2
		`
		args = append(args, beaconID, limit)
	} else {
		query = `
			SELECT * FROM sensor_data
			ORDER BY timestamp DESC
			LIMIT $1
		`
		args = append(args, limit)
	}

	err = database.DB.Select(&data, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func PostSensorData(c *gin.Context) {
	var input models.SensorData
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Timestamp.IsZero() {
		input.Timestamp = time.Now()
	}

	var id int64
	query := `
		INSERT INTO sensor_data (beacon_id, timestamp, visibility, wind_speed, wind_direction, temperature, humidity, terrain_elevation)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err := database.DB.Get(&id, query,
		input.BeaconID, input.Timestamp,
		input.Visibility, input.WindSpeed,
		input.WindDirection, input.Temperature,
		input.Humidity, input.TerrainElevation,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Sensor data recorded"})
}

func GetLatestSensorData(c *gin.Context) {
	var data []models.SensorData
	query := `
		SELECT DISTINCT ON (beacon_id) *
		FROM sensor_data
		ORDER BY beacon_id, timestamp DESC
	`

	err := database.DB.Select(&data, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func GetSignalReception(c *gin.Context) {
	fromIDStr := c.Query("from_id")
	toIDStr := c.Query("to_id")
	limitStr := c.DefaultQuery("limit", "50")

	limit, _ := strconv.Atoi(limitStr)
	if limit > 500 {
		limit = 500
	}

	var receptions []models.SignalReception
	var query string
	var args []interface{}

	if fromIDStr != "" && toIDStr != "" {
		fromID, _ := strconv.Atoi(fromIDStr)
		toID, _ := strconv.Atoi(toIDStr)
		query = `
			SELECT * FROM signal_reception
			WHERE from_beacon_id = $1 AND to_beacon_id = $2
			ORDER BY timestamp DESC
			LIMIT $3
		`
		args = append(args, fromID, toID, limit)
	} else {
		query = `
			SELECT * FROM signal_reception
			ORDER BY timestamp DESC
			LIMIT $1
		`
		args = append(args, limit)
	}

	err := database.DB.Select(&receptions, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, receptions)
}

func PostSignalReception(c *gin.Context) {
	var input models.SignalReception
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Timestamp.IsZero() {
		input.Timestamp = time.Now()
	}

	var id int64
	query := `
		INSERT INTO signal_reception (from_beacon_id, to_beacon_id, timestamp, signal_strength, is_received, interference_level, weather_factor)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err := database.DB.Get(&id, query,
		input.FromBeaconID, input.ToBeaconID, input.Timestamp,
		input.SignalStrength, input.IsReceived,
		input.InterferenceLevel, input.WeatherFactor,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Signal reception recorded"})
}
