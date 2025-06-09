package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
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
	mu        sync.RWMutex
	Lists     map[string]*List
	delayMu   sync.RWMutex
	itemDelay time.Duration // throttle between tasks
}

var startPings sync.Once
var debug bool         //debug flag
func NewStore() *Store { return &Store{Lists: make(map[string]*List)} }

/* --------------------------------------------------------------------- */
/* 2.  Embedded index.html                                               */
/*	yeah...i put it here...												 */
/* --------------------------------------------------------------------- */

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Review-Board</title>
<style>
body{font-family:sans-serif;margin:2rem}
table{border-collapse:collapse;width:100%;margin-bottom:2rem}
th,td{border:1px solid #ddd;padding:.4rem;text-align:left}
tr:hover{background:#f3f3f3}
.badge{padding:2px 6px;border-radius:4px;color:#fff;font-size:.8rem}
.open{background:#28a745}.closed{background:#6c757d}
form{margin-top:1rem}
</style>
</head>
<body>


<h1>OpenHands MCP List Tool</h1>

<!-- Route cheat-sheet -->
<table class="routes">
<thead><tr><th>Verb & Path</th><th>Description</th></tr></thead>
<tbody>
<tr><td><code>GET /open/{list}</code></td>   <td>Return first open item with its index</td></tr>
<tr><td><code>GET /close/{list}?index=n</code></td><td>Close item (<code>index</code> optional)</td></tr>
<tr><td><code>GET /add/{list}</code></td>    <td>Create empty list</td></tr>
<tr><td><code>POST /add/{list}</code></td>   <td>Create list, seed with JSON array</td></tr>
<tr><td><code>GET /delete/{list}</code></td> <td>Delete list</td></tr>
<tr><td><code>GET /list/{list}</code></td>   <td>Return full list (JSON)</td></tr>
<tr><td><code>GET /timeout/{seconds}</code></td><td>Set throttle delay (0-600 s)</td></tr>
<tr><td><code>GET /meta</code></td>          <td>Summary for index page</td></tr>
<tr><td><code>/ or /index.html</code></td>   <td>This web UI</td></tr>
</tbody>
</table>

<table id="lists">
<thead><tr><th>Name</th><th>Total</th><th>Open</th></tr></thead>
<tbody></tbody></table>

<h2>Throttle Open / Close</h2>
<div>
<p> The main function of the throttle is to slow down the AI tool</p>
<p> This way you can use gemini without running immediately into limits</p>
</div>
<form id="delayForm">
<label>Delay&nbsp;(seconds):
  <input id="delaySeconds" type="number" min="0" max="600" value="0" required>
</label>
<button type="submit">Set&nbsp;delay</button>
<span id="currentDelay" style="margin-left:1rem;color:#555"></span>
</form>

<h2>Add / Seed List</h2>
<form id="seedForm">
<label>List&nbsp;name:
  <input id="listName" required>
</label><br><br>
<label>JSON&nbsp;array&nbsp;of&nbsp;items:<br>
  <textarea id="jsonBody" rows="10" cols="80"
   placeholder='[{"Document":"a.md","conflict":"x","new_statement":"y"}]'></textarea>
</label><br><br>
<button type="submit">POST /add/{list}</button>
</form>

<script>
async function refresh(){
  const res=await fetch('/meta');
  const data=await res.json();

  // update list table
  const tbody=document.querySelector('#lists tbody');
  tbody.innerHTML='';
  data.lists.forEach(l=>{
    const tr=document.createElement('tr');
    tr.innerHTML=
      '<td>'+l.name+'</td>'+
      '<td>'+l.count+'</td>'+
      '<td><span class="badge '+(l.open? "open":"closed")+'">'+l.open+'</span></td>';
    tbody.appendChild(tr);
  });

  // update delay display
  document.getElementById('currentDelay').textContent='current: '+data.delay+' s';
  document.getElementById('delaySeconds').value=data.delay;
}

document.getElementById('delayForm').addEventListener('submit',async e=>{
  e.preventDefault();
  const secs=document.getElementById('delaySeconds').value;
  try{
    const res=await fetch('/timeout/'+secs);
    if(!res.ok) throw new Error(await res.text());
    refresh();
  }catch(err){alert(err);}
});

document.getElementById('seedForm').addEventListener('submit',async e=>{
  e.preventDefault();
  const name=document.getElementById('listName').value.trim();
  const body=document.getElementById('jsonBody').value.trim()||'[]';
  try{
    const res=await fetch('/add/'+encodeURIComponent(name),{
      method:'POST',
      headers:{'Content-Type':'application/json'},
      body:body
    });
    if(!res.ok) throw new Error(await res.text());
    alert('Success!');
    document.getElementById('jsonBody').value='';
    refresh();
  }catch(err){alert(err);}
});

refresh(); setInterval(refresh,5000);
</script>
</body></html>`

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

/*───────────────────────────────────────────────────────────────*/
/*  Functions for HTTP handlers                                  */
/*───────────────────────────────────────────────────────────────*/

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

/* ------------------------------------------------------------------ */
/* 0-bis.  SSE hub (thread-safe)                           */
/* ------------------------------------------------------------------ */

type sseHub struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

func newHub() *sseHub { return &sseHub{clients: make(map[chan string]struct{})} }

func (h *sseHub) add(ch chan string) {
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
}

func (h *sseHub) remove(ch chan string) {
	h.mu.Lock()
	delete(h.clients, ch)
	close(ch)
	h.mu.Unlock()
}

func (h *sseHub) broadcast(msg string) {
	h.mu.RLock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default: /* slow client, drop */
		}
	}
	h.mu.RUnlock()
}

// sendCommentPing starts a goroutine that broadcasts an SSE “comment” (":\n")
// this functionas as a keep alive.  Call it **once**!!!!! after the first handshake.
func sendCommentPing(hub *sseHub, interval time.Duration) {
	startPings.Do(func() {
		go func() {
			sendPing := func(tag string) {
				hub.mu.RLock()
				empty := len(hub.clients) == 0
				hub.mu.RUnlock()
				if empty {
					return
				}
				hub.broadcast(":\n\n")
				if debug {
					log.Printf("[DEBUG] %s SSE keep-alive ping sent at %v\n", tag, time.Now())
				}
			}

			// Send initial ping, This keeps openhands happy:P
			sendPing("Initial")

			t := time.NewTicker(interval)
			for range t.C {
				sendPing("Periodic")
			}
		}()
	})
}

// loggingMiddleware loggs all connection request comming in.
// handy to review all openhands communication
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if debug {
			log.Printf("[TRACE] %s %s", r.Method, r.URL.String())
			log.Printf(" ↳ RemoteAddr: %s", r.RemoteAddr)
			log.Printf(" ↳ Host: %s", r.Host)
			log.Printf(" ↳ User-Agent: %s", r.UserAgent())
			log.Printf(" ↳ Accept: %s", r.Header.Get("Accept"))
			log.Printf(" ↳ Content-Type: %s", r.Header.Get("Content-Type"))
			log.Printf(" ↳ X-Forwarded-For: %s", r.Header.Get("X-Forwarded-For"))
		}
		next.ServeHTTP(w, r)
	})
}

/* --------------------------------------------------------------------- */
/* 5.  main                                                              */
/* --------------------------------------------------------------------- */

func main() {
	debug = true
	fmt.Println("Starting Roelof Jan Boer's list MCP tool")
	defPort := "3002" // compile-time default
	flagPort := flag.String("port", defPort, "TCP port to listen on")
	flag.BoolVar(&debug, "debug", true, "enable debug logging output")
	flag.Parse()
	fmt.Println("Logging is: ", debug)

	port := os.Getenv("PORT")
	if *flagPort != defPort { // user passed --port
		port = *flagPort
	}
	if port == "" { // neither flag nor env
		port = defPort
	}

	addr := ":" + port
	store := NewStore()
	events := newHub()

	log.Printf("🔗  Listening on %s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(store.route(events))); err != nil {
		log.Fatal(err)
	}
}
