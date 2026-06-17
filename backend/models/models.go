package models

import "time"

type Beacon struct {
	ID                   int       `db:"id" json:"id"`
	Name                 string    `db:"name" json:"name"`
	Code                 string    `db:"code" json:"code"`
	Dynasty              string    `db:"dynasty" json:"dynasty"`
	Lon                  float64   `db:"lon" json:"lon"`
	Lat                  float64   `db:"lat" json:"lat"`
	Elevation            float64   `db:"elevation" json:"elevation"`
	Height               float64   `db:"height" json:"height"`
	Description          string    `db:"description" json:"description"`
	Status               string    `db:"status" json:"status"`
	CoordinateSource     string    `db:"coordinate_source" json:"coordinate_source"`
	CoordinatePrecisionM float64   `db:"coordinate_precision_m" json:"coordinate_precision_m"`
	GpsVerified          bool      `db:"gps_verified" json:"gps_verified"`
	ArchaeologicalSiteID string    `db:"archaeological_site_id" json:"archaeological_site_id"`
	SurveyYear           int       `db:"survey_year" json:"survey_year"`
	CreatedAt            time.Time `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time `db:"updated_at" json:"updated_at"`
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

type NetworkTopology struct {
	ID          int       `db:"id" json:"id"`
	Version     int       `db:"version" json:"version"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	IsActive    bool      `db:"is_active" json:"is_active"`
	DynastyCode string    `db:"dynasty_code" json:"dynasty_code"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type NetworkLink struct {
	ID              int       `db:"id" json:"id"`
	TopologyID      int       `db:"topology_id" json:"topology_id"`
	FromBeaconID    int       `db:"from_beacon_id" json:"from_beacon_id"`
	ToBeaconID      int       `db:"to_beacon_id" json:"to_beacon_id"`
	LinkType        string    `db:"link_type" json:"link_type"`
	Capacity        float64   `db:"capacity" json:"capacity"`
	BaseReliability float64   `db:"base_reliability" json:"base_reliability"`
	IsBidirectional bool      `db:"is_bidirectional" json:"is_bidirectional"`
	IsCritical      bool      `db:"is_critical" json:"is_critical"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
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

type Dynasty struct {
	Code        string `db:"code" json:"code"`
	Name        string `db:"name" json:"name"`
	Period      string `db:"period" json:"period"`
	Description string `db:"description" json:"description"`
	Color       string `db:"color" json:"color"`
	SortOrder   int    `db:"sort_order" json:"sort_order"`
}

type ModernBaseStation struct {
	ID                  int       `db:"id" json:"id"`
	Name                string    `db:"name" json:"name"`
	StationType         string    `db:"station_type" json:"station_type"`
	TypeName            string    `db:"type_name,omitempty" json:"type_name,omitempty"`
	Lon                 float64   `db:"lon" json:"lon"`
	Lat                 float64   `db:"lat" json:"lat"`
	Height              float64   `db:"height" json:"height"`
	CoverageRadiusKm    float64   `db:"coverage_radius_km" json:"coverage_radius_km"`
	CapacityMbps        float64   `db:"capacity_mbps" json:"capacity_mbps"`
	LatencyMs           float64   `db:"latency_ms" json:"latency_ms"`
	FrequencyGhz        float64   `db:"frequency_ghz" json:"frequency_ghz"`
	PowerKw             float64   `db:"power_kw" json:"power_kw"`
	IsStandardCompliant bool      `db:"is_standard_compliant" json:"is_standard_compliant"`
	StandardVersion     string    `db:"standard_version" json:"standard_version"`
	Status              string    `db:"status" json:"status"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
}

type BaseStationType struct {
	TypeCode             string  `db:"type_code" json:"type_code"`
	TypeName             string  `db:"type_name" json:"type_name"`
	StandardVersion      string  `db:"standard_version" json:"standard_version"`
	Description          string  `db:"description" json:"description"`
	MinCoverageRadiusKm  float64 `db:"min_coverage_radius_km" json:"min_coverage_radius_km"`
	MaxCoverageRadiusKm  float64 `db:"max_coverage_radius_km" json:"max_coverage_radius_km"`
	StdCoverageRadiusKm  float64 `db:"standard_coverage_radius_km" json:"standard_coverage_radius_km"`
	MinCapacityMbps      float64 `db:"min_capacity_mbps" json:"min_capacity_mbps"`
	MaxCapacityMbps      float64 `db:"max_capacity_mbps" json:"max_capacity_mbps"`
	StdCapacityMbps      float64 `db:"standard_capacity_mbps" json:"standard_capacity_mbps"`
	MinLatencyMs         float64 `db:"min_latency_ms" json:"min_latency_ms"`
	MaxLatencyMs         float64 `db:"max_latency_ms" json:"max_latency_ms"`
	StdLatencyMs         float64 `db:"standard_latency_ms" json:"standard_latency_ms"`
	FrequencyBand        string  `db:"frequency_band" json:"frequency_band"`
	TypicalHeightM       float64 `db:"typical_height_m" json:"typical_height_m"`
	TypicalPowerKw       float64 `db:"typical_power_kw" json:"typical_power_kw"`
	TechnologyGeneration string  `db:"technology_generation" json:"technology_generation"`
	SortOrder            int     `db:"sort_order" json:"sort_order"`
}

type ResilienceAnalysis struct {
	ID                 int       `db:"id" json:"id"`
	TopologyID         int       `db:"topology_id" json:"topology_id"`
	AnalysisType       string    `db:"analysis_type" json:"analysis_type"`
	AttackType         string    `db:"attack_type" json:"attack_type"`
	NodeRemovalRatio   float64   `db:"node_removal_ratio" json:"node_removal_ratio"`
	ConnectivityIndex  float64   `db:"connectivity_index" json:"connectivity_index"`
	GiantComponentSize int       `db:"giant_component_size" json:"giant_component_size"`
	RobustnessScore    float64   `db:"robustness_score" json:"robustness_score"`
	Iterations         int       `db:"iterations" json:"iterations"`
	Details            string    `db:"details" json:"details"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
}

type UserBeaconIgnition struct {
	ID                     int64     `db:"id" json:"id"`
	SessionID              string    `db:"session_id" json:"session_id"`
	BeaconID               int       `db:"beacon_id" json:"beacon_id"`
	TopologyID             int       `db:"topology_id" json:"topology_id"`
	UserNote               string    `db:"user_note" json:"user_note"`
	WeatherFactor          float64   `db:"weather_factor" json:"weather_factor"`
	ReachedCount           int       `db:"reached_count" json:"reached_count"`
	TotalPropagationTimeMs float64   `db:"total_propagation_time_ms" json:"total_propagation_time_ms"`
	PropagationPath        string    `db:"propagation_path" json:"propagation_path"`
	CreatedAt              time.Time `db:"created_at" json:"created_at"`
}

type DynastyComparison struct {
	DynastyCode     string  `json:"dynasty_code"`
	DynastyName     string  `json:"dynasty_name"`
	Color           string  `json:"color"`
	NodeCount       int     `json:"node_count"`
	LinkCount       int     `json:"link_count"`
	ConnectivityIdx float64 `json:"connectivity_index"`
	AvgPathLength   float64 `json:"avg_path_length"`
	Diameter        int     `json:"diameter"`
	Density         float64 `json:"density"`
	Reliability     float64 `json:"reliability"`
	AvgReliability  float64 `json:"avg_link_reliability"`
	TopologyID      int     `json:"topology_id"`
}

type CrossEraComparison struct {
	BeaconNetwork map[string]interface{} `json:"beacon_network"`
	ModernNetwork map[string]interface{} `json:"modern_network"`
	Comparison    map[string]interface{} `json:"comparison"`
}

type ResilienceCurvePoint struct {
	RemovalRatio      float64 `json:"removal_ratio"`
	ConnectivityIndex float64 `json:"connectivity_index"`
	GiantComponentPct float64 `json:"giant_component_pct"`
}

type ResilienceResult struct {
	AttackType        string                 `json:"attack_type"`
	CurvePoints       []ResilienceCurvePoint `json:"curve_points"`
	RobustnessScore   float64                `json:"robustness_score"`
	CriticalThreshold float64                `json:"critical_threshold"`
	TotalNodes        int                    `json:"total_nodes"`
	Iterations        int                    `json:"iterations"`
}

type IgnitionPropagationStep struct {
	BeaconID   int     `json:"beacon_id"`
	BeaconName string  `json:"beacon_name"`
	Step       int     `json:"step"`
	DelayMs    float64 `json:"delay_ms"`
}

type IgnitionResult struct {
	IgnitionID    int64                     `json:"ignition_id"`
	StartBeaconID int                       `json:"start_beacon_id"`
	ReachedCount  int                       `json:"reached_count"`
	TotalTimeMs   float64                   `json:"total_time_ms"`
	Path          []IgnitionPropagationStep `json:"path"`
	TopologyID    int                       `json:"topology_id"`
	WeatherFactor float64                   `json:"weather_factor"`
}
