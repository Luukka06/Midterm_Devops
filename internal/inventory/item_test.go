package inventory

import "testing"

func TestStatus(t *testing.T) {
	tests := []struct {
		name              string
		quantity, minimum int
		want              string
	}{
		{"healthy inventory", 14, 5, "healthy"},
		{"exact threshold", 5, 5, "low"},
		{"below threshold", 2, 5, "low"},
		{"empty inventory", 0, 5, "out"},
		{"zero with zero minimum", 0, 0, "out"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Status(Item{Quantity: tt.quantity, MinStock: tt.minimum})
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSummarize(t *testing.T) {
	items := []Item{
		{Quantity: 20, MinStock: 4},
		{Quantity: 3, MinStock: 4},
		{Quantity: 0, MinStock: 2},
	}
	got := Summarize(items)
	want := Summary{TotalItems: 3, TotalUnits: 23, LowStock: 1, OutOfStock: 1}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	got := Summarize(nil)
	if got != (Summary{}) {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestNormalizeTrimsAndDefaults(t *testing.T) {
	got, err := (ItemInput{Name: "  Notebook  ", Quantity: 3, MinStock: 1}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Notebook" || got.Category != "General" || got.Unit != "pcs" {
		t.Fatalf("unexpected normalized value: %+v", got)
	}
}

func TestNormalizeRejectsInvalidValues(t *testing.T) {
	inputs := []ItemInput{
		{Name: ""},
		{Name: "mouse", Quantity: -1},
		{Name: "mouse", MinStock: -1},
		{Name: "mouse", Quantity: 1000001},
	}
	for _, in := range inputs {
		if _, err := in.Normalize(); err == nil {
			t.Fatalf("should reject %+v", in)
		}
	}
}
