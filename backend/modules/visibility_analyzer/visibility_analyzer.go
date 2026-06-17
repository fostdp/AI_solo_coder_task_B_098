package visibility_analyzer

import (
	"beacon-system/analysis"
	"beacon-system/config"
	"beacon-system/database"
	"beacon-system/models"
	"beacon-system/modules/eventbus"
	"fmt"
	"log"
	"time"
)

type VisibilityAnalyzer struct {
	cfg *config.Config
	bus *eventbus.EventBus
}

func New(cfg *config.Config) *VisibilityAnalyzer {
	return &VisibilityAnalyzer{
		cfg: cfg,
		bus: eventbus.Get(),
	}
}

func (v *VisibilityAnalyzer) GetBeaconByID(id int) (*models.Beacon, error) {
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

func (v *VisibilityAnalyzer) QueryDEMPoints(fromLon, fromLat, toLon, toLat float64) ([]models.DEMPoint, error) {
	var demPoints []models.DEMPoint
	radius := v.cfg.Params.Terrain.DEMSearchRadiusMeters

	query := `
		WITH line AS (
			SELECT ST_MakeLine(
				ST_SetSRID(ST_MakePoint($1, $2), 4326),
				ST_SetSRID(ST_MakePoint($3, $4), 4326)
			)::geography as geom
		)
		SELECT d.id, d.lon, d.lat, d.elevation, d.resolution
		FROM dem_data d, line l
		WHERE ST_DWithin(ST_SetSRID(ST_MakePoint(d.lon, d.lat), 4326)::geography, l.geom, $5)
		ORDER BY d.lon, d.lat
	`

	err := database.DB.Select(&demPoints, query, fromLon, fromLat, toLon, toLat, radius)
	if err != nil {
		log.Printf("[Visibility] DEM query warning: %v, falling back to refraction-only", err)
		return []models.DEMPoint{}, nil
	}

	return demPoints, nil
}

func (v *VisibilityAnalyzer) Calculate(fromID, toID int) (*models.VisibilityAnalysis, error) {
	fromBeacon, err := v.GetBeaconByID(fromID)
	if err != nil {
		return nil, fmt.Errorf("from beacon not found: %w", err)
	}

	toBeacon, err := v.GetBeaconByID(toID)
	if err != nil {
		return nil, fmt.Errorf("to beacon not found: %w", err)
	}

	demPoints, err := v.QueryDEMPoints(fromBeacon.Lon, fromBeacon.Lat, toBeacon.Lon, toBeacon.Lat)
	if err != nil {
		demPoints = []models.DEMPoint{}
	}

	result := analysis.CalculateVisibility(fromBeacon, toBeacon, demPoints)
	result.CalculatedAt = time.Now()

	v.persistResult(&result)

	v.bus.Publish(eventbus.Event{
		Type: eventbus.EventVisibilityCalculated,
		Payload: eventbus.VisibilityPayload{
			Result: result,
		},
		Time: time.Now().UnixNano(),
	})

	log.Printf("[Visibility] %d→%d visible=%v dist=%.2fkm clearance=%.1fm",
		fromID, toID, result.IsVisible, result.DistanceKm, result.MinClearance)

	return &result, nil
}

func (v *VisibilityAnalyzer) persistResult(result *models.VisibilityAnalysis) {
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
	err := database.DB.Get(&id, upsertQuery,
		result.FromBeaconID, result.ToBeaconID, result.IsVisible,
		result.DistanceKm, result.Bearing, result.EarthCurvatureDrop,
		result.MinClearance, result.MaxTerrainElevation,
		result.VisibilityAngle, result.CalculationMethod, result.CalculatedAt,
	)
	if err == nil {
		result.ID = id
	}
}

func (v *VisibilityAnalyzer) CalculateMatrix() ([]models.VisibilityAnalysis, error) {
	var beacons []struct {
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
		WHERE status = 'active'
		ORDER BY id
	`

	err := database.DB.Select(&beacons, query)
	if err != nil {
		return nil, err
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

	log.Printf("[Visibility] Matrix calculated: %d pairs", len(results))
	return results, nil
}

func (v *VisibilityAnalyzer) GetViewShed(beaconID int, azimuthStart, azimuthEnd, maxDistanceKm float64) ([][2]float64, error) {
	if maxDistanceKm <= 0 {
		maxDistanceKm = v.cfg.Params.Visibility.DefaultViewShedDistKm
	}
	if maxDistanceKm < v.cfg.Params.Visibility.MinViewShedDistKm {
		maxDistanceKm = v.cfg.Params.Visibility.MinViewShedDistKm
	}
	if maxDistanceKm > v.cfg.Params.Visibility.MaxViewShedDistKm {
		maxDistanceKm = v.cfg.Params.Visibility.MaxViewShedDistKm
	}

	beacon, err := v.GetBeaconByID(beaconID)
	if err != nil {
		return nil, fmt.Errorf("beacon not found: %w", err)
	}

	sectorPoints := analysis.CalculateViewShedSector(beacon, azimuthStart, azimuthEnd, maxDistanceKm)
	return sectorPoints, nil
}

func (v *VisibilityAnalyzer) GetAllResults() ([]models.VisibilityAnalysis, error) {
	var results []models.VisibilityAnalysis
	query := `
		SELECT * FROM visibility_analysis
		ORDER BY from_beacon_id, to_beacon_id
	`

	err := database.DB.Select(&results, query)
	return results, err
}

func (v *VisibilityAnalyzer) Start() {
	log.Println("[Visibility] Analyzer module started")
}
