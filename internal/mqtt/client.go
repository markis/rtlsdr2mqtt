// Package mqtt implements an MQTT client using the Paho MQTT library.
package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var ErrParseCACertificate = errors.New("failed to parse CA certificate")

// PahoClient wraps the Paho MQTT client.
type PahoClient struct {
	client mqtt.Client
	config ClientConfig
	logger *slog.Logger

	// Message handlers
	onConnect          ConnectHandler
	onConnectionLost   ConnectionLostHandler
	messageHandlers    map[string]MessageHandler
	messageHandlersMux sync.RWMutex
}

// NewClient creates a new MQTT client.
func NewClient(config *ClientConfig, logger *slog.Logger) (Client, error) {
	if logger == nil {
		logger = slog.Default()
	}

	client := &PahoClient{
		config:          *config,
		logger:          logger,
		messageHandlers: make(map[string]MessageHandler),
	}

	// Configure TLS if enabled
	var tlsConfig *tls.Config
	if config.TLSEnabled {
		var err error
		tlsConfig, err = client.configureTLS()
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
	}

	// Set default timeouts
	if config.KeepAlive == 0 {
		config.KeepAlive = 60 * time.Second
	}
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 10 * time.Second
	}

	// Create MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://" + net.JoinHostPort(config.Host, strconv.Itoa(config.Port)))
	opts.SetClientID(config.ClientID)

	if config.Username != "" {
		opts.SetUsername(config.Username)
		opts.SetPassword(config.Password)
	}

	opts.SetKeepAlive(config.KeepAlive)
	opts.SetConnectTimeout(config.ConnectTimeout)
	opts.SetAutoReconnect(true)
	opts.SetDefaultPublishHandler(client.defaultMessageHandler)

	if tlsConfig != nil {
		if config.TLSEnabled {
			opts.AddBroker("ssl://" + net.JoinHostPort(config.Host, strconv.Itoa(config.Port)))
		}
		opts.SetTLSConfig(tlsConfig)
	}

	// Set Last Will and Testament
	if config.WillTopic != "" {
		opts.SetWill(config.WillTopic, config.WillPayload, config.WillQoS, config.WillRetain)
	}

	// Set connection handlers
	opts.SetOnConnectHandler(client.onConnectHandler)
	opts.SetConnectionLostHandler(client.onConnectionLostHandler)

	client.client = mqtt.NewClient(opts)
	return client, nil
}

// Connect establishes connection to the MQTT broker.
func (c *PahoClient) Connect() error {
	c.logger.Info("Connecting to MQTT broker", "host", c.config.Host, "port", c.config.Port)

	token := c.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	c.logger.Info("Connected to MQTT broker")
	return nil
}

// Disconnect closes the connection to the MQTT broker.
func (c *PahoClient) Disconnect() {
	c.logger.Info("Disconnecting from MQTT broker")
	c.client.Disconnect(1000) // 1 second timeout
}

// IsConnected returns true if the client is connected.
func (c *PahoClient) IsConnected() bool {
	return c.client.IsConnected()
}

// Publish publishes a message to the specified topic.
func (c *PahoClient) Publish(topic string, payload any, qos byte, retain bool) error {
	var payloadBytes []byte
	var err error

	switch v := payload.(type) {
	case string:
		payloadBytes = []byte(v)
	case []byte:
		payloadBytes = v
	default:
		// Assume it's a struct that needs JSON encoding
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	c.logger.Debug("Publishing MQTT message", "topic", topic, "qos", qos, "retain", retain)

	token := c.client.Publish(topic, qos, retain, payloadBytes)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish to topic %s: %w", topic, token.Error())
	}

	return nil
}

// Subscribe subscribes to a topic with a message handler.
func (c *PahoClient) Subscribe(topic string, qos byte, handler MessageHandler) error {
	c.messageHandlersMux.Lock()
	c.messageHandlers[topic] = handler
	c.messageHandlersMux.Unlock()

	c.logger.Debug("Subscribing to MQTT topic", "topic", topic, "qos", qos)

	token := c.client.Subscribe(topic, qos, func(_ mqtt.Client, msg mqtt.Message) {
		c.messageHandlersMux.RLock()
		h, exists := c.messageHandlers[msg.Topic()]
		c.messageHandlersMux.RUnlock()

		if exists && h != nil {
			h(msg.Topic(), msg.Payload())
		}
	})

	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", topic, token.Error())
	}

	return nil
}

// Unsubscribe unsubscribes from a topic.
func (c *PahoClient) Unsubscribe(topic string) error {
	c.messageHandlersMux.Lock()
	delete(c.messageHandlers, topic)
	c.messageHandlersMux.Unlock()

	c.logger.Debug("Unsubscribing from MQTT topic", "topic", topic)

	token := c.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe from topic %s: %w", topic, token.Error())
	}

	return nil
}

// SetOnConnectHandler sets the handler for connect events.
func (c *PahoClient) SetOnConnectHandler(handler ConnectHandler) {
	c.onConnect = handler
}

// SetOnConnectionLostHandler sets the handler for connection lost events.
func (c *PahoClient) SetOnConnectionLostHandler(handler ConnectionLostHandler) {
	c.onConnectionLost = handler
}

// configureTLS configures TLS settings.
func (c *PahoClient) configureTLS() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if c.config.TLSInsecure {
		tlsConfig.InsecureSkipVerify = true
	}

	// Load CA certificate
	if c.config.TLSCA != "" {
		caCert, err := os.ReadFile(c.config.TLSCA)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, ErrParseCACertificate
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate
	if c.config.TLSCert != "" && c.config.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(c.config.TLSCert, c.config.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// onConnectHandler handles connection events.
func (c *PahoClient) onConnectHandler(_ mqtt.Client) {
	c.logger.Info("MQTT client connected")
	if c.onConnect != nil {
		c.onConnect()
	}
}

// onConnectionLostHandler handles connection lost events.
func (c *PahoClient) onConnectionLostHandler(_ mqtt.Client, err error) {
	c.logger.Warn("MQTT connection lost", "error", err)
	if c.onConnectionLost != nil {
		c.onConnectionLost(err)
	}
}

// defaultMessageHandler handles messages for topics without specific handlers.
func (c *PahoClient) defaultMessageHandler(_ mqtt.Client, msg mqtt.Message) {
	c.logger.Debug("Received unhandled MQTT message", "topic", msg.Topic())
}
