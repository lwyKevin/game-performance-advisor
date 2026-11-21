package input

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Read loads frame times from a CSV file, JSON file, or stdin.
// CSV input requires a frametime_ms header; JSON input is a plain array of floats.
func Read(path, format string) ([]float64, error) {
	var reader io.Reader
	if path == "-" || path == "" {
		reader = os.Stdin
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open %q: %w", path, err)
		}
		defer file.Close()
		reader = file
	}

	if format == "auto" {
		if path == "-" || path == "" {
			data, err := io.ReadAll(reader)
			if err != nil {
				return nil, fmt.Errorf("read stdin: %w", err)
			}
			reader = bytes.NewReader(data)
			if bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) {
				format = "json"
			} else {
				format = "csv"
			}
		} else {
			lower := strings.ToLower(path)
			if strings.HasSuffix(lower, ".json") {
				format = "json"
			} else {
				format = "csv"
			}
		}
	}

	switch strings.ToLower(format) {
	case "csv":
		return readCSV(reader)
	case "json":
		return readJSON(reader)
	default:
		return nil, fmt.Errorf("unsupported input format %q", format)
	}
}

func readCSV(r io.Reader) ([]float64, error) {
	cr := csv.NewReader(bufio.NewReader(r))
	cr.TrimLeadingSpace = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must contain a header and at least one frame")
	}

	header := records[0]
	msColumn := -1
	for i, column := range header {
		if strings.EqualFold(strings.TrimSpace(column), "frametime_ms") {
			msColumn = i
			break
		}
	}
	if msColumn < 0 {
		return nil, fmt.Errorf("CSV header must contain frametime_ms")
	}

	values := make([]float64, 0, len(records)-1)
	for i, row := range records[1:] {
		if msColumn >= len(row) {
			return nil, fmt.Errorf("row %d is missing the frametime_ms value", i+2)
		}
		text := strings.TrimSpace(row[msColumn])
		if text == "" {
			continue
		}
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, fmt.Errorf("row %d has invalid frametime_ms %q", i+2, text)
		}
		if value <= 0 {
			return nil, fmt.Errorf("row %d frametime_ms must be greater than zero", i+2)
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("no frame times were found")
	}
	return values, nil
}

func readJSON(r io.Reader) ([]float64, error) {
	var values []float64
	if err := json.NewDecoder(r).Decode(&values); err != nil {
		return nil, fmt.Errorf("read JSON: %w", err)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("JSON array must contain at least one frame time")
	}
	for i, value := range values {
		if value <= 0 {
			return nil, fmt.Errorf("JSON value %d must be greater than zero", i)
		}
	}
	return values, nil
}

// Parse validates and parses frame times from an in-memory text reader.
func Parse(r io.Reader, format string) ([]float64, error) {
	switch strings.ToLower(format) {
	case "csv":
		return readCSV(r)
	case "json":
		return readJSON(r)
	default:
		return nil, fmt.Errorf("unsupported input format %q", format)
	}
}
