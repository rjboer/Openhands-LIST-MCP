
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

/*───────────────────────────────────────────────────────────────*/
/*  Functions for HTTP handlers                                  */
/*───────────────────────────────────────────────────────────────*/

/* -------------------- /index  (handle the index) ----------------------- */
func (s *Store) handleIndex(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")

	// 4.1  UI
	if path == "" || path == "index.html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(indexHTML))
		return
	}
}

/* -------------------- /meta  (summary + delay) ----------------------- */

func (s *Store) handleMeta(w http.ResponseWriter, r *http.Request) {
	type entry struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
		Open  int    `json:"open"`
	}
	out := struct {
		Lists []entry `json:"lists"`
		Delay int     `json:"delay"`
	}{}

	s.mu.RLock()
	for name, lst := range s.Lists {
		open := 0
		for _, it := range lst.Items {
			if it.Status == "open" {
				open++
			}
		}
		out.Lists = append(out.Lists, entry{name, len(lst.Items), open})
	}
	s.mu.RUnlock()

	s.delayMu.RLock()
	out.Delay = int(s.itemDelay.Seconds())
	s.delayMu.RUnlock()

	json.NewEncoder(w).Encode(out)
}

/* ---------------------- /timeout/{seconds} --------------------------- */

func (s *Store) handleTimeout(w http.ResponseWriter, secStr string) {
	secs, err := strconv.Atoi(secStr)
	if err != nil || secs < 0 || secs > 600 {
		http.Error(w, "invalid timeout (0-600)", http.StatusBadRequest)
		return
	}
	s.delayMu.Lock()
	s.itemDelay = time.Duration(secs) * time.Second
	s.delayMu.Unlock()
	fmt.Fprintf(w, `{"message":"delay set to %d s"}`, secs)
	//this works
	fmt.Println("delay set at: ", secs)
}

/* -------------------------- /add/{list} ------------------------------ */
func (s *Store) handleAdd(w http.ResponseWriter, r *http.Request,
	listName string, hub *sseHub) {

	s.mu.Lock()
	defer s.mu.Unlock()

	switch r.Method {
	case http.MethodGet: // create empty list
		if _, exists := s.Lists[listName]; exists {
			http.Error(w, "list exists", http.StatusConflict)
			return
		}
		s.Lists[listName] = &List{} // empty
		writeJSON(w, http.StatusCreated, map[string]string{
			"name": listName, "status": "created",
		})
		send(hub, map[string]string{"event": "add_list", "name": listName})

	case http.MethodPost: // seed with JSON array
		if _, exists := s.Lists[listName]; exists {
			http.Error(w, "list exists", http.StatusConflict)
			return
		}
		var items []Item
		if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// assign indices + default status
		exists := map[int]struct{}{}
		for i := range items {
			if items[i].Index == 0 {
				items[i].Index = i + 1
			}
			if _, dup := exists[items[i].Index]; dup {
				http.Error(w, "duplicate index in payload", http.StatusBadRequest)
				return
			}
			exists[items[i].Index] = struct{}{}
			if items[i].Status == "" {
				items[i].Status = "open"
			}
		}

		s.Lists[listName] = &List{Items: items}
		writeJSON(w, http.StatusCreated, items)
		send(hub, map[string]any{"event": "add_list", "name": listName, "items": items})

	default:
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
	}
}

/* ------------------------ /delete/{list} ----------------------------- */

func (s *Store) handleDelete(w http.ResponseWriter, r *http.Request,
	listName string, hub *sseHub) {

	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.Lists[listName]; !ok {
		http.Error(w, "list not found", http.StatusNotFound)
		return
	}
	delete(s.Lists, listName)
	resp := map[string]any{"name": listName, "deleted": true}
	writeJSON(w, http.StatusOK, resp)
	send(hub, resp) // ← broadcast
}

/* -------------------------- /list/{list} ----------------------------- */

func (s *Store) handleList(w http.ResponseWriter, r *http.Request,
	listName string, hub *sseHub) {

	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	list, ok := s.Lists[listName]
	if !ok {
		http.Error(w, "list not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, list.Items)
	send(hub, list.Items) // ← broadcast
}

/* -------------------------- /open/{list} ----------------------------- */

func (s *Store) handleOpen(w http.ResponseWriter, r *http.Request, name string, hub *sseHub) {
	if r.Method != http.MethodGet {
		writeUsage(w)
		return
	}
	s.delayMu.RLock()
	time.Sleep(s.itemDelay)
	s.delayMu.RUnlock()

	s.mu.RLock()
	list, ok := s.Lists[name]
	s.mu.RUnlock()
	if !ok {
		http.Error(w, "list not found", http.StatusNotFound)
		return
	}
	for _, it := range list.Items {
		if it.Status == "open" {
			json.NewEncoder(w).Encode(it)
			// Stream the observation back to OpenHands using SSE:
			b, _ := json.Marshal(it)
			hub.broadcast("data: " + string(b) + "\n\n")
			return
		}
	}
	http.Error(w, "no open item", http.StatusNotFound)
}

/* -------------------------- /close/{list} ---------------------------- */

func (s *Store) handleClose(w http.ResponseWriter, r *http.Request,
	listName string, hub *sseHub) {

	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	list, ok := s.Lists[listName]
	if !ok {
		http.Error(w, "list not found", http.StatusNotFound)
		return
	}

	// optional ?index=n query
	if idxStr := r.URL.Query().Get("index"); idxStr != "" {
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 1 || idx > len(list.Items) {
			http.Error(w, "bad index", http.StatusBadRequest)
			return
		}
		list.Items[idx-1].Status = "closed"
		writeJSON(w, http.StatusOK, list.Items[idx-1])
		send(hub, list.Items[idx-1]) // ← broadcast
		return
	}

	// otherwise close first open item
	for i := range list.Items {
		if list.Items[i].Status == "open" {
			list.Items[i].Status = "closed"
			writeJSON(w, http.StatusOK, list.Items[i])
			send(hub, list.Items[i]) // ← broadcast
			return
		}
	}
	http.Error(w, "no open item", http.StatusNotFound)
}

