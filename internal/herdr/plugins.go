package herdr

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
)

const PlusPluginID = "cloudmanic.herdr-plus"

func (c *Client) HasHerdrPlus(ctx context.Context) (bool, error) {
	data, err := c.Runner.Run(ctx, "plugin", "list", "--json")
	if err != nil {
		return false, err
	}
	var decoded interface{}
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&decoded); err != nil {
		return false, err
	}
	return containsPluginID(decoded, PlusPluginID), nil
}

func containsPluginID(value interface{}, expected string) bool {
	switch item := value.(type) {
	case map[string]interface{}:
		for key, child := range item {
			if (key == "id" || key == "plugin_id") && strings.EqualFold(strings.TrimSpace(stringValue(child)), expected) {
				return true
			}
			if containsPluginID(child, expected) {
				return true
			}
		}
	case []interface{}:
		for _, child := range item {
			if containsPluginID(child, expected) {
				return true
			}
		}
	case string:
		return strings.EqualFold(strings.TrimSpace(item), expected)
	}
	return false
}

func stringValue(value interface{}) string {
	text, _ := value.(string)
	return text
}
