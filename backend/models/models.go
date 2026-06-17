package models

import "time"

type Beacon struct {
	ID          int       `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Code        string    `db:"code" json:"code"`
	Dynasty     string    `db:"dynasty" json:"dynasty"`
	Lon         float64   `db:"lon" json:"lon"`
	Lat         float64   `db:"lat" json:"lat"`
	Elevation   float64   `db:"elevation" json:"elevation"`
	Height      float64   `db:"height" json:"height"`
	Description string    `db:"description" json:"description"`
	Status      string    `db:"status" json:"status"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type SensorData struct {
	ID               int64     `db:"id" json:"id"`
	BeaconID         int       `db:"beacon_id" json:"beacon_id"`
	Timestamp        time.Time `db:"timestamp" json:"timestamp"`
	Visibility       float64   `db:"visibility" json:"visibility"`
	WindSpeed        float64   `db:"wind_speed" json:"wind_speed"`
	WindDirection    float64   `db:"wind_direction" json:"wind_direction"`
	Temperature      float64   `db:"temperature" json:"temperature"`
	Humidity         float64   `db:"humidity" json:"humidity"`
	TerrainElevation float64   `db:"terrain_elevation" json:"terrain_elevation"`
}

type SignalReception struct {
	ID                int64     `db:"id" json:"id"`
	FromBeaconID      int       `db:"from_beacon_id" json:"from_beacon_id"`
	ToBeaconID        int       `db:"to_beacon_id" json:"to_beacon_id"`
	Timestamp         time.Time `db:"timestamp" json:"timestamp"`
	SignalStrength    float64   `db:"signal_strength" json:"signal_strength"`
	IsReceived        bool      `db:"is_received" json:"is_received"`
	InterferenceLevel float64   `db:"interference_level" json:"interference_level"`
	WeatherFactor     float64   `db:"weather_factor" json:"weather_factor"`
}

type VisibilityAnalysis struct {
	ID                  int       `db:"id" json:"id"`
	FromBeaconID        int       `db:"from_beacon_id" json:"from_beacon_id"`
	ToBeaconID          int       `db:"to_beacon_id" json:"to_beacon_id"`
	IsVisible           bool      `db:"is_visible" json:"is_visible"`
	DistanceKm          float64   `db:"distance_km" json:"distance_km"`
	Bearing             float64   `db:"bearing" json:"bearing"`
	EarthCurvatureDrop  float64   `db:"earth_curvature_drop" json:"earth_curvature_drop"`
	MinClearance        float64   `db:"min_clearance" json:"min_clearance"`
	MaxTerrainElevation float64   `db:"max_terrain_elevation" json:"max_terrain_elevation"`
	VisibilityAngle     float64   `db:"visibility_angle" json:"visibility_angle"`
	CalculationMethod   string    `db:"calculation_method" json:"calculation_method"`
	CalculatedAt        time.Time `db:"calculated_at" json:"calculated_at"`
}

type NetworkLink struct {
	ID              int     `db:"id" json:"id"`
	TopologyID      int     `db:"topology_id" json:"topology_id"`
	FromBeaconID    int     `db:"from_beacon_id" json:"from_beacon_id"`
	ToBeaconID      int     `db:"to_beacon_id" json:"to_beacon_id"`
	LinkType        string  `db:"link_type" json:"link_type"`
	Capacity        float64 `db:"capacity" json:"capacity"`
	BaseReliability float64 `db:"base_reliability" json:"base_reliability"`
	IsBidirectional bool    `db:"is_bidirectional" json:"is_bidirectional"`
	IsCritical      bool    `db:"is_critical" json:"is_critical"`
}

type ReliabilityAnalysis struct {
	ID                   int       `db:"id" json:"id"`
	TopologyID           int       `db:"topology_id" json:"topology_id"`
	AnalysisType         string    `db:"analysis_type" json:"analysis_type"`
	Timestamp            time.Time `db:"timestamp" json:"timestamp"`
	OverallReliability   float64   `db:"overall_reliability" json:"overall_reliability"`
	ConnectivityIndex    float64   `db:"connectivity_index" json:"connectivity_index"`
	AveragePathLength    float64   `db:"average_path_length" json:"average_path_length"`
	NodeCount            int       `db:"node_count" json:"node_count"`
	LinkCount            int       `db:"link_count" json:"link_count"`
	MonteCarloIterations int       `db:"monte_carlo_iterations" json:"monte_carlo_iterations"`
	WeatherCondition     string    `db:"weather_condition" json:"weather_condition"`
	Details              string    `db:"details" json:"details"`
}

type Alert struct {
	ID          int64     `db:"id" json:"id"`
	AlertType   string    `db:"alert_type" json:"alert_type"`
	Severity    string    `db:"severity" json:"severity"`
	Title       string    `db:"title" json:"title"`
	Description string    `db:"description" json:"description"`
	BeaconID    int       `db:"beacon_id" json:"beacon_id"`
	LinkID      int       `db:"link_id" json:"link_id"`
	RelatedData string    `db:"related_data" json:"related_data"`
	IsResolved  bool      `db:"is_resolved" json:"is_resolved"`
	ResolvedAt  time.Time `db:"resolved_at" json:"resolved_at"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type DEMPoint struct {
	ID         int     `db:"id" json:"id"`
	Lon        float64 `db:"lon" json:"lon"`
	Lat        float64 `db:"lat" json:"lat"`
	Elevation  float64 `db:"elevation" json:"elevation"`
	Resolution float64 `db:"resolution" json:"resolution"`
}

type ViewShed struct {
	ID           int     `db:"id" json:"id"`
	BeaconID     int     `db:"beacon_id" json:"beacon_id"`
	AzimuthStart float64 `db:"azimuth_start" json:"azimuth_start"`
	AzimuthEnd   float64 `db:"azimuth_end" json:"azimuth_end"`
	MaxDistance  float64 `db:"max_distance" json:"max_distance"`
}

type MonteCarloResult struct {
	Iterations         int        `json:"iterations"`
	SuccessRate        float64    `json:"success_rate"`
	ConfidenceInterval [2]float64 `json:"confidence_interval"`
	FailedLinks        []int      `json:"failed_links,omitempty"`
}
