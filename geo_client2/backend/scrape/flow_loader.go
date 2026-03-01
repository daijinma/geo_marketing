package scrape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func flowPath(version, platform string) string {
	return filepath.Join(bundlesDir(), version, "flows", platform+".json")
}

// LoadScrapeFlow loads the active scrape flow for a platform.
// Returns (nil, "", nil) if no active version or flow file exists.
func LoadScrapeFlow(platform string) (*ScrapeFlow, string, error) {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return nil, "", fmt.Errorf("empty platform")
	}
	active, err := readActiveVersion()
	if err != nil {
		return nil, "", err
	}
	if active == "" {
		return nil, "", nil
	}
	path := flowPath(active, platform)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("read flow %s: %w", path, err)
	}
	var flow ScrapeFlow
	if err := json.Unmarshal(b, &flow); err != nil {
		return nil, "", fmt.Errorf("parse flow %s: %w", path, err)
	}
	if flow.SchemaVersion != 1 {
		return nil, "", fmt.Errorf("unsupported flow schemaVersion=%d", flow.SchemaVersion)
	}
	if strings.TrimSpace(flow.Platform) == "" {
		flow.Platform = platform
	}
	if flow.Platform != platform {
		return nil, "", fmt.Errorf("flow platform mismatch: want=%s got=%s", platform, flow.Platform)
	}
	if len(flow.Pipeline) == 0 {
		return nil, "", fmt.Errorf("flow pipeline empty for platform=%s", platform)
	}
	return &flow, active, nil
}
