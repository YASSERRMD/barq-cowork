package generator

import "testing"

func TestInferColumnKinds_MixedTypes(t *testing.T) {
	rows := [][]any{
		{"Jan 2026", "$1,200.50", "12%", "2026-01-15", 42, "hello"},
		{"Feb 2026", "$2,340.00", "14.5%", "2026-02-01", 99, "world"},
		{"Mar 2026", "$1,890.75", "9.25%", "2026-03-10", 17, "foo"},
	}
	kinds := InferColumnKinds(rows, 6, 20)
	want := []ColumnKind{
		ColumnText,     // "Jan 2026" etc — not a recognised date layout
		ColumnCurrency,
		ColumnPercent,
		ColumnDate,
		ColumnInteger,
		ColumnText,
	}
	for i, k := range kinds {
		if k != want[i] {
			t.Errorf("col %d: got %v, want %v", i, k, want[i])
		}
	}
}

func TestInferColumnKinds_StrayStringPinsText(t *testing.T) {
	rows := [][]any{
		{100},
		{200},
		{"N/A"}, // one string flips the whole column to text
		{400},
	}
	kinds := InferColumnKinds(rows, 1, 20)
	if kinds[0] != ColumnText {
		t.Errorf("expected text (one stray string), got %v", kinds[0])
	}
}

func TestInferColumnKinds_EmptyColumn(t *testing.T) {
	rows := [][]any{
		{""},
		{nil},
		{"  "},
	}
	kinds := InferColumnKinds(rows, 1, 20)
	if kinds[0] != ColumnText {
		t.Errorf("empty column should default to text, got %v", kinds[0])
	}
}

func TestInferColumnKinds_FloatNotInt(t *testing.T) {
	rows := [][]any{{1.5}, {2.25}, {3.125}}
	kinds := InferColumnKinds(rows, 1, 20)
	if kinds[0] != ColumnFloat {
		t.Errorf("expected float, got %v", kinds[0])
	}
}

func TestLooksLikeCurrency(t *testing.T) {
	cases := map[string]bool{
		"$1,234.56": true,
		"€12.50":    true,
		"12.50":     false, // no symbol — plain number, not currency
		"abc":       false,
		"":          false,
	}
	for in, want := range cases {
		if got := looksLikeCurrency(in); got != want {
			t.Errorf("looksLikeCurrency(%q) = %v, want %v", in, got, want)
		}
	}
}
