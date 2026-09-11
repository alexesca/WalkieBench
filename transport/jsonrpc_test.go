package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"walkiebench/contract"
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

func TestV2JSONRPCMethodsStayOnContractSurface(t *testing.T) {
	var method string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		method = request.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"id":"srv-1","name":"demo","owner_id":"agent-a","join_policy":"public"}}`))
	}))
	defer server.Close()
	client := New(Config{Endpoint: server.URL})
	got, err := client.CreateServer(context.Background(), contract.ServerSpec{Name: "demo", JoinPolicy: contract.JoinPublic})
	if err != nil {
		t.Fatal(err)
	}
	if method != "CreateServer" || got.ID != "srv-1" || got.JoinPolicy != contract.JoinPublic {
		t.Fatalf("method=%q result=%+v", method, got)
	}
}
