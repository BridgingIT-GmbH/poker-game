# Poker Game

Dieses Projekt bietet eine Umgebung für kompetitive Poker-Partien, bei denen
mehrere Entwickler jeweils einen eigenen Poker-Bot schreiben, der gegen die
Bots der anderen Teilnehmer antritt. Ein zentraler Server übernimmt die
Spiellogik und kommuniziert mit den Bots über UDP.

## Einstieg

1. **Eigenen Bot wählen/bauen** – es stehen Starter-Clients in drei Sprachen
   zur Verfügung, die jeweils eine einfache "immer CALL"-Strategie
   implementieren und als Ausgangspunkt dienen:
   - [Go-Client](go-client/README.md)
   - [Java-Client](java-client/README.md)
   - [TypeScript-Client](ts-client/README.md)
2. **Eigene Strategie implementieren**, indem die Entscheidungslogik des
   jeweiligen Clients angepasst wird (siehe verlinkte Client-READMEs für
   Details zu Struktur, Bauen und Starten).
3. Mit dem Bot gegen den (bereits laufenden) Server antreten – die
   Server-Adresse wird beim Start des Clients angegeben.

## Protokoll

Die Kommunikation zwischen Server und Bots erfolgt über ein einfaches,
textbasiertes UDP-Protokoll (`;`-getrennte Felder). Details und alle
Nachrichtentypen sind in [`MESSAGES.md`](MESSAGES.md) beschrieben.

## Spielregeln & Limitierungen

- Gespielt wird **Texas Hold'em** mit den Phasen Pre-Flop, Flop, Turn, River
  und Showdown.
- Pro Spieler ist **nur 1 Raise pro Betting-Runde (Street)** erlaubt. Ein
  Raise erhöht den aktuellen Einsatz immer um genau **einen Big Blind**
  (keine freie Raise-Höhe).
- **All-in wird nicht unterstützt**: Ein Call/Raise ist nur erlaubt, wenn er
  vollständig mit dem Guthaben gedeckt ist. Reicht das Guthaben exakt (Chips
  danach = 0), ist die Aktion gültig. Reicht es nicht aus, wird der Bot
  automatisch gefoldet.
- Ungültige oder zum falschen Zeitpunkt gesendete Aktionen (z. B. `CHECK`
  trotz offenem Einsatz, oder ein zweiter `RAISE` in derselben Street)
  führen ebenfalls zum automatischen Fold.
