package protocol

import (
	"errors"
	"log/slog"
	"strings"
)

type CommandPlayer string
type CommandServer string

const (
	CmdRegister CommandPlayer = "REGISTER"
	CmdJoin     CommandPlayer = "JOIN"
)

const (
	CmdRegistered    CommandServer = "REGISTERED"
	CmdRejected      CommandServer = "REJECTED"
	CmdRoundStarting CommandServer = "ROUND STARTING"
	CmdRoundStarted  CommandServer = "ROUND STARTED"
)

var (
	ErrInvalidMessageFormat = errors.New("invalid message format")
)

// ParseMessage splits a raw protocol message by ';'.
// The minimal format is: CMD;ARG.
func ParseMessage(data []byte) ([]string, error) {
	message := string(data)
	minParts := 2

	parts := strings.Split(message, ";")
	if len(parts) < minParts {
		slog.Warn("Invalid message format", "message", message)

		return nil, ErrInvalidMessageFormat
	}

	return parts, nil
}
