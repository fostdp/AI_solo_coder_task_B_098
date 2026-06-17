package dtu_receiver

import (
	"beacon-system/config"
	"beacon-system/database"
	"beacon-system/models"
	"beacon-system/modules/eventbus"
	"fmt"
	"log"
	"time"
)

type DTUReceiver struct {
	cfg    *config.Config
	bus    *eventbus.EventBus
	errors map[int]int64
}

func New(cfg *config.Config) *DTUReceiver {
	return &DTUReceiver{
		cfg:    cfg,
		bus:    eventbus.Get(),
		errors: make(map[int]int64),
	}
}

func (r *DTUReceiver) ValidateSensorData(data *models.SensorData) error {
	if data.BeaconID <= 0 {
		return fmt.Errorf("invalid beacon_id: %d", data.BeaconID)
	}

	if data.Visibility < 0 || data.Visibility > 100 {
		return fmt.Errorf("visibility out of range [0, 100]: %f", data.Visibility)
	}

	if data.WindSpeed < 0 || data.WindSpeed > 50 {
		return fmt.Errorf("wind_speed out of range [0, 50]: %f", data.WindSpeed)
	}

	if data.Temperature < -50 || data.Temperature > 60 {
		return fmt.Errorf("temperature out of range [-50, 60]: %f", data.Temperature)
	}

	if data.Humidity < 0 || data.Humidity > 100 {
		return fmt.Errorf("humidity out of range [0, 100]: %f", data.Humidity)
	}

	if data.WindDirection < 0 || data.WindDirection > 360 {
		return fmt.Errorf("wind_direction out of range [0, 360]: %f", data.WindDirection)
	}

	return nil
}

func (r *DTUReceiver) ValidateSignalReception(sig *models.SignalReception) error {
	if sig.FromBeaconID <= 0 {
		return fmt.Errorf("invalid from_beacon_id: %d", sig.FromBeaconID)
	}
	if sig.ToBeaconID <= 0 {
		return fmt.Errorf("invalid to_beacon_id: %d", sig.ToBeaconID)
	}
	if sig.FromBeaconID == sig.ToBeaconID {
		return fmt.Errorf("from_beacon_id equals to_beacon_id")
	}
	if sig.SignalStrength < 0 || sig.SignalStrength > 100 {
		return fmt.Errorf("signal_strength out of range [0, 100]: %f", sig.SignalStrength)
	}
	if sig.WeatherFactor < 0 || sig.WeatherFactor > 1 {
		return fmt.Errorf("weather_factor out of range [0, 1]: %f", sig.WeatherFactor)
	}
	return nil
}

func (r *DTUReceiver) ProcessSensorData(data *models.SensorData) (int64, error) {
	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	if err := r.ValidateSensorData(data); err != nil {
		r.errors[data.BeaconID]++
		return 0, fmt.Errorf("validation failed: %w", err)
	}

	var id int64
	query := `
		INSERT INTO sensor_data (beacon_id, timestamp, visibility, wind_speed, wind_direction, temperature, humidity, terrain_elevation)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err := database.DB.Get(&id, query,
		data.BeaconID, data.Timestamp,
		data.Visibility, data.WindSpeed,
		data.WindDirection, data.Temperature,
		data.Humidity, data.TerrainElevation,
	)
	if err != nil {
		return 0, fmt.Errorf("insert sensor_data failed: %w", err)
	}

	r.bus.Publish(eventbus.Event{
		Type: eventbus.EventSensorDataReceived,
		Payload: eventbus.SensorDataPayload{
			Data: *data,
		},
		Time: time.Now().UnixNano(),
	})

	log.Printf("[DTU] Sensor data recorded: beacon=%d visibility=%.1fkm wind=%.1fm/s",
		data.BeaconID, data.Visibility, data.WindSpeed)

	return id, nil
}

func (r *DTUReceiver) ProcessSignalReception(sig *models.SignalReception) (int64, error) {
	if sig.Timestamp.IsZero() {
		sig.Timestamp = time.Now()
	}

	if err := r.ValidateSignalReception(sig); err != nil {
		return 0, fmt.Errorf("validation failed: %w", err)
	}

	var id int64
	query := `
		INSERT INTO signal_reception (from_beacon_id, to_beacon_id, timestamp, signal_strength, is_received, interference_level, weather_factor)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err := database.DB.Get(&id, query,
		sig.FromBeaconID, sig.ToBeaconID, sig.Timestamp,
		sig.SignalStrength, sig.IsReceived,
		sig.InterferenceLevel, sig.WeatherFactor,
	)
	if err != nil {
		return 0, fmt.Errorf("insert signal_reception failed: %w", err)
	}

	r.bus.Publish(eventbus.Event{
		Type: eventbus.EventSignalReceptionReceived,
		Payload: eventbus.SignalReceptionPayload{
			Data: *sig,
		},
		Time: time.Now().UnixNano(),
	})

	log.Printf("[DTU] Signal reception recorded: %d→%d strength=%.1f received=%v",
		sig.FromBeaconID, sig.ToBeaconID, sig.SignalStrength, sig.IsReceived)

	return id, nil
}

func (r *DTUReceiver) Start() {
	log.Println("[DTU] Receiver module started")
}

func (r *DTUReceiver) GetErrorCount(beaconID int) int64 {
	return r.errors[beaconID]
}

func (r *DTUReceiver) ResetErrors() {
	r.errors = make(map[int]int64)
}
