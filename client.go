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

// waitgroup so that we wait till all the items are queried
var wg = &sync.WaitGroup{}

// backoff params
var maxBackoff int = 3
var backoffSchedule = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	10 * time.Second,
}

// mutex for 'dict'; will update item as 'not processed' in goRoutine if it's query fails
var mu sync.RWMutex
var dict map[string]bool = make(map[string]bool) // dictionary for already queried items

func sendRequest(baseUrl string, itemID string) []interface{} {

	// generate authorization header using the id (convert id to base64)
	auth := base64.URLEncoding.EncodeToString([]byte(itemID))

	// prepare the target URL using the item id
	url := baseUrl + itemID

	// create a Client, define request, set header and perform req
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", auth)

	// perfrom the get request
	res, err := client.Do(req)

	// log the error if any during the GET request
	if err != nil {
		log.Fatal(err)
	}

	// read the response and close the response body
	body, _ := ioutil.ReadAll(res.Body)
	code := res.StatusCode
	res.Body.Close()

	// send response through the channel for printing
	return []interface{}{string(body[:]), itemID, code}
}

func requestInfo(baseUrl string, channel chan []interface{}) {
	// keep picking available item query requests from the channel (per goRoutine)
	for item := range channel {
		// perform the GET request and store the response in buff
		buff := sendRequest(baseUrl, item[0].(string))
		if buff[2].(int) != 200 {
			// if status code is not 200, perfrom backoff
			if item[1].(int) < maxBackoff {
				// start a goRoutine to add this item back to channel after backoff timer
				go backoffHandler(item[0].(string), item[1].(int), channel)
				// debug print
				// fmt.Println("Retrying for item:", item[0].(string))
			} else {
				// reached end of trial for current item (failed), mark its wg as done
				wg.Done()
				// mutex lock/unlock to write to 'dict'
				mu.Lock()
				// mark as false so that if this item re-appears in list, let it proceed
				dict[item[0].(string)] = false
				mu.Unlock()
				// debug print
				// fmt.Println("Fetch failed for item:", item[0].(string))
			}
		} else {
			// reached end of trial for current item (success), mark its wg as done
			wg.Done()
			// debug print
			fmt.Printf("Info for item[ %s ] received as [ %s ]\n", buff[1].(string), buff[0].(string))
		}
	}
}

func backoffHandler(item string, attempt int, channel chan []interface{}) {
	// start backoff timer according to the current attempt
	time.Sleep(backoffSchedule[attempt])
	// push the item back to the channel after backoff timer expires
	channel <- []interface{}{item, attempt + 1}
}

func main() {

	/* Idea is to create 5 Goroutines to keep sending GET requests simultaneously
	and keep sending requests to these routines using a channel (ideally with same capacity i.e. 5) */

	// baseUrl := "http://localhost:8080/items/"   // use when testing with test_server/server.go
	baseUrl := "https://challenges.qluv.io/items/" // given test server

	// genereate a sample itemList for testing
	itemList := make([]string, 1000)
	for i := 0; i < len(itemList); i++ {
		// random set of ints, less than itemList size so that some item id's repeat
		itemList[i] = fmt.Sprint(rand.Intn(380))
	}
	// make a channel with a capacity to hold as many items
	channel := make(chan []interface{}, 5)

	// to measure time taken for all the queries
	start := time.Now()

	// start 5 goRoutines for sending the query requests
	for idx := 0; idx < 5; idx++ {
		go requestInfo(baseUrl, channel)
	}

	// run loop to start pushing item id's for the above goRoutines
	for current := 0; current < len(itemList); current++ {
		// check if current id is already queried
		if !dict[itemList[current]] {
			// lock/unlock mutex because goRoutine could be simultaneously writing in failure case
			mu.Lock()
			// if new item, store in dict as 'visited' now
			dict[itemList[current]] = true
			mu.Unlock()

			// add to the wg; we will mark it as done iff its queried successfully or timed out
			wg.Add(1)

			// debug print
			// fmt.Println("Sent request from item:", itemList[current], " and index:", fmt.Sprint(current))

			// send current item to channel
			channel <- []interface{}{itemList[current], 0}
		}
	}

	// wait till all the queued items are processed (responseSuccess or backoffTimedOut)
	wg.Wait()

	// close the channel so that the go routines can close and 'wg' is all done
	close(channel)

	// print total time taken for all the queries
	duration := time.Since(start)
	fmt.Println("Time taken = ", duration)
}
