package domain

import (
	"encoding/json"
	"fmt"
)

// replyObject reads an AI reply as one JSON object.
func replyObject(reply string) (map[string]any, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(reply), &raw); err != nil {
		return nil, fmt.Errorf("khối JSON không đọc được (%v)", err)
	}
	return raw, nil
}
