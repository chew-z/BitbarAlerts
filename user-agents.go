package main

import (
	"math/rand"
	"time"
)

var userAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5.2 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/115.0",
}

func randUserAgent() string {
	i := intN(len(userAgents) - 1)
	return userAgents[i]
}

// Generates a pseudo-random int, where 0 <= x < `n`.
func intN(n int) int {
	seed := rand.NewSource(time.Now().UnixNano())
	rnew := rand.New(seed)
	return rnew.Intn(n)
}
