package backend

// Version information injected at build time via -ldflags
var (
	// Version is the application version (from package.json)
	Version = "dev"
	// BuildTime is the build timestamp (ISO 8601 format)
	BuildTime = "unknown"
)
