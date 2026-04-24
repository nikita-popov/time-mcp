# time-mcp-go

Минимальный MCP server на Go.

Особенности:
- transport: stdio;
- один tool: `now`;
- конфиг timezone через переменную окружения `TZ`;
- если `timezone` не передан в аргументах tool, используется `TZ`, иначе `UTC`.

## Запуск

```bash
TZ=Europe/Moscow go run .
```

или:

```bash
go build -o time-mcp-go .
TZ=Europe/Moscow ./time-mcp-go
```

## Пример Claude Desktop / совместимого клиента

```json
{
  "mcpServers": {
    "time": {
      "command": "/absolute/path/to/time-mcp-go",
      "env": {
        "TZ": "Europe/Moscow"
      }
    }
  }
}
```

## Tool

### `now`

Аргументы:

```json
{
  "timezone": "Europe/Moscow"
}
```

`timezone` опционален.

## Ответ

Tool возвращает JSON строкой в `content[0].text`, например:

```json
{
  "timezone": "Europe/Moscow",
  "iso": "2026-04-24T16:52:00+03:00",
  "unix": 1777038720,
  "date": "2026-04-24",
  "time": "16:52:00",
  "offset": "+0300",
  "weekday": "Friday"
}
```

## Почему так

Это максимально простой вариант: без внешних зависимостей, без HTTP, без лишнего конфига. Один бинарник, один env var, stdin/stdout. Подход хорошо ложится на UNIX/KISS.
