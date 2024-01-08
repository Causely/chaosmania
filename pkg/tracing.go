package pkg

import "os"

func IsDatadogEnabled() bool {
	return os.Getenv("DATADOG_ENABLED") == "true"
}

func IsOpenTelemetryEnabled() bool {
	return os.Getenv("OTEL_ENABLED") == "true"
}
