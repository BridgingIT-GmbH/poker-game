package pokergame;

public final class Protocol {

    private Protocol() {
    }

    // ── client -> server ────────────────────────────────────────────────
    public static final String CMD_REGISTER = "REGISTER";
    public static final String CMD_JOIN = "JOIN";
    public static final String CMD_FOLD = "FOLD";
    public static final String CMD_CHECK = "CHECK";
    public static final String CMD_CALL = "CALL";
    public static final String CMD_RAISE = "RAISE";

    // ── server -> client ─────────────────────────────────────────────────
    public static final String CMD_REGISTERED = "REGISTERED";
    public static final String CMD_REJECTED = "REJECTED";
    public static final String CMD_ROUND_STARTING = "ROUND STARTING";
    public static final String CMD_ROUND_STARTED = "ROUND STARTED";
    public static final String CMD_CARDS = "CARDS";
    public static final String CMD_YOUR_TURN = "YOUR TURN";
    public static final String CMD_BOARD = "BOARD";
    public static final String CMD_SCORE = "SCORE";
}
