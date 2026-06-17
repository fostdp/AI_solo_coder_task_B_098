package config

import (
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
}

func Load() *Config {
	return &Config{
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
