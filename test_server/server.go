package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var count int = 0
var mu sync.RWMutex

func itemFetcher(w http.ResponseWriter, r *http.Request) {
	itemId := r.URL.Path[7:]
	auth := r.Header.Get("Authorization")
	decodeAuth, err := base64.StdEncoding.DecodeString(auth)
	if err != nil || string(decodeAuth) != itemId {
		fmt.Fprintf(w, "Invalid authorization: Authorization header must be base 64 encoded ID")
	} else {
		// print out the request details on the server side for debugging etc.
		fmt.Println(r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())

		// mutex so that count is not accessed/updated by conc. req. handlers at the same time
		mu.Lock()
		// maintain count => number of concurrent processes currently
		count++
		if count > 5 {
			mu.Unlock()
			// if number of concurrent requests >= 5, return max requests error
			w.WriteHeader(429)
			fmt.Fprintf(w, "429 Too Many Requests")
		} else if rand.Intn(5) == 3 {
			mu.Unlock()
			// randomly return errors every once in a while, could happen practically
			w.WriteHeader(404)
			fmt.Fprintf(w, "404 Random Error Occured")
		} else {
			mu.Unlock()
			// write the item info to the response
			fmt.Fprintf(w, "%d", rand.Intn(100000000000000000))
		}

		// added delay, as if it's processing the request
		time.Sleep(2 * time.Second)

		// reduce count before exiting handler
		mu.Lock()
		count--
		mu.Unlock()
	}
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Usage: http://localhost:8080/items/<item_id>")
}

func main() {
	http.HandleFunc("/", defaultHandler)
	http.HandleFunc("/items/", itemFetcher)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
