package links

import "testing"

func TestNormalizeTags(t *testing.T) {
	got, err := NormalizeTags([]string{"ads", "br_2026", "ads", "", "growth-ops"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ads", "br_2026", "growth-ops"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tag[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNormalizeTagsRejectsInvalidTags(t *testing.T) {
	tests := [][]string{
		{"Upper"},
		{"com espaço"},
		{"tag.marcada"},
		{"abcdefghijklmnopqrstuvwxyzabcdefg"},
		{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"},
	}
	for _, tags := range tests {
		if _, err := NormalizeTags(tags); err == nil {
			t.Fatalf("NormalizeTags(%#v) sem erro; want ErrInvalidTags", tags)
		}
	}
}

func TestListParamsValidatesFilters(t *testing.T) {
	got, err := listParams(ListInput{
		Q:        " docs ",
		Status:   "ACTIVE",
		Tag:      "ads",
		Campaign: " launch ",
		Limit:    20,
		Offset:   5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Q != "docs" || got.Status != StatusActive || got.Tag != "ads" || got.Campaign != "launch" {
		t.Fatalf("params = %#v, filtros normalizados incorretamente", got)
	}

	if _, err := listParams(ListInput{Status: "paused"}); err != ErrInvalidStatus {
		t.Fatalf("status invalido err = %v, want ErrInvalidStatus", err)
	}
	if _, err := listParams(ListInput{Tag: "Ads"}); err != ErrInvalidTags {
		t.Fatalf("tag invalida err = %v, want ErrInvalidTags", err)
	}
}
