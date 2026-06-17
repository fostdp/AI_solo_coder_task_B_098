package handlers

import (
	"beacon-system/analysis"
	"beacon-system/database"
	"beacon-system/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func GetVisibilityMatrix(c *gin.Context) {
	var results []models.VisibilityAnalysis
	query := `
		SELECT * FROM visibility_analysis
		ORDER BY from_beacon_id, to_beacon_id
	`

	err := database.DB.Select(&results, query)
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

	fromBeacon, err := getBeaconByID(fromID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "From beacon not found"})
		return
	}

	toBeacon, err := getBeaconByID(toID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "To beacon not found"})
		return
	}

	var demPoints []models.DEMPoint
	query := `
		WITH line AS (
			SELECT ST_MakeLine(
				ST_SetSRID(ST_MakePoint($1, $2), 4326),
				ST_SetSRID(ST_MakePoint($3, $4), 4326)
			)::geography as geom
		)
		SELECT d.id, d.lon, d.lat, d.elevation, d.resolution
		FROM dem_data d, line l
		WHERE ST_DWithin(ST_SetSRID(ST_MakePoint(d.lon, d.lat), 4326)::geography, l.geom, 5000)
		ORDER BY d.lon, d.lat
	`

	err = database.DB.Select(&demPoints, query,
		fromBeacon.Lon, fromBeacon.Lat,
		toBeacon.Lon, toBeacon.Lat,
	)
	if err != nil {
		demPoints = []models.DEMPoint{}
	}

	result := analysis.CalculateVisibility(fromBeacon, toBeacon, demPoints)
	result.CalculatedAt = time.Now()

	upsertQuery := `
		INSERT INTO visibility_analysis (
			from_beacon_id, to_beacon_id, is_visible, distance_km, bearing,
			earth_curvature_drop, min_clearance, max_terrain_elevation,
			visibility_angle, calculation_method, calculated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (from_beacon_id, to_beacon_id) DO UPDATE SET
			is_visible = EXCLUDED.is_visible,
			distance_km = EXCLUDED.distance_km,
			bearing = EXCLUDED.bearing,
			earth_curvature_drop = EXCLUDED.earth_curvature_drop,
			min_clearance = EXCLUDED.min_clearance,
			max_terrain_elevation = EXCLUDED.max_terrain_elevation,
			visibility_angle = EXCLUDED.visibility_angle,
			calculated_at = EXCLUDED.calculated_at
		RETURNING id
	`

	var id int
	err = database.DB.Get(&id, upsertQuery,
		result.FromBeaconID, result.ToBeaconID, result.IsVisible,
		result.DistanceKm, result.Bearing, result.EarthCurvatureDrop,
		result.MinClearance, result.MaxTerrainElevation,
		result.VisibilityAngle, result.CalculationMethod, result.CalculatedAt,
	)
	if err == nil {
		result.ID = id
	}

	c.JSON(http.StatusOK, result)
}

func CalculateVisibilityMatrix(c *gin.Context) {
	var beacons []struct {
		models.Beacon
		Lon float64 `db:"lon" json:"lon"`
		Lat float64 `db:"lat" json:"lat"`
	}

	query := `
		SELECT id, name, code, dynasty,
			ST_X(location::geometry) as lon,
			ST_Y(location::geometry) as lat,
			elevation, height, description, status, created_at, updated_at
		FROM beacons
		WHERE status = 'active'
		ORDER BY id
	`

	err := database.DB.Select(&beacons, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	results := make([]models.VisibilityAnalysis, 0)
	for i := 0; i < len(beacons); i++ {
		for j := 0; j < len(beacons); j++ {
			if i == j {
				continue
			}
			from := &beacons[i].Beacon
			from.Lon = beacons[i].Lon
			from.Lat = beacons[i].Lat
			to := &beacons[j].Beacon
			to.Lon = beacons[j].Lon
			to.Lat = beacons[j].Lat

			result := analysis.CalculateVisibility(from, to, nil)
			results = append(results, result)
		}
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

	beacon, err := getBeaconByID(beaconID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Beacon not found"})
		return
	}

	sectorPoints := analysis.CalculateViewShedSector(
		beacon, azimuthStart, azimuthEnd, maxDistance,
	)

	c.JSON(http.StatusOK, gin.H{
		"beacon_id":     beaconID,
		"azimuth_start": azimuthStart,
		"azimuth_end":   azimuthEnd,
		"max_distance":  maxDistance,
		"polygon":       sectorPoints,
	})
}

func getBeaconByID(id int) (*models.Beacon, error) {
	var beacon struct {
		models.Beacon
		Lon float64 `db:"lon"`
		Lat float64 `db:"lat"`
	}

	query := `
		SELECT id, name, code, dynasty,
			ST_X(location::geometry) as lon,
			ST_Y(location::geometry) as lat,
			elevation, height, description, status, created_at, updated_at
		FROM beacons
		WHERE id = $1
	`

	err := database.DB.Get(&beacon, query, id)
	if err != nil {
		return nil, err
	}

	beacon.Beacon.Lon = beacon.Lon
	beacon.Beacon.Lat = beacon.Lat

	return &beacon.Beacon, nil
}
