package mqtt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/isa-cloud/isa_cloud/internal/config"
	"github.com/isa-cloud/isa_cloud/pkg/logger"
)

// Adapter handles MQTT protocol adaptation for IoT devices
type Adapter struct {
	config       *config.MQTTConfig
	logger       *logger.Logger
	client       mqtt.Client
	handlers     map[string]MessageHandler
	mu           sync.RWMutex
	httpClient   *http.Client
	deviceConfig *config.DeviceManagementConfig
	isConnected  bool
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

// NewAdapter creates a new MQTT adapter
func NewAdapter(cfg *config.MQTTConfig, deviceCfg *config.DeviceManagementConfig, logger *logger.Logger) *Adapter {
	adapter := &Adapter{
		config:       cfg,
		deviceConfig: deviceCfg,
		logger:       logger,
		handlers:     make(map[string]MessageHandler),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		isConnected:  false,
	}

	// Setup default handlers
	adapter.setupDefaultHandlers()

	return adapter
}

// Connect establishes connection to MQTT broker
func (a *Adapter) Connect() error {
	if !a.config.Enabled {
		a.logger.Info("MQTT adapter disabled in configuration")
		return nil
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(a.config.BrokerURL)
	// Add timestamp to make client ID unique
	clientID := fmt.Sprintf("%s_%d", a.config.ClientID, time.Now().Unix())
	opts.SetClientID(clientID)

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

	a.logger.Info("Connected to MQTT broker", "broker", a.config.BrokerURL)
	return nil
}

// Disconnect closes the MQTT connection
func (a *Adapter) Disconnect() {
	if a.client != nil && a.client.IsConnected() {
		a.client.Disconnect(250)
		a.logger.Info("Disconnected from MQTT broker")
	}
	a.isConnected = false
}

// Subscribe to MQTT topics
func (a *Adapter) Subscribe(topic string, handler MessageHandler) error {
	a.mu.Lock()
	a.handlers[topic] = handler
	a.mu.Unlock()

	token := a.client.Subscribe(topic, a.config.QoS, func(client mqtt.Client, msg mqtt.Message) {
		a.handleMessage(msg.Topic(), msg.Payload())
	})

	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %v", topic, token.Error())
	}

	a.logger.Debug("Subscribed to MQTT topic", "topic", topic)
	return nil
}

// Publish message to MQTT topic
func (a *Adapter) Publish(topic string, payload interface{}) error {
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

	a.logger.Debug("Published MQTT message", "topic", topic, "size", len(data))
	return nil
}

// setupDefaultHandlers configures default message handlers
func (a *Adapter) setupDefaultHandlers() {
	// Device telemetry handler
	a.handlers[a.config.Topics.DeviceTelemetry] = a.handleTelemetry

	// Device status handler
	a.handlers[a.config.Topics.DeviceStatus] = a.handleDeviceStatus

	// Device commands response handler
	a.handlers[a.config.Topics.DeviceCommandsResponse] = a.handleCommandResponse

	// Device authentication handler
	a.handlers[a.config.Topics.DeviceAuth] = a.handleDeviceAuth

	// Device registration handler
	a.handlers[a.config.Topics.DeviceRegistration] = a.handleDeviceRegistration
}

// handleMessage processes incoming MQTT messages
func (a *Adapter) handleMessage(topic string, payload []byte) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Find matching handler
	for pattern, handler := range a.handlers {
		if a.matchTopic(pattern, topic) {
			if err := handler(topic, payload); err != nil {
				a.logger.Error("Error handling MQTT message", 
					"topic", topic, 
					"error", err,
				)
			}
			return
		}
	}

	a.logger.Debug("No handler found for MQTT topic", "topic", topic)
}

// handleTelemetry processes device telemetry data
func (a *Adapter) handleTelemetry(topic string, payload []byte) error {
	deviceID := a.extractDeviceID(topic)

	var telemetry map[string]interface{}
	if err := json.Unmarshal(payload, &telemetry); err != nil {
		return err
	}

	// Add device ID if not present
	if _, exists := telemetry["device_id"]; !exists {
		telemetry["device_id"] = deviceID
	}

	// Forward to telemetry service via HTTP
	return a.forwardToService("telemetry_service", 
		fmt.Sprintf("/api/v1/devices/%s/telemetry", deviceID), 
		telemetry)
}

// handleDeviceStatus processes device status updates
func (a *Adapter) handleDeviceStatus(topic string, payload []byte) error {
	deviceID := a.extractDeviceID(topic)

	var status map[string]interface{}
	if err := json.Unmarshal(payload, &status); err != nil {
		return err
	}

	// Add device ID if not present
	if _, exists := status["device_id"]; !exists {
		status["device_id"] = deviceID
	}

	// Forward to device management service
	return a.forwardToService("device_service", 
		fmt.Sprintf("/api/v1/devices/%s/status", deviceID), 
		status)
}

// handleCommandResponse processes command responses from devices
func (a *Adapter) handleCommandResponse(topic string, payload []byte) error {
	deviceID := a.extractDeviceID(topic)

	var response map[string]interface{}
	if err := json.Unmarshal(payload, &response); err != nil {
		return err
	}

	a.logger.Info("Command response received", 
		"device_id", deviceID, 
		"response", response,
	)

	// Store or forward command response as needed
	// This could be forwarded to a command tracking service
	return nil
}

// handleDeviceAuth handles device authentication requests
func (a *Adapter) handleDeviceAuth(topic string, payload []byte) error {
	deviceID := a.extractDeviceID(topic)

	var authData map[string]interface{}
	if err := json.Unmarshal(payload, &authData); err != nil {
		return err
	}

	// Forward to auth service for device authentication
	authResult, err := a.authenticateDevice(deviceID, authData)
	if err != nil {
		// Send auth failure response
		return a.Publish(fmt.Sprintf("devices/%s/auth/response", deviceID), map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Send auth success response
	return a.Publish(fmt.Sprintf("devices/%s/auth/response", deviceID), authResult)
}

// handleDeviceRegistration handles new device registration
func (a *Adapter) handleDeviceRegistration(topic string, payload []byte) error {
	var regData map[string]interface{}
	if err := json.Unmarshal(payload, &regData); err != nil {
		return err
	}

	// Forward to device management service for registration
	result, err := a.registerDevice(regData)
	if err != nil {
		a.logger.Error("Device registration failed", "error", err)
		return err
	}

	// Send registration result back to device if device_id is available
	if deviceID, ok := result["device_id"].(string); ok {
		return a.Publish(fmt.Sprintf("devices/%s/register/response", deviceID), result)
	}

	a.logger.Info("Device registration completed", "result", result)
	return nil
}

// SendCommandToDevice sends a command to a device via MQTT
func (a *Adapter) SendCommandToDevice(deviceID string, command map[string]interface{}) error {
	topic := fmt.Sprintf("devices/%s/commands", deviceID)
	return a.Publish(topic, command)
}

// Connection event handlers
func (a *Adapter) onConnect(client mqtt.Client) {
	a.logger.Info("MQTT client connected")
	a.isConnected = true

	// Subscribe to all configured topics
	topics := []string{
		a.config.Topics.DeviceTelemetry,
		a.config.Topics.DeviceStatus,
		a.config.Topics.DeviceCommandsResponse,
		a.config.Topics.DeviceAuth,
		a.config.Topics.DeviceRegistration,
	}

	for _, topic := range topics {
		if handler, exists := a.handlers[topic]; exists {
			if err := a.Subscribe(topic, handler); err != nil {
				a.logger.Error("Failed to subscribe to topic", "topic", topic, "error", err)
			}
		}
	}
}

func (a *Adapter) onConnectionLost(client mqtt.Client, err error) {
	a.logger.Warn("MQTT connection lost", "error", err)
	a.isConnected = false
}

func (a *Adapter) onReconnecting(client mqtt.Client, opts *mqtt.ClientOptions) {
	a.logger.Info("MQTT client reconnecting...")
}

// Helper functions

func (a *Adapter) matchTopic(pattern, topic string) bool {
	patternParts := strings.Split(pattern, "/")
	topicParts := strings.Split(topic, "/")

	if len(patternParts) != len(topicParts) {
		return false
	}

	for i, part := range patternParts {
		if part != "+" && part != "#" && part != topicParts[i] {
			return false
		}
		if part == "#" {
			return true // # matches everything after
		}
	}

	return true
}

func (a *Adapter) extractDeviceID(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) >= 2 && parts[0] == "devices" {
		return parts[1]
	}
	return ""
}

// forwardToService forwards a message to an HTTP service
func (a *Adapter) forwardToService(serviceName, endpoint string, data interface{}) error {
	if !a.deviceConfig.Enabled {
		a.logger.Debug("Device management disabled, skipping service forward")
		return nil
	}

	var serviceEndpoint *config.ServiceEndpoint
	switch serviceName {
	case "device_service":
		serviceEndpoint = &a.deviceConfig.DeviceService
	case "telemetry_service":
		serviceEndpoint = &a.deviceConfig.TelemetryService
	case "ota_service":
		serviceEndpoint = &a.deviceConfig.OTAService
	default:
		return fmt.Errorf("unknown service: %s", serviceName)
	}

	url := fmt.Sprintf("http://%s:%d%s", 
		serviceEndpoint.Host, 
		serviceEndpoint.HTTPPort, 
		endpoint)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ISA-Gateway-MQTT-Adapter")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to forward to service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("service returned error: %d", resp.StatusCode)
	}

	a.logger.Debug("Successfully forwarded to service", 
		"service", serviceName, 
		"endpoint", endpoint,
		"status", resp.StatusCode,
	)

	return nil
}

// authenticateDevice authenticates a device via auth service
func (a *Adapter) authenticateDevice(deviceID string, authData map[string]interface{}) (map[string]interface{}, error) {
	// This would integrate with the auth service
	// For now, return a mock successful authentication
	return map[string]interface{}{
		"success":   true,
		"token":     "device-jwt-token",
		"device_id": deviceID,
		"expires_at": time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

// registerDevice registers a new device via device management service
func (a *Adapter) registerDevice(regData map[string]interface{}) (map[string]interface{}, error) {
	// Forward to device management service
	err := a.forwardToService("device_service", "/api/v1/devices", regData)
	if err != nil {
		return nil, err
	}

	// Return mock successful registration
	// In real implementation, this would parse the response from the service
	return map[string]interface{}{
		"success":   true,
		"device_id": fmt.Sprintf("device_%d", time.Now().Unix()),
		"message":   "Device registered successfully",
	}, nil
}

// IsConnected returns the connection status
func (a *Adapter) IsConnected() bool {
	return a.isConnected && a.client != nil && a.client.IsConnected()
}

// GetStats returns adapter statistics
func (a *Adapter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"connected":        a.IsConnected(),
		"broker_url":       a.config.BrokerURL,
		"client_id":        a.config.ClientID,
		"subscribed_topics": len(a.handlers),
	}
}