//nolint:dupl
package server

import (
	"fmt"
	"strings"
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

	expected := fmt.Sprintf("%s;Alice:990;Bob:1280\n", protocol.CmdScore)
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

	expected := fmt.Sprintf("%s;Alice:1290:Ah,Kd;Bob:980:3c,2s\n", protocol.CmdScore)
	for _, pkg := range pkgs {
		if string(pkg.Data) != expected {
			t.Errorf("expected %q, got %q", expected, string(pkg.Data))
		}
	}
}

func TestHandleShowdown_TiedWinnersSplitPot(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)
	round.Table.Pot = 301
	round.Table.CommunityCards = []model.Card{
		card(model.RankTwo, model.Hearts),
		card(model.RankThree, model.Diamonds),
		card(model.RankFour, model.Clubs),
		card(model.RankFive, model.Hearts),
		card(model.RankSix, model.Spades),
	}
	round.JoinedPlayers["Alice"].HoleCards = [2]model.Card{
		card(model.RankAce, model.Hearts),
		card(model.RankKing, model.Diamonds),
	}
	round.JoinedPlayers["Bob"].HoleCards = [2]model.Card{
		card(model.RankNine, model.Clubs),
		card(model.RankEight, model.Spades),
	}

	core.handleShowdown(round)

	pkgs := drainN(t, outbound, 2)

	expected := fmt.Sprintf("%s;Alice:1140.5:Ah,Kd;Bob:1130.5:9c,8s\n", protocol.CmdScore)
	for _, pkg := range pkgs {
		if string(pkg.Data) != expected {
			t.Errorf("expected %q, got %q", expected, string(pkg.Data))
		}
	}
}

func TestHandleShowdown_IncludesFoldedPlayerWithoutCards(t *testing.T) {
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
	foldedCards := [2]model.Card{
		card(model.RankQueen, model.Hearts),
		card(model.RankJack, model.Diamonds),
	}
	round.JoinedPlayers["Alice"].HoleCards = foldedCards
	round.JoinedPlayers["Bob"].HoleCards = [2]model.Card{
		card(model.RankThree, model.Clubs),
		card(model.RankTwo, model.Spades),
	}
	game.ApplyFold(round.Table, round.JoinedPlayers["Alice"])

	core.handleShowdown(round)

	pkgs := drainN(t, outbound, 2)

	expected := fmt.Sprintf("%s;Alice:990;Bob:1280:3c,2s\n", protocol.CmdScore)
	for _, pkg := range pkgs {
		if string(pkg.Data) != expected {
			t.Errorf("expected %q, got %q", expected, string(pkg.Data))
		}

		if strings.Contains(string(pkg.Data), foldedCards[0].String()) ||
			strings.Contains(string(pkg.Data), foldedCards[1].String()) {
			t.Errorf("folded player's hole cards were revealed: %q", string(pkg.Data))
		}
	}
}

func TestBuildScoreMessage_UsesCurrentRoundScore(t *testing.T) {
	t.Parallel()

	core, firstRound, _ := twoPlayerRoundInProgress(t)
	firstRound.Table.Pot = 301
	firstRound.Table.CommunityCards = []model.Card{
		card(model.RankTwo, model.Hearts),
		card(model.RankThree, model.Diamonds),
		card(model.RankFour, model.Clubs),
		card(model.RankFive, model.Hearts),
		card(model.RankSix, model.Spades),
	}
	firstResult := &game.ShowdownResult{
		Winners: []*model.Player{firstRound.JoinedPlayers["Alice"], firstRound.JoinedPlayers["Bob"]},
		Pot:     firstRound.Table.Pot,
	}

	firstMessage := core.buildScoreMessage(firstRound, calculatePayouts(firstResult), false)
	if expected := fmt.Sprintf("%s;Alice:1140.5;Bob:1130.5\n", protocol.CmdScore); firstMessage != expected {
		t.Fatalf("expected first round score %q, got %q", expected, firstMessage)
	}

	secondRound := game.NewRound(2)
	secondRound.PlayerOrder = []string{"Alice", "Bob"}
	secondRound.JoinedPlayers["Alice"] = model.NewPlayer(0, "Alice", 900)
	secondRound.JoinedPlayers["Bob"] = model.NewPlayer(1, "Bob", 800)
	secondResult := &game.ShowdownResult{
		Winners: []*model.Player{secondRound.JoinedPlayers["Alice"]},
		Pot:     100,
	}
	secondMessage := core.buildScoreMessage(secondRound, calculatePayouts(secondResult), false)

	if expected := fmt.Sprintf("%s;Alice:1000;Bob:800\n", protocol.CmdScore); secondMessage != expected {
		t.Errorf("expected second round score %q, got %q", expected, secondMessage)
	}
}
