package pokergame;

import java.io.IOException;
import java.net.DatagramPacket;
import java.net.DatagramSocket;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.List;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Low-level network and protocol plumbing (connect, receive loop, send,
 * parsing helpers). Mirrors client/internal/bot/transport.go.
 *
 * Players don't need to touch this class — see Bot.java for the parts
 * relevant to your strategy.
 */
public final class Connection {

    private static final Logger LOGGER = Logger.getLogger(Connection.class.getName());
    private static final int BUFFER_SIZE = 4096;

    private final DatagramSocket socket;
    private final String name;

    public Connection(DatagramSocket socket, String name) {
        this.socket = socket;
        this.name = name;
    }

    /** Registers the bot with the server. */
    public void connect() throws IOException {
        send(Protocol.CMD_REGISTER + ";" + name);
        LOGGER.info(() -> "Sent REGISTER;" + name);
    }

    /** Starts the blocking receive loop, dispatching each message to the handler. */
    public void run(MessageHandler handler) throws IOException {
        byte[] buf = new byte[BUFFER_SIZE];
        DatagramPacket packet = new DatagramPacket(buf, buf.length);

        while (true) {
            socket.receive(packet);

            String raw = stripTrailingNewline(
                    new String(packet.getData(), 0, packet.getLength(), StandardCharsets.UTF_8));

            LOGGER.info(() -> "<- server: " + raw);

            handler.handle(raw);
        }
    }

    /** Sends a raw protocol message, appending the trailing newline. */
    public void send(String msg) throws IOException {
        byte[] data = (msg + "\n").getBytes(StandardCharsets.UTF_8);
        socket.send(new DatagramPacket(data, data.length, socket.getRemoteSocketAddress()));
    }

    /** Sends a message, logging (instead of throwing) on failure. */
    public void mustSend(String msg) {
        try {
            send(msg);
        } catch (IOException e) {
            LOGGER.log(Level.SEVERE, "Failed to send message: " + msg, e);
        }
    }

    static String stripTrailingNewline(String s) {
        return s.replaceAll("[\r\n]+$", "");
    }

    /**
     * Splits a comma-separated card string into individual cards.
     * The server sends cards in the format "As,Kh" (CARDS) and "Td,7c,2h" (BOARD).
     */
    static List<String> parseCards(String raw) {
        if (raw == null || raw.isEmpty()) {
            return List.of();
        }

        return Arrays.asList(raw.split(","));
    }

    @FunctionalInterface
    public interface MessageHandler {
        void handle(String raw);
    }
}
