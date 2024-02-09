package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Message struct {
	Status string `json:"status"`
	Body   string `json:"body"`
}

func perClientRateLimiter(next func(w http.ResponseWriter, r *http.Request)) http.Handler {
	fmt.Println("testing rate limiter")
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)
	go func() {
		// for { ... }: This is a forever loop, also known as an infinite loop. 
		// It continues indefinitely until the program is terminated externally
		// use it if u want something in a middleware to persist
		fmt.Println("create a clean up routine")
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
	// in a middleware only return is run everysingle time its called. for everything else,
	// u might have to use for ever loops like above
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("handle func")
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		mu.Lock()
		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(2, 4)}
		}
		clients[ip].lastSeen = time.Now()
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			message := Message{
				Status: "Request failed",
				Body:   "the API is at capacity",
			}
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(message)
			return
		}
		mu.Unlock()
		next(w, r)
	})
}
func endpointHandler(writer http.ResponseWriter, request *http.Request) {
	fmt.Println("testing endpoint")
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	message := Message{
		Status: "Successful",
		Body:   "Hi, you have reached the API. How may I help you?",
	}
	err := json.NewEncoder(writer).Encode(&message)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	// since perCleintRateLimiter is middle ware, it is actually run only once
	// but its effects such as the clean up and rate limiting logic persists.
	http.Handle("/ping", perClientRateLimiter(endpointHandler))
	fmt.Println("starting server at port 9000")
	err := http.ListenAndServe(":9000", nil)
	if err != nil {
		log.Println(err)
	}
}