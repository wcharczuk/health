#Health

This is a simple healtcheck utility for OS-X / Linux only right now (no windows support right now because of ANSI clear code).

##Installation

Install using standard `go get && go install`. Make sure that your `$GOPATH/bin` directory is in your `$PATH`

```bash
> go get -u github.com/wcharczuk/health
> go install github.com/wcharczuk/health
> health --host http://google.com --interval 1000
```

##Usage

```bash
> health --host <server>
```

Health will ping the servers every 30 seconds by default. If a server goes down it will raise a notification via AppleScript. 

##Example Output:

```bash
http://fooserver.com/status/postgres   UP Last: 164.018477ms Average: 274.073916ms StdDev: 110.055439ms
http://barserver.com/status/postgres   UP Last: 733.29607ms  Average: 584.391811ms StdDev: 148.904259ms
http://bazserver.com/status/postgres   UP Last: 155.562902ms Average: 296.666452ms StdDev: 141.10355ms
```

The screen will clear everytime the interval/2 timer fires, so you should always basically see that (the list of the servers you're monitoring).

##Config File Format

Optionally you can create a config file with the following format:

```json
{
  "interval": 30000,
  "show_notification": true,
  "hosts": [
    "http://www.google.com",
    "http://www.apple.com"
  ]
}
```

Interval, by default, is set in milliseconds. 

You specify the config file when invoking via:

```bash
> health --config my_config.json
```
