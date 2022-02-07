package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"
)

var count int = 0

func itemFetcher(w http.ResponseWriter, r *http.Request) {
	itemId := r.URL.Path[7:]
	auth := r.Header.Get("Authorization")
	decodeAuth, err := base64.StdEncoding.DecodeString(auth)
	if err != nil || string(decodeAuth) != itemId {
		fmt.Fprintf(w, "Invalid authorization: Authorization header must be base 64 encoded ID")
	} else {
		count = count + 1
		fmt.Fprintf(w, "Some info about item: %s! ; Count: %d", itemId, count)
		fmt.Println(r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
		time.Sleep(3 * time.Second)
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
