package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	ServerPort            string
	DBHost                string
	DBPort                string
	DBUser                string
	DBPassword            string
	DBName                string
	MQTTBroker            string
	MQTTPort              int
	MQTTUser              string
	MQTTPass              string
	MQTTTopic             string
	ConnectivityThreshold float64
	DemoMode              bool
	Params                *Params
}

type Params struct {
	Terrain     TerrainConfig     `json:"terrain"`
	Atmosphere  AtmosphereConfig  `json:"atmosphere"`
	Visibility  VisibilityConfig  `json:"visibility"`
	Reliability ReliabilityConfig `json:"reliability"`
	Weather     WeatherConfig     `json:"weather"`
}

type TerrainConfig struct {
	DEMResolutionMeters      float64 `json:"dem_resolution_meters"`
	DEMSearchRadiusMeters    float64 `json:"dem_search_radius_meters"`
	DefaultElevationMeters   float64 `json:"default_terrain_elevation_meters"`
	ClearanceThresholdMeters float64 `json:"terrain_clearance_threshold_meters"`
	MaxAnalysisDistanceKm    float64 `json:"max_analysis_distance_km"`
	MinAnalysisDistanceKm    float64 `json:"min_analysis_distance_km"`
}

type AtmosphereConfig struct {
	EffectiveEarthFactorK  float64            `json:"itu_r_effective_earth_factor_k"`
	RefractionGradient     float64            `json:"refraction_gradient_n_units_per_km"`
	StandardTemperatureK   float64            `json:"standard_temperature_kelvin"`
	StandardPressureHPa    float64            `json:"standard_pressure_hpa"`
	StandardHumidityPct    float64            `json:"standard_humidity_percent"`
	TemperatureLapseRate   float64            `json:"temperature_lapse_rate_k_per_km"`
	RefractionModels       map[string]KFactor `json:"refraction_models"`
	DefaultRefractionModel string             `json:"default_refraction_model"`
}

type KFactor struct {
	KFactor     float64 `json:"k_factor"`
	Description string  `json:"description"`
}

type VisibilityConfig struct {
	ViewShedAzimuthSteps  int     `json:"viewshed_azimuth_steps"`
	DefaultViewShedDistKm float64 `json:"default_viewshed_distance_km"`
	MinViewShedDistKm     float64 `json:"min_viewshed_distance_km"`
	MaxViewShedDistKm     float64 `json:"max_viewshed_distance_km"`
	BearingToleranceDeg   float64 `json:"bearing_tolerance_degrees"`
}

type ReliabilityConfig struct {
	DefaultMCIterations       int     `json:"default_monte_carlo_iterations"`
	MaxMCIterations           int     `json:"max_monte_carlo_iterations"`
	ISEdgeThreshold           int     `json:"importance_sampling_edge_threshold"`
	ISBiasFactor              float64 `json:"importance_sampling_bias_factor"`
	ConfidenceLevel           float64 `json:"confidence_level"`
	CriticalLinkSensitivity   float64 `json:"critical_link_sensitivity_threshold"`
	ConnectivityWarningThresh float64 `json:"connectivity_warning_threshold"`
}

type WeatherConfig struct {
	VisibilityFactors     map[string]float64 `json:"visibility_factors"`
	WindSpeedThresholdMPS float64            `json:"wind_speed_threshold_mps"`
	MaxWindPenaltyFactor  float64            `json:"max_wind_penalty_factor"`
	VisibilityThresholdKm float64            `json:"visibility_threshold_km"`
	MinVisibilityFactor   float64            `json:"min_visibility_factor"`
}

func Load() *Config {
	cfg := &Config{
		ServerPort:            getEnv("SERVER_PORT", "8080"),
		DBHost:                getEnv("DB_HOST", "localhost"),
		DBPort:                getEnv("DB_PORT", "5432"),
		DBUser:                getEnv("DB_USER", "postgres"),
		DBPassword:            getEnv("DB_PASSWORD", "postgres"),
		DBName:                getEnv("DB_NAME", "beacon_system"),
		MQTTBroker:            getEnv("MQTT_BROKER", "localhost"),
		MQTTPort:              getEnvInt("MQTT_PORT", 1883),
		MQTTUser:              getEnv("MQTT_USER", ""),
		MQTTPass:              getEnv("MQTT_PASS", ""),
		MQTTTopic:             getEnv("MQTT_TOPIC", "beacon/alerts"),
		ConnectivityThreshold: getEnvFloat("CONNECTIVITY_THRESHOLD", 0.7),
		DemoMode:              getEnvBool("DEMO_MODE", true),
	}

	params, err := LoadParams("config/params.json")
	if err != nil {
		fmt.Printf("Warning: failed to load params.json, using defaults: %v\n", err)
		params = defaultParams()
	}
	cfg.Params = params

	return cfg
}

func LoadParams(path string) (*Params, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read params file: %w", err)
	}

	var params Params
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("parse params json: %w", err)
	}

	return &params, nil
}

func defaultParams() *Params {
	return &Params{
		Terrain: TerrainConfig{
			DEMResolutionMeters:      30,
			DEMSearchRadiusMeters:    5000,
			DefaultElevationMeters:   1200,
			ClearanceThresholdMeters: 5,
			MaxAnalysisDistanceKm:    100,
			MinAnalysisDistanceKm:    0.5,
		},
		Atmosphere: AtmosphereConfig{
			EffectiveEarthFactorK:  4.0 / 3.0,
			RefractionGradient:     -40e-6,
			StandardTemperatureK:   288.15,
			StandardPressureHPa:    1013.25,
			StandardHumidityPct:    60,
			TemperatureLapseRate:   -6.5,
			DefaultRefractionModel: "standard",
		},
		Visibility: VisibilityConfig{
			ViewShedAzimuthSteps:  36,
			DefaultViewShedDistKm: 20,
			MinViewShedDistKm:     5,
			MaxViewShedDistKm:     50,
			BearingToleranceDeg:   1,
		},
		Reliability: ReliabilityConfig{
			DefaultMCIterations:       1000,
			MaxMCIterations:           100000,
			ISEdgeThreshold:           20,
			ISBiasFactor:              2.0,
			ConfidenceLevel:           0.95,
			CriticalLinkSensitivity:   0.1,
			ConnectivityWarningThresh: 0.7,
		},
		Weather: WeatherConfig{
			VisibilityFactors: map[string]float64{
				"clear": 1.0, "light_haze": 0.8, "foggy": 0.6, "heavy_fog": 0.4, "sandstorm": 0.2,
			},
			WindSpeedThresholdMPS: 5,
			MaxWindPenaltyFactor:  0.5,
			VisibilityThresholdKm: 20,
			MinVisibilityFactor:   0.1,
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
