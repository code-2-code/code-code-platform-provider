package providerobservability

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func decodeGoogleAIStudioRPCBody(body []byte) ([]any, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, fmt.Errorf("empty response body")
	}
	var direct []any
	if err := json.Unmarshal([]byte(trimmed), &direct); err == nil {
		return direct, nil
	}
	var encoded string
	if err := json.Unmarshal([]byte(trimmed), &encoded); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(encoded)
	}
	if err != nil {
		return nil, fmt.Errorf("decode base64 response: %w", err)
	}
	if err := json.Unmarshal(decoded, &direct); err != nil {
		return nil, fmt.Errorf("decode wrapped json response: %w", err)
	}
	return direct, nil
}

func googleAIStudioPayloadRows(payload []any) ([]any, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("payload is empty")
	}
	rows, ok := payload[0].([]any)
	if !ok {
		return nil, fmt.Errorf("payload[0] is %T, want []any", payload[0])
	}
	return rows, nil
}

func googleAIStudioPayloadRow(item any) ([]any, bool) {
	row, ok := item.([]any)
	return row, ok
}

func googleAIStudioStringAt(row []any, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(stringFromAny(row[index]))
}

func googleAIStudioIntAt(row []any, index int) (int, bool) {
	if index < 0 || index >= len(row) {
		return 0, false
	}
	value, ok := numberFromAny(row[index])
	return int(value), ok
}
