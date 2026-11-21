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
