package httpapi

import "testing"

func TestSplitCSV(t *testing.T) {
	in := " SP500, ,NASDAQ ,US10Y"
	got := splitCSV(in)
	want := []string{"SP500", "NASDAQ", "US10Y"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("idx=%d got=%q want=%q", i, got[i], want[i])
		}
	}
}
