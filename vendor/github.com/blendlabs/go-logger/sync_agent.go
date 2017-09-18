package logger

import (
	"fmt"
	"net/http"
	"os"
	"sync"
)

// NewSync returns a new sync agent with a given set of enabled flags.
func NewSync(events ...Event) *SyncAgent {
	return &SyncAgent{
		eventsLock:         &sync.Mutex{},
		events:             NewEventSet(events...),
		eventListenersLock: &sync.Mutex{},
		eventListeners:     map[Event][]EventListener{},
		writer:             NewWriterWithErrorOutput(os.Stdout, os.Stderr),
	}
}

// NewSyncWithWriter returns a new sync agent with a given set of enabled flags.
func NewSyncWithWriter(events *EventSet, writer *Writer) *SyncAgent {
	return &SyncAgent{
		eventsLock:         &sync.Mutex{},
		events:             events,
		eventListenersLock: &sync.Mutex{},
		eventListeners:     map[Event][]EventListener{},
		writer:             writer,
	}
}

// SyncAll returns a valid sync agent that fires any and all events.
func SyncAll(writer ...*Writer) *SyncAgent {
	if len(writer) > 0 {
		return NewSyncWithWriter(NewEventSetAll(), writer[0])
	}
	return NewSyncWithWriter(NewEventSetAll(), NewWriterFromEnv())
}

// SyncNone returns a valid agent that won't fire any events.
func SyncNone(writer ...*Writer) *SyncAgent {
	if len(writer) > 0 {
		return NewSyncWithWriter(NewEventSetNone(), writer[0])
	}
	return NewSyncWithWriter(NewEventSetNone(), NewWriterFromEnv())
}

// NewSyncFromEnv returns a new agent with settings read from the environment.
func NewSyncFromEnv() *SyncAgent {
	return NewSyncWithWriter(NewEventSetFromEnv(), NewWriterFromEnv())
}

// SyncAgent is an agent that fires events synchronously.
// It wraps a regular agent.
type SyncAgent struct {
	eventsLock         *sync.Mutex
	events             *EventSet
	eventListenersLock *sync.Mutex
	eventListeners     map[Event][]EventListener
	writer             *Writer
}

// Writer returns the underlying writer.
func (sa *SyncAgent) Writer() *Writer {
	return sa.writer
}

// Events returns the EventSet.
func (sa *SyncAgent) Events() *EventSet {
	if sa == nil {
		return NewEventSet()
	}
	return sa.events
}

// EnableEvent flips the bit flag for a given event.
func (sa *SyncAgent) EnableEvent(eventFlag Event) {
	sa.eventsLock.Lock()
	sa.events.Enable(eventFlag)
	sa.eventsLock.Unlock()
}

// DisableEvent flips the bit flag for a given event.
func (sa *SyncAgent) DisableEvent(eventFlag Event) {
	sa.eventsLock.Lock()
	sa.events.Disable(eventFlag)
	sa.eventsLock.Unlock()
}

// IsEnabled asserts if a flag value is set or not.
func (sa *SyncAgent) IsEnabled(flagValue Event) bool {
	if sa == nil {
		return false
	}
	sa.eventsLock.Lock()
	enabled := sa.events.IsEnabled(flagValue)
	sa.eventsLock.Unlock()
	return enabled
}

// HasListener returns if there are registered listener for an event.
func (sa *SyncAgent) HasListener(event Event) bool {
	if sa == nil {
		return false
	}
	if sa.eventListeners == nil {
		return false
	}
	listeners, hasHandler := sa.eventListeners[event]
	if !hasHandler {
		return false
	}
	return len(listeners) > 0
}

// AddEventListener adds a listener for errors.
func (sa *SyncAgent) AddEventListener(eventFlag Event, listener EventListener) {
	sa.eventListenersLock.Lock()
	sa.eventListeners[eventFlag] = append(sa.eventListeners[eventFlag], listener)
	sa.eventListenersLock.Unlock()
}

// RemoveListeners clears *all* listeners for an Event.
func (sa *SyncAgent) RemoveListeners(eventFlag Event) {
	delete(sa.eventListeners, eventFlag)
}

// OnEvent fires the currently configured event listeners.
func (sa *SyncAgent) OnEvent(eventFlag Event, state ...interface{}) {
	if sa == nil {
		return
	}
	if sa.IsEnabled(eventFlag) && sa.HasListener(eventFlag) {
		sa.triggerListeners(append([]interface{}{TimeNow(), eventFlag}, state...)...)
	}
}

// Infof logs an informational message to the output stream.
func (sa *SyncAgent) Infof(format string, args ...interface{}) {
	if sa == nil {
		return
	}
	sa.WriteEventf(EventInfo, ColorLightWhite, format, args...)
}

// Debugf logs a debug message to the output stream.
func (sa *SyncAgent) Debugf(format string, args ...interface{}) {
	if sa == nil {
		return
	}
	sa.WriteEventf(EventDebug, ColorLightYellow, format, args...)
}

// Warningf logs a debug message to the output stream.
func (sa *SyncAgent) Warningf(format string, args ...interface{}) error {
	if sa == nil {
		return nil
	}
	return sa.Warning(fmt.Errorf(format, args...))
}

// Warning logs a warning error to std err.
func (sa *SyncAgent) Warning(err error) error {
	if sa == nil {
		return err
	}
	return sa.ErrorEventWithState(EventWarning, ColorLightYellow, err)
}

// WarningWithReq logs a warning error to std err with a request.
func (sa *SyncAgent) WarningWithReq(err error, req *http.Request) error {
	if sa == nil {
		return err
	}
	return sa.ErrorEventWithState(EventWarning, ColorLightYellow, err, req)
}

// Errorf writes an event to the log and triggers event listeners.
func (sa *SyncAgent) Errorf(format string, args ...interface{}) error {
	if sa == nil {
		return nil
	}
	return sa.Error(fmt.Errorf(format, args...))
}

// Error logs an error to std err.
func (sa *SyncAgent) Error(err error) error {
	if sa == nil {
		return err
	}
	return sa.ErrorEventWithState(EventError, ColorRed, err)
}

// ErrorWithReq logs an error to std err with a request.
func (sa *SyncAgent) ErrorWithReq(err error, req *http.Request) error {
	if sa == nil {
		return err
	}
	return sa.ErrorEventWithState(EventError, ColorRed, err, req)
}

// Fatalf writes an event to the log and triggers event listeners.
func (sa *SyncAgent) Fatalf(format string, args ...interface{}) error {
	if sa == nil {
		return nil
	}
	return sa.Fatal(fmt.Errorf(format, args...))
}

// Fatal logs the result of a panic to std err.
func (sa *SyncAgent) Fatal(err error) error {
	if sa == nil {
		return err
	}
	return sa.ErrorEventWithState(EventFatalError, ColorRed, err)
}

// FatalWithReq logs the result of a fatal error to std err with a request.
func (sa *SyncAgent) FatalWithReq(err error, req *http.Request) error {
	if sa == nil {
		return err
	}
	return sa.ErrorEventWithState(EventFatalError, ColorRed, err, req)
}

// FatalExit logs the result of a fatal error to std err and calls `exit(1)`.
// NOTE: this terminates the program.
func (sa *SyncAgent) FatalExit(err error) {
	if sa == nil {
		os.Exit(1)
	}

	sa.ErrorEventWithState(EventFatalError, ColorRed, err)
	os.Exit(1)
}

// WriteEventf writes to the standard output and triggers events.
func (sa *SyncAgent) WriteEventf(event Event, color AnsiColorCode, format string, args ...interface{}) {
	if sa == nil {
		return
	}
	if sa.events == nil {
		return
	}
	if sa.events.IsEnabled(event) {
		sa.write(append([]interface{}{TimeNow(), event, color, format}, args...)...)

		if sa.HasListener(event) {
			sa.triggerListeners(append([]interface{}{TimeNow(), event, format}, args...)...)
		}
	}
}

// WriteErrorEventf writes to the error output and triggers events.
func (sa *SyncAgent) WriteErrorEventf(event Event, color AnsiColorCode, format string, args ...interface{}) {
	if sa == nil {
		return
	}
	if sa.events == nil {
		return
	}
	if sa.IsEnabled(event) {
		sa.writeError(append([]interface{}{TimeNow(), event, color, format}, args...)...)

		if sa.HasListener(event) {
			sa.triggerListeners(append([]interface{}{TimeNow(), event, format}, args...)...)
		}
	}
}

// ErrorEventWithState writes an error and triggers events with a given state.
func (sa *SyncAgent) ErrorEventWithState(event Event, color AnsiColorCode, err error, state ...interface{}) error {
	if sa == nil {
		return err
	}
	if sa.events == nil {
		return nil
	}
	if err != nil {
		if sa.IsEnabled(event) {
			sa.writeError(TimeNow(), event, color, "%+v", err)
			if sa.HasListener(event) {
				sa.triggerListeners(append([]interface{}{TimeNow(), event, err}, state...)...)
			}
		}
	}
	return err
}

// triggerListeners triggers the currently configured event listeners.
func (sa *SyncAgent) triggerListeners(actionState ...interface{}) error {
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

	sa.eventListenersLock.Lock()
	listeners := sa.eventListeners[eventFlag]
	sa.eventListenersLock.Unlock()

	for x := 0; x < len(listeners); x++ {
		listener := listeners[x]
		listener(sa.writer, timeSource, eventFlag, actionState[2:]...)
	}

	return nil
}

func (sa *SyncAgent) write(actionState ...interface{}) error {
	return sa.writeWithOutput(sa.writer.PrintfWithTimeSource, actionState...)
}

func (sa *SyncAgent) writeError(actionState ...interface{}) error {
	return sa.writeWithOutput(sa.writer.ErrorfWithTimeSource, actionState...)
}

// writeEventMessage writes an event message.
func (sa *SyncAgent) writeWithOutput(output loggerOutputWithTimeSource, actionState ...interface{}) error {
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

	_, err = output(timeSource, "%s %s", sa.writer.FormatEvent(eventFlag, labelColor), fmt.Sprintf(format, actionState[4:]...))
	return err
}
