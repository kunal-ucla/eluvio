package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var wg = &sync.WaitGroup{}

func sendRequest(baseUrl string, itemID string) []interface{} {

	// generate authorization header using the id (convert id to base64)
	auth := base64.URLEncoding.EncodeToString([]byte(itemID))

	// prepare the target URL using the item id
	url := baseUrl + itemID

	// create a Client, define request, set header and perform req
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", auth)

	var body []byte
	var err error
	var res *http.Response
	var backoffSchedule = []time.Duration{
		1 * time.Second,
		3 * time.Second,
		10 * time.Second,
	}

	for _, backoff := range backoffSchedule {
		// perfrom the get request
		res, err = client.Do(req)

		// log the error if any during the GET request
		if err != nil {
			log.Fatal(err)
		}

		// read the response and close the response body
		body, _ = ioutil.ReadAll(res.Body)
		code := res.StatusCode
		res.Body.Close()

		// if success, break out
		if code == 200 {
			break
		}

		// sleep till backoff timer for retrying
		time.Sleep(backoff)
	}

	// send response through the channel for printing
	return []interface{}{string(body[:]), itemID}
}

func requestInfo(baseUrl string, item string) {
	channel := make(chan []interface{})
	buff := sendRequest(baseUrl, item)
	fmt.Printf("Info for item[ %s ] received as [ %s ]\n", buff[1].(string), buff[0].(string))
	close(channel)
	wg.Done()
}

func main() {

	/* Idea is to create 5 Goroutines to keep sending GET requests simultaneously */

	baseUrl := "http://localhost:8080/items/"
	// baseUrl := os.Args[1]
	// baseUrl := "https://challenges.qluv.io/items/"

	// declare 5 items to send per Goroutine
	var items [5]string

	// genereate a sample itemList for testing
	itemList := make([]string, 100)
	for i := 0; i < 100; i++ {
		itemList[i] = fmt.Sprint(rand.Intn(18))
	}

	// dictionary for already queried items
	dict := make(map[string]bool)

	// run through itemList, 5 at a time
	for current := 0; current < len(itemList); {
		// copy top 5 ids from itemList to items[]

		var idx int // declaring this outside for loop to retain it's value afterwards
		for idx = 0; idx < 5 && current < len(itemList); current++ {
			// check if current id is already queried
			if !dict[itemList[current]] {
				// if new, store in dict as queried now
				dict[itemList[current]] = true
				// copy from itemList to items and increment idx
				items[idx], idx = itemList[current], idx+1
			}
		}

		// create 5 Goroutines
		for i := 0; i < idx; i++ {
			wg.Add(1)
			go requestInfo(baseUrl, items[i])
		}
	}
	wg.Wait()
}
