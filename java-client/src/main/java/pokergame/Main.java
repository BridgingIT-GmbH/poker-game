package pokergame;

import java.io.IOException;
import java.net.DatagramSocket;
import java.net.InetSocketAddress;
import java.net.SocketException;
import java.net.UnknownHostException;
import java.util.logging.Logger;

/**
 * Entry point for the Java poker client.
 * Usage: java -cp out pokergame.Main <host:port> [name]
 *
 * Mirrors client/main.go.
 */
public final class Main {

    private static final Logger LOGGER = Logger.getLogger(Main.class.getName());
    private static final String DEFAULT_NAME = "CallBot";

    private Main() {
    }

    public static void main(String[] args) {
        if (args.length < 1) {
            LOGGER.severe("Usage: java -cp out pokergame.Main <host:port> [name]");
            System.exit(1);
        }

        String addr = args[0];
        String name = args.length >= 2 ? args[1] : DEFAULT_NAME;

        InetSocketAddress serverAddress = resolveAddress(addr);

        try (DatagramSocket socket = new DatagramSocket()) {
            socket.connect(serverAddress);

            LOGGER.info(() -> "Connecting to server: addr=" + addr + " player=" + name);

            Connection connection = new Connection(socket, name);
            Bot bot = new Bot(name, connection);

            bot.connect();
            bot.run();
        } catch (IOException e) {
            LOGGER.severe("run: " + e.getMessage());
            System.exit(1);
        }
    }

    private static InetSocketAddress resolveAddress(String addr) {
        int sep = addr.lastIndexOf(':');
        if (sep < 0) {
            LOGGER.severe("resolve address: invalid format, expected host:port");
            System.exit(1);
        }

        String host = addr.substring(0, sep);
        String portStr = addr.substring(sep + 1);

        try {
            int port = Integer.parseInt(portStr);
            InetSocketAddress resolved = new InetSocketAddress(host, port);

            if (resolved.isUnresolved()) {
                throw new UnknownHostException(host);
            }

            return resolved;
        } catch (NumberFormatException | UnknownHostException e) {
            LOGGER.severe("resolve address: " + e.getMessage());
            System.exit(1);
            return null;
        }
    }
}
