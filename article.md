
# Rate Limiting in Golang

Recently I was working on a problem where I neeeded a rate limiter to limit the number of concurrent connections I could maintain at once for a an API. I’ve solved similar problems before using Node.js, however, I wanted to see how I could manage rate limiting with Golang. If you’re new to Go and want to learn more, you can check out A Tour of Go for a great introduction.

## What We’re going to Build

We’re going to build a Rate Limiter in Golang that implements 3 basic rate limiting algorithms:

1. Throttle Rate Limiter - limits based on a single request per specified time interaval.

1. Max Concurrency Rate Limiter - limits the number of active concurrent requests at any given time.

1. Fixed Window Rate Limiter - specifies a fixed number of requests that can be processed in a given time window. Once that limit is reached no more requests can be processed until the next window.

The diagram below outlines the flow control of the rate limiter.

![Rate Limiter Flow Overview](https://cdn-images-1.medium.com/max/3904/1*V90OQp4ZJG0UspAhs97HOQ.png)*Rate Limiter Flow Overview*

* The rate limit should expose an *Acquire() *method that, when called, will block until a rate limit token is available.

* Interally, we’ll have two separate channels for synchronization: 1) an incoming request channel, and 2) an outgoing token channel.

* Each Rate Limiter will define an *await* function that listens for incoming requests, determines when a rate limit token is available, and sends the token to the *out* channel.

## Rate Limiter Interface

Let’s start by defining the RateLimiter Interface and Config struct.

<iframe src="https://medium.com/media/a9f8f061aaccd3107efc07b8f9f3375b" frameborder=0></iframe>

Great, we have a method for aquiring a Token. Let’s keep going and define a Manager struct that implements a Rate Limiter Interface. We will need to add the *in* channel and *out* channel fields to the manager. Additionally, let’s specify a field called *makeToken, *which is a factory function for creating tokens that will allow different rate limiter implementations to define their own custom logic for token creation. For now, we’ll default *makeToken *to the *NewToken()* factory function.

<iframe src="https://medium.com/media/d9b42b0591cd8a86f3286fa7c4be9b58" frameborder=0></iframe>

When *Acquire() *is called, an empty struct{} is sent to the *in *channel and we wait for either a token from the *out *channel or an error from the *error *channel. Each rate limiter will need to define an await function that receives messages from the *in* channel, handles the scheduling for when a new token will beavailable, and calls manager.tryGenerateToken()* *method which creates a new token and sends it to the *out *channel.

The last thing we need to define before we can implement the first rate limiter is the Rate Limit Token.

<iframe src="https://medium.com/media/8824dfcedddac6296b7e02b6154dc2f2" frameborder=0></iframe>

We only need two fields for now, a unique id and the creation time, that’s it!

## Throttle Rate Limiter

Now we can implement the first rate limiter. The basic idea behind a throttler is to handle bursty programs by only allowing a certain number of requests to be processed per time duration. For example, a public API may try to regulate the server load by only allowing 1 request per second per client. With Go, we can accomplish this by using a time [Ticker](https://golang.org/src/time/tick.go), which schedules ticks at a specified interval.Then we can synchronize our throttler to only allow 1 rate limit token per tick.

<iframe src="https://medium.com/media/a4b9be0082bd615be5013d10ef49d789" frameborder=0></iframe>

Here we’ve defined the *await()* function that loops over the ticker channel and blocks while waiting to receive a message from the *in* channel. Once a message is received we’ll call *tryGenerateToken() *and continue looping to wait for the next ticker.

Now that we have all this let’s try it out to see if it works! We’ll try creating a new ThrottleRateLimiter with a throttle duration of 1 second and generate 10 request workers that need a Rate Limit Token.

<iframe src="https://medium.com/media/860cb645e968a4bbd645f994e40c2ec7" frameborder=0></iframe>

And here’s the output we get when running the program.

![](https://cdn-images-1.medium.com/max/3788/1*aMRHe8nGohAB0fpIgvoc2Q.png)

Notice that each worker receives a token throttled at 1 second intervals, as desired!

## Max Concurrency Rate Limiter

A Max Concurrency Rate Limiter puts a limit on how many active rate limits can be in use at a given time. If a rate limit is requested and the active count is less than the limit, the rate limit is immediately granted, otherwise the request must wait until a previously granted rate limit is finished. This means that we need to add the ability to release a rate limit token. Let’s add that to the rate limiter interface.

<iframe src="https://medium.com/media/2c6f3aaeb9efe7ff6f9bfc09620c2f69" frameborder=0></iframe>

Notice that in addition to the *Release() *method on the RateLimiter Interface, we added two fields to the config struct: Limit and TokenResetsIn. The Limit option specifies the max number of tokens that can be in circulation at a time. However, we need to think about the case when a Token is acquired but is failed to be released. We don’t want that Token to starve when it could be released and given to something else waiting, so we need to specify a max time-to-live for each token. Once that max time is reached, the token will be released and added back into the pool of available tokens. The config option TokenResetsIn defines the max duration before a token is forcibly released.

Let’s update the Rate Limit Manager to add the functionality for releasing a token.

<iframe src="https://medium.com/media/4d742c30c6e77008166001018d0dc5dc" frameborder=0></iframe>

Notice the new fields:

* *releaseChan - *for synchronizing the release of tokens.

* *limit - *represents the maximum number of tokens that can be active and is used to determine whether a token can be generated or not.

* *activeTokens *- a map containing all the active tokens in circulation.

* *needToken *- a counter representing the number of pending or waiting token requests.

In the tryGenerateToken method there is now a check to see if the manager has exceeded the limit, and if so we will increment the needToken counter and exit. Otherwise we add the new token to the activeToken map. There is also a new method called *releaseToken() *where the token is removed from the activeMap, and then checks to see if there are any pending requests that can be processed.

The releasing and generating of tokens needs to be serialized so there are not concurrent writes and reads to the *activeToken *map,* *which will cause a runtime crash. Let’s add a NewMaxConcurrencyRateLimiter factory and update the *await()* function.

<iframe src="https://medium.com/media/95382626feb4353ef7f26c4aac8e132a" frameborder=0></iframe>

Perfect, now that we’ve synchronized the token creation / release, let’s test it out! We’ll call NewMaxConcurrencyRateLimiter with a Limit of 3.

<iframe src="https://medium.com/media/fd62c9124e737d4c4244a1f0c705da2d" frameborder=0></iframe>

Here’s the result I get when running:

![](https://cdn-images-1.medium.com/max/4056/1*XugfXtebnKDsTj2vRJ3I0g.png)

Notice that the first 3 token requests are processed immediately, then once the limit is reached each subsequent request has to wait until a token becomes available. At any time we never have more than 3 tokens in use, which is exactly what we want!

So this is great, but we’re still missing one important thing; what happens when the caller fails to release the token? Notice in the example we have the following line:

    r.Release(token)

What do you think happens if we remove that line and don’t release the tokens when finished?

![](https://cdn-images-1.medium.com/max/4060/1*SJz9ECaMy26tVnUG0Wubng.png)

Uh oh! Once the first three tokens are complete we deadlock waiting indefinitely on the *releaseChan*. I mentioned above that we will need the ability to manually remove a token after a specified period of time. Let’s go ahead and add that now.

<iframe src="https://medium.com/media/c02baf84b00dc6b52813b78748252088" frameborder=0></iframe>

We create a ticker periodically runs and checks all the active tokens to see if they need to be reset. If, so we send them to the release chanel and move on. The token *NeedReset() *method looks like this:

<iframe src="https://medium.com/media/5ce07bbf17e405e0fc167929ce3d176a" frameborder=0></iframe>

Let’s try running the max concurrency rate limiter again with the reset config option to see if we fixed our deadlock issues!

    conf := &ratelimiter.Config{
        Limit: 5,
        // Reset tokens manually after 10 seconds
        TokenResetsAfter: 10 * time.Second,
    }

    r, _ := NewMaxConcurrencyRateLimiter(conf)

    DoWork(r, 10)

![](https://cdn-images-1.medium.com/max/4076/1*Rv--5vyqaW8NWjRwsv5hhA.png)

Notice the first five requests are granted, then each of the workers finish **without **releasing their rate limit token. Then after 10 seconds the reset task runs, and forcibly resets each of the stale tokens so that the last 5 requests can then be processed.

## Fixed Window Rate Limiter

In a fixed window rate limiter, a window of time *t* is defined along with a max request limit per window. Each request that occurs within the window time increments the counter until the limit is reached, then each subsequent request has to wait until the next available window.

We can define a FixedWindowInterval with a start time, end time and an interval duration.

<iframe src="https://medium.com/media/227979927d0fa91cdd64af7e3af1d2f1" frameborder=0></iframe>

The run method starts a ticker of interval n that sets the fixed window time and calls a callback function per each interval.

In the throttle and max concurrency rate limiters a token could be released at any point by just calling manager.Release(). However, in the fixed window rate limiter, a token can only be released if the fixed window in which the token was requested has passed. In other words, the time at which a token expires is the endTime of the current fixed window interval.

Let’s go ahead and update our Token and Manager to only allow tokens to be released once they have expired.

<iframe src="https://medium.com/media/4d1f604e529ceaf1528853590c5009fc" frameborder=0></iframe>

<iframe src="https://medium.com/media/bbc5fbb952a4d59624d99364b2343423" frameborder=0></iframe>

The Token now has an IsExpired method, and on release, the manager will first verify the token is expired before releasing it. We also defined a new method called releaseExpiredTokens on the manager will loops over the active tokens and releases them if they have expired.

Let’s go ahead and defined our fixed window rate limiter implementation.

<iframe src="https://medium.com/media/f2f29f4dc152c39275ef524229b9f7d2" frameborder=0></iframe>

In *NewFixedWindowRateLimiter() *we define a FixedWindowInterval and override manager.makeToken to set the token ExpiresAt to be the fixed window end time. Then we call window.run and pass *manager.releaseExpiredTokens* as the callback.

Let’s try an example:

<iframe src="https://medium.com/media/6bc917d9b2f1825221fde92faf974bb2" frameborder=0></iframe>

![](https://cdn-images-1.medium.com/max/3900/1*hvTaaH-VGi4zVOzGzNae0g.png)

Notice the first 5 rate limit requests are granted. Then there is a 15 second waiting period, then the next 5 are granted in the following fixed window interval.

## Conclusion

In this article we looked at how to implement three different types of rate limiting algorithms using Golang. In all three of our examples we were able to extend the functionality of the rate limit manager to meet our requirements. All of the code samples and examples for this article can be found at [https://github.com/jpg013/go_rate_limiter](https://github.com/jpg013/go_rate_limiter). Feel free to leave comments or questions and thanks for reading!
