package main

import (
	"time"

	ratelimit "github.com/bygui86/go-rate-limit"
)

func main() {
	r, err := ratelimit.NewThrottleRateLimiter(&ratelimit.Config{
		Throttle: 1 * time.Second,
	})

	if err != nil {
		panic(err)
	}

	ratelimit.DoWork(r, 10)
}
