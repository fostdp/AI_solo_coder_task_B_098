package handlers

import (
	"beacon-system/database"
	"beacon-system/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetBeacons(c *gin.Context) {
	var beacons []struct {
		models.Beacon
		Lon float64 `db:"lon" json:"lon"`
		Lat float64 `db:"lat" json:"lat"`
	}

	query := `
		SELECT id, name, code, dynasty,
			ST_X(location::geometry) as lon,
			ST_Y(location::geometry) as lat,
			elevation, height, description, status,
			coordinate_source, coordinate_precision_m, gps_verified,
			archaeological_site_id, survey_year,
			created_at, updated_at
		FROM beacons
		WHERE status = 'active'
		ORDER BY id
	`

	err := database.DB.Select(&beacons, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, beacons)
}

func GetBeacon(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid beacon ID"})
		return
	}

	var beacon struct {
		models.Beacon
		Lon float64 `db:"lon" json:"lon"`
		Lat float64 `db:"lat" json:"lat"`
	}

	query := `
		SELECT id, name, code, dynasty,
			ST_X(location::geometry) as lon,
			ST_Y(location::geometry) as lat,
			elevation, height, description, status,
			coordinate_source, coordinate_precision_m, gps_verified,
			archaeological_site_id, survey_year,
			created_at, updated_at
		FROM beacons
		WHERE id = $1
	`

	err = database.DB.Get(&beacon, query, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Beacon not found"})
		return
	}

	c.JSON(http.StatusOK, beacon)
}

func CreateBeacon(c *gin.Context) {
	var input struct {
		Name                 string  `json:"name" binding:"required"`
		Code                 string  `json:"code" binding:"required"`
		Dynasty              string  `json:"dynasty"`
		Lon                  float64 `json:"lon" binding:"required"`
		Lat                  float64 `json:"lat" binding:"required"`
		Elevation            float64 `json:"elevation" binding:"required"`
		Height               float64 `json:"height"`
		Description          string  `json:"description"`
		CoordinateSource     string  `json:"coordinate_source"`
		CoordinatePrecisionM float64 `json:"coordinate_precision_m"`
		GpsVerified          bool    `json:"gps_verified"`
		ArchaeologicalSiteID string  `json:"archaeological_site_id"`
		SurveyYear           int     `json:"survey_year"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Height == 0 {
		input.Height = 10.0
	}
	if input.Dynasty == "" {
		input.Dynasty = "汉代"
	}
	if input.CoordinateSource == "" {
		input.CoordinateSource = "archaeological_survey"
	}
	if input.CoordinatePrecisionM == 0 {
		input.CoordinatePrecisionM = 5.0
	}

	var id int
	query := `
		INSERT INTO beacons (name, code, dynasty, location, elevation, height, description, status,
			coordinate_source, coordinate_precision_m, gps_verified, archaeological_site_id, survey_year)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326), $6, $7, $8, 'active',
			$9, $10, $11, $12, $13)
		RETURNING id
	`

	err := database.DB.Get(&id, query,
		input.Name, input.Code, input.Dynasty,
		input.Lon, input.Lat, input.Elevation,
		input.Height, input.Description,
		input.CoordinateSource, input.CoordinatePrecisionM,
		input.GpsVerified, input.ArchaeologicalSiteID, input.SurveyYear,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Beacon created successfully"})
}
