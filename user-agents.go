package main

import (
	"math/rand"
	"time"
)

var userAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.2 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:95.0) Gecko/20100101 Firefox/95.0",
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
