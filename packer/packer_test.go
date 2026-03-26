package packer

import (
	"testing"
)

func TestCalculatePacks(t *testing.T) {
	defaultSizes := []int{250, 500, 1000, 2000, 5000}

	tests := []struct {
		name     string
		order    int
		sizes    []int
		expected map[int]int
	}{
		{
			name:     "1 item orders smallest pack",
			order:    1,
			sizes:    defaultSizes,
			expected: map[int]int{250: 1},
		},
		{
			name:     "exact match 250",
			order:    250,
			sizes:    defaultSizes,
			expected: map[int]int{250: 1},
		},
		{
			name:     "251 rounds up to 500",
			order:    251,
			sizes:    defaultSizes,
			expected: map[int]int{500: 1},
		},
		{
			name:     "501 uses 500 + 250",
			order:    501,
			sizes:    defaultSizes,
			expected: map[int]int{500: 1, 250: 1},
		},
		{
			name:     "12001 uses 5000x2 + 2000 + 250",
			order:    12001,
			sizes:    defaultSizes,
			expected: map[int]int{5000: 2, 2000: 1, 250: 1},
		},
		{
			name:     "exact match 5000",
			order:    5000,
			sizes:    defaultSizes,
			expected: map[int]int{5000: 1},
		},
		{
			name:     "750 uses 500 + 250",
			order:    750,
			sizes:    defaultSizes,
			expected: map[int]int{500: 1, 250: 1},
		},
		{
			name:     "zero order returns empty",
			order:    0,
			sizes:    defaultSizes,
			expected: map[int]int{},
		},
		{
			name:     "negative order returns empty",
			order:    -5,
			sizes:    defaultSizes,
			expected: map[int]int{},
		},
		{
			name:     "empty pack sizes returns empty",
			order:    100,
			sizes:    []int{},
			expected: map[int]int{},
		},
		{
			name:     "custom pack sizes",
			order:    263,
			sizes:    []int{100, 200, 500},
			expected: map[int]int{200: 1, 100: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePacks(tt.order, tt.sizes)

			if len(result) != len(tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
				return
			}

			for size, count := range tt.expected {
				if result[size] != count {
					t.Errorf("got %v, want %v", result, tt.expected)
					return
				}
			}
		})
	}
}

func TestCalculatePacksMinimizesItems(t *testing.T) {
	sizes := []int{250, 500, 1000, 2000, 5000}

	result := CalculatePacks(501, sizes)

	totalItems := 0
	for size, count := range result {
		totalItems += size * count
	}

	if totalItems != 750 {
		t.Errorf("expected 750 total items, got %d (packs: %v)", totalItems, result)
	}
}

func TestCalculatePacksMinimizesPacks(t *testing.T) {
	sizes := []int{250, 500, 1000, 2000, 5000}

	// 500 should be 1x500, not 2x250
	result := CalculatePacks(500, sizes)

	totalPacks := 0
	for _, count := range result {
		totalPacks += count
	}

	if totalPacks != 1 {
		t.Errorf("expected 1 pack, got %d (packs: %v)", totalPacks, result)
	}
}

// TestEdgeCase verifies the algorithm against the edge case provided in the assessment:
// pack sizes [23, 31, 53], order 500,000 → expected {23:2, 31:7, 53:9429}.
//
// Verification: 53*9429 + 31*7 + 23*2 = 499,737 + 217 + 46 = 500,000 items, 9,438 packs.
// This is provably the minimum number of packs: the linear equation 30a+8b=282,949
// has no integer solution for 9,437 packs (right-hand side is odd, GCD(30,8)=2).
func TestEdgeCase(t *testing.T) {
	result := CalculatePacks(500_000, []int{23, 31, 53})

	expected := map[int]int{23: 2, 31: 7, 53: 9429}

	totalItems := 0
	for size, count := range result {
		totalItems += size * count
	}

	if totalItems != 500_000 {
		t.Errorf("expected exactly 500,000 items shipped, got %d (packs: %v)", totalItems, result)
	}

	if len(result) != len(expected) {
		t.Errorf("got %v, want %v", result, expected)
		return
	}
	for size, count := range expected {
		if result[size] != count {
			t.Errorf("got %v, want %v", result, expected)
			return
		}
	}
}
