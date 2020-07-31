package main

import (
	"time"

	ratelimit "github.com/bygui86/go-rate-limit"
)

func main() {
	r, err := ratelimit.NewMaxConcurrencyRateLimiter(&ratelimit.Config{
		Limit:            4,
		TokenResetsAfter: 10 * time.Second,
	})

	if err != nil {
		panic(err)
	}

	ratelimit.DoWork(r, 10)
}
