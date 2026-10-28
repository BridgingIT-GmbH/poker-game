import type { Socket } from 'node:dgram';
import { CmdRegister } from './protocol.js';

/**
 * Low-level network and protocol plumbing (connect, receive loop, send,
 * parsing helpers). Mirrors client/internal/bot/transport.go and
 * java-client/.../Connection.java.
 *
 * Players don't need to touch this class — see bot.ts for the parts
 * relevant to your strategy.
 */

export type MessageHandler = (raw: string) => void;

export class Connection {
    private readonly socket: Socket;
    private readonly name: string;

    constructor(socket: Socket, name: string) {
        this.socket = socket;
        this.name = name;
    }

    /** Registers the bot with the server. */
    connect(): void {
        this.mustSend(`${CmdRegister};${this.name}`);
        console.log(`Sent REGISTER;${this.name}`);
    }

    /** Starts the (non-blocking, event-based) receive loop. */
    run(handler: MessageHandler): void {
        this.socket.on('message', (msg: Buffer) => {
            const raw = stripTrailingNewline(msg.toString('utf8'));
            console.log(`<- server: ${raw}`);
            handler(raw);
        });
    }

    /** Sends a raw protocol message, appending the trailing newline. */
    send(msg: string, callback?: (err: Error | null) => void): void {
        const data = Buffer.from(`${msg}\n`, 'utf8');
        this.socket.send(data, callback);
    }

    /** Sends a message, logging (instead of throwing) on failure. */
    mustSend(msg: string): void {
        this.send(msg, (err) => {
            if (err) {
                console.error(`Failed to send message: ${msg}`, err);
            }
        });
    }

    /**
     * Splits a comma-separated card string into individual cards.
     * The server sends cards in the format "As,Kh" (CARDS) and "Td,7c,2h" (BOARD).
     */
    static parseCards(raw: string): string[] {
        if (!raw) {
            return [];
        }

        return raw.split(',');
    }
}

function stripTrailingNewline(s: string): string {
    return s.replace(/[\r\n]+$/, '');
}
