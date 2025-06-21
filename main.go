package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var debug bool //debug flag

func main() {
	var (
		addr      = flag.String("addr", ":54077", "server listen address")
		origin    = flag.String("origin", "*", "CORS origin")
		assetsDir = flag.String("assets", "./assets", "path to assets")
	)
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.Parse()

	// 0.  Server setup
	log.SetFlags(log.Lshortfile)
	fmt.Println("starting server at", *addr)

	// 0-bis.  SSE hub (thread-safe)
	hub := newHub()
	go hub.run()

	// 1.  Data store
	s := NewStore()

	// 2.  HTTP router
	mux := s.route(hub)

	// CORS
	corsMux := newCORSHandler(mux, *origin)

	// 4.  Static assets
	fs := http.FileServer(http.Dir(*assetsDir))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	// 5.  Start server
	server := &http.Server{
		Addr:           *addr,
		Handler:        corsMux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		ErrorLog:       log.New(os.Stderr, "ERR: ", log.LstdFlags),
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}



	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
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
// Send initial ping, This keeps openhands happy:P
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

			// Send initial ping immediately
			sendPing("Initial")

			// Then start periodic pinging
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
