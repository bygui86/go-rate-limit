package main

import (
	"time"

	ratelimit "github.com/bygui86/go-rate-limit"
)

func main() {
	r, err := ratelimit.NewFixedWindowRateLimiter(&ratelimit.Config{
		Limit:         5,
		FixedInterval: 15 * time.Second,
	})

	if err != nil {
		panic(err)
	}

	ratelimit.DoWork(r, 10)
}
