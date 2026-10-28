// Protocol constants shared between all clients.
// Mirrors server/pkg/protocol/protocol.go and java-client/.../Protocol.java.

export type CommandPlayer =
    | 'REGISTER'
    | 'JOIN'
    | 'FOLD'
    | 'CHECK'
    | 'CALL'
    | 'RAISE';

export type CommandServer =
    | 'REGISTERED'
    | 'REJECTED'
    | 'ROUND STARTING'
    | 'ROUND STARTED'
    | 'CARDS'
    | 'YOUR TURN'
    | 'BOARD'
    | 'SCORE';

// ── client -> server ────────────────────────────────────────────────────────
export const CmdRegister: CommandPlayer = 'REGISTER';
export const CmdJoin: CommandPlayer = 'JOIN';
export const CmdFold: CommandPlayer = 'FOLD';
export const CmdCheck: CommandPlayer = 'CHECK';
export const CmdCall: CommandPlayer = 'CALL';
export const CmdRaise: CommandPlayer = 'RAISE';

// ── server -> client ────────────────────────────────────────────────────────
export const CmdRegistered: CommandServer = 'REGISTERED';
export const CmdRejected: CommandServer = 'REJECTED';
export const CmdRoundStarting: CommandServer = 'ROUND STARTING';
export const CmdRoundStarted: CommandServer = 'ROUND STARTED';
export const CmdCards: CommandServer = 'CARDS';
export const CmdYourTurn: CommandServer = 'YOUR TURN';
export const CmdBoard: CommandServer = 'BOARD';
export const CmdScore: CommandServer = 'SCORE';
