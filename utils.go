
package main

import (
	"encoding/json"
	"net/http"
)

/* --------------------------------------------------------------------- */
/* 3.  Usage helper                                                      */
/* --------------------------------------------------------------------- */

func writeUsage(w http.ResponseWriter) {
	const help = `Valid endpoints (all JSON):

GET  /open/{list}              → first open item with its index
GET  /close/{list}?index=n     → close item (index optional)
GET  /add/{list}               → create empty list
POST /add/{list}               → create list, seed JSON array
GET  /delete/{list}            → delete list
GET  /list/{list}              → full list JSON
GET  /timeout/{seconds}        → set throttle delay (0-600 s)
GET  /meta                     → summary for index page
/ or /index.html               → web UI`
	http.Error(w, help, http.StatusBadRequest)
}

// writeJSON writes v as application/json and ignores secondary errors.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// send broadcasts v as an SSE data frame if hub != nil.
func send(hub *sseHub, v any) {
	if hub == nil {
		return
	}
	if b, err := json.Marshal(v); err == nil {
		hub.broadcast("data: " + string(b) + "\n\n")
	}
}
