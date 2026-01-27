// Package version provides version information for the application.
package version

import (
	"fmt"
	"runtime"
)

const (
	// Version represents the current version of the application.
	Version = "1.0.0"
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
