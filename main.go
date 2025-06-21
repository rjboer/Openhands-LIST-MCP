

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"./cors"
	"./sse"
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
	hub := sse.newHub()
	go hub.run()

	// 1.  Data store
	s := NewStore()

	// 2.  HTTP router
	mux := s.route(hub)

	// CORS
	corsMux := cors.newCORSHandler(mux, *origin)

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

