# time-mcp

Minimal MCP server written in Go that provides current date and time to language models.

**Transport:** stdio  
**Dependencies:** none (stdlib only)  
**Config:** `TZ` environment variable

## Tools

### `datetime_now`

Returns current date and time in the server's configured timezone. Takes no parameters — the timezone is controlled exclusively by the `TZ` env var.

```json
{}
```

### `datetime_now_tz`

Returns current date and time in an arbitrary IANA timezone. Useful when the model needs to show time in a different location without affecting the server's default.

```json
{ "timezone": "America/New_York" }
```

`timezone` is **required**.

## Response format

Both tools return a JSON object as a text string in `content[0].text`:

```json
{
  "timezone": "Asia/Omsk",
  "iso":      "2026-04-24T20:52:00+06:00",
  "unix":     1777038720,
  "date":     "2026-04-24",
  "time":     "20:52:00",
  "offset":   "+0600",
  "weekday":  "Friday"
}
```

## Usage

```bash
TZ=Asia/Omsk go run .
```

or build first:

```bash
go build -o time-mcp .
TZ=Asia/Omsk ./time-mcp
```

## Client configuration

### Claude Desktop

```json
{
  "mcpServers": {
    "time": {
      "command": "/absolute/path/to/time-mcp",
      "env": {
        "TZ": "Asia/Omsk"
      }
    }
  }
}
```

### Any stdio MCP client

Point the client at the binary and set `TZ` in the process environment. No other configuration needed.

## Why

Language models have no reliable access to the current time. This server provides it as an explicit tool call — a clean external dependency rather than an internal guess. One binary, one env var, stdin/stdout.
