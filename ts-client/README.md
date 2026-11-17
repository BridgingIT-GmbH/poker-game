# Poker TypeScript-Client

# Poker TypeScript-Client

Starter-Client mit "immer CALL"-Strategie als Ausgangspunkt für deinen
eigenen Bot. Deine Strategie implementierst du in `src/bot.ts`, Methode `act`.

Voraussetzung: Node.js 18+ und npm.

## Starten

```bash
npm install
npx tsc
node dist/main.js localhost:8083 Alice
```

Alternativ über `make`:

```bash
make run ARGS="localhost:8083 Alice"
```
