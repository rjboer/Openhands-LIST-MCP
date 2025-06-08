package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

func NewStore() *Store { return &Store{Lists: make(map[string]*List)} }

/* --------------------------------------------------------------------- */
/* 2.  Embedded index.html                                               */
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
/* 4.  HTTP router                                                       */
/* --------------------------------------------------------------------- */

func (s *Store) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")

	// 4.1  UI
	if path == "" || path == "index.html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(indexHTML))
		return
	}

	// 4.2  helpers
	switch {
	case path == "meta":
		s.handleMeta(w)
		return
	case strings.HasPrefix(path, "timeout/"):
		s.handleTimeout(w, strings.TrimPrefix(path, "timeout/"))
		return
	}

	// 4.3  two-segment REST
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		writeUsage(w)
		return
	}
	action, list := parts[0], parts[1]

	switch action {
	case "add":
		s.handleAdd(w, r, list)
	case "delete":
		s.handleDelete(w, r, list)
	case "list":
		s.handleList(w, r, list)
	case "open":
		s.handleOpen(w, r, list)
	case "close":
		s.handleClose(w, r, list)
	default:
		writeUsage(w)
	}
}

/* -------------------- /meta  (summary + delay) ----------------------- */

func (s *Store) handleMeta(w http.ResponseWriter) {
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
		fmt.Fprintf(w, `{"message":"list %q created"}`, name)

	case http.MethodPost:
		var items []Item
		if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		if _, exists := s.Lists[name]; !exists {
			s.Lists[name] = &List{Name: name}
		}
		base := len(s.Lists[name].Items)
		for i := range items {
			items[i].Index = base + i + 1
			if items[i].Status == "" {
				items[i].Status = "open"
			}
			s.Lists[name].Items = append(s.Lists[name].Items, items[i])
		}
		s.mu.Unlock()

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
	list, ok := s.Lists[name]
	s.mu.RUnlock()
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
	s.delayMu.RLock()
	time.Sleep(s.itemDelay)
	s.delayMu.RUnlock()

	q := r.URL.Query().Get("index")

	s.mu.Lock()
	defer s.mu.Unlock()
	list, ok := s.Lists[name]
	if !ok {
		http.Error(w, "list not found", http.StatusNotFound)
		return
	}

	reply := func(it *Item) { json.NewEncoder(w).Encode(it) }

	if q != "" { // explicit index
		idx, err := strconv.Atoi(q)
		if err != nil || idx <= 0 || idx > len(list.Items) {
			http.Error(w, "invalid index", http.StatusBadRequest)
			return
		}
		list.Items[idx-1].Status = "closed"
		reply(&list.Items[idx-1])
		return
	}

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
/* 5.  main                                                              */
/* --------------------------------------------------------------------- */

func main() {
	store := NewStore()
	addr := ":3002"
	fmt.Printf("🔗  Listening at http://localhost%s  – UI on /\n", addr)
	if err := http.ListenAndServe(addr, store); err != nil {
		log.Fatal(err)
	}
}
