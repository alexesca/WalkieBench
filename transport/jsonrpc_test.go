package transport

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSecureWireRoundTrip(t *testing.T) {
	const token = "test-session-token"
	input := map[string]any{
		"content": "wire-secret-marker",
		"nested":  map[string]any{"title": "nested-title"},
	}
	sealed, err := protectJSON(input, token, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sealed), "wire-secret-marker") || strings.Contains(string(sealed), "nested-title") {
		t.Fatalf("secure payload contains plaintext: %s", sealed)
	}
	opened, err := protectJSON(json.RawMessage(sealed), token, false)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(opened, &got); err != nil {
		t.Fatal(err)
	}
	if got["content"] != input["content"] || got["nested"].(map[string]any)["title"] != "nested-title" {
		t.Fatalf("round trip = %#v", got)
	}
}
