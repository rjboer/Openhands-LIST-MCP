package main

import (
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
