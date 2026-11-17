# Poker Java Client

# Poker Java Client

Starter-Client mit "immer CALL"-Strategie als Ausgangspunkt für deinen
eigenen Bot. Deine Strategie implementierst du in `Bot.java`, Methode `act`.

Voraussetzung: JDK 17+ (keine weiteren Abhängigkeiten).

## Starten

```bash
mkdir -p out
javac -d out -sourcepath src/main/java src/main/java/pokergame/Main.java
java -cp out pokergame.Main localhost:8083 Alice
```

Alternativ über `make`:

```bash
make run ARGS="localhost:8083 Alice"
```
