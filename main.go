package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      any         `json:"id,omitempty"`
	Result  any         `json:"result,omitempty"`
	Error   *RespError  `json:"error,omitempty"`
}

type RespError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type NowArgs struct {
	Timezone string `json:"timezone,omitempty"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func main() {
	log.SetOutput(io.Discard)
	reader := bufio.NewScanner(os.Stdin)

	for reader.Scan() {
		line := strings.TrimSpace(reader.Text())
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			write(Response{JSONRPC: "2.0", Error: &RespError{Code: -32700, Message: "parse error"}})
			continue
		}

		handle(req)
	}

	if err := reader.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func handle(req Request) {
	switch req.Method {
	case "initialize":
		write(Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2025-03-26",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "time-stdio-go",
					"version": "0.1.0",
				},
			},
		})
	case "notifications/initialized":
		return
	case "tools/list":
		write(Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": []Tool{nowTool()},
			},
		})
	case "tools/call":
		handleToolCall(req)
	default:
		write(Response{JSONRPC: "2.0", ID: req.ID, Error: &RespError{Code: -32601, Message: "method not found"}})
	}
}

func nowTool() Tool {
	return Tool{
		Name:        "now",
		Description: "Return current date and time. If timezone is omitted, use TZ env var or UTC.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timezone": map[string]any{
					"type":        "string",
					"description": "IANA timezone, for example Europe/Moscow",
				},
			},
		},
	}
}

func handleToolCall(req Request) {
	var p CallToolParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		write(Response{JSONRPC: "2.0", ID: req.ID, Error: &RespError{Code: -32602, Message: "invalid params"}})
		return
	}

	if p.Name != "now" {
		write(Response{JSONRPC: "2.0", ID: req.ID, Error: &RespError{Code: -32602, Message: "unknown tool"}})
		return
	}

	var args NowArgs
	if len(p.Arguments) > 0 {
		if err := json.Unmarshal(p.Arguments, &args); err != nil {
			write(Response{JSONRPC: "2.0", ID: req.ID, Error: &RespError{Code: -32602, Message: "invalid tool arguments"}})
			return
		}
	}

	locName := strings.TrimSpace(args.Timezone)
	if locName == "" {
		locName = strings.TrimSpace(os.Getenv("TZ"))
	}
	if locName == "" {
		locName = "UTC"
	}

	loc, err := time.LoadLocation(locName)
	if err != nil {
		write(Response{JSONRPC: "2.0", ID: req.ID, Error: &RespError{Code: -32602, Message: "invalid timezone: " + locName}})
		return
	}

	now := time.Now().In(loc)
	payload := map[string]any{
		"timezone": locName,
		"iso":      now.Format(time.RFC3339),
		"unix":     now.Unix(),
		"date":     now.Format("2006-01-02"),
		"time":     now.Format("15:04:05"),
		"offset":   now.Format("-0700"),
		"weekday":  now.Weekday().String(),
	}

	b, _ := json.Marshal(payload)
	write(Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"content": []TextContent{{Type: "text", Text: string(b)}},
		},
	})
}

func write(v any) {
	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(v)
}
