package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// decode decodes a single JSON line written by handleTo into a Response.
func decode(t *testing.T, buf *bytes.Buffer) Response {
	t.Helper()
	var resp Response
	if err := json.NewDecoder(buf).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v (raw: %q)", err, buf.String())
	}
	return resp
}

// call is a helper that builds a tools/call request and returns the response.
func call(t *testing.T, toolName string, args string) Response {
	t.Helper()
	params, _ := json.Marshal(map[string]any{
		"name":      toolName,
		"arguments": json.RawMessage(args),
	})
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}
	var buf bytes.Buffer
	handleTo(&buf, req)
	return decode(t, &buf)
}

// payload unmarshals content[0].text from a successful tool response.
func payload(t *testing.T, resp Response) map[string]any {
	t.Helper()
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	result, _ := resp.Result.(map[string]any)
	contents, _ := result["content"].([]any)
	if len(contents) == 0 {
		t.Fatal("content is empty")
	}
	item, _ := contents[0].(map[string]any)
	text, _ := item["text"].(string)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return out
}

// --- initialize ---

func TestInitialize(t *testing.T) {
	var buf bytes.Buffer
	params, _ := json.Marshal(map[string]any{
		"protocolVersion": "2025-03-26",
		"clientInfo":      map[string]any{"name": "test", "version": "0"},
	})
	handleTo(&buf, Request{JSONRPC: "2.0", ID: 1, Method: "initialize", Params: params})
	resp := decode(t, &buf)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	result := resp.Result.(map[string]any)
	if result["protocolVersion"] != "2025-03-26" {
		t.Errorf("protocolVersion = %v", result["protocolVersion"])
	}
}

// --- tools/list ---

func TestToolsList(t *testing.T) {
	var buf bytes.Buffer
	handleTo(&buf, Request{JSONRPC: "2.0", ID: 1, Method: "tools/list"})
	resp := decode(t, &buf)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	result := resp.Result.(map[string]any)
	tools := result["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("want 2 tools, got %d", len(tools))
	}
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.(map[string]any)["name"].(string)
	}
	for _, want := range []string{"datetime_now", "datetime_now_tz"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Errorf("tool %q not found in list %v", want, names)
		}
	}
}

// --- datetime_now ---

func TestDatetimeNow_UTC(t *testing.T) {
	t.Setenv("TZ", "")
	p := payload(t, call(t, "datetime_now", "{}"))
	if p["timezone"] != "UTC" {
		t.Errorf("timezone = %v, want UTC", p["timezone"])
	}
	for _, key := range []string{"iso", "unix", "date", "time", "offset", "weekday"} {
		if p[key] == nil {
			t.Errorf("missing key %q", key)
		}
	}
}

func TestDatetimeNow_TZEnv(t *testing.T) {
	t.Setenv("TZ", "Asia/Omsk")
	p := payload(t, call(t, "datetime_now", "{}"))
	if p["timezone"] != "Asia/Omsk" {
		t.Errorf("timezone = %v, want Asia/Omsk", p["timezone"])
	}
	offset := p["offset"].(string)
	if !strings.HasPrefix(offset, "+06") {
		t.Errorf("offset = %v, want +06xx", offset)
	}
}

func TestDatetimeNow_IgnoresArguments(t *testing.T) {
	// datetime_now takes no params; any extra JSON in args must be ignored
	t.Setenv("TZ", "Asia/Omsk")
	p := payload(t, call(t, "datetime_now", `{"timezone":"Europe/Moscow"}`))
	if p["timezone"] != "Asia/Omsk" {
		t.Errorf("timezone = %v, want Asia/Omsk (TZ env must win)", p["timezone"])
	}
}

// --- datetime_now_tz ---

func TestDatetimeNowTz_ValidTimezone(t *testing.T) {
	p := payload(t, call(t, "datetime_now_tz", `{"timezone":"America/New_York"}`))
	if p["timezone"] != "America/New_York" {
		t.Errorf("timezone = %v", p["timezone"])
	}
	offset := p["offset"].(string)
	// New York is UTC-4 (EDT) or UTC-5 (EST)
	if !strings.HasPrefix(offset, "-04") && !strings.HasPrefix(offset, "-05") {
		t.Errorf("unexpected offset %v for America/New_York", offset)
	}
}

func TestDatetimeNowTz_MissingTimezone(t *testing.T) {
	resp := call(t, "datetime_now_tz", `{}`)
	if resp.Error == nil {
		t.Fatal("expected error for missing timezone, got nil")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("error code = %d, want -32602", resp.Error.Code)
	}
}

func TestDatetimeNowTz_InvalidTimezone(t *testing.T) {
	resp := call(t, "datetime_now_tz", `{"timezone":"Not/Real"}`)
	if resp.Error == nil {
		t.Fatal("expected error for invalid timezone")
	}
	if !strings.Contains(resp.Error.Message, "Not/Real") {
		t.Errorf("error message %q should contain timezone name", resp.Error.Message)
	}
}

// --- unknown tool / method ---

func TestUnknownTool(t *testing.T) {
	resp := call(t, "datetime_what", `{}`)
	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("code = %d, want -32602", resp.Error.Code)
	}
}

func TestUnknownMethod(t *testing.T) {
	var buf bytes.Buffer
	handleTo(&buf, Request{JSONRPC: "2.0", ID: 1, Method: "rpc.unknown"})
	resp := decode(t, &buf)
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("code = %d, want -32601", resp.Error.Code)
	}
}
