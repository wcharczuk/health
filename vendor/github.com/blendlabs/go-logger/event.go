package logger

const (
	// EventAll is a special flag that allows all events to fire.
	EventAll Event = "all"
	// EventNone is a special flag that allows no events to fire.
	EventNone Event = "none"

	// EventFatalError fires for fatal errors (panics or errors returned to users).
	EventFatalError Event = "fatal"
	// EventFatal fires for fatal errors and is an alias to `Fatal`.
	EventFatal = EventFatalError
	// EventError fires for errors that are severe enough to log but not so severe as to abort a process.
	EventError Event = "error"
	// EventWarning fires for warnings.
	EventWarning Event = "warning"
	// EventDebug fires for debug messages.
	EventDebug Event = "debug"
	// EventInfo fires for informational messages (app startup etc.)
	EventInfo Event = "info"
	// EventSilly is for when you just need to log something weird.
	EventSilly Event = "silly"

	// EventWebRequestStart fires when an app starts handling a request.
	EventWebRequestStart Event = "web.request.start"
	// EventWebRequest fires when an app completes handling a request.
	EventWebRequest Event = "web.request"
	// EventWebRequestPostBody fires when a request has a post body.
	EventWebRequestPostBody Event = "web.request.postbody"
	// EventWebResponse fires to provide the raw response to a request.
	EventWebResponse Event = "web.response"
)

// Event is a unit of work for the logger; it represents actions that can be enabled or disabled.
type Event string
