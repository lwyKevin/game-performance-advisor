package input

import (
	"strings"
	"testing"
)

func TestReadCSV(t *testing.T) {
	r := strings.NewReader("frame,frametime_ms\n1,16.67\n2,33.33\n")
	values, err := readCSV(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0] != 16.67 || values[1] != 33.33 {
		t.Fatalf("unexpected values: %v", values)
	}
}

func TestReadCSVRequiresHeader(t *testing.T) {
	_, err := readCSV(strings.NewReader("frame\n1\n"))
	if err == nil {
		t.Fatal("expected missing frametime_ms header error")
	}
}

func TestReadCSVRejectsMalformedRows(t *testing.T) {
	tests := []struct {
		name string
		csv  string
	}{
		{name: "missing frametime", csv: "frame,frametime_ms\n1\n"},
		{name: "blank frametime", csv: "frame,frametime_ms\n1,\n"},
		{name: "invalid frametime", csv: "frame,frametime_ms\n1,fast\n"},
		{name: "zero frametime", csv: "frame,frametime_ms\n1,0\n"},
		{name: "negative frametime", csv: "frame,frametime_ms\n1,-1\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := readCSV(strings.NewReader(tt.csv)); err == nil {
				t.Fatal("expected malformed CSV row to fail")
			}
		})
	}
}

func TestReadCSVAllowsExtraColumns(t *testing.T) {
	values, err := readCSV(strings.NewReader("frame,frametime_ms,extra\n1,16.67,ignored\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0] != 16.67 {
		t.Fatalf("unexpected values: %v", values)
	}
}

func TestReadJSON(t *testing.T) {
	values, err := readJSON(strings.NewReader("[16.67, 16.71, 33.2]"))
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 3 {
		t.Fatalf("got %d values, want 3", len(values))
	}
}

func TestReadJSONRejectsInvalidValues(t *testing.T) {
	if _, err := readJSON(strings.NewReader("[16.67, 0]")); err == nil {
		t.Fatal("expected invalid JSON frame time error")
	}
}
