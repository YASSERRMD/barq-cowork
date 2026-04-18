package generator

import "testing"

func TestComputeColumnWidths_BasicShape(t *testing.T) {
	headers := []string{"Name", "Description"}
	rows := [][]any{
		{"Alice", "a short note"},
		{"Bob", "a much much longer descriptive sentence that sprawls out way beyond the clamp ceiling yes indeed"},
	}
	widths := ComputeColumnWidths(headers, rows, 100)

	if len(widths) != 2 {
		t.Fatalf("want 2 widths, got %d", len(widths))
	}
	if widths[0] < minColumnWidth || widths[0] > maxColumnWidth {
		t.Errorf("col0 out of clamp: %v", widths[0])
	}
	if widths[1] != maxColumnWidth {
		t.Errorf("long text should clamp to max (%v), got %v", maxColumnWidth, widths[1])
	}
	if widths[0] >= widths[1] {
		t.Errorf("col0 (short) should be narrower than col1 (long): %v vs %v", widths[0], widths[1])
	}
}

func TestComputeColumnWidths_MinimumFloor(t *testing.T) {
	widths := ComputeColumnWidths([]string{"x"}, [][]any{{"y"}}, 10)
	if widths[0] != minColumnWidth {
		t.Errorf("expected min floor %v, got %v", minColumnWidth, widths[0])
	}
}

func TestComputeColumnWidths_CountsRunesNotBytes(t *testing.T) {
	headers := []string{"أسم"} // 3 runes, 6 bytes
	widths := ComputeColumnWidths(headers, nil, 10)
	// minColumnWidth floor applies, but rune counting must not inflate past
	// its maximum rune width (3 + 2 padding = 5, below floor — this proves
	// we didn't accidentally use byte length which would be 8).
	if widths[0] != minColumnWidth {
		t.Errorf("rune count should yield min floor, got %v", widths[0])
	}
}
