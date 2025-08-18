package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func getenv(k, def string) string { if v := os.Getenv(k); v != "" { return v }; return def }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func preview(b []byte) string {
	s := string(b)
	if len(s) > 256 { s = s[:256] + "...(truncated)" }
	return strings.TrimSpace(s)
}
func tokenFingerprint(h string) string {
	// don’t log tokens; show short fingerprint for debugging
	if !strings.HasPrefix(h, "Bearer ") { return "" }
	sum := sha256.Sum256([]byte(h[7:]))
	return hex.EncodeToString(sum[:])[:10]
}

func main() {
	listen := getenv("LISTEN_ADDR", ":8080")
	userinfoUp := getenv("DEX_USERINFO_URL", "https://dex.meyeringh.org/userinfo")
	tokenUp := getenv("DEX_TOKEN_URL", "https://dex.meyeringh.org/token")
	insecure := os.Getenv("TLS_INSECURE_SKIP_VERIFY") == "true"

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		},
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	http.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[shim] /userinfo hit; method=%s authPresent=%v authFp=%s",
			r.Method, r.Header.Get("Authorization") != "", tokenFingerprint(r.Header.Get("Authorization")))

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

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, userinfoUp, nil)
		if err != nil { writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "build upstream request"}); return }
		req.Header.Set("Authorization", auth)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil { writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream error contacting Dex"}); return }
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		log.Printf("[shim] Dex /userinfo -> %d; bodyPreview=%q", resp.StatusCode, preview(body))

		if resp.StatusCode != http.StatusOK {
			var asJSON any
			if json.Unmarshal(body, &asJSON) == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(resp.StatusCode)
				_, _ = w.Write(body)
			} else {
				writeJSON(w, resp.StatusCode, map[string]any{"error": "upstream non-JSON", "status": resp.Status, "raw_preview": preview(body)})
			}
			return
		}

		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "upstream returned non-JSON", "raw_preview": preview(body)})
			return
		}
		if _, ok := m["id"]; !ok {
			if sub, ok := m["sub"].(string); ok && sub != "" { m["id"] = sub }
		}
		writeJSON(w, http.StatusOK, m)
	})

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		bodyBytes, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		upReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, tokenUp, bytes.NewReader(bodyBytes))
		if err != nil { writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "build upstream request"}); return }
		upReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		upReq.Header.Set("Accept", "application/json")

		resp, err := client.Do(upReq)
		if err != nil { writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream error contacting Dex"}); return }
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("[shim] Dex /token -> %d; bodyPreview=%q", resp.StatusCode, preview(respBody))

		var parsed any
		if json.Unmarshal(respBody, &parsed) == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			_, _ = w.Write(respBody)
		} else {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error":       "upstream returned non-JSON at /token",
				"status":      resp.Status,
				"raw_preview": preview(respBody),
			})
		}
	})

	log.Printf("[shim] listening on %s; /userinfo -> %s; /token -> %s; insecureTLS=%v", listen, userinfoUp, tokenUp, insecure)
	srv := &http.Server{ Addr: listen, ReadHeaderTimeout: 5 * time.Second }
	log.Fatal(srv.ListenAndServe())
}