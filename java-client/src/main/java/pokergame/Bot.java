package pokergame;

import java.io.IOException;
import java.util.Arrays;
import java.util.List;
import java.util.logging.Logger;

/**
 * Bot handles all network communication and protocol details.
 * Players are free to implement their own decision logic directly inside
 * act() below (see the CMD_YOUR_TURN case in handleMessage).
 */
public final class Bot {

    private static final Logger LOGGER = Logger.getLogger(Bot.class.getName());

    private final String name;
    private final Connection connection;

    public Bot(String name, Connection connection) {
        this.name = name;
        this.connection = connection;
    }

    /** Registers the bot with the server. */
    public void connect() throws IOException {
        connection.connect();
    }

    /** Starts the blocking receive loop. */
    public void run() throws IOException {
        connection.run(this::handleMessage);
    }

    // ─── Message dispatcher ─────────────────────────────────────────────

    /**
     * Routes an incoming server message to the appropriate handler.
     * Handles both server->client commands and server-broadcast player
     * actions (FOLD/CALL/RAISE/CHECK with the acting player's name as payload).
     */
    void handleMessage(String raw) {
        // Split into at most 3 parts to handle ROUND STARTED;roundNum;player1,player2
        String[] parts = raw.split(";", 3);
        String cmd = parts[0];
        String payload = parts.length > 1 ? parts[1] : "";

        switch (cmd) {
            // ── server -> client ─────────────────────────────────────────
            case Protocol.CMD_REGISTERED:
                LOGGER.info("Registered successfully");
                break;

            case Protocol.CMD_REJECTED:
                LOGGER.warning("Rejected by server: reason=" + payload);
                break;

            case Protocol.CMD_ROUND_STARTING:
                connection.mustSend(Protocol.CMD_JOIN + ";" + payload);
                LOGGER.info("Sent JOIN");
                break;

            case Protocol.CMD_ROUND_STARTED: {
                // format: ROUND STARTED;<roundNumber>;<player1,player2,...>
                String playerList = parts.length > 2 ? parts[2] : "";
                List<String> players = Arrays.asList(playerList.split(","));
                LOGGER.info("Round started: round=" + payload + " players=" + players);
                break;
            }

            case Protocol.CMD_CARDS:
                LOGGER.info("Hole cards received: cards=" + Connection.parseCards(payload));
                break;

            case Protocol.CMD_BOARD:
                LOGGER.info("Board updated: cards=" + Connection.parseCards(payload));
                break;

            case Protocol.CMD_SCORE:
                LOGGER.info("Score received: result=" + payload);
                break;

            case Protocol.CMD_YOUR_TURN:
                act(payload);
                break;

            // ── server-broadcast player actions ─────────────────────────
            case Protocol.CMD_FOLD:
                LOGGER.info("Player folded: player=" + payload);
                break;

            case Protocol.CMD_CALL:
                LOGGER.info("Player called: player=" + payload);
                break;

            case Protocol.CMD_RAISE:
                LOGGER.info("Player raised: player=" + payload);
                break;

            case Protocol.CMD_CHECK:
                LOGGER.info("Player checked: player=" + payload);
                break;

            default:
                LOGGER.fine("Unknown message: raw=" + raw);
        }
    }

    // ╔════════════════════════════════════════════════════════════════════╗
    // ║  HIER GEHT'S LOS – baue hier deine eigene Poker-Strategie ein!     ║
    // ╚════════════════════════════════════════════════════════════════════╝
    //
    // Verfügbare Aktionen (Protocol-Klasse):
    //
    //   Protocol.CMD_FOLD   – Hand aufgeben
    //   Protocol.CMD_CHECK  – passen (nur gültig, wenn kein Einsatz aussteht)
    //   Protocol.CMD_CALL   – aktuellen Einsatz mitgehen (Standard)
    //   Protocol.CMD_RAISE  – Einsatz erhöhen
    //
    // Tipp: In handleMessage() oben siehst du alle Nachrichten, die der Server
    // schickt (CARDS, BOARD, FOLD, ...). Wenn du dir Informationen zwischen
    // Zügen merken möchtest, ergänze einfach eigene Felder in dieser Klasse
    // und aktualisiere sie in den passenden Cases.
    void act(String token) {
        if (token == null || token.isEmpty()) {
            LOGGER.warning("YOUR TURN received without token, skipping action");
            return;
        }

        String action = Protocol.CMD_CALL; // Standard: immer CALL

        // TODO: Ersetze die Zeile oben durch deine eigene Entscheidungslogik.

        LOGGER.info("Decided action: action=" + action);

        connection.mustSend(action + ";" + token);
    }
}
