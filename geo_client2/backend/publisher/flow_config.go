package publisher

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"geo_client2/backend/config"
)

//go:embed flows/*.json
var embeddedFlows embed.FS

type Flow struct {
	SchemaVersion int        `json:"schemaVersion"`
	Platform      string     `json:"platform"`
	Steps         []FlowStep `json:"steps"`
}

type FlowStep struct {
	ID           string   `json:"id"`
	Action       string   `json:"action"`
	Optional     bool     `json:"optional,omitempty"`
	TimeoutMS    int      `json:"timeoutMs,omitempty"`
	URL          string   `json:"url,omitempty"`
	MS           int      `json:"ms,omitempty"`
	Selector     string   `json:"selector,omitempty"`
	Regex        string   `json:"regex,omitempty"`
	Value        string   `json:"value,omitempty"`
	Script       string   `json:"script,omitempty"`
	Args         []string `json:"args,omitempty"`
	Files        []string `json:"files,omitempty"`
	Mode         string   `json:"mode,omitempty"` // input|value|innerText|innerHTML
	ClickFirst   bool     `json:"clickFirst,omitempty"`
	Frame        string   `json:"frameSelector,omitempty"`
	Prompt       string   `json:"prompt,omitempty"`       // shown to user for needs_manual action
	CheckSuccess bool     `json:"checkSuccess,omitempty"` // if true, eval must return {success:true} or error is raised
}

func flowOverridePath(platform string) (string, error) {
	if strings.TrimSpace(platform) == "" {
		return "", fmt.Errorf("empty platform")
	}
	// ~/.geo_client2/flows/publish/{platform}.json
	return filepath.Join(config.GetAppDir(), "flows", "publish", platform+".json"), nil
}

func LoadPublishFlow(platform string) (*Flow, error) {
	// 1) local override (server-downloaded / user-provided)
	if p, err := flowOverridePath(platform); err == nil {
		if b, readErr := os.ReadFile(p); readErr == nil {
			f := &Flow{}
			if err := json.Unmarshal(b, f); err != nil {
				return nil, fmt.Errorf("parse publish flow override %s: %w", p, err)
			}
			return validateFlow(platform, f)
		}
	}

	// 2) embedded default flow
	path := filepath.ToSlash(filepath.Join("flows", platform+".json"))
	b, err := embeddedFlows.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("publish flow not found for platform=%s", platform)
	}
	f := &Flow{}
	if err := json.Unmarshal(b, f); err != nil {
		return nil, fmt.Errorf("parse embedded publish flow %s: %w", path, err)
	}
	return validateFlow(platform, f)
}

func validateFlow(platform string, f *Flow) (*Flow, error) {
	if f == nil {
		return nil, fmt.Errorf("nil flow")
	}
	if f.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported flow schemaVersion=%d", f.SchemaVersion)
	}
	if f.Platform == "" {
		f.Platform = platform
	}
	if f.Platform != platform {
		return nil, fmt.Errorf("flow platform mismatch: want=%s got=%s", platform, f.Platform)
	}
	if len(f.Steps) == 0 {
		return nil, fmt.Errorf("empty flow steps")
	}
	for i, s := range f.Steps {
		if strings.TrimSpace(s.ID) == "" {
			return nil, fmt.Errorf("flow step[%d] missing id", i)
		}
		if strings.TrimSpace(s.Action) == "" {
			return nil, fmt.Errorf("flow step[%d] missing action", i)
		}
	}
	return f, nil
}

func interpolateValue(raw string, article Article, tempVars map[string]string) string {
	out := raw
	out = strings.ReplaceAll(out, "{{title}}", article.Title)
	out = strings.ReplaceAll(out, "{{content}}", article.Content)
	out = strings.ReplaceAll(out, "{{cover_image}}", article.CoverImage)
	for k, v := range tempVars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

func interpolateScript(raw string, article Article, tempVars map[string]string) string {
	out := raw
	out = strings.ReplaceAll(out, "{{title}}", article.Title)
	out = strings.ReplaceAll(out, "{{cover_image}}", article.CoverImage)
	for k, v := range tempVars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}
