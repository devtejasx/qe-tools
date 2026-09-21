package coffeebreak

import (
	"slices"
	"testing"
)

func TestNonEmptyLines(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{name: "no trailing newline", content: "a, b, c\nd, e, f", want: []string{"a, b, c", "d, e, f"}},
		{name: "trailing newline", content: "a, b, c\nd, e, f\n", want: []string{"a, b, c", "d, e, f"}},
		{name: "CRLF line endings", content: "a, b, c\r\nd, e, f\r\n", want: []string{"a, b, c", "d, e, f"}},
		{name: "blank lines", content: "\n\na, b, c\n\n", want: []string{"a, b, c"}},
		{name: "empty file", content: "", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nonEmptyLines(tt.content); !slices.Equal(got, tt.want) {
				t.Errorf("nonEmptyLines(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestPickGroupExcludesLastGroup(t *testing.T) {
	participants := []string{"a", "b", "c", "d", "e", "f"}
	// The file used to be split on "\n" as is, so a trailing newline made the
	// last "group" empty and nobody was excluded.
	history := nonEmptyLines("x, y, z\nd, e, f\n")

	for range 50 {
		group, err := pickGroup(participants, history)
		if err != nil {
			t.Fatal(err)
		}
		if len(group) != 3 {
			t.Fatalf("got group %q, want 3 members", group)
		}
		for _, member := range group {
			if slices.Contains([]string{"d", "e", "f"}, member) {
				t.Fatalf("group %q contains %q from the last group", group, member)
			}
		}
	}
}

func TestPickGroupNoHistory(t *testing.T) {
	group, err := pickGroup([]string{"a", "b", "c"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(group) != 3 {
		t.Fatalf("got group %q, want 3 members", group)
	}
}

func TestPickGroupNotEnoughEligible(t *testing.T) {
	// Five participants, three of whom were in the last group, leave only
	// two: this used to panic slicing eligibleParticipants[:3].
	_, err := pickGroup([]string{"a", "b", "c", "d", "e"}, []string{"c, d, e"})
	if err == nil {
		t.Fatal("expected an error when fewer than 3 participants are eligible")
	}
}
