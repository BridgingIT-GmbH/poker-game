package server

import (
	"fmt"
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

func card(rank model.Rank, suit model.Suit) model.Card {
	return model.Card{Rank: rank, Suit: suit}
}

func TestHandleLastPlayerStanding_ScoreFormat(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)
	round.Table.Pot = 300
	round.Table.CommunityCards = []model.Card{
		card(model.RankTwo, model.Hearts),
		card(model.RankAce, model.Diamonds),
		card(model.RankFour, model.Clubs),
		card(model.RankSeven, model.Hearts),
		card(model.RankSix, model.Spades),
	}
	round.JoinedPlayers["Alice"].HoleCards = [2]model.Card{card(model.RankAce, model.Hearts),
		card(model.RankKing, model.Diamonds)}
	round.JoinedPlayers["Bob"].HoleCards = [2]model.Card{card(model.RankAce, model.Clubs),
		card(model.RankKing, model.Spades)}

	game.ApplyFold(round.Table, round.Table.Players[0]) // Alice folds → Bob wins

	core.handleLastPlayerStanding(round)

	pkgs := drainN(t, outbound, 2) // broadcast to both joined players

	expected := fmt.Sprintf("%s;Bob:300\n", protocol.CmdScore)
	for _, pkg := range pkgs {
		if string(pkg.Data) != expected {
			t.Errorf("expected %q, got %q", expected, string(pkg.Data))
		}
	}
}

func TestHandleShowdown_BroadcastsScore(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)
	round.Table.Pot = 300
	round.Table.CommunityCards = []model.Card{
		card(model.RankTwo, model.Hearts),
		card(model.RankAce, model.Diamonds),
		card(model.RankFour, model.Clubs),
		card(model.RankSeven, model.Hearts),
		card(model.RankSix, model.Spades),
	}
	round.JoinedPlayers["Alice"].HoleCards = [2]model.Card{card(model.RankAce, model.Hearts),
		card(model.RankKing, model.Diamonds)}
	round.JoinedPlayers["Bob"].HoleCards = [2]model.Card{card(model.RankThree, model.Clubs),
		card(model.RankTwo, model.Spades)}

	core.handleShowdown(round)

	pkgs := drainN(t, outbound, 2) // SCORE to both players

	for _, pkg := range pkgs {
		data := string(pkg.Data)
		if data[:len(string(protocol.CmdScore))] != string(protocol.CmdScore) {
			t.Errorf("expected message to start with SCORE, got %q", data)
		}
	}
}
