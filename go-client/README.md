# Poker Go-Client

Starter-Client mit "immer CALL"-Strategie als Ausgangspunkt für deinen
eigenen Bot. Deine Strategie implementierst du in
`internal/bot/bot.go`.

Voraussetzung: Go 1.25+.

## Starten

```bash
go run ./main.go localhost:8083 Alice
```

Alternativ über `make`:

```bash
make run ARGS="localhost:8083 Alice"
```


