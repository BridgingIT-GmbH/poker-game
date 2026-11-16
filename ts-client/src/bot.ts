import { Connection } from './connection.js';
import {
    CmdBoard,
    CmdCall,
    CmdCards,
    CmdCheck,
    CmdFold,
    CmdJoin,
    CmdRaise,
    CmdRegistered,
    CmdRejected,
    CmdRoundStarted,
    CmdRoundStarting,
    CmdScore,
    CmdYourTurn,
} from './protocol.js';

/**
 * Bot handles all network communication and protocol details.
 * Players are free to implement their own decision logic directly inside
 * act() below (see the CmdYourTurn case in handleMessage).
 */
export class Bot {
    private readonly name: string;
    private readonly connection: Connection;

    constructor(name: string, connection: Connection) {
        this.name = name;
        this.connection = connection;
    }

    /** Registers the bot with the server. */
    connect(): void {
        this.connection.connect();
    }

    /** Starts the receive loop. */
    run(): void {
        this.connection.run((raw) => this.handleMessage(raw));
    }

    // ─── Message dispatcher ─────────────────────────────────────────────────

    /**
     * Routes an incoming server message to the appropriate handler.
     * Handles both server->client commands and server-broadcast player
     * actions (FOLD/CALL/RAISE/CHECK with the acting player's name as payload).
     */
    private handleMessage(raw: string): void {
        // Split into at most 3 parts to handle ROUND STARTED;roundNum;player1,player2
        const parts = raw.split(';');
        const cmd = parts[0] ?? '';
        const payload = parts[1] ?? '';

        switch (cmd) {
            // ── server -> client ─────────────────────────────────────────────
            case CmdRegistered:
                console.log('Registered successfully');
                break;

            case CmdRejected:
                console.warn(`Rejected by server: reason=${payload}`);
                break;

            case CmdRoundStarting:
                this.connection.mustSend(`${CmdJoin};${payload}`);
                console.log('Sent JOIN');
                break;

            case CmdRoundStarted: {
                // format: ROUND STARTED;<roundNumber>;<player1,player2,...>
                const playerList = parts.slice(2).join(';') || '';
                const players = playerList.split(',');
                console.log(`Round started: round=${payload} players=${players}`);
                break;
            }

            case CmdCards:
                console.log(`Hole cards received: cards=${Connection.parseCards(payload)}`);
                break;

            case CmdBoard:
                console.log(`Board updated: cards=${Connection.parseCards(payload)}`);
                break;

            case CmdScore:
                console.log(`Score received: result=${payload}`);
                break;

            case CmdYourTurn:
                this.act(payload);
                break;

            // ── server-broadcast player actions ─────────────────────────────
            case CmdFold:
                console.log(`Player folded: player=${payload}`);
                break;

            case CmdCall:
                console.log(`Player called: player=${payload}`);
                break;

            case CmdRaise:
                console.log(`Player raised: player=${payload}`);
                break;

            case CmdCheck:
                console.log(`Player checked: player=${payload}`);
                break;

            default:
                console.debug(`Unknown message: raw=${raw}`);
        }
    }

    // ╔════════════════════════════════════════════════════════════════════╗
    // ║  HIER GEHT'S LOS – baue hier deine eigene Poker-Strategie ein!     ║
    // ╚════════════════════════════════════════════════════════════════════╝
    //
    // Verfügbare Aktionen (protocol.ts):
    //
    //   CmdFold   – Hand aufgeben
    //   CmdCheck  – passen (nur gültig, wenn kein Einsatz aussteht)
    //   CmdCall   – aktuellen Einsatz mitgehen (Standard)
    //   CmdRaise  – Einsatz erhöhen
    //
    // Tipp: In handleMessage() oben siehst du alle Nachrichten, die der Server
    // schickt (CARDS, BOARD, FOLD, ...). Wenn du dir Informationen zwischen
    // Zügen merken möchtest, ergänze einfach eigene Felder in dieser Klasse
    // und aktualisiere sie in den passenden Cases.
    private act(token: string): void {
        if (!token) {
            console.warn('YOUR TURN received without token, skipping action');
            return;
        }

        const action = CmdCall; // Standard: immer CALL

        // TODO: Ersetze die Zeile oben durch deine eigene Entscheidungslogik.

        console.log(`Decided action: action=${action}`);

        this.connection.mustSend(`${action};${token}`);
    }
}
