# Poker Java Client

Ein reiner Java-Client (kein Build-Tool, keine externen Dependencies –
nur JDK) mit derselben Netzwerk-/Protokoll-Logik wie der Go-Client unter
`../client`. Er verbindet sich per UDP mit dem Poker-Server und implementiert
standardmäßig eine simple "immer CALL"-Strategie, die als Ausgangspunkt für
einen eigenen Poker-Bot dient.

## Struktur

- `Protocol.java` – Protokoll-Konstanten (siehe `../server/pkg/protocol/protocol.go`
  und `../server/MESSAGES.md`).
- `Connection.java` – Netzwerk-Plumbing (UDP-Socket, Empfangsschleife, Senden,
  Parsing-Helfer). Muss i. d. R. nicht verändert werden.
- `Bot.java` – Nachrichtenverarbeitung und Entscheidungslogik. **Hier baust du
  deine eigene Poker-Strategie ein** (Methode `act`).
- `Main.java` – Einstiegspunkt, parst CLI-Argumente und startet den Bot.

## Bauen & Starten

Voraussetzung: JDK 17+ (keine weiteren Abhängigkeiten).

```bash
# Bauen
make build

# Starten (Standardname "CallBot")
make run ARGS="localhost:8083"

# Starten mit eigenem Namen
make run ARGS="localhost:8083 Alice"
```

Ohne `make` funktioniert es auch direkt – plattformunabhängig (Linux/macOS/Windows),
da `javac` über `-sourcepath` alle von `Main.java` referenzierten Klassen automatisch
mitkompiliert (kein `find`, kein Glob nötig):

```bash
# Linux / macOS
mkdir -p out
javac -d out -sourcepath src/main/java src/main/java/pokergame/Main.java
java -cp out pokergame.Main localhost:8083 Alice
```

```powershell
# Windows (PowerShell oder cmd)
mkdir out
javac -d out -sourcepath src\main\java src\main\java\pokergame\Main.java
java -cp out pokergame.Main localhost:8083 Alice
```
