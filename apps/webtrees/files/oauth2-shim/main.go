package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	listenAddr := getenv("LISTEN_ADDR", ":80")
	dexUserinfo := getenv("DEX_USERINFO_URL", "https://dex.meyeringh.org/userinfo")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	http.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, dexUserinfo, nil)
		if err != nil {
			http.Error(w, "failed to build upstream request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Authorization", auth)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "upstream error contacting Dex", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// If Dex didn't return 200, just relay it.
		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
			_, _ = io.Copy(w, resp.Body)
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "failed to read upstream body", http.StatusBadGateway)
			return
		}

		// Parse, inject id=sub, and return.
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			// If JSON is somehow invalid, just pass through as-is.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}

		if _, ok := m["id"]; !ok {
			if sub, ok := m["sub"].(string); ok && sub != "" {
				m["id"] = sub
			}
		}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(m); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	log.Printf("oauth2 userinfo shim listening on %s; proxying to %s", listenAddr, dexUserinfo)
	srv := &http.Server{
		Addr:              listenAddr,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
