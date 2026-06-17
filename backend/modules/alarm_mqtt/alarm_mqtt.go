package alarm_mqtt

import (
	"beacon-system/config"
	"beacon-system/database"
	"beacon-system/models"
	"beacon-system/modules/eventbus"
	"beacon-system/mqtt"
	"encoding/json"
	"log"
	"sync"
	"time"
)

type AlarmMQTT struct {
	cfg                *config.Config
	bus                *eventbus.EventBus
	mqttClient         *mqtt.Client
	connectivityThresh float64
	lastAlertTime      map[string]int64
	alertCooldownSec   int64
	mu                 sync.Mutex
	eventChan          chan eventbus.Event
	done               chan struct{}
}

func New(cfg *config.Config, mqttClient *mqtt.Client) *AlarmMQTT {
	return &AlarmMQTT{
		cfg:                cfg,
		bus:                eventbus.Get(),
		mqttClient:         mqttClient,
		connectivityThresh: cfg.Params.Reliability.ConnectivityWarningThresh,
		lastAlertTime:      make(map[string]int64),
		alertCooldownSec:   60,
		eventChan:          make(chan eventbus.Event, 256),
		done:               make(chan struct{}),
	}
}

func (a *AlarmMQTT) Start() {
	go a.eventLoop()

	go func() {
		ch1 := a.bus.Subscribe(eventbus.EventSensorDataReceived, 100)
		ch2 := a.bus.Subscribe(eventbus.EventSignalReceptionReceived, 100)
		ch3 := a.bus.Subscribe(eventbus.EventReliabilityAnalyzed, 100)
		ch4 := a.bus.Subscribe(eventbus.EventConnectivityCheck, 100)

		for {
			select {
			case <-a.done:
				return
			case ev := <-ch1:
				a.forward(ev)
			case ev := <-ch2:
				a.forward(ev)
			case ev := <-ch3:
				a.forward(ev)
			case ev := <-ch4:
				a.forward(ev)
			}
		}
	}()

	log.Println("[Alarm] Alarm & MQTT module started")
}

func (a *AlarmMQTT) forward(ev eventbus.Event) {
	select {
	case a.eventChan <- ev:
	default:
	}
}

func (a *AlarmMQTT) eventLoop() {
	for {
		select {
		case <-a.done:
			return
		case ev := <-a.eventChan:
			switch ev.Type {
			case eventbus.EventSensorDataReceived:
				a.handleSensorEvent(ev)
			case eventbus.EventSignalReceptionReceived:
				a.handleSignalEvent(ev)
			case eventbus.EventReliabilityAnalyzed:
				a.handleReliabilityEvent(ev)
			case eventbus.EventConnectivityCheck:
				a.handleConnectivityEvent(ev)
			}
		}
	}
}

func (a *AlarmMQTT) handleSensorEvent(ev eventbus.Event) {
	payload, ok := ev.Payload.(eventbus.SensorDataPayload)
	if !ok {
		return
	}

	data := payload.Data
	if data.Visibility < 2.0 {
		alert := &models.Alert{
			AlertType:   "low_visibility",
			Severity:    "medium",
			Title:       "能见度异常降低",
			Description: "烽火台传感器检测到能见度低于安全阈值",
			BeaconID:    data.BeaconID,
		}
		relData, _ := json.Marshal(map[string]interface{}{
			"visibility": data.Visibility,
			"threshold":  2.0,
		})
		alert.RelatedData = string(relData)
		a.triggerAlert(alert)
	}

	if data.WindSpeed > 15.0 {
		alert := &models.Alert{
			AlertType:   "high_wind_speed",
			Severity:    "medium",
			Title:       "风速超过警戒值",
			Description: "高风速可能影响烟火信号传递",
			BeaconID:    data.BeaconID,
		}
		relData, _ := json.Marshal(map[string]interface{}{
			"wind_speed": data.WindSpeed,
			"threshold":  15.0,
		})
		alert.RelatedData = string(relData)
		a.triggerAlert(alert)
	}
}

func (a *AlarmMQTT) handleSignalEvent(ev eventbus.Event) {
	payload, ok := ev.Payload.(eventbus.SignalReceptionPayload)
	if !ok {
		return
	}

	sig := payload.Data
	if !sig.IsReceived {
		isCritical := a.checkLinkCritical(sig.FromBeaconID, sig.ToBeaconID)
		if isCritical {
			alert := &models.Alert{
				AlertType:   "critical_link_down",
				Severity:    "high",
				Title:       "关键链路信号中断",
				Description: "关键烽火台链路未能接收到信号",
				BeaconID:    sig.FromBeaconID,
			}
			relData, _ := json.Marshal(map[string]interface{}{
				"from_beacon_id":  sig.FromBeaconID,
				"to_beacon_id":    sig.ToBeaconID,
				"signal_strength": sig.SignalStrength,
				"is_critical":     true,
			})
			alert.RelatedData = string(relData)
			a.triggerAlert(alert)
		}
	}
}

func (a *AlarmMQTT) handleReliabilityEvent(ev eventbus.Event) {
	payload, ok := ev.Payload.(eventbus.ReliabilityPayload)
	if !ok {
		return
	}

	if payload.ConnectivityIdx < a.connectivityThresh {
		alert := &models.Alert{
			AlertType:   "connectivity_low",
			Severity:    "high",
			Title:       "网络连通度低于阈值",
			Description: "烽火台通信网络连通度已低于安全阈值",
		}
		relData, _ := json.Marshal(map[string]float64{
			"connectivity_index": payload.ConnectivityIdx,
			"threshold":          a.connectivityThresh,
		})
		alert.RelatedData = string(relData)
		a.triggerAlert(alert)
	}
}

func (a *AlarmMQTT) handleConnectivityEvent(ev eventbus.Event) {
	payload, ok := ev.Payload.(eventbus.ReliabilityPayload)
	if !ok {
		return
	}

	if payload.ConnectivityIdx < a.connectivityThresh {
		alert := &models.Alert{
			AlertType:   "connectivity_low",
			Severity:    "high",
			Title:       "网络连通度低于阈值",
			Description: "烽火台通信网络连通度已低于安全阈值",
		}
		relData, _ := json.Marshal(map[string]float64{
			"connectivity_index": payload.ConnectivityIdx,
			"threshold":          a.connectivityThresh,
		})
		alert.RelatedData = string(relData)
		a.triggerAlert(alert)
	}
}

func (a *AlarmMQTT) triggerAlert(alert *models.Alert) bool {
	a.mu.Lock()
	now := time.Now().Unix()
	key := alert.AlertType + ":" + itoa(alert.BeaconID)
	if lastTime, exists := a.lastAlertTime[key]; exists {
		if now-lastTime < a.alertCooldownSec {
			a.mu.Unlock()
			return false
		}
	}
	a.lastAlertTime[key] = now
	a.mu.Unlock()

	alert.CreatedAt = time.Now()
	a.saveAlert(alert)

	if a.mqttClient != nil {
		if err := a.mqttClient.PublishAlert(alert); err != nil {
			log.Printf("[Alarm] MQTT publish failed: %v", err)
		}
	}

	a.bus.Publish(eventbus.Event{
		Type: eventbus.EventAlertTriggered,
		Payload: eventbus.AlertPayload{
			Alert:           *alert,
			ConnectivityIdx: 0,
			Threshold:       a.connectivityThresh,
		},
		Time: time.Now().UnixNano(),
	})

	log.Printf("[Alarm] Triggered: type=%s severity=%s beacon=%d title=%s",
		alert.AlertType, alert.Severity, alert.BeaconID, alert.Title)

	return true
}

func (a *AlarmMQTT) saveAlert(alert *models.Alert) error {
	query := `
		INSERT INTO alerts (
			alert_type, severity, title, description, beacon_id, link_id, related_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	var id int64
	var createdAt time.Time
	err := database.DB.QueryRow(query,
		alert.AlertType, alert.Severity, alert.Title,
		alert.Description, alert.BeaconID, alert.LinkID, alert.RelatedData,
	).Scan(&id, &createdAt)
	if err == nil {
		alert.ID = id
		alert.CreatedAt = createdAt
	}
	return err
}

func (a *AlarmMQTT) checkLinkCritical(fromID, toID int) bool {
	query := `
		SELECT COUNT(*) FROM network_links l
		JOIN network_topology t ON l.topology_id = t.id
		WHERE t.is_active = true AND l.is_critical = true
		  AND ((l.from_beacon_id = $1 AND l.to_beacon_id = $2)
		   OR (l.from_beacon_id = $2 AND l.to_beacon_id = $1 AND l.is_bidirectional = true))
	`
	var count int
	err := database.DB.Get(&count, query, fromID, toID)
	if err != nil {
		return false
	}
	return count > 0
}

func (a *AlarmMQTT) GetAlerts(limit int, resolved *bool) ([]models.Alert, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var alerts []models.Alert
	var query string
	var args []interface{}

	if resolved != nil {
		query = `
			SELECT * FROM alerts
			WHERE is_resolved = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		args = append(args, *resolved, limit)
	} else {
		query = `
			SELECT * FROM alerts
			ORDER BY created_at DESC
			LIMIT $1
		`
		args = append(args, limit)
	}

	err := database.DB.Select(&alerts, query, args...)
	return alerts, err
}

func (a *AlarmMQTT) ResolveAlert(id int64) error {
	query := `
		UPDATE alerts
		SET is_resolved = true, resolved_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := database.DB.Exec(query, id)
	return err
}

func (a *AlarmMQTT) Stop() {
	close(a.done)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
