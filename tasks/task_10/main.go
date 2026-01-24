package main

import (
	"log"
	"net/http"
	"time"
)

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func main() {
	addr := ":8080"
	repo := NewInMemoryTaskRepo(realClock{})
	handler := NewHTTPHandler(repo)

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
