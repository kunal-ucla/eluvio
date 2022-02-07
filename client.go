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
var wg2 = &sync.WaitGroup{}
var wg3 = &sync.WaitGroup{}
var maxBackoff int = 3
var backoffSchedule = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	10 * time.Second,
}
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

func requestInfo(baseUrl string, channel <-chan []interface{}, response chan<- []interface{}) {
	defer wg.Done()
	for item := range channel {
		buff := sendRequest(baseUrl, item[0].(string))
		response <- []interface{}{buff[0].(string), buff[1].(string), buff[2].(int), item[1].(int)}
	}
}

func main() {

	/* Idea is to create 5 Goroutines to keep sending GET requests simultaneously
	and keep sending requests to these routines using a channel with a high capacity */

	baseUrl := "http://localhost:8080/items/"
	// baseUrl := os.Args[1]
	// baseUrl := "https://challenges.qluv.io/items/"

	// genereate a sample itemList for testing
	itemList := make([]string, 100)
	for i := 0; i < 100; i++ {
		itemList[i] = fmt.Sprint(rand.Intn(18))
	}
	// make a channel with a capacity to hold as many items
	channel := make(chan []interface{}, 5)
	response := make(chan []interface{}, 5)

	// to measure time taken for all the queries
	start := time.Now()

	for idx := 0; idx < 5; idx++ {
		wg.Add(1)
		go requestInfo(baseUrl, channel, response)
	}

	wg2.Add(1)
	go func() {
		for current := 0; current < len(itemList); current++ {
			// check if current id is already queried
			if !dict[itemList[current]] {
				// if new, store in dict as queried now
				mu.Lock()
				dict[itemList[current]] = true
				mu.Unlock()
				// send current item to channel
				wg3.Add(1)
				channel <- []interface{}{itemList[current], 0}
			}
		}
		wg2.Done()
	}()

	wg2.Add(1)
	go func() {
		for resp := range response {
			if resp[2].(int) == 200 {
				wg3.Done()
				fmt.Printf("Info for item[ %s ] received as [ %s ]\n", resp[1].(string), resp[0].(string))
			} else if resp[3].(int) < maxBackoff {
				time.Sleep(backoffSchedule[resp[3].(int)])
				channel <- []interface{}{resp[0].(string), resp[3].(int) + 1}
			} else {
				wg3.Done()
				mu.Lock()
				dict[resp[0].(string)] = false
				mu.Unlock()
			}
		}
		wg2.Done()
	}()

	wg3.Wait()

	close(response)
	wg2.Wait()

	close(channel)
	wg.Wait()

	// print total time taken for all the queries
	duration := time.Since(start)
	fmt.Println("Time taken = ", duration)
}
