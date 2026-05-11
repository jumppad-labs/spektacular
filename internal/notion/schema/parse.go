package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// ParseDataSource parses either a direct data-source-state JSON object or the
// JSON/text wrapper returned by the Notion MCP fetch tool.
func ParseDataSource(raw []byte) (DataSource, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return DataSource{}, fmt.Errorf("empty data source snapshot")
	}

	if ds, ok := parseDirect(raw); ok {
		return ds, nil
	}

	var textWrapper struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &textWrapper); err == nil && textWrapper.Text != "" {
		return ParseDataSource([]byte(textWrapper.Text))
	}

	var mcpItems []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &mcpItems); err == nil {
		for _, item := range mcpItems {
			if item.Text == "" {
				continue
			}
			ds, err := ParseDataSource([]byte(item.Text))
			if err == nil {
				return ds, nil
			}
		}
	}

	if state := extractBetween(string(raw), "<data-source-state>", "</data-source-state>"); state != "" {
		if ds, ok := parseDirect([]byte(state)); ok {
			return ds, nil
		}
	}

	return DataSource{}, fmt.Errorf("unsupported Notion data source snapshot")
}

func parseDirect(raw []byte) (DataSource, bool) {
	var ds DataSource
	if err := json.Unmarshal(raw, &ds); err != nil {
		return DataSource{}, false
	}
	if ds.URL == "" && len(ds.Schema) == 0 {
		return DataSource{}, false
	}
	normalizePropertyNames(ds.Schema)
	return ds, true
}

func normalizePropertyNames(props map[string]Property) {
	for name, prop := range props {
		if prop.Name == "" {
			prop.Name = name
			props[name] = prop
		}
	}
}

func extractBetween(s, start, end string) string {
	startIdx := strings.Index(s, start)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(start)
	endIdx := strings.Index(s[startIdx:], end)
	if endIdx == -1 {
		return ""
	}
	return s[startIdx : startIdx+endIdx]
}
