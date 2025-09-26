package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTAdapter handles MQTT protocol adaptation for IoT devices
type MQTTAdapter struct {
	client        mqtt.Client
	config        *MQTTConfig
	handlers      map[string]MessageHandler
	mu            sync.RWMutex
	messageRouter *MessageRouter
	isConnected   bool
}

// MQTTConfig holds MQTT adapter configuration
type MQTTConfig struct {
	BrokerURL    string        `yaml:"broker_url"`
	ClientID     string        `yaml:"client_id"`
	Username     string        `yaml:"username"`
	Password     string        `yaml:"password"`
	KeepAlive    time.Duration `yaml:"keep_alive"`
	PingTimeout  time.Duration `yaml:"ping_timeout"`
	CleanSession bool          `yaml:"clean_session"`
	AutoReconnect bool         `yaml:"auto_reconnect"`
	QoS          byte          `yaml:"qos"`
}

// MessageHandler processes MQTT messages
type MessageHandler func(topic string, payload []byte) error

// DeviceMessage represents a message from/to IoT device
type DeviceMessage struct {
	DeviceID    string                 `json:"device_id"`
	MessageType string                 `json:"message_type"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
}

// MessageRouter routes messages between MQTT and HTTP services
type MessageRouter struct {
	httpClient    *HTTPClient
	deviceService string
	authService   string
}

// NewMQTTAdapter creates a new MQTT adapter
func NewMQTTAdapter(config *MQTTConfig, router *MessageRouter) *MQTTAdapter {
	adapter := &MQTTAdapter{
		config:        config,
		handlers:      make(map[string]MessageHandler),
		messageRouter: router,
		isConnected:   false,
	}
	
	// Setup default handlers
	adapter.setupDefaultHandlers()
	
	return adapter
}

// Connect establishes connection to MQTT broker
func (a *MQTTAdapter) Connect() error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(a.config.BrokerURL)
	opts.SetClientID(a.config.ClientID)
	
	if a.config.Username != "" {
		opts.SetUsername(a.config.Username)
		opts.SetPassword(a.config.Password)
	}
	
	opts.SetKeepAlive(a.config.KeepAlive)
	opts.SetPingTimeout(a.config.PingTimeout)
	opts.SetCleanSession(a.config.CleanSession)
	opts.SetAutoReconnect(a.config.AutoReconnect)
	
	// Set connection handlers
	opts.SetOnConnectHandler(a.onConnect)
	opts.SetConnectionLostHandler(a.onConnectionLost)
	opts.SetReconnectingHandler(a.onReconnecting)
	
	// Create and connect client
	a.client = mqtt.NewClient(opts)
	
	token := a.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %v", token.Error())
	}
	
	log.Printf("Connected to MQTT broker at %s", a.config.BrokerURL)
	return nil
}

// Disconnect closes the MQTT connection
func (a *MQTTAdapter) Disconnect() {
	if a.client != nil && a.client.IsConnected() {
		a.client.Disconnect(250)
		log.Println("Disconnected from MQTT broker")
	}
	a.isConnected = false
}

// Subscribe to MQTT topics
func (a *MQTTAdapter) Subscribe(topic string, handler MessageHandler) error {
	a.mu.Lock()
	a.handlers[topic] = handler
	a.mu.Unlock()
	
	token := a.client.Subscribe(topic, a.config.QoS, func(client mqtt.Client, msg mqtt.Message) {
		a.handleMessage(msg.Topic(), msg.Payload())
	})
	
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %v", topic, token.Error())
	}
	
	log.Printf("Subscribed to topic: %s", topic)
	return nil
}

// Publish message to MQTT topic
func (a *MQTTAdapter) Publish(topic string, payload interface{}) error {
	var data []byte
	var err error
	
	switch v := payload.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %v", err)
		}
	}
	
	token := a.client.Publish(topic, a.config.QoS, false, data)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish to topic %s: %v", topic, token.Error())
	}
	
	return nil
}

// setupDefaultHandlers configures default message handlers
func (a *MQTTAdapter) setupDefaultHandlers() {
	// Device telemetry handler
	a.handlers["devices/+/telemetry"] = a.handleTelemetry
	
	// Device status handler
	a.handlers["devices/+/status"] = a.handleDeviceStatus
	
	// Device commands response handler
	a.handlers["devices/+/commands/response"] = a.handleCommandResponse
	
	// Device authentication handler
	a.handlers["devices/+/auth"] = a.handleDeviceAuth
	
	// Device registration handler
	a.handlers["devices/register"] = a.handleDeviceRegistration
}

// handleMessage processes incoming MQTT messages
func (a *MQTTAdapter) handleMessage(topic string, payload []byte) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	// Find matching handler
	for pattern, handler := range a.handlers {
		if matchTopic(pattern, topic) {
			if err := handler(topic, payload); err != nil {
				log.Printf("Error handling message on topic %s: %v", topic, err)
			}
			return
		}
	}
	
	log.Printf("No handler found for topic: %s", topic)
}

// handleTelemetry processes device telemetry data
func (a *MQTTAdapter) handleTelemetry(topic string, payload []byte) error {
	deviceID := extractDeviceID(topic)
	
	var telemetry map[string]interface{}
	if err := json.Unmarshal(payload, &telemetry); err != nil {
		return err
	}
	
	// Forward to telemetry service via HTTP
	endpoint := fmt.Sprintf("/api/v1/devices/%s/telemetry", deviceID)
	return a.messageRouter.ForwardToService("telemetry_service", endpoint, telemetry)
}

// handleDeviceStatus processes device status updates
func (a *MQTTAdapter) handleDeviceStatus(topic string, payload []byte) error {
	deviceID := extractDeviceID(topic)
	
	var status map[string]interface{}
	if err := json.Unmarshal(payload, &status); err != nil {
		return err
	}
	
	// Forward to device management service
	endpoint := fmt.Sprintf("/api/v1/devices/%s/status", deviceID)
	return a.messageRouter.ForwardToService("device_management_service", endpoint, status)
}

// handleCommandResponse processes command responses from devices
func (a *MQTTAdapter) handleCommandResponse(topic string, payload []byte) error {
	deviceID := extractDeviceID(topic)
	
	var response map[string]interface{}
	if err := json.Unmarshal(payload, &response); err != nil {
		return err
	}
	
	log.Printf("Command response from device %s: %v", deviceID, response)
	return nil
}

// handleDeviceAuth handles device authentication requests
func (a *MQTTAdapter) handleDeviceAuth(topic string, payload []byte) error {
	deviceID := extractDeviceID(topic)
	
	var authData map[string]interface{}
	if err := json.Unmarshal(payload, &authData); err != nil {
		return err
	}
	
	// Authenticate device
	authResult, err := a.messageRouter.AuthenticateDevice(deviceID, authData)
	if err != nil {
		// Send auth failure response
		a.Publish(fmt.Sprintf("devices/%s/auth/response", deviceID), map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return err
	}
	
	// Send auth success response
	return a.Publish(fmt.Sprintf("devices/%s/auth/response", deviceID), authResult)
}

// handleDeviceRegistration handles new device registration
func (a *MQTTAdapter) handleDeviceRegistration(topic string, payload []byte) error {
	var regData map[string]interface{}
	if err := json.Unmarshal(payload, &regData); err != nil {
		return err
	}
	
	// Forward to device management service for registration
	result, err := a.messageRouter.RegisterDevice(regData)
	if err != nil {
		return err
	}
	
	// Send registration result back to device
	deviceID := result["device_id"].(string)
	return a.Publish(fmt.Sprintf("devices/%s/register/response", deviceID), result)
}

// SendCommandToDevice sends a command to a device via MQTT
func (a *MQTTAdapter) SendCommandToDevice(deviceID string, command map[string]interface{}) error {
	topic := fmt.Sprintf("devices/%s/commands", deviceID)
	return a.Publish(topic, command)
}

// Connection event handlers
func (a *MQTTAdapter) onConnect(client mqtt.Client) {
	log.Println("MQTT client connected")
	a.isConnected = true
	
	// Re-subscribe to topics
	for topic := range a.handlers {
		a.Subscribe(topic, a.handlers[topic])
	}
}

func (a *MQTTAdapter) onConnectionLost(client mqtt.Client, err error) {
	log.Printf("MQTT connection lost: %v", err)
	a.isConnected = false
}

func (a *MQTTAdapter) onReconnecting(client mqtt.Client, opts *mqtt.ClientOptions) {
	log.Println("MQTT client reconnecting...")
}

// Helper functions

func matchTopic(pattern, topic string) bool {
	patternParts := strings.Split(pattern, "/")
	topicParts := strings.Split(topic, "/")
	
	if len(patternParts) != len(topicParts) {
		return false
	}
	
	for i, part := range patternParts {
		if part != "+" && part != topicParts[i] {
			return false
		}
	}
	
	return true
}

func extractDeviceID(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// HTTPClient wrapper for making HTTP requests to services
type HTTPClient struct {
	// Implementation would use the existing gateway's HTTP client
	// This is a placeholder for the actual implementation
}

// ForwardToService forwards a message to an HTTP service
func (r *MessageRouter) ForwardToService(service, endpoint string, data interface{}) error {
	// This would use the gateway's existing service discovery and HTTP forwarding
	log.Printf("Forwarding to %s: %s", service, endpoint)
	return nil
}

// AuthenticateDevice authenticates a device
func (r *MessageRouter) AuthenticateDevice(deviceID string, authData map[string]interface{}) (map[string]interface{}, error) {
	// Forward to auth service for device authentication
	endpoint := fmt.Sprintf("/api/v1/devices/%s/auth", deviceID)
	// Implementation would call the auth service
	return map[string]interface{}{
		"success": true,
		"token":   "device-jwt-token",
		"expires": time.Now().Add(24 * time.Hour),
	}, nil
}

// RegisterDevice registers a new device
func (r *MessageRouter) RegisterDevice(regData map[string]interface{}) (map[string]interface{}, error) {
	// Forward to device management service
	// Implementation would call the device management service
	return map[string]interface{}{
		"success":   true,
		"device_id": "new-device-id",
	}, nil
}