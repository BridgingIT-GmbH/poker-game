import { createSocket } from 'node:dgram';
import { Bot } from './bot.js';
import { Connection } from './connection.js';

/**
 * Entry point for the TypeScript poker client.
 * Usage: node dist/main.js <host:port> [name]
 *
 * Mirrors client/main.go and java-client/.../Main.java.
 */

const DEFAULT_NAME = 'CallBot';

function main(): void {
    const { host, port, name } = parseArgs();

    console.log(`Connecting to server: addr=${host}:${port} player=${name}`);

    const socket = createSocket('udp4');

    socket.on('error', (err) => {
        console.error('socket error', err);
        socket.close();
        process.exit(1);
    });

    // "Connecting" a UDP socket fixes the remote peer, so send() no longer
    // needs an explicit address/port (mirrors net.DialUDP in Go and
    // DatagramSocket#connect in Java).
    socket.connect(port, host, () => {
        const connection = new Connection(socket, name);
        const bot = new Bot(name, connection);

        bot.connect();
        bot.run();
    });
}

/** Returns the server address and player name from CLI arguments. */
function parseArgs(): { host: string; port: number; name: string } {
    const args = process.argv.slice(2);

    if (args.length < 1) {
        console.error('Usage: node dist/main.js <host:port> [name]');
        process.exit(1);
    }

    const addr = args[0]!;
    const name = args[1] ?? DEFAULT_NAME;

    const sep = addr.lastIndexOf(':');
    if (sep < 0) {
        console.error('resolve address: invalid format, expected host:port');
        process.exit(1);
    }

    const host = addr.slice(0, sep);
    const port = Number(addr.slice(sep + 1));

    if (!host || Number.isNaN(port) || port <= 0 || port > 65535) {
        console.error('resolve address: invalid host or port');
        process.exit(1);
    }

    return { host, port, name };
}

main();
