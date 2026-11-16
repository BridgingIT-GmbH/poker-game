# Poker TypeScript-Client

Ein TypeScript-Client mit derselben Netzwerk-/Protokoll-Logik wie der Go-Client
unter `../client`. Er verbindet
sich per UDP mit dem Poker-Server und implementiert standardmäßig eine simple
"immer CALL"-Strategie, die als Ausgangspunkt für einen eigenen Poker-Bot dient.

TypeScript wird mit dem offiziellen Compiler (`tsc`) zu reinem JavaScript
kompiliert. `typescript` und `@types/node` sind **Dev-Dependencies**, die nur
für den Build gebraucht werden – der kompilierte Bot (`dist/`) hat zur
Laufzeit **keine** Abhängigkeiten und nutzt ausschließlich Node-Core-Module
(`node:dgram`).

## Struktur

- `src/protocol.ts` – Protokoll-Konstanten (siehe
  `../server/pkg/protocol/protocol.go` und `../server/MESSAGES.md`).
- `src/connection.ts` – Netzwerk-Plumbing (UDP-Socket, Empfangsschleife,
  Senden, Parsing-Helfer). Muss i. d. R. nicht verändert werden.
- `src/bot.ts` – Nachrichtenverarbeitung und Entscheidungslogik. **Hier baust
  du deine eigene Poker-Strategie ein** (Methode `act`).
- `src/main.ts` – Einstiegspunkt, parst CLI-Argumente und startet den Bot.

## Voraussetzungen

- Node.js 18+ (nur zur Laufzeit; für den Build reicht jede Node-Version, die
  npm mitbringt)
- npm (zum einmaligen Installieren von `typescript`/`@types/node`)

## Bauen & Starten

```bash
# Abhängigkeiten installieren (einmalig, nur Dev-Dependencies)
npm install

# Bauen (TypeScript -> dist/*.js)
make build

# Starten (Standardname "CallBot")
make run ARGS="localhost:8083"

# Starten mit eigenem Namen
make run ARGS="localhost:8083 Alice"
```

Ohne `make` funktioniert es auch direkt, plattformunabhängig:

```bash
npm install
npx tsc
node dist/main.js localhost:8083 Alice
```

## Eigene Strategie

Die Entscheidungslogik befindet sich ausschließlich in `src/bot.ts`, Methode
`act(token)`. Dort kannst du anhand der zuvor empfangenen Nachrichten
(`CARDS`, `BOARD`, ...) eigene Felder in der `Bot`-Klasse pflegen und eine
eigene Strategie statt des Standard-`CALL` implementieren.
