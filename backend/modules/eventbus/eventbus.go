package eventbus

import (
	"beacon-system/models"
	"sync"
)

type EventType string

const (
	EventSensorDataReceived      EventType = "sensor_data_received"
	EventSignalReceptionReceived EventType = "signal_reception_received"
	EventVisibilityCalculated    EventType = "visibility_calculated"
	EventReliabilityAnalyzed     EventType = "reliability_analyzed"
	EventConnectivityCheck       EventType = "connectivity_check"
	EventAlertTriggered          EventType = "alert_triggered"
	EventBeaconIgnited           EventType = "beacon_ignited"
	EventResilienceAnalyzed      EventType = "resilience_analyzed"
	EventDynastyCompared         EventType = "dynasty_compared"
)

type Event struct {
	Type    EventType
	Payload interface{}
	Time    int64
}

type SensorDataPayload struct {
	Data models.SensorData
}

type SignalReceptionPayload struct {
	Data models.SignalReception
}

type VisibilityPayload struct {
	Result models.VisibilityAnalysis
}

type ReliabilityPayload struct {
	Result          interface{}
	Metrics         map[string]float64
	ConnectivityIdx float64
}

type AlertPayload struct {
	Alert           models.Alert
	ConnectivityIdx float64
	Threshold       float64
}

type EventBus struct {
	subscribers map[EventType][]chan Event
	mu          sync.RWMutex
}

var (
	instance *EventBus
	once     sync.Once
)

func Get() *EventBus {
	once.Do(func() {
		instance = &EventBus{
			subscribers: make(map[EventType][]chan Event),
		}
	})
	return instance
}

func (eb *EventBus) Subscribe(eventType EventType, bufferSize int) chan Event {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan Event, bufferSize)
	eb.subscribers[eventType] = append(eb.subscribers[eventType], ch)
	return ch
}

func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	subs, ok := eb.subscribers[event.Type]
	if !ok {
		return
	}

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, subs := range eb.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}
	eb.subscribers = make(map[EventType][]chan Event)
}
