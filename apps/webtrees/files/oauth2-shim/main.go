package main

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	listen := getenv("LISTEN_ADDR", ":80")
	upstream := getenv("DEX_USERINFO_URL", "https://dex.meyeringh.org/userinfo")
	insecure := os.Getenv("TLS_INSECURE_SKIP_VERIFY") == "false"

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		},
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	http.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, POST")
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing Authorization header"})
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstream, nil)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build upstream request"})
			return
		}
		req.Header.Set("Authorization", auth)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream error contacting Dex"})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		preview := string(body)
		if len(preview) > 256 {
			preview = preview[:256] + "...(truncated)"
		}
		log.Printf("Dex /userinfo -> %d, body preview: %q", resp.StatusCode, preview)

		if resp.StatusCode != http.StatusOK {
			// try to pass through JSON error bodies; otherwise wrap
			var asJSON any
			if json.Unmarshal(body, &asJSON) == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(resp.StatusCode)
				_, _ = w.Write(body)
			} else {
				writeJSON(w, resp.StatusCode, map[string]string{
					"error":       "upstream non-JSON",
					"status":      http.StatusText(resp.StatusCode),
					"raw_preview": strings.TrimSpace(preview),
				})
			}
			return
		}

		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			// If Dex returned non-JSON, wrap it so the client still gets JSON.
			writeJSON(w, http.StatusBadGateway, map[string]string{
				"error":       "upstream returned non-JSON",
				"raw_preview": strings.TrimSpace(preview),
			})
			return
		}

		if _, ok := m["id"]; !ok {
			if sub, ok := m["sub"].(string); ok && sub != "" {
				m["id"] = sub
			}
		}

		if _, ok := m["username"]; !ok {
			if sub, ok := m["preferred_username"].(string); ok && sub != "" {
				m["username"] = sub
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(m)
	})

	log.Printf("userinfo shim on %s -> %s (insecureTLS=%v)", listen, upstream, insecure)
	srv := &http.Server{Addr: listen, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
