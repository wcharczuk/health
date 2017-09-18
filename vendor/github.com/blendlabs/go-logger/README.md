Go Logger
=========

Go logger is not well named. It's really an event queue that is managed by a hashset of event names. It lets you register event handlers for common events, generally these event handlers write to stdout.

# Example

```golang
logger.SetDefault(logger.NewAgentFromEnvironment()) // set the singleton to a environment configured default.
logger.Default().AddEventListener(logger.EventError, func(wr logger.Logger, ts logger.TimeSource, flag logger.EventFlag, args ...interface{}) {
    //ping an external service?
    //log something to the db?
    //this action will be handled by a separate go-routine
})
```

Then, elsewhere in our code:

```golang
logger.Default().Error(fmt.Errorf("this is an exception"))   // this will write the error to stderr, but also
                                                             // will trigger the handler from before.
```

# What can I do with this?

You can defer writing a bunch of log messages to stdout to unblock requests in high-throughput scenarios. `logger` is very careful to preserve timing state so that actions that live in the queue for multiple seconds are logged with the correct  timestamp.

# What else can I do with this?

You can standardize how you write log messages across multiple packages / services.

# Benchmarks

In `_example/main.go` you'll find a sample webserver with a bunch of endpoints, two worth looking at are `/bench/logged` and `/bench/stdout`. The former tests with the event queue driven logger, the later tests just writing to stdout.

Results:

Stdout / Printf:

```bash
> wrk -c 32 -t 4 -d 30 http://localhost:8888/bench/stdout
Running 30s test @ http://localhost:8888/bench/stdout
  4 threads and 32 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.71ms    6.22ms 157.31ms   96.62%
    Req/Sec    10.11k     1.11k   32.83k    88.84%
  1208885 requests in 30.10s, 153.33MB read
Requests/sec:  40161.82
Transfer/sec:      5.09MB
```

Versus event queue:

```bash
> wrk -c 32 -t 4 -d 30 http://localhost:8888/bench/logged
Running 30s test @ http://localhost:8888/bench/logged
  4 threads and 32 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.37ms    8.53ms 151.83ms   99.02%
    Req/Sec    12.80k     1.71k   34.19k    81.98%
  1527807 requests in 30.10s, 193.79MB read
Requests/sec:  50758.87
Transfer/sec:      6.44MB
```