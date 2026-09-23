package markdown

import (
	"slices"
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		in   string
		want Size
		ok   bool
	}{
		{in: "20", want: Size{SizeCells, 20}, ok: true},
		{in: "30%", want: Size{SizePercent, 30}, ok: true},
		{in: "2fr", want: Size{SizeFraction, 2}, ok: true},
		{in: " 1.5fr ", want: Size{SizeFraction, 1.5}, ok: true},
		{in: "auto"},
		{in: ""},
		{in: "wide"},
		{in: "-4"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := parseSize(tt.in)
			if ok != tt.ok || got != tt.want {
				t.Errorf("parseSize(%q) = %v, %t; want %v, %t", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParsePadding(t *testing.T) {
	tests := []struct {
		in   string
		x, y int
	}{
		{in: "", x: 0, y: 0},
		{in: "2", x: 2, y: 2},
		{in: "1 4", x: 4, y: 1},
		{in: "nope", x: 0, y: 0},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			x, y := parsePadding(tt.in)
			if x != tt.x || y != tt.y {
				t.Errorf("parsePadding(%q) = %d, %d; want %d, %d", tt.in, x, y, tt.x, tt.y)
			}
		})
	}
}

func TestDistribute(t *testing.T) {
	tests := []struct {
		name    string
		total   int
		gap     int
		sizes   []Size
		natural []int
		want    []int
	}{
		{
			name:  "equal fractions split evenly",
			total: 90,
			sizes: []Size{{SizeFraction, 1}, {SizeFraction, 1}, {SizeFraction, 1}},
			want:  []int{30, 30, 30},
		},
		{
			name:  "a remainder is handed out rather than lost",
			total: 100,
			sizes: []Size{{SizeFraction, 1}, {SizeFraction, 1}, {SizeFraction, 1}},
			want:  []int{34, 33, 33},
		},
		{
			name:  "master takes twice the stack",
			total: 90,
			sizes: []Size{{SizeFraction, 2}, {SizeFraction, 1}},
			want:  []int{60, 30},
		},
		{
			name:  "gaps come out of the total",
			total: 100,
			gap:   2,
			sizes: []Size{{SizeFraction, 1}, {SizeFraction, 1}},
			want:  []int{49, 49},
		},
		{
			name:  "exact and percent tracks are served first",
			total: 100,
			sizes: []Size{{SizeCells, 20}, {SizePercent, 30}, {SizeFraction, 1}},
			want:  []int{20, 30, 50},
		},
		{
			name:    "auto tracks keep their natural size",
			total:   50,
			sizes:   []Size{{}, {SizeFraction, 1}},
			natural: []int{12},
			want:    []int{12, 38},
		},
		{
			name:  "overflowing tracks shrink to fit",
			total: 30,
			sizes: []Size{{SizeCells, 40}, {SizeCells, 20}},
			want:  []int{20, 10},
		},
		{
			name:  "no room at all",
			total: 1,
			gap:   2,
			sizes: []Size{{SizeFraction, 1}, {SizeFraction, 1}},
			want:  []int{0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := distribute(tt.total, tt.gap, tt.sizes, tt.natural)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("distribute() = %v, want %v", got, tt.want)
			}

			avail := max(tt.total-tt.gap*(len(got)-1), 0)
			if used := sum(got); used > avail {
				t.Errorf("distribute() used %d cells of the %d available", used, avail)
			}
		})
	}
}
