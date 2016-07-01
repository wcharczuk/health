#Health

Health is a simple commandline healtcheck utility that gives you basic stats about response times (uptime %, last, average, 99th percentile, 90th percentile, 75th percentile).

It is only for OS-X / Linux right now (no windows support because of the ANSI clear code used to wipe the console).

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

Health will ping the servers every 30 seconds by default.

You can have multiple `--host` values:

```bash
> health --host <server> --host <another server> --host <yet another server>
```

##Example Output:

```bash
http://fooserver.com/status/postgres   UP (100.0) Last: 1ms    Average: 2ms    99th: 2ms     90th: 2ms    75th: 2ms
http://barserver.com/status/postgres   UP (100.0) Last: 1ms    Average: 2ms    99th: 3ms     90th: 2ms    75th: 2ms
http://bazserver.com/status/postgres   UP (100.0) Last: 1ms    Average: 2ms    99th: 4ms     90th: 2ms    75th: 1ms
```

The screen will clear every 500ms. The polling interval will also be used as the timeout for the pings, with the difference between the elapsed time for the ping and the interval comprising the rest of the sleep time.

##Config File Format

Optionally you can create a config file with the following format:

```json
{
  "interval": "30s",
  "show_notification": true,
  "hosts": [
    "http://www.google.com",
    "http://www.apple.com"
  ]
}
```

Interval is set in milliseconds. 

You can specify the config file when invoking `health` as follows:

```bash
> health --config my_config.json
```

Note: changes to `my_config.json` will result in `health` reloading and resetting statistics. 
