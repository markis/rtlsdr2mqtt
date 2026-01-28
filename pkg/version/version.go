// Package version provides version information for the application.
package version

import (
	"fmt"
	"runtime"
)

// Version represents the current version of the application.
// This is set at build time via ldflags: -ldflags "-X rtlsdr2mqtt/pkg/version.Version=v1.2.3"
var Version = "dev"

const (
	// ApplicationName is the name of the application.
	ApplicationName = "rtlsdr2mqtt"
)

// Info returns version information about the application.
func Info() string {
	return fmt.Sprintf("%s %s", ApplicationName, Version)
}

// BuildInfo returns detailed build information.
func BuildInfo() string {
	return fmt.Sprintf("%s %s (built with %s)", ApplicationName, Version, runtime.Version())
}
