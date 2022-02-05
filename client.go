package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
)

func getInfo(baseUrl string, itemID string, ch chan<- string) {

	// generate authorization header using the id (convert id to base64)
	auth := base64.URLEncoding.EncodeToString([]byte(itemID))

	// prepare the target URL using the item id
	url := baseUrl + itemID

	// create a Client, define request, set header and perform req
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", auth)
	res, err := client.Do(req)

	// log the error if any during the GET request
	if err != nil {
		log.Fatal(err)
	}

	// read the response and close the response body
	body, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()

	// send response through the channel for printing
	ch <- string(body[:])
}

func main() {

	/* Idea is to create 5 Goroutines to keep sending GET requests simultaneously */

	// baseUrl := "http://localhost:8080/items/"
	// baseUrl := os.Args[1]
	baseUrl := "https://challenges.qluv.io/items/"

	// create 5 channels per Goroutine
	var chans [5]chan string
	for i := range chans {
		chans[i] = make(chan string)
	}

	// declare 5 items to send per Goroutine
	var items [5]string

	// genereate a sample itemList for testing
	var itemList [100]string
	for i := 0; i < 100; i++ {
		itemList[i] = fmt.Sprint(rand.Intn(18))
	}

	// dictionary for already queried items
	dict := make(map[string]bool)

	// run through itemList, 5 at a time
	for current := 0; current < len(itemList); {
		// copy top 5 ids from itemList to items[]

		// declaring this outside for loop to retain it's value afterwards
		var idx int
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
			go getInfo(baseUrl, items[i], chans[i])
		}

		// create 5 buffer to store results per Goroutine
		var buff [5]string
		for i := 0; i < idx; i++ {
			buff[i] = <-chans[i]
		}

		// print responses per Goroutine
		for i := 0; i < idx; i++ {
			fmt.Println(buff[i])
		}
	}

}
