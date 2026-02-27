package aiassist

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type DifyClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type WorkflowRequest struct {
	Inputs       map[string]interface{} `json:"inputs"`
	ResponseMode string                 `json:"response_mode"`
	User         string                 `json:"user"`
}

// sseEvent represents a single SSE event parsed from the stream.
type sseEvent struct {
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"-"`
	raw   string
}

func NewDifyClient(baseURL, apiKey string) (*DifyClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("dify base url is empty")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("dify api key is empty")
	}
	return &DifyClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

func (c *DifyClient) RunWorkflow(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
	reqBody := WorkflowRequest{
		Inputs:       inputs,
		ResponseMode: "streaming",
		User:         "geo_client2",
	}

	buf, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/workflows/run"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dify request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dify response status: %s", resp.Status)
	}

	// Parse SSE stream line by line, looking for workflow_finished event.
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var envelope struct {
			Event string                 `json:"event"`
			Data  map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
			// Skip malformed lines
			continue
		}

		if envelope.Event == "workflow_finished" {
			// data.outputs contains the result
			if envelope.Data == nil {
				return nil, fmt.Errorf("workflow_finished event has no data")
			}
			status, _ := envelope.Data["status"].(string)
			if status != "" && status != "succeeded" {
				return nil, fmt.Errorf("workflow status: %s", status)
			}
			outputs, _ := envelope.Data["outputs"].(map[string]interface{})
			if outputs == nil {
				// outputs may be absent for workflows with no output variables
				outputs = make(map[string]interface{})
			}
			return outputs, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading SSE stream: %w", err)
	}

	return nil, fmt.Errorf("stream ended without workflow_finished event")
}
