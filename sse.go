package main

import (
        "time"
)

// sseHub manages SSE clients and broadcasts messages.
type sseHub struct {
        clients   map[chan string]bool
        addClient chan chan string
        rmClient  chan chan string
        broadcast chan string
}

func NewHub() *sseHub {
        return &sseHub{
                clients:   make(map[chan string]bool),
                addClient: make(chan chan string),
                rmClient:  make(chan chan string),
                broadcast: make(chan string),
        }
}

func (h *sseHub) add(ch chan string) {
        h.addClient <- ch
}

func (h *sseHub) remove(ch chan string) {
        h.rmClient <- ch
}

func (h *sseHub) run() {
        for {
                select {
                case ch := <-h.addClient:
                        h.clients[ch] = true
                case ch := <-h.rmClient:
                        delete(h.clients, ch)
                case msg := <-h.broadcast:
                        for ch := range h.clients {
                                ch <- msg
                        }
                }
        }
}

// sendCommentPing sends a keep-alive comment to the SSE stream every interval.
func sendCommentPing(hub *sseHub, interval time.Duration) {
        startPings.Do(func() {
                go func() {
                        for range time.Tick(interval) {
                                hub.broadcast <- ": keep-alive\n\n" // comment ping
                        }
                }()
        })
}

