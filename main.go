package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

/* --------------------------------------------------------------------- */
/* 1.  Data model                                                        */
/* --------------------------------------------------------------------- */

type Item struct {
	Index        int    `json:"index"`
	Document     string `json:"Document"`
	Conflict     string `json:"conflict"`
	NewStatement string `json:"new_statement"`
	Status       string `json:"status"`
}

type List struct {
	Name  string
	Items []Item
}

type Store struct {
	mu    sync.RWMutex
	Lists map[string]*List
}

func NewStore() *Store { return &Store{Lists: make(map[string]*List)} }

/* --------------------------------------------------------------------- */
/* 2.  Helper: usage text                                                */
/* --------------------------------------------------------------------- */

func writeUsage(w http.ResponseWriter) {
	const help = `Valid endpoints (all JSON replies):

GET  /open/{list}              → first open item with its index
GET  /close/{list}?index=n     → set item.status="closed" (index optional)
GET  /add/{list}               → create empty list
POST /add/{list}               → create list and seed with JSON array
GET  /delete/{list}            → delete list
GET  /list/{list}              → full list as JSON`
	http.Error(w, help, http.StatusBadRequest)
}

/* --------------------------------------------------------------------- */
/* 3.  HTTP handler                                                      */
/* --------------------------------------------------------------------- */

func (s *Store) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		writeUsage(w)
		return
	}
	action, listName := parts[0], parts[1]

	switch action {
	case "add":
		s.handleAdd(w, r, listName)
	case "delete":
		s.handleDelete(w, r, listName)
	case "list":
		s.handleList(w, r, listName)
	case "open":
		s.handleOpen(w, r, listName)
	case "close":
		s.handleClose(w, r, listName)
	default:
		writeUsage(w)
	}
}

/* -------------------------- /add/{list} ------------------------------ */

func (s *Store) handleAdd(w http.ResponseWriter, r *http.Request, name string) {
	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, exists := s.Lists[name]; exists {
			http.Error(w, "list already exists", http.StatusConflict)
			return
		}
		s.Lists[name] = &List{Name: name}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"message":"list %q created empty"}`, name)

	case http.MethodPost:
		var items []Item
		if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, exists := s.Lists[name]; exists {
			http.Error(w, "list already exists", http.StatusConflict)
			return
		}
		// ensure sequential indices
		for i := range items {
			items[i].Index = i + 1
			if items[i].Status == "" {
				items[i].Status = "open"
			}
		}
		s.Lists[name] = &List{Name: name, Items: items}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(items)

	default:
		writeUsage(w)
	}
}

/* ------------------------ /delete/{list} ----------------------------- */

func (s *Store) handleDelete(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		writeUsage(w)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Lists[name]; !ok {
		http.Error(w, "list not found", http.StatusNotFound)
		return
	}
	delete(s.Lists, name)
	fmt.Fprintf(w, `{"message":"list %q deleted"}`, name)
}

/* -------------------------- /list/{list} ----------------------------- */

func (s *Store) handleList(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		writeUsage(w)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	list, ok := s.Lists[name]
	if !ok {
		http.Error(w, "list not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(list.Items)
}

/* -------------------------- /open/{list} ----------------------------- */

func (s *Store) handleOpen(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		writeUsage(w)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	list, ok := s.Lists[name]
	if !ok {
		http.Error(w, "list not found", http.StatusNotFound)
		return
	}
	for _, it := range list.Items {


// handleTimeout handles the /timeout/{list} endpoint.
func (s *Store) handleTimeout(w http.ResponseWriter, r *http.Request, name string) {
	// TODO: Implement timeout functionality
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"timeout not implemented"}`)
}


		if it.Status == "open" {
			json.NewEncoder(w).Encode(it)
			return
		}
	}
	http.Error(w, "no open item", http.StatusNotFound)
}

/* -------------------------- /close/{list} ---------------------------- */

func (s *Store) handleClose(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		writeUsage(w)
		return
	}
	q := r.URL.Query().Get("index")

	s.mu.Lock()
	defer s.mu.Unlock()
	list, ok := s.Lists[name]
	if !ok {
		http.Error(w, "list not found", http.StatusNotFound)
		return
	}

	// helper to encode and reply
	reply := func(it *Item) {
		json.NewEncoder(w).Encode(it)
	}

	// close by explicit index
	if q != "" {
		idx, err := strconv.Atoi(q)
		if err != nil || idx <= 0 || idx > len(list.Items) {
			http.Error(w, "invalid index", http.StatusBadRequest)
			return
		}
		list.Items[idx-1].Status = "closed"
		reply(&list.Items[idx-1])
		return
	}

	// otherwise close first open
	for i := range list.Items {
		if list.Items[i].Status == "open" {
			list.Items[i].Status = "closed"
			reply(&list.Items[i])
			return
		}
	}
	http.Error(w, "no open item to close", http.StatusNotFound)
}

/* --------------------------------------------------------------------- */
/* 4.  main                                                              */
/* --------------------------------------------------------------------- */

func main() {
	store := NewStore()

	addr := ":8080"
	fmt.Printf("🔗  Listening on http://localhost%s  (try /add/todo)\n", addr)
	if err := http.ListenAndServe(addr, store); err != nil {
		log.Fatal(err)
	}
}

// SetTimeout sets the timeout between serving list items.
func (s *Store) SetTimeout(w http.ResponseWriter, r *http.Request, timeout int) {
	// TODO: Implement timeout functionality
	fmt.Fprintf(w, `{"message":"timeout set to %d seconds"}`, timeout)
}

