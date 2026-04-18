package generator

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestStyleMapper_AllRolesBuildable excelize must accept every built-in role
// the mapper knows about. If any one fails excelize validation we want a
// failing test, not a runtime panic when a real workbook is rendered.
func TestStyleMapper_AllRolesBuildable(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	m := NewXlsxStyleMapper(f, XlsxTheme{
		AccentColor:     "BE123C",
		HeaderFillColor: "1F2937",
		ZebraFillColor:  "F3F4F6",
	})

	roles := []StyleRole{
		StyleHeader, StyleBody, StyleZebra, StyleTotals, StyleAccent,
		StyleNegative, StylePositive,
		StyleBodyNumber, StyleZebraNumber, StyleBodyCurrency, StyleBodyPercent,
		StyleBodyDate, StyleMono,
	}
	for _, r := range roles {
		if _, err := m.ID(r); err != nil {
			t.Errorf("role %q: %v", r, err)
		}
	}
}

// TestStyleMapper_Cached calling ID twice for the same role must return the
// same excelize style index (we don't want a new style registered per cell).
func TestStyleMapper_Cached(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	m := NewXlsxStyleMapper(f, XlsxTheme{})

	first, err := m.ID(StyleHeader)
	if err != nil {
		t.Fatalf("first ID: %v", err)
	}
	second, err := m.ID(StyleHeader)
	if err != nil {
		t.Fatalf("second ID: %v", err)
	}
	if first != second {
		t.Fatalf("expected cached id, got %d then %d", first, second)
	}
}

// TestStyleMapper_UnknownRole asking for a role that doesn't exist must
// surface as an error, never a default zero-id style.
func TestStyleMapper_UnknownRole(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	m := NewXlsxStyleMapper(f, XlsxTheme{})
	if _, err := m.ID(StyleRole("totally-made-up")); err == nil {
		t.Fatal("expected error for unknown role")
	}
}

// TestMergedTheme_KeepsCallerFields blanks are filled but caller values must
// survive verbatim.
func TestMergedTheme_KeepsCallerFields(t *testing.T) {
	out := XlsxTheme{AccentColor: "BE123C", HeaderFillColor: "222222"}.mergedTheme()
	if out.AccentColor != "BE123C" {
		t.Errorf("accent: got %q", out.AccentColor)
	}
	if out.HeaderFillColor != "222222" {
		t.Errorf("header fill: got %q", out.HeaderFillColor)
	}
	if out.ZebraFillColor == "" {
		t.Error("zebra fill should have defaulted, got empty")
	}
	if out.BodyFont == "" {
		t.Error("body font should have defaulted, got empty")
	}
}
