package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
)

// JSONRPCRequest mirrors the JSON-RPC 2.0 request shape from internal/trigger.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      int64           `json:"id"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	Result  string `json:"result,omitempty"`
	ID      int64  `json:"id"`
}

// CellWrittenParams is the notification payload for cell.written.
type CellWrittenParams struct {
	AddedID    int64           `json:"added_id"`
	RowKey     string          `json:"row_key"`
	ColumnName string          `json:"column_name"`
	RefKey     int64           `json:"ref_key"`
	Body       json.RawMessage `json:"body"`
	CreatedAt  string          `json:"created_at"`
	ShardID    int             `json:"shard_id"`
}

// BillingEvent is the expected body shape for billing column writes.
// TODO we've integrated this with seed_users for now, so it's not ideal, but
// it demonstrates immediately that BillingEvents are possible for customers.
type BillingEvent struct {
	Customer    string  `json:"email"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// ledger tracks accumulated charges per customer.
type ledger struct {
	mu      sync.Mutex
	charges map[string]float64
}

func (l *ledger) add(customer string, amount float64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.charges[customer] += amount
}

func (l *ledger) printSummary() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.charges) == 0 {
		fmt.Println("\nNo charges recorded.")
		return
	}

	customers := make([]string, 0, len(l.charges))
	for c := range l.charges {
		customers = append(customers, c)
	}
	sort.Strings(customers)

	fmt.Println("\n=== Billing Summary ===")
	var total float64
	for _, c := range customers {
		amt := l.charges[c]
		total += amt
		fmt.Printf("  %-20s $%.2f\n", c, amt)
	}
	fmt.Printf("  %-20s -------\n", "")
	fmt.Printf("  %-20s $%.2f\n", "TOTAL", total)
}

func main() {
	mezzanineURL := "http://localhost:8080"
	if u := os.Getenv("MEZZANINE_URL"); u != "" {
		mezzanineURL = u
	}

	port := "9001"
	if p := os.Getenv("PLUGIN_PORT"); p != "" {
		port = p
	}

	l := &ledger{charges: make(map[string]float64)}

	// Start JSON-RPC server.
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", rpcHandler(l))

	server := &http.Server{Addr: ":" + port, Handler: mux}

	go func() {
		fmt.Printf("Billing plugin listening on :%s/rpc\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Register with Mezzanine.
	endpoint := fmt.Sprintf("http://localhost:%s/rpc", port)
	registerPlugin(mezzanineURL, endpoint)

	// Wait for shutdown signal, then print summary.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	l.printSummary()
}

func rpcHandler(l *ledger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if req.Method != "cell.written" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				Result:  "unknown method",
				ID:      req.ID,
			})
			return
		}

		var params CellWrittenParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			fmt.Printf("  [warn] failed to parse params: %v\n", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(JSONRPCResponse{JSONRPC: "2.0", Result: "ok", ID: req.ID})
			return
		}

		var event BillingEvent
		if err := json.Unmarshal(params.Body, &event); err != nil {
			fmt.Printf("  [warn] failed to parse billing event body: %v\n", err)
		} else {
			l.add(event.Customer, event.Amount)
			fmt.Printf("  [charge] customer=%s amount=$%.2f desc=%q row_key=%s\n",
				event.Customer, event.Amount, event.Description, params.RowKey)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(JSONRPCResponse{JSONRPC: "2.0", Result: "ok", ID: req.ID})
	}
}

func registerPlugin(mezzanineURL, endpoint string) {
	body, err := json.Marshal(map[string]any{
		"name":               "billing",
		"endpoint":           endpoint,
	//	"subscribed_columns": []string{"billing"},
		"subscribed_columns": []string{"profile"},
	})
	if err != nil {
		log.Fatalf("failed to marshal registration body: %v", err)
	}

	resp, err := http.Post(mezzanineURL+"/v1/plugins", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Fatalf("failed to register plugin: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// Allow StatusConflict so that we can restart
		if resp.StatusCode != http.StatusConflict {
			log.Fatalf("unexpected status registering plugin: %d", resp.StatusCode)
		}
	}

	fmt.Printf("Registered billing plugin with Mezzanine at %s\n", mezzanineURL)
}
