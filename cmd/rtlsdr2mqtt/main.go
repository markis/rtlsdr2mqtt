package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"rtlsdr2mqtt/internal/config"
	"rtlsdr2mqtt/internal/controller"
	"rtlsdr2mqtt/pkg/version"
)

func main() {
	// Parse command line flags
	var (
		configPath  = flag.String("config", "", "Path to configuration file")
		showVersion = flag.Bool("version", false, "Show version and exit")
		showHelp    = flag.Bool("help", false, "Show help and exit")
	)
	flag.Parse()

	// Show version
	if *showVersion {
		_, _ = os.Stdout.WriteString(version.BuildInfo() + "\n")
		os.Exit(0)
	}

	// Show help
	if *showHelp {
		showUsage()
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logger based on configuration
	logger := setupLogger(cfg.General.Verbosity)

	logger.Info("Starting application", "version", version.Version)
	logger.Debug("Configuration loaded",
		"meters", len(cfg.Meters),
		"mqtt_host", cfg.MQTT.Host,
		"usb_device", cfg.SDR.USBDevice)

	// Create and run controller
	ctrl, err := controller.New(cfg, logger)
	if err != nil {
		logger.Error("Failed to create controller", "error", err)
		os.Exit(1)
	}

	// Run the application
	if err := ctrl.Run(); err != nil {
		logger.Error("Application error", "error", err)
		os.Exit(1)
	}
}

// setupLogger configures the logger based on verbosity level.
func setupLogger(verbosity string) *slog.Logger {
	var level slog.Level
	var output io.Writer = os.Stdout

	switch strings.ToLower(verbosity) {
	case "none":
		output = io.Discard
		level = slog.LevelDebug // Doesn't matter since output is discarded
	case "error", "critical":
		level = slog.LevelError
	case "warning":
		level = slog.LevelWarn
	case "info":
		level = slog.LevelInfo
	case "debug":
		level = slog.LevelDebug
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(output, opts)
	return slog.New(handler)
}

// showUsage displays usage information.
func showUsage() {
	usage := version.BuildInfo() + `
RTL-SDR to MQTT Bridge for Home Assistant

Usage: rtlsdr2mqtt [OPTIONS]

Options:
  -config string    Path to configuration file
  -version         Show version information and exit
  -help            Show this help message and exit

Configuration file search order:
  1. /data/options.json (Home Assistant add-on)
  2. /data/options.yaml
  3. /data/options.yml
  4. /etc/rtlsdr2mqtt.yaml
  5. Path specified by -config flag

For more information, visit: https://github.com/markis/rtlsdr2mqtt
`
	_, _ = os.Stdout.WriteString(usage)
}
