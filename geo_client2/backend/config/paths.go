package config

import (
	"os"
	"path/filepath"
)

// AppName is the name of the application, used for directory naming.
// This can be changed to white-label the application.
var AppName = "duanjiegeo"

// GetAppDir returns the root directory for the application configuration and data.
// Default: ~/.geo_client2/
func GetAppDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home cannot be determined
		return "." + AppName
	}
	return filepath.Join(home, "."+AppName)
}

// GetDBPath returns the full path to the database file.
// Default: ~/.geo_client2/cache.db
func GetDBPath() string {
	return filepath.Join(GetAppDir(), "cache.db")
}

// GetLogDir returns the directory for log files.
// Default: ~/.geo_client2/logs/
func GetLogDir() string {
	return filepath.Join(GetAppDir(), "logs")
}

// GetBrowserDataDir returns the root directory for all browser profiles.
// Default: ~/.geo_client2/browser_data/
func GetBrowserDataDir() string {
	return filepath.Join(GetAppDir(), "browser_data")
}
