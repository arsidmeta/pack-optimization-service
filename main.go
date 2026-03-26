package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"

	"github.com/arsid/pack-optimization-service/packer"
)

type PackSize struct {
	Size int `json:"size"`
}

type OrderRequest struct {
	Items int `json:"items"`
}

type PackResult struct {
	Size     int `json:"size"`
	Quantity int `json:"quantity"`
}

type OrderResponse struct {
	OrderedItems int          `json:"ordered_items"`
	TotalItems   int          `json:"total_items"`
	Packs        []PackResult `json:"packs"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

var (
	packSizes = []int{250, 500, 1000, 2000, 5000}
	mu        sync.RWMutex
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/calculate", handleCalculate)
	mux.HandleFunc("/api/packs", handlePacks)
	mux.Handle("/", http.FileServer(http.Dir("static")))

	handler := corsMiddleware(mux)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

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

	mu.RLock()
	sizes := make([]int, len(packSizes))
	copy(sizes, packSizes)
	mu.RUnlock()

	result := packer.CalculatePacks(req.Items, sizes)

	packs := make([]PackResult, 0, len(result))
	totalItems := 0
	for size, qty := range result {
		packs = append(packs, PackResult{Size: size, Quantity: qty})
		totalItems += size * qty
	}

	sort.Slice(packs, func(i, j int) bool {
		return packs[i].Size > packs[j].Size
	})

	resp := OrderResponse{
		OrderedItems: req.Items,
		TotalItems:   totalItems,
		Packs:        packs,
	}

	writeJSON(w, http.StatusOK, resp)
}

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

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
