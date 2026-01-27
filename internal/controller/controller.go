// Package controller manages the application lifecycle and coordinates all components.
package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"rtlsdr2mqtt/internal/config"
	"rtlsdr2mqtt/internal/decoder"
	"rtlsdr2mqtt/internal/discovery"
	"rtlsdr2mqtt/internal/mqtt"
	"rtlsdr2mqtt/internal/parser"
	"rtlsdr2mqtt/internal/usb"
	"rtlsdr2mqtt/pkg/version"
)

var (
	ErrMeterNotFound = errors.New("meter not found in configuration")
	ErrReadLine      = errors.New("error reading line from rtlamr output")
)

// Controller manages the application lifecycle and coordinates all components.
type Controller struct {
	config     *config.Config
	mqttClient mqtt.Client
	decoder    *decoder.Decoder

	cancel context.CancelFunc

	// Tracking
	readCounter map[string]bool // Track which meters have been read in current cycle
	mu          sync.RWMutex

	// Cached values
	healthCheckFile string // Cached HEALTHCHECK_FILE env var

	logger *slog.Logger
}

// New creates a new controller instance.
func New(cfg *config.Config, logger *slog.Logger) (*Controller, error) {
	if logger == nil {
		logger = slog.Default()
	}

	controller := &Controller{
		config:          cfg,
		readCounter:     make(map[string]bool),
		healthCheckFile: os.Getenv("HEALTHCHECK_FILE"), // Cache at startup
		logger:          logger,
	}

	// Setup MQTT client
	mqttConfig := mqtt.ClientConfig{
		Host:        cfg.MQTT.Host,
		Port:        cfg.MQTT.Port,
		Username:    cfg.MQTT.User,
		Password:    cfg.MQTT.Password,
		ClientID:    fmt.Sprintf("rtlsdr2mqtt-%d", time.Now().Unix()),
		TLSEnabled:  cfg.MQTT.TLS.Enabled,
		TLSInsecure: cfg.MQTT.TLS.Insecure,
		TLSCA:       cfg.MQTT.TLS.CA,
		TLSCert:     cfg.MQTT.TLS.Cert,
		TLSKey:      cfg.MQTT.TLS.Keyfile,
		WillTopic:   discovery.GenerateStatusTopic(cfg.MQTT.BaseTopic),
		WillPayload: "offline",
		WillQoS:     1,
		WillRetain:  true,
	}

	var err error
	controller.mqttClient, err = mqtt.NewClient(&mqttConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %w", err)
	}

	// Setup decoder (direct rtlamr integration with SDR)
	controller.decoder = decoder.NewDecoder(cfg, logger)

	return controller, nil
}

// Run starts the main application.
func (c *Controller) Run() error {
	c.logger.Info("Starting rtlsdr2mqtt", "version", version.Version)

	// Create context for the application
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	// Setup signal handling
	c.setupSignalHandling(ctx)

	// Connect to MQTT
	if err := c.mqttClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to MQTT: %w", err)
	}

	// Set up MQTT handlers
	c.setupMQTTHandlers()

	// Subscribe to Home Assistant status topic for restart detection
	if err := c.mqttClient.Subscribe(c.config.MQTT.HomeAssistant.StatusTopic, 1, c.handleHAStatus); err != nil {
		c.logger.Warn("Failed to subscribe to HA status topic", "topic", c.config.MQTT.HomeAssistant.StatusTopic, "error", err)
	}

	// Publish discovery messages
	c.publishDiscovery()

	// Publish initial status (retained so subscribers see current state)
	statusTopic := discovery.GenerateStatusTopic(c.config.MQTT.BaseTopic)
	if err := c.mqttClient.Publish(statusTopic, "online", 1, true); err != nil {
		c.logger.Warn("Failed to publish online status", "error", err)
	}

	// Reset USB device if needed (skip in containers where USB enumeration is unreliable)
	if !isRunningInContainer() {
		if err := c.resetUSBDevice(); err != nil {
			c.logger.Warn("USB device reset failed", "error", err)
		}
	} else {
		c.logger.Debug("Skipping USB device reset (running in container)")
	}

	// Start the main loop
	return c.mainLoop(ctx)
}

// Shutdown performs graceful shutdown.
func (c *Controller) Shutdown() {
	c.logger.Info("Shutting down rtlsdr2mqtt...")

	// Cancel context to signal shutdown
	c.cancel()

	// Publish offline status (retained so subscribers see current state)
	statusTopic := discovery.GenerateStatusTopic(c.config.MQTT.BaseTopic)
	if c.mqttClient.IsConnected() {
		if err := c.mqttClient.Publish(statusTopic, "offline", 1, true); err != nil {
			c.logger.Debug("Failed to publish offline status", "error", err)
		}
	}

	// Stop decoder
	if c.decoder != nil {
		if err := c.decoder.Stop(); err != nil {
			c.logger.Debug("Error stopping decoder", "error", err)
		}
	}

	// Disconnect MQTT
	if c.mqttClient != nil {
		c.mqttClient.Disconnect()
	}

	c.logger.Info("Shutdown complete")
}

// setupSignalHandling sets up signal handlers for graceful shutdown.
func (c *Controller) setupSignalHandling(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		defer signal.Stop(sigChan)
		select {
		case sig := <-sigChan:
			c.logger.Info("Received signal", "signal", sig)
			c.Shutdown()
		case <-ctx.Done():
			return
		}
	}()
}

// setupMQTTHandlers configures MQTT event handlers.
func (c *Controller) setupMQTTHandlers() {
	c.mqttClient.SetOnConnectHandler(func() {
		c.logger.Info("MQTT client connected")
		// Re-publish discovery on reconnect
		c.publishDiscovery()
	})

	c.mqttClient.SetOnConnectionLostHandler(func(err error) {
		c.logger.Warn("MQTT connection lost", "error", err)
	})
}

// mainLoop runs the main application loop.
// It blocks on decoder channels instead of polling, reducing CPU usage.
func (c *Controller) mainLoop(ctx context.Context) error {
	// Ticker for periodic tasks (sleep check, process health)
	sleepCheckTicker := time.NewTicker(5 * time.Second)
	defer sleepCheckTicker.Stop()

	// Start the decoder initially
	if err := c.ensureProcessesRunning(ctx); err != nil {
		c.logger.Error("Failed to start decoder", "error", err)
		return err
	}

	for {
		// Get channels from decoder (may be recreated after sleep)
		msgChan, errChan, doneChan := c.decoder.Channels()

		select {
		case <-ctx.Done():
			return nil
		case msg := <-msgChan:
			c.handleMessage(msg)
		case err := <-errChan:
			c.handleDecoderError(ctx, err)
		case <-doneChan:
			c.handleDecoderStopped(ctx)
		case <-sleepCheckTicker.C:
			c.handlePeriodicCheck(ctx)
		}
	}
}

// handleMessage processes an incoming meter reading message.
func (c *Controller) handleMessage(msg *decoder.Message) {
	if msg == nil {
		return
	}

	if err := c.writeHealthCheck(); err != nil {
		c.logger.Warn("Failed to write health check", "error", err)
	}

	c.logger.Info("Received meter reading", "meter_id", msg.MeterID, "consumption", msg.Consumption)

	if err := c.processReadingFromDecoder(msg); err != nil {
		c.logger.Error("Failed to process reading", "error", err)
	}
}

// handleDecoderError handles errors from the decoder and attempts recovery.
func (c *Controller) handleDecoderError(ctx context.Context, err error) {
	c.logger.Error("Decoder error", "error", err)

	if stopErr := c.decoder.Stop(); stopErr != nil {
		c.logger.Debug("Error stopping decoder", "error", stopErr)
	}

	if restartErr := c.ensureProcessesRunning(ctx); restartErr != nil {
		c.logger.Error("Failed to restart decoder", "error", restartErr)
	}
}

// handleDecoderStopped handles unexpected decoder stops and attempts restart.
func (c *Controller) handleDecoderStopped(ctx context.Context) {
	if c.shouldSleep() {
		return // Expected stop for sleep mode
	}

	c.logger.Warn("Decoder stopped unexpectedly, restarting...")

	if err := c.ensureProcessesRunning(ctx); err != nil {
		c.logger.Error("Failed to restart decoder", "error", err)
	}
}

// handlePeriodicCheck performs periodic health checks and sleep mode transitions.
func (c *Controller) handlePeriodicCheck(ctx context.Context) {
	if c.shouldSleep() {
		if err := c.enterSleepMode(ctx); err != nil {
			c.logger.Error("Error during sleep mode", "error", err)
		}
		if err := c.ensureProcessesRunning(ctx); err != nil {
			c.logger.Error("Failed to restart decoder after sleep", "error", err)
		}
		return
	}

	if !c.decoder.IsRunning() {
		if err := c.ensureProcessesRunning(ctx); err != nil {
			c.logger.Error("Failed to ensure processes running", "error", err)
		}
	}
}

// ensureProcessesRunning starts/restarts processes if they're not running.
func (c *Controller) ensureProcessesRunning(ctx context.Context) error {
	// Check decoder - it now manages SDR directly
	if !c.decoder.IsRunning() {
		c.logger.Info("Starting decoder with direct SDR access...")
		if err := c.decoder.Start(ctx); err != nil {
			return fmt.Errorf("failed to start decoder: %w", err)
		}
	}

	return nil
}

// writeHealthCheck touches the health check file to update its modification time.
// This is called each time a message is successfully decoded, indicating the system is working.
func (c *Controller) writeHealthCheck() error {
	if c.healthCheckFile == "" {
		// No healthcheck file configured
		return nil
	}

	filePath := filepath.Clean(c.healthCheckFile)
	now := time.Now()

	err := os.Chtimes(filePath, now, now)
	if err == nil {
		c.logger.Debug("Health check file touched", "file", filePath)
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to update health check file: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create health check file: %w", err)
	}
	file.Close()

	c.logger.Debug("Health check file created", "file", filePath)
	return nil
}

// processReadingFromDecoder processes a single meter reading from the decoder.
func (c *Controller) processReadingFromDecoder(msg *decoder.Message) error {
	meterID := msg.MeterIDString()

	meterConfig, exists := c.config.FindMeterByID(meterID)
	if !exists {
		return fmt.Errorf("%w: %s", ErrMeterNotFound, meterID)
	}

	// Format the reading
	var formattedReading any
	consumption := msg.ConsumptionInt64()
	if meterConfig.Format != "" {
		formattedReading = parser.FormatNumber(consumption, meterConfig.Format)
	} else {
		formattedReading = consumption
	}

	// Create state payload
	statePayload := mqtt.StatePayload{
		Reading:  formattedReading,
		LastSeen: time.Now().Format(time.RFC3339),
	}

	// Publish state (retained so HA sees last known value after restart)
	stateTopic := discovery.GenerateStateTopic(c.config.MQTT.BaseTopic, meterID)
	if err := c.mqttClient.Publish(stateTopic, statePayload, 1, true); err != nil {
		return fmt.Errorf("failed to publish state: %w", err)
	}

	// Publish attributes (retained so HA sees last known value after restart)
	attributesTopic := discovery.GenerateAttributesTopic(c.config.MQTT.BaseTopic, meterID)
	if err := c.mqttClient.Publish(attributesTopic, msg.Attributes, 1, true); err != nil {
		return fmt.Errorf("failed to publish attributes: %w", err)
	}

	// Update read counter
	c.mu.Lock()
	c.readCounter[meterID] = true
	c.mu.Unlock()

	return nil
}

// shouldSleep checks if we should enter sleep mode.
func (c *Controller) shouldSleep() bool {
	if c.config.General.SleepFor <= 0 {
		return false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if all meters have been read
	for i := range c.config.Meters {
		if !c.readCounter[c.config.Meters[i].ID] {
			return false
		}
	}

	return true
}

// enterSleepMode stops processes, sleeps, and resets counters.
func (c *Controller) enterSleepMode(ctx context.Context) error {
	c.logger.Info("All meters read, entering sleep mode", "duration", c.config.General.SleepFor)

	// Stop decoder (which will close the SDR device)
	if err := c.decoder.Stop(); err != nil {
		c.logger.Warn("Failed to stop decoder for sleep", "error", err)
	}

	// Reset read counter
	c.mu.Lock()
	c.readCounter = make(map[string]bool)
	c.mu.Unlock()

	// Sleep
	sleepTimer := time.NewTimer(time.Duration(c.config.General.SleepFor) * time.Second)
	defer sleepTimer.Stop()

	select {
	case <-sleepTimer.C:
		c.logger.Info("Sleep complete, resuming operations")
	case <-ctx.Done():
		return nil // Shutdown requested
	}

	return nil
}

// publishDiscovery publishes Home Assistant discovery messages for all meters.
func (c *Controller) publishDiscovery() {
	c.logger.Info("Publishing Home Assistant discovery messages")

	for i := range c.config.Meters {
		meter := &c.config.Meters[i]
		payload := discovery.GenerateDiscoveryPayload(c.config.MQTT.BaseTopic, meter)
		topic := discovery.GenerateDiscoveryTopic(c.config.MQTT.HomeAssistant.DiscoveryPrefix, meter.ID)

		if err := c.mqttClient.Publish(topic, payload, 1, true); err != nil {
			c.logger.Warn("Failed to publish discovery for meter", "meter_id", meter.ID, "error", err)
			continue
		}

		c.logger.Debug("Published discovery for meter", "meter_id", meter.ID, "topic", topic)
	}
}

// handleHAStatus handles Home Assistant status messages.
func (c *Controller) handleHAStatus(_ string, payload []byte) {
	c.logger.Debug("Home Assistant status received", "payload", string(payload))
	c.publishDiscovery()
}

// resetUSBDevice resets the USB device if configured.
func (c *Controller) resetUSBDevice() error {
	if c.config.SDR.USBDevice == "" {
		// Auto-detect and reset first device
		if err := usb.ResetFirstDevice(c.logger); err != nil {
			return fmt.Errorf("failed to reset first USB device: %w", err)
		}
	} else {
		// Reset specific device
		if err := usb.ResetDeviceByID(c.config.SDR.USBDevice, c.logger); err != nil {
			return fmt.Errorf("failed to reset USB device %s: %w", c.config.SDR.USBDevice, err)
		}
	}

	return nil
}

// isRunningInContainer detects if the application is running inside a container.
// It checks for Docker (/.dockerenv), Podman (/run/.containerenv), and Kubernetes.
func isRunningInContainer() bool {
	// Check for Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check for Podman
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}

	// Check for Kubernetes (env var is set automatically in all pods)
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	return false
}
