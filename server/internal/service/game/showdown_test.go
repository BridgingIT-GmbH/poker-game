package game

import (
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
)

func card(rank model.Rank, suit model.Suit) model.Card {
	return model.Card{Rank: rank, Suit: suit}
}

func TestDetermineWinner_LastStanding(t *testing.T) {
	t.Parallel()

	alice := &model.Player{IsActive: true,
		HasFolded: false,
		HoleCards: [2]model.Card{card(model.RankTwo, model.Clubs), card(model.RankFour, model.Diamonds)}}
	bob := &model.Player{IsActive: false,
		HasFolded: true,
		HoleCards: [2]model.Card{card(model.RankKing, model.Clubs), card(model.RankAce, model.Diamonds)}}
	table := &model.Table{Players: []*model.Player{alice, bob},
		Pot: 200,
		CommunityCards: []model.Card{
			c(model.RankTwo, model.Hearts),
			c(model.RankThree, model.Diamonds),
			c(model.RankAce, model.Clubs),
			c(model.RankKing, model.Hearts),
			c(model.RankKing, model.Spades),
		}}

	round := &Round{
		Table:       table,
		PlayerOrder: []string{"Alice", "Bob"},
	}

	result, err := DetermineWinner(round)
	if err != nil {
		t.Error("err: ", err)
	}

	if len(result.Winners) != 1 && result.Winners[0].Name != "Alice" {
		t.Errorf("expected winner Alice, got %s", result.Winners[0].Name)
	}

	if result.Pot != 200 {
		t.Errorf("expected pot 200, got %d", result.Pot)
	}
}

func TestDetermineWinner_Showdown(t *testing.T) {
	t.Parallel()

	alice := &model.Player{IsActive: true,
		HasFolded: false,
		HoleCards: [2]model.Card{card(model.RankTwo, model.Clubs), card(model.RankFour, model.Diamonds)}}
	bob := &model.Player{IsActive: true,
		HasFolded: false,
		HoleCards: [2]model.Card{card(model.RankKing, model.Clubs), card(model.RankAce, model.Diamonds)}}
	table := &model.Table{Players: []*model.Player{alice, bob},
		Pot: 300,
		CommunityCards: []model.Card{
			c(model.RankTwo, model.Hearts),
			c(model.RankThree, model.Diamonds),
			c(model.RankAce, model.Clubs),
			c(model.RankKing, model.Hearts),
			c(model.RankKing, model.Spades),
		}}

	round := &Round{
		Table:       table,
		PlayerOrder: []string{"Alice", "Bob"},
	}

	result, err := DetermineWinner(round)
	if err != nil {
		t.Error("err: ", err)
	}

	if len(result.Winners) != 1 && result.Winners[0].Name != "Bob" {
		t.Errorf("expected winner Bob, got %s", result.Winners[0].Name)
	}

	if len(result.Survivors) != 2 {
		t.Errorf("expected 2 survivors, got %d", len(result.Survivors))
	}
}
