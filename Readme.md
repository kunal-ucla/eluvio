# Eluvio Problem Statement
Imagine you have a program that needs to look up information about items using their item ID, often in large batches.

Unfortunately, the only API available for returning this data takes one item at a time, which means you will have to perform one query per item. Additionally, the API is limited to five simultaneous requests. Any additional requests will be served with `HTTP 429` (too many requests).

Write a client utility for your program to use that will retrieve the information for all given IDs as quickly as possible without triggering the simultaneous requests limit, and without performing unnecessary queries for item IDs that have already been seen.

## API Usage:
GET https://challenges.qluv.io/items/:id

## Required headers:
Authorization: Base64(:id)
Example:
```
curl https://challenges.qluv.io/items/cRF2dvDZQsmu37WGgK6MTcL7XjH -H "Authorization: Y1JGMmR2RFpRc211MzdXR2dLNk1UY0w3WGpI"
```

# My Solution in Go

My idea is to create 5 GoRoutines that will independently scheduling the HTTP requests one by one from the same queue. I have used a channel with 5 buffer size for implementing this queue and each of these GoRoutines will keep popping off a user ID and sending the HTTP GET request for it.

## GET request
Here is the function that will be invoked by each of the GoRoutine to send an individual HTTP GET request.
```
func sendRequest(baseUrl string, itemID string) []interface{} {...}
```
It takes as input the base URL and the item ID (it will concatenate the item ID to the base string to form the final URL). It first generates the Authorization header using the item ID (by converting to base64 as mentioned in the problem statement). And then it sends the GET request to the final URL with this auth header, and captures the response body and status code inside an interface structure that it returns back to the caller.

## GoRoutine
Here is the function whose 5 concurrent instances will be running and this will keep invoking the `sendRequest()` function to send the GET requests and use the response status code to either print out the returned item info if success, or start a (concurrent) backoff mechanism where the item will be re-queried after certain duration and will be dropped after certain number of max attempts.
```
func requestInfo(baseUrl string, channel chan []interface{}) {...}
```
It takes as input the base URL and the channel on which it keeps monitoring for any available jobs (itemIDs) to perform on (send the query req). It will further invoke another concurrent goroutine for backoff process if a query is not responded with a success code. If the maximum number of attempts is reached for a particular itemID, it will drop it and mark it as 'not visited' in a dictionary - which can later be used by the main function to re-consider this same userID (if it comes again).

## Backoff Handler
Here is the function that performs the backoff functionality. It's invoked by the GoRoutine `requestInfo()` if a particular itemID's query is not successful.
```
func backoffHandler(item string, attempt int, channel chan []interface{}) {...}
```
It takes as input the itemID (so that it can re-queue it to the channel), the attempt count (it will increment this before re-queueing) and the channel on which it will requeue the itemID. It also runs the backoff timer according to the attempt count. Note that this runs as another (sub) GoRoutine since other items scheduling shouldn't be held on waiting for this to be re-queued.

## Main
The main function will invoke the 5 main GoRoutines (`requestInfo()`) as shown below:
```
for idx := 0; idx < 5; idx++ {
    go requestInfo(baseUrl, channel)
}
```
A channel with size 5 is also created along with this which will be used for continuous queueing of jobs (itemIDs to be queried):
```
channel := make(chan []interface{}, 5)
```

It's assumed that the itemID's are all listed in the string array `itemList`. Then each itemID's are sent to the channel using a for loop to be queried by the GoRoutines:
```
for current := 0; current < len(itemList); current++ {...}
```

A dictionary is maintained globally to ensure that the same itemID that's already been queried is not re-queried:
```
var dict map[string]bool = make(map[string]bool)
```
It will be written by the main function once to mark the itemID while sending it to the channel first, and also used by the GoRoutine to mark an itemID in case of failure of query. Hence a mutex is used to use this dictionary at both the places.

Finally, a Waitgroup is used to ensure that all the queries are finished completely (either successfully or failed after all the backoffs) before we close the channel and end the main function. It's 'added' once at each addition of itemID from the itemList, and is marked 'Done' only when it's either successully or failed completely.