package protocol

import (
	"errors"
	"log/slog"
	"strings"
)

type CommandPlayer string
type CommandServer string

const (
	CmdRegister          CommandPlayer = "REGISTER"
	CmdRegisterSpectator CommandPlayer = "REGISTER_SPECTATOR"
	CmdJoin              CommandPlayer = "JOIN"
	CmdFold              CommandPlayer = "FOLD"
	CmdCheck             CommandPlayer = "CHECK"
	CmdCall              CommandPlayer = "CALL"
	CmdRaise             CommandPlayer = "RAISE"
)

const (
	CmdRegistered    CommandServer = "REGISTERED"
	CmdRejected      CommandServer = "REJECTED"
	CmdRoundStarting CommandServer = "ROUND STARTING"
	CmdRoundStarted  CommandServer = "ROUND STARTED"
	CmdCards         CommandServer = "CARDS"
	CmdYourTurn      CommandServer = "YOUR TURN"
	CmdTurn          CommandServer = "TURN"
	CmdBoard         CommandServer = "BOARD"
	CmdScore         CommandServer = "SCORE"
)

var (
	ErrInvalidMessageFormat = errors.New("invalid message format")
)

// ParseMessage splits a raw protocol message by ';'.
// The minimal format is: CMD;ARG.
// A trailing newline is stripped before parsing.
func ParseMessage(data []byte) ([]string, error) {
	message := strings.TrimRight(string(data), "\n")
	minParts := 2

	parts := strings.Split(message, ";")
	if len(parts) < minParts {
		slog.Warn("Invalid message format", "message", message)

		return nil, ErrInvalidMessageFormat
	}

	return parts, nil
}
