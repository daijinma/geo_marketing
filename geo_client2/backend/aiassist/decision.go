package aiassist

import (
	"encoding/json"
	"fmt"
)

type Decision struct {
	Action     string  `json:"action"`
	Selector   string  `json:"selector,omitempty"`
	Value      string  `json:"value,omitempty"`
	MS         int     `json:"ms,omitempty"`
	Reason     string  `json:"reason,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

func ParseDecision(outputs map[string]interface{}) (*Decision, error) {
	if outputs == nil {
		return nil, fmt.Errorf("empty outputs")
	}

	var raw interface{}
	if v, ok := outputs["output"]; ok {
		raw = v
	} else if v, ok := outputs["text"]; ok {
		raw = v
	} else if v, ok := outputs["answer"]; ok {
		raw = v
	} else {
		return nil, fmt.Errorf("no decision output field")
	}

	// output may already be a JSON object or a JSON string
	switch t := raw.(type) {
	case map[string]interface{}:
		buf, _ := json.Marshal(t)
		var d Decision
		if err := json.Unmarshal(buf, &d); err != nil {
			return nil, fmt.Errorf("decode decision object: %w", err)
		}
		return &d, nil
	case string:
		var d Decision
		if err := json.Unmarshal([]byte(t), &d); err != nil {
			return nil, fmt.Errorf("decode decision string: %w", err)
		}
		return &d, nil
	default:
		return nil, fmt.Errorf("unsupported decision type: %T", raw)
	}
}
