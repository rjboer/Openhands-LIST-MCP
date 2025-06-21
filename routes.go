

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

/* --------------------------------------------------------------------- */
/* 4.  HTTP router  s                                                     */
/* --------------------------------------------------------------------- */

// route returns a fully-wired *http.ServeMux*.
// Pass the shared hub so handlers can stream events.
func (s *Store) route(hub *sseHub) *http.ServeMux {
	mux := http.NewServeMux()

	/*──────────────── MCP endpoints ───────────────*/

	// 3.1  POST /mcp  — handshake
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}

		// Tell OpenHands which tools exist
		manifest := map[string]any{
			"tools": []map[string]string{
				{"name": "open_item", "description": "Return first open item"},
				{"name": "close_item", "description": "Close an item"},
				{"name": "list_items", "description": "Return full list"},
			},
		}
		b, _ := json.Marshal(manifest)
		hub.broadcast("data: " + string(b) + "\n\n")

		// Start 25-second keep-alive pings *only once*
		sendCommentPing(hub, 10*time.Second)

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// 3.2  GET /mcp/sse  — event stream
	sseHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "stream unsupported", 500)
			return
		}
		//register client
		ch := make(chan string, 8)
		hub.add(ch)
		defer hub.remove(ch)

		fmt.Fprint(w, ":\n\n") // initial ping
		flusher.Flush()

		for {
			select {
			case <-r.Context().Done():
				return
			case msg := <-ch:
				fmt.Fprint(w, msg)
				flusher.Flush()
			}
		}
	}

	mux.HandleFunc("/mcp/sse", sseHandler)  // SSE handler...
	mux.HandleFunc("/mcp/sse/", sseHandler) // handles the trailing slash sometimes produced by openhands....

	/*───────────── Existing REST / UI handlers ─────────────*/
	mux.HandleFunc("/open/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/open/")
		s.handleOpen(w, r, name, hub) // pass hub for broadcast
	})
	mux.HandleFunc("/close/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/close/")
		s.handleClose(w, r, name, hub) // pass hub for broadcast
	})
	mux.HandleFunc("/add/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/add/")
		s.handleAdd(w, r, name, hub)
	})
	mux.HandleFunc("/delete/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/delete/")
		s.handleDelete(w, r, name, hub)
	})
	mux.HandleFunc("/list/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/list/")
		s.handleList(w, r, name, hub)
	})
	mux.HandleFunc("/timeout/", func(w http.ResponseWriter, r *http.Request) {
		secs := strings.TrimPrefix(r.URL.Path, "/timeout/")
		s.handleTimeout(w, secs)
	})
	mux.HandleFunc("/meta", s.handleMeta)
	mux.HandleFunc("/", s.handleIndex)

	return mux
}

