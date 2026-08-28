package bot

import (
	"reflect"
	"testing"
)

func TestGameState_ActivePlayers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		players []string
		folded  map[string]bool
		want    []string
	}{
		{
			name:    "no folds",
			players: []string{"Alice", "Bob", "Carol"},
			folded:  map[string]bool{},
			want:    []string{"Alice", "Bob", "Carol"},
		},
		{
			name:    "all folded",
			players: []string{"Alice", "Bob"},
			folded:  map[string]bool{"Alice": true, "Bob": true},
			want:    []string{},
		},
		{
			name:    "some folded",
			players: []string{"Alice", "Bob", "Carol"},
			folded:  map[string]bool{"Bob": true},
			want:    []string{"Alice", "Carol"},
		},
		{
			name:    "nil folded map",
			players: []string{"Alice", "Bob"},
			folded:  nil,
			want:    []string{"Alice", "Bob"},
		},
		{
			name:    "empty player list",
			players: []string{},
			folded:  map[string]bool{},
			want:    []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			state := GameState{Players: test.players, FoldedPlayers: test.folded}
			got := state.ActivePlayers()

			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("ActivePlayers() = %v, want %v", got, test.want)
			}
		})
	}
}

