package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func WrapUserError(message string, err error) error {
	if err == nil {
		return fmt.Errorf(message)
	}
	return fmt.Errorf("%s: %w", message, err)
}

func WrapHTTPError(message string, statusCode int, body []byte) error {
	msg := fmt.Sprintf("%s: status=%d", message, statusCode)
	if compact, ok := compactJSON(body); ok {
		return fmt.Errorf("%s, body=%s", msg, compact)
	}
	return fmt.Errorf(msg)
}

func compactJSON(body []byte) (string, bool) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return "", false
	}

	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return "", false
	}

	compact, err := json.Marshal(v)
	if err != nil {
		return "", false
	}
	return string(compact), true
}
