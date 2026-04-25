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
	JSONRPC string     `json:"jsonrpc"`
	ID      any        `json:"id,omitempty"`
	Result  any        `json:"result,omitempty"`
	Error   *RespError `json:"error,omitempty"`
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

type NowTzArgs struct {
	Timezone string `json:"timezone"`
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
			writeTo(os.Stdout, Response{JSONRPC: "2.0", Error: &RespError{Code: -32700, Message: "parse error"}})
			continue
		}

		handleTo(os.Stdout, req)
	}

	if err := reader.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func handleTo(w io.Writer, req Request) {
	switch req.Method {
	case "initialize":
		writeTo(w, Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2025-03-26",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "time-stdio-go",
					"version": "0.2.0",
				},
			},
		})
	case "notifications/initialized":
		return
	case "tools/list":
		writeTo(w, Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": []Tool{datetimeNowTool(), datetimeNowTzTool()},
			},
		})
	case "tools/call":
		handleToolCallTo(w, req)
	default:
		writeTo(w, Response{JSONRPC: "2.0", ID: req.ID, Error: &RespError{Code: -32601, Message: "method not found"}})
	}
}

func datetimeNowTool() Tool {
	return Tool{
		Name:        "datetime_now",
		Description: "Return current date and time in the server's configured timezone (TZ env var, fallback UTC). Takes no parameters.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func datetimeNowTzTool() Tool {
	return Tool{
		Name:        "datetime_now_tz",
		Description: "Return current date and time in the given IANA timezone, e.g. America/New_York.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timezone": map[string]any{
					"type":        "string",
					"description": "IANA timezone name, e.g. America/New_York or Asia/Tokyo",
				},
			},
			"required": []string{"timezone"},
		},
	}
}

func handleToolCallTo(w io.Writer, req Request) {
	var p CallToolParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		writeTo(w, Response{JSONRPC: "2.0", ID: req.ID, Error: &RespError{Code: -32602, Message: "invalid params"}})
		return
	}

	switch p.Name {
	case "datetime_now":
		locName := strings.TrimSpace(os.Getenv("TZ"))
		if locName == "" {
			locName = "UTC"
		}
		writeTimeTo(w, req.ID, locName)

	case "datetime_now_tz":
		var args NowTzArgs
		if err := json.Unmarshal(p.Arguments, &args); err != nil || strings.TrimSpace(args.Timezone) == "" {
			writeTo(w, Response{JSONRPC: "2.0", ID: req.ID, Error: &RespError{Code: -32602, Message: "timezone is required"}})
			return
		}
		writeTimeTo(w, req.ID, strings.TrimSpace(args.Timezone))

	default:
		writeTo(w, Response{JSONRPC: "2.0", ID: req.ID, Error: &RespError{Code: -32602, Message: "unknown tool: " + p.Name}})
	}
}

func writeTimeTo(w io.Writer, id any, locName string) {
	loc, err := time.LoadLocation(locName)
	if err != nil {
		writeTo(w, Response{JSONRPC: "2.0", ID: id, Error: &RespError{Code: -32602, Message: "invalid timezone: " + locName}})
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
	writeTo(w, Response{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"content": []TextContent{{Type: "text", Text: string(b)}},
		},
	})
}

func writeTo(w io.Writer, v any) {
	_ = json.NewEncoder(w).Encode(v)
}
