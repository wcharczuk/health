package logger

import (
	"sync"
	"time"
)

const (
	// EventAverageQueueLatency is an event that fires when we collect average queue latencies.
	EventAverageQueueLatency Event = "queue_latency"
)

// AverageQueueLatencyListener is a listener for EventAverageQueueLatency.
type AverageQueueLatencyListener func(*Writer, TimeSource, time.Duration)

// NewAverageQueueLatencyListener returns a new listener for Average Queue Latency events.
func NewAverageQueueLatencyListener(listener AverageQueueLatencyListener) EventListener {
	return func(wr *Writer, ts TimeSource, eventFlag Event, state ...interface{}) {
		if len(state) > 0 {
			if typed, isTyped := state[0].(time.Duration); isTyped {
				listener(wr, ts, typed)
			}
		}
	}
}

// DebugPrintAverageLatency prints the average queue latency for an agent.
func DebugPrintAverageLatency(agent *Agent) {
	var (
		debugLatenciesLock sync.Mutex
		debugLatencies     = []time.Duration{}
	)

	agent.EnableEvent(EventAverageQueueLatency)
	agent.AddDebugListener(func(_ *Writer, ts TimeSource, ef Event, _ ...interface{}) {
		if ef != EventAverageQueueLatency {
			debugLatenciesLock.Lock()
			debugLatencies = append(debugLatencies, time.Now().UTC().Sub(ts.UTCNow()))
			debugLatenciesLock.Unlock()
		}
	})

	var averageLatency time.Duration
	poll := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-poll.C:
				{
					debugLatenciesLock.Lock()
					averageLatency = MeanOfDuration(debugLatencies)
					debugLatencies = []time.Duration{}
					debugLatenciesLock.Unlock()
					if averageLatency != time.Duration(0) {
						agent.WriteEventf(EventAverageQueueLatency, ColorLightBlack, "%v", averageLatency)
					}
				}
			}
		}
	}()
}
