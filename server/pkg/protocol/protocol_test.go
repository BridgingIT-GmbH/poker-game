package protocol

import (
	"errors"
	"testing"
)

func TestParseMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       []byte
		expected    []string
		expectedErr error
	}{
		{"valid two parts", []byte("REGISTER;Player"), []string{"REGISTER", "Player"}, nil},
		{"valid multiple parts", []byte("REGISTER;Player;Extra"), []string{"REGISTER", "Player", "Extra"}, nil},
		{"invalid one part", []byte("REGISTER"), nil, ErrInvalidMessageFormat},
		{"invalid empty", []byte(""), nil, ErrInvalidMessageFormat},
		{"valid empty second part", []byte("REGISTER;"), []string{"REGISTER", ""}, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual, err := ParseMessage(test.input)

			if !errors.Is(err, test.expectedErr) {
				t.Errorf("parseMessage() error = %v, expectedErr %v", err, test.expectedErr)

				return
			}

			if err == nil && len(actual) != len(test.expected) {
				t.Errorf("parseMessage() actual length = %v, expected length %v", len(actual), len(test.expected))

				return
			}

			for i := range actual {
				if actual[i] != test.expected[i] {
					t.Errorf("parseMessage() actual[%d] = %v, expected[%d] %v", i, actual[i], i, test.expected[i])
				}
			}
		})
	}
}
