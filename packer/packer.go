// Package packer provides the core algorithm for calculating the optimal
// combination of packs needed to fulfil a customer order.
package packer

import (
	"sort"
)

// CalculatePacks determines the optimal combination of packs to fulfil an order.
//
// The algorithm follows three rules in priority order:
//  1. Only whole packs can be sent — no partial packs.
//  2. Minimise the total number of items shipped (least overshoot above the order).
//  3. Among solutions with equal item counts, use as few packs as possible.
//
// It uses an unbounded knapsack / coin-change style dynamic programming approach:
//   - dp[i] holds the fewest packs needed to reach exactly i items (-1 = unreachable).
//   - parent[i] records which pack size was added last to reach i items, enabling
//     reconstruction of the full combination after the DP pass.
//
// The search space is bounded: we only need to consider totals up to the order
// quantity rounded up to the nearest multiple of the smallest pack, plus one
// extra largest pack for headroom. Any total beyond that is guaranteed to overshoot
// more than the minimum achievable solution.
func CalculatePacks(orderQty int, packSizes []int) map[int]int {
	if orderQty <= 0 || len(packSizes) == 0 {
		return map[int]int{}
	}

	// Work on a sorted copy (descending) so the DP favours larger packs first,
	// which helps parent tracking choose fewer, bigger packs during reconstruction.
	sizes := make([]int, len(packSizes))
	copy(sizes, packSizes)
	sort.Sort(sort.Reverse(sort.IntSlice(sizes)))

	smallest := sizes[len(sizes)-1]

	// Round the order up to the nearest multiple of the smallest pack.
	// This is the tightest upper bound on the minimum achievable total.
	maxItems := orderQty
	if maxItems%smallest != 0 {
		maxItems = orderQty + (smallest - orderQty%smallest)
	}

	// Add one largest-pack worth of headroom so we never miss a valid combination
	// that crosses the maxItems boundary with a big pack.
	limit := maxItems + sizes[0]

	// Initialise the DP table. dp[0] = 0 (zero packs to ship zero items).
	dp := make([]int, limit+1)
	parent := make([]int, limit+1)
	for i := range dp {
		dp[i] = -1
		parent[i] = -1
	}
	dp[0] = 0

	// Forward DP: for each reachable total i, try adding every pack size.
	// Because we iterate i in ascending order and all pack sizes are positive,
	// dp[i] is already optimal by the time we process it.
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
			// Only update when we strictly improve — this preserves the
			// best (fewest-packs) path found so far for each total.
			if dp[next] == -1 || newPacks < dp[next] {
				dp[next] = newPacks
				parent[next] = size
			}
		}
	}

	// Find the smallest reachable total >= orderQty (rule 2: minimise items).
	bestTotal := -1
	for i := orderQty; i <= limit; i++ {
		if dp[i] != -1 {
			bestTotal = i
			break
		}
	}

	if bestTotal == -1 {
		// No solution found — pack sizes cannot cover this order (shouldn't happen
		// with a valid set of pack sizes, but guard against empty/incompatible sets).
		return map[int]int{}
	}

	// Reconstruct the pack combination by walking backwards through the parent table.
	result := make(map[int]int)
	remaining := bestTotal
	for remaining > 0 {
		size := parent[remaining]
		result[size]++
		remaining -= size
	}

	return result
}
