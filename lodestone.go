package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"
)

var c http.Client

type options struct {
	url      string
	requests int
	clients  int
}

type result struct {
	url         string
	statusCode  int
	timeElapsed time.Duration
}

type summary struct {
	successes uint32
	failures  uint32
	meanTime  time.Duration
	minTime   time.Duration
	maxTime   time.Duration
}

func parseOptions() options {
	requests := flag.Int("requests", 1, "Number of requests to make")
	clients := flag.Int("clients", 1, "Number of concurent clients")
	flag.Parse()
	return options{
		url:      flag.Arg(0),
		requests: *requests,
		clients:  *clients,
	}
}

func makeRequest(url string) result {
	started := time.Now()
	resp, err := c.Get(url)
	finished := time.Now()
	timeElapsed := finished.Sub(started)
	if err != nil {
		return result{url, 0, timeElapsed}
	}
	resp.Body.Close()
	return result{url, resp.StatusCode, timeElapsed}
}

func calculateStatistics(results []result) map[string]summary {
	urls := map[string][]result{}
	for _, res := range results {
		current := urls[res.url]
		current = append(current, res)
		urls[res.url] = current
	}

	stats := map[string]summary{}
	for url, urlResults := range urls {
		var maxTime, minTime, timesSum time.Duration
		var successes, failures uint32
		for _, res := range urlResults {
			if 200 <= res.statusCode && res.statusCode < 300 {
				successes++
				if res.timeElapsed < minTime || minTime == 0 {
					minTime = res.timeElapsed
				}
				if res.timeElapsed > maxTime {
					maxTime = res.timeElapsed
				}
				timesSum += res.timeElapsed
			} else {
				failures++
			}
		}

		var meanTime time.Duration
		if successes > 0 {
			meanTime = timesSum / time.Duration(successes)
		}
		stats[url] = summary{
			successes: successes,
			failures:  failures,
			minTime:   minTime,
			maxTime:   maxTime,
			meanTime:  meanTime,
		}
	}
	return stats
}

func performLoadTests(opts options) []result {
	requests, clients, url := opts.requests, opts.clients, opts.url

	fmt.Printf("Making %v requests to %v\n", requests, url)

	urls := makeURLsSource(opts)
	done := make(chan result, clients)

	for i := 0; i < clients; i++ {
		go getURL(urls, done)
	}

	results := make([]result, 0, requests)
	for i := 0; i < requests; i++ {
		res := <-done
		results = append(results, res)
	}
	return results
}

func makeURLsSource(opts options) <-chan string {
	urls := make(chan string)
	go func() {
		for i := 0; i < opts.requests; i++ {
			urls <- opts.url
		}
		close(urls)
	}()
	return urls
}

func getURL(urls <-chan string, done chan<- result) {
	for url := range urls {
		done <- makeRequest(url)
	}
}

func printStats(stats map[string]summary) {
	for url, urlSummary := range stats {
		fmt.Printf(
			"%s, successes: %d, failures: %d\n"+
				"\tminTime: %d ms\n"+
				"\tmeanTime: %d ms\n"+
				"\tmaxTime: %d ms\n",
			url, urlSummary.successes, urlSummary.failures,
			urlSummary.minTime/time.Millisecond,
			urlSummary.meanTime/time.Millisecond,
			urlSummary.maxTime/time.Millisecond,
		)
	}
}

func main() {
	opts := parseOptions()
	results := performLoadTests(opts)
	stats := calculateStatistics(results)
	printStats(stats)
}
