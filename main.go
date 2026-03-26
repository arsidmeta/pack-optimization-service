package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/arsid/pack-optimization-service/packer"
)

// defaultPackSizes are used on first run when no persisted configuration exists.
var defaultPackSizes = []int{250, 500, 1000, 2000, 5000}

// packsFile is the path to the JSON file used to persist pack sizes across restarts.
const packsFile = "packs.json"

// PackSize represents a single pack size in API requests and responses.
type PackSize struct {
	Size int `json:"size"`
}

// OrderRequest is the payload for POST /api/calculate.
type OrderRequest struct {
	Items int `json:"items"`
}

// PackResult describes how many packs of a given size are needed.
type PackResult struct {
	Size     int `json:"size"`
	Quantity int `json:"quantity"`
}

// OrderResponse is the response payload for a successful calculation.
type OrderResponse struct {
	OrderedItems int          `json:"ordered_items"`
	TotalItems   int          `json:"total_items"`
	Packs        []PackResult `json:"packs"`
}

// ErrorResponse wraps an error message for API error responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

var (
	packSizes []int
	mu        sync.RWMutex
)

func main() {
	// Load pack sizes from disk; fall back to defaults if the file doesn't exist.
	packSizes = loadPackSizes()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/calculate", handleCalculate)
	mux.HandleFunc("/api/packs", handlePacks)
	// Serve the static UI from the "static" directory.
	mux.Handle("/", http.FileServer(http.Dir("static")))

	handler := corsMiddleware(mux)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

// corsMiddleware adds CORS headers so the API can be called from any origin.
// This is required when the UI is served from a different host during development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleCalculate accepts a POST request with the number of items ordered and
// returns the optimal pack breakdown using the current pack size configuration.
func handleCalculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Items <= 0 {
		writeError(w, http.StatusBadRequest, "Items must be a positive number")
		return
	}

	// Take a snapshot of pack sizes under the read lock to avoid holding the
	// lock during the (potentially expensive) calculation.
	mu.RLock()
	sizes := make([]int, len(packSizes))
	copy(sizes, packSizes)
	mu.RUnlock()

	result := packer.CalculatePacks(req.Items, sizes)

	// Build the response, tallying the total items that will actually be shipped.
	packs := make([]PackResult, 0, len(result))
	totalItems := 0
	for size, qty := range result {
		packs = append(packs, PackResult{Size: size, Quantity: qty})
		totalItems += size * qty
	}

	// Return packs in descending size order for readability.
	sort.Slice(packs, func(i, j int) bool {
		return packs[i].Size > packs[j].Size
	})

	writeJSON(w, http.StatusOK, OrderResponse{
		OrderedItems: req.Items,
		TotalItems:   totalItems,
		Packs:        packs,
	})
}

// handlePacks manages the set of available pack sizes.
//
//	GET    /api/packs         — list current pack sizes
//	POST   /api/packs         — add a new pack size
//	DELETE /api/packs?size=N  — remove a pack size
func handlePacks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		sizes := make([]int, len(packSizes))
		copy(sizes, packSizes)
		mu.RUnlock()

		sort.Sort(sort.Reverse(sort.IntSlice(sizes)))
		result := make([]PackSize, len(sizes))
		for i, s := range sizes {
			result[i] = PackSize{Size: s}
		}
		writeJSON(w, http.StatusOK, result)

	case http.MethodPost:
		var ps PackSize
		if err := json.NewDecoder(r.Body).Decode(&ps); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if ps.Size <= 0 {
			writeError(w, http.StatusBadRequest, "Pack size must be a positive number")
			return
		}

		mu.Lock()
		for _, s := range packSizes {
			if s == ps.Size {
				mu.Unlock()
				writeError(w, http.StatusConflict, "Pack size already exists")
				return
			}
		}
		packSizes = append(packSizes, ps.Size)
		savePackSizes(packSizes)
		mu.Unlock()

		writeJSON(w, http.StatusCreated, ps)

	case http.MethodDelete:
		sizeStr := r.URL.Query().Get("size")
		size, err := strconv.Atoi(sizeStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid size parameter")
			return
		}

		mu.Lock()
		found := false
		for i, s := range packSizes {
			if s == size {
				packSizes = append(packSizes[:i], packSizes[i+1:]...)
				found = true
				break
			}
		}
		if found {
			savePackSizes(packSizes)
		}
		mu.Unlock()

		if !found {
			writeError(w, http.StatusNotFound, "Pack size not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// loadPackSizes reads pack sizes from packsFile. If the file doesn't exist or
// cannot be parsed, the default pack sizes are returned and saved to disk.
func loadPackSizes() []int {
	data, err := os.ReadFile(packsFile)
	if err != nil {
		log.Printf("No %s found, using defaults", packsFile)
		savePackSizes(defaultPackSizes)
		return append([]int{}, defaultPackSizes...)
	}

	var sizes []int
	if err := json.Unmarshal(data, &sizes); err != nil || len(sizes) == 0 {
		log.Printf("Invalid %s, using defaults", packsFile)
		savePackSizes(defaultPackSizes)
		return append([]int{}, defaultPackSizes...)
	}

	log.Printf("Loaded pack sizes from %s: %v", packsFile, sizes)
	return sizes
}

// savePackSizes writes the current pack sizes to packsFile so they survive restarts.
// Errors are logged but not returned — a failed write is non-fatal.
func savePackSizes(sizes []int) {
	data, err := json.MarshalIndent(sizes, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal pack sizes: %v", err)
		return
	}
	if err := os.WriteFile(packsFile, data, 0644); err != nil {
		log.Printf("Failed to write %s: %v", packsFile, err)
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
