package logger

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/blendlabs/go-workqueue"
)

var (
	// DefaultAgentQueueWorkers is the number of consumers (goroutines) for the agent work queue.
	DefaultAgentQueueWorkers = 4

	// DefaultAgentQueueLength is the maximum number of items to buffer in the event queue.
	DefaultAgentQueueLength = 1 << 20 // 1mm items
)

var (
	_default     *Agent
	_defaultLock sync.Mutex
)

var (
	// DefaultAgentVerbosity is the default verbosity for a diagnostics agent inited from the environment.
	DefaultAgentVerbosity = NewEventSet(EventFatalError, EventError, EventWebRequest, EventInfo)
)

// Default returnes a default Agent singleton.
func Default() *Agent {
	return _default
}

// SetDefault sets the diagnostics singleton.
func SetDefault(agent *Agent) {
	_defaultLock.Lock()
	defer _defaultLock.Unlock()
	_default = agent
}

// New returns a new agent with a given set of enabled flags.
func New(events ...Event) *Agent {
	return &Agent{
		events:         NewEventSet(events...),
		eventQueue:     newEventQueue(),
		eventListeners: map[Event][]EventListener{},
		debugListeners: []EventListener{},
		writer:         NewWriterWithErrorOutput(os.Stdout, os.Stderr),
	}
}

// NewWithEventSet returns a new agent with a given event set.
func NewWithEventSet(events *EventSet) *Agent {
	return &Agent{
		events:         events,
		eventQueue:     newEventQueue(),
		eventListeners: map[Event][]EventListener{},
		debugListeners: []EventListener{},
		writer:         NewWriterWithErrorOutput(os.Stdout, os.Stderr),
	}
}

// NewWithWriter returns a new agent with a given event set and writer.
func NewWithWriter(events *EventSet, writer *Writer) *Agent {
	return &Agent{
		events:         events,
		eventQueue:     newEventQueue(),
		eventListeners: map[Event][]EventListener{},
		debugListeners: []EventListener{},
		writer:         writer,
	}
}

// NewFromEnv returns a new agent with settings read from the environment.
func NewFromEnv() *Agent {
	return NewWithWriter(NewEventSetFromEnv(), NewWriterFromEnv())
}

// All returns a valid agent that fires any and all events.
func All(writer ...*Writer) *Agent {
	if len(writer) > 0 {
		return NewWithWriter(NewEventSetAll(), writer[0])
	}
	return NewWithWriter(NewEventSetAll(), NewWriterFromEnv())
}

// None returns a valid agent that won't fire any events.
func None(writer ...*Writer) *Agent {
	if len(writer) > 0 {
		return NewWithWriter(NewEventSetNone(), writer[0])
	}
	return NewWithWriter(NewEventSetNone(), NewWriterFromEnv())
}

// Agent is a handler for various logging events with descendent handlers.
type Agent struct {
	writer             *Writer
	eventsLock         sync.Mutex
	events             *EventSet
	eventListenersLock sync.Mutex
	eventListeners     map[Event][]EventListener
	debugListeners     []EventListener
	eventQueue         *workqueue.Queue

	syncLock sync.Mutex
	sync     *SyncAgent
}

// Writer returns the inner Logger for the diagnostics agent.
func (da *Agent) Writer() *Writer {
	return da.writer
}

// EventQueue returns the inner event queue for the agent.
func (da *Agent) EventQueue() *workqueue.Queue {
	return da.eventQueue
}

// Events returns the EventSet
func (da *Agent) Events() *EventSet {
	if da == nil {
		return NewEventSet()
	}
	return da.events
}

// SetVerbosity sets the agent verbosity synchronously.
func (da *Agent) SetVerbosity(events *EventSet) {
	da.eventsLock.Lock()
	da.events = events
	da.eventsLock.Unlock()
}

// EnableEvent flips the bit flag for a given event.
func (da *Agent) EnableEvent(eventFlag Event) {
	da.eventsLock.Lock()
	da.events.Enable(eventFlag)
	da.eventsLock.Unlock()
}

// DisableEvent flips the bit flag for a given event.
func (da *Agent) DisableEvent(eventFlag Event) {
	da.eventsLock.Lock()
	da.events.Disable(eventFlag)
	da.eventsLock.Unlock()
}

// IsEnabled asserts if a flag value is set or not.
func (da *Agent) IsEnabled(flagValue Event) bool {
	if da == nil {
		return false
	}
	return da.events.IsEnabled(flagValue)
}

// HasListener returns if there are registered listener for an event.
func (da *Agent) HasListener(event Event) bool {
	if da == nil {
		return false
	}
	if da.eventListeners == nil {
		return false
	}
	listeners, hasHandler := da.eventListeners[event]
	if !hasHandler {
		return false
	}
	return len(listeners) > 0
}

// AddEventListener adds a listener for errors.
func (da *Agent) AddEventListener(eventFlag Event, listener EventListener) {
	da.eventListenersLock.Lock()
	da.eventListeners[eventFlag] = append(da.eventListeners[eventFlag], listener)
	da.eventListenersLock.Unlock()
}

// AddDebugListener adds a listener that will fire on *all* events.
func (da *Agent) AddDebugListener(listener EventListener) {
	da.eventListenersLock.Lock()
	da.debugListeners = append(da.debugListeners, listener)
	da.eventListenersLock.Unlock()
}

// RemoveListeners clears *all* listeners for an Event.
func (da *Agent) RemoveListeners(eventFlag Event) {
	delete(da.eventListeners, eventFlag)
}

// OnEvent fires the currently configured event listeners.
func (da *Agent) OnEvent(eventFlag Event, state ...interface{}) {
	if da == nil {
		return
	}
	if da.IsEnabled(eventFlag) && da.HasListener(eventFlag) {
		da.eventQueue.Enqueue(da.triggerListeners, append([]interface{}{TimeNow(), eventFlag}, state...)...)
	}
}

// Infof logs an informational message to the output stream.
func (da *Agent) Infof(format string, args ...interface{}) {
	if da == nil {
		return
	}
	da.WriteEventf(EventInfo, ColorLightWhite, format, args...)
}

// Debugf logs a debug message to the output stream.
func (da *Agent) Debugf(format string, args ...interface{}) {
	if da == nil {
		return
	}
	da.WriteEventf(EventDebug, ColorLightYellow, format, args...)
}

// Warningf logs a debug message to the output stream.
func (da *Agent) Warningf(format string, args ...interface{}) error {
	if da == nil {
		return nil
	}
	return da.Warning(fmt.Errorf(format, args...))
}

// Warning logs a warning error to std err.
func (da *Agent) Warning(err error) error {
	if da == nil {
		return err
	}
	return da.ErrorEventWithState(EventWarning, ColorLightYellow, err)
}

// WarningWithReq logs a warning error to std err with a request.
func (da *Agent) WarningWithReq(err error, req *http.Request) error {
	if da == nil {
		return err
	}
	return da.ErrorEventWithState(EventWarning, ColorLightYellow, err, req)
}

// Errorf writes an event to the log and triggers event listeners.
func (da *Agent) Errorf(format string, args ...interface{}) error {
	if da == nil {
		return nil
	}
	return da.Error(fmt.Errorf(format, args...))
}

// Error logs an error to std err.
func (da *Agent) Error(err error) error {
	if da == nil {
		return err
	}
	return da.ErrorEventWithState(EventError, ColorRed, err)
}

// ErrorWithReq logs an error to std err with a request.
func (da *Agent) ErrorWithReq(err error, req *http.Request) error {
	if da == nil {
		return err
	}
	return da.ErrorEventWithState(EventError, ColorRed, err, req)
}

// Fatalf writes an event to the log and triggers event listeners.
func (da *Agent) Fatalf(format string, args ...interface{}) error {
	if da == nil {
		return nil
	}
	return da.Fatal(fmt.Errorf(format, args...))
}

// Fatal logs the result of a panic to std err.
func (da *Agent) Fatal(err error) error {
	if da == nil {
		return err
	}
	return da.ErrorEventWithState(EventFatalError, ColorRed, err)
}

// FatalWithReq logs the result of a fatal error to std err with a request.
func (da *Agent) FatalWithReq(err error, req *http.Request) error {
	if da == nil {
		return err
	}
	return da.ErrorEventWithState(EventFatalError, ColorRed, err, req)
}

// FatalExit logs the result of a fatal error to std err and calls `exit(1)`
func (da *Agent) FatalExit(err error) {
	if da == nil || da.events == nil {
		os.Exit(1)
	}

	da.ErrorEventWithState(EventFatalError, ColorRed, err)
	da.Drain()
	os.Exit(1)
}

// --------------------------------------------------------------------------------
// meta methods
// --------------------------------------------------------------------------------

// WriteEventf writes to the standard output and triggers events.
func (da *Agent) WriteEventf(event Event, color AnsiColorCode, format string, args ...interface{}) {
	if da == nil {
		return
	}
	if da.IsEnabled(event) {
		da.queueWrite(event, ColorLightYellow, format, args...)

		if da.HasListener(event) {
			da.eventQueue.Enqueue(da.triggerListeners, append([]interface{}{TimeNow(), event, format}, args...)...)
		}
	}
}

// WriteErrorEventf writes to the error output and triggers events.
func (da *Agent) WriteErrorEventf(event Event, color AnsiColorCode, format string, args ...interface{}) {
	if da == nil {
		return
	}
	if da.IsEnabled(event) {
		da.queueWriteError(event, ColorLightYellow, format, args...)

		if da.HasListener(event) {
			da.eventQueue.Enqueue(da.triggerListeners, append([]interface{}{TimeNow(), event, format}, args...)...)
		}
	}
}

// ErrorEventWithState writes an error and triggers events with a given state.
func (da *Agent) ErrorEventWithState(event Event, color AnsiColorCode, err error, state ...interface{}) error {
	if da == nil {
		return err
	}
	if err != nil {
		if da.IsEnabled(event) {
			da.queueWriteError(event, color, "%+v", err)
			if da.HasListener(event) {
				da.eventQueue.Enqueue(da.triggerListeners, append([]interface{}{TimeNow(), event, err}, state...)...)
			}
		}
	}
	return err
}

// --------------------------------------------------------------------------------
// synchronous methods
// --------------------------------------------------------------------------------

// Sync returns a synchronous agent.
func (da *Agent) Sync() *SyncAgent {
	if da.sync == nil {
		da.syncLock.Lock()
		if da.sync == nil {
			da.sync = &SyncAgent{
				eventsLock:         &da.eventsLock,
				events:             da.events,
				eventListenersLock: &da.eventListenersLock,
				eventListeners:     da.eventListeners,
				writer:             da.writer,
			}
		}
		da.syncLock.Unlock()
	}
	return da.sync
}

// --------------------------------------------------------------------------------
// finalizers
// --------------------------------------------------------------------------------

// Close releases shared resources for the agent.
func (da *Agent) Close() (err error) {
	if da.eventQueue != nil {
		err = da.eventQueue.Close()
		if err != nil {
			return
		}
	}
	if da.writer != nil {
		err = da.writer.Close()
	}
	return
}

// Drain waits for the agent to finish it's queue of events before closing.
func (da *Agent) Drain() error {
	if da == nil {
		return nil
	}
	da.SetVerbosity(NewEventSetNone())

	for da.eventQueue.Len() > 0 {
		time.Sleep(time.Millisecond)
	}
	return da.Close()
}

// --------------------------------------------------------------------------------
// internal methods
// --------------------------------------------------------------------------------

// triggerListeners triggers the currently configured event listeners.
func (da *Agent) triggerListeners(actionState ...interface{}) error {
	if len(actionState) < 2 {
		return nil
	}

	timeSource, err := stateAsTimeSource(actionState[0])
	if err != nil {
		return err
	}

	eventFlag, err := stateAsEvent(actionState[1])
	if err != nil {
		return err
	}

	da.eventListenersLock.Lock()
	listeners := da.eventListeners[eventFlag]
	da.eventListenersLock.Unlock()

	for x := 0; x < len(listeners); x++ {
		listener := listeners[x]
		listener(da.writer, timeSource, eventFlag, actionState[2:]...)
	}

	if len(da.debugListeners) > 0 {
		for x := 0; x < len(da.debugListeners); x++ {
			listener := da.debugListeners[x]
			listener(da.writer, timeSource, eventFlag, actionState[2:]...)
		}
	}

	return nil
}

// printf checks an event flag and writes a message with a given color.
func (da *Agent) queueWrite(eventFlag Event, color AnsiColorCode, format string, args ...interface{}) {
	if len(format) > 0 {
		da.eventQueue.Enqueue(da.write, append([]interface{}{TimeNow(), eventFlag, color, format}, args...)...)
	}
}

// errorf checks an event flag and writes a message to the error stream (if one is configured) with a given color.
func (da *Agent) queueWriteError(eventFlag Event, color AnsiColorCode, format string, args ...interface{}) {
	if len(format) > 0 {
		da.eventQueue.Enqueue(da.writeError, append([]interface{}{TimeNow(), eventFlag, color, format}, args...)...)
	}
}

func (da *Agent) write(actionState ...interface{}) error {
	return da.writeWithOutput(da.writer.PrintfWithTimeSource, actionState...)
}

func (da *Agent) writeError(actionState ...interface{}) error {
	return da.writeWithOutput(da.writer.ErrorfWithTimeSource, actionState...)
}

type loggerOutputWithTimeSource func(ts TimeSource, format string, args ...interface{}) (int64, error)

// writeEventMessage writes an event message.
func (da *Agent) writeWithOutput(output loggerOutputWithTimeSource, actionState ...interface{}) error {
	if len(actionState) < 4 {
		return nil
	}

	timeSource, err := stateAsTimeSource(actionState[0])
	if err != nil {
		return err
	}

	eventFlag, err := stateAsEvent(actionState[1])
	if err != nil {
		return err
	}

	labelColor, err := stateAsAnsiColorCode(actionState[2])
	if err != nil {
		return err
	}

	format, err := stateAsString(actionState[3])
	if err != nil {
		return err
	}

	_, err = output(timeSource, "%s %s", da.writer.FormatEvent(eventFlag, labelColor), fmt.Sprintf(format, actionState[4:]...))
	return err
}

func newEventQueue() *workqueue.Queue {
	eq := workqueue.NewWithWorkers(DefaultAgentQueueWorkers)
	eq.SetMaxWorkItems(DefaultAgentQueueLength) //more than this and queuing will block
	eq.Start()
	return eq
}
