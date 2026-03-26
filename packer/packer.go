package packer

import (
	"sort"
)

// CalculatePacks determines the optimal combination of packs to fulfill an order.
// Rules (in priority order):
//  1. Only whole packs can be sent.
//  2. Minimize total items shipped (least overshoot).
//  3. Minimize total number of packs.
func CalculatePacks(orderQty int, packSizes []int) map[int]int {
	if orderQty <= 0 || len(packSizes) == 0 {
		return map[int]int{}
	}

	sizes := make([]int, len(packSizes))
	copy(sizes, packSizes)
	sort.Sort(sort.Reverse(sort.IntSlice(sizes)))

	smallest := sizes[len(sizes)-1]

	// Upper bound: we never need more items than the order rounded up to the smallest pack
	maxItems := orderQty
	if maxItems%smallest != 0 {
		maxItems = orderQty + (smallest - orderQty%smallest)
	}

	// BFS/DP approach: find all achievable totals >= orderQty, pick the smallest total,
	// then among solutions hitting that total, pick the one with fewest packs.
	// We use DP over item counts from 0..maxItems.

	type solution struct {
		total int
		packs int
		combo map[int]int
	}

	// For efficiency, use a bounded DP.
	// dp[i] = minimum packs to reach exactly i items, -1 if unreachable.
	limit := maxItems + sizes[0] // give some headroom above maxItems
	dp := make([]int, limit+1)
	parent := make([]int, limit+1) // which pack size was used to reach this total
	for i := range dp {
		dp[i] = -1
		parent[i] = -1
	}
	dp[0] = 0

	for i := 0; i <= limit; i++ {
		if dp[i] == -1 {
			continue
		}
		for _, size := range sizes {
			next := i + size
			if next > limit {
				continue
			}
			newPacks := dp[i] + 1
			if dp[next] == -1 || newPacks < dp[next] {
				dp[next] = newPacks
				parent[next] = size
			}
		}
	}

	// Find the smallest achievable total >= orderQty with fewest packs
	bestTotal := -1
	for i := orderQty; i <= limit; i++ {
		if dp[i] != -1 {
			bestTotal = i
			break
		}
	}

	if bestTotal == -1 {
		return map[int]int{}
	}

	// Reconstruct the combination
	result := make(map[int]int)
	remaining := bestTotal
	for remaining > 0 {
		size := parent[remaining]
		result[size]++
		remaining -= size
	}

	return result
}
