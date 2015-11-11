#Health

This is a simple healtcheck utility for OS-X / Linux only right now (no windows support right now because of ANSI clear code).

##Usage

```bash
> health --host <server>
```

Health will ping the servers every 30 seconds by default. If a server goes down it will raise a notification via AppleScript. 

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
