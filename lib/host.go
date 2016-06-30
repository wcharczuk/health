package health

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/blendlabs/go-request"
	"github.com/blendlabs/go-util"
	"github.com/blendlabs/go-util/collections"
)

var (
	label99th     = util.Color("99th", util.ColorLightBlack)
	label90th     = util.Color("90th", util.ColorLightBlack)
	label75th     = util.Color("75th", util.ColorLightBlack)
	labelAverage  = util.Color("Average", util.ColorLightBlack)
	labelLast     = util.Color("Last", util.ColorLightBlack)
	labelUptime   = util.Color("Uptime", util.ColorLightBlack)
	unknownStatus = util.Color("UNKNOWN", util.ColorLightBlack)
	statusUP      = util.Color("UP", util.ColorGreen)
	statusDOWN    = util.Color("DOWN", util.ColorRed)
)

// NewHost returns a new host.
func NewHost(host string, timeout Duration, maxStats int) *Host {
	hostURL, _ := url.Parse(host)
	return &Host{
		url:       hostURL,
		maxStats:  maxStats,
		timeout:   timeout,
		startedAt: time.Now().UTC(),
		stats:     collections.NewRingBufferWithCapacity(maxStats),
	}
}

// Host is a server to ping
type Host struct {
	url       *url.URL
	startedAt time.Time
	downAt    *time.Time
	downtime  time.Duration
	stats     collections.Queue
	transport *http.Transport
	timeout   Duration
	maxStats  int
}

// SetTimeout sets the timeout used by `ping`.
func (h *Host) SetTimeout(timeout Duration) {
	h.timeout = timeout
}

// URL returns the URL.
func (h Host) URL() *url.URL {
	return h.url
}

// IsUp returns if the host is up or not.
func (h Host) IsUp() bool {
	return h.downAt == nil
}

// TotalDowntime returns the total downtime including a current down window.
func (h *Host) TotalDowntime() time.Duration {
	dt := h.downtime
	if h.downAt != nil {
		dt += time.Now().UTC().Sub(*h.downAt)
	}
	return dt
}

// TotalTime returns the total time the check has been active for.
func (h *Host) TotalTime() time.Duration {
	return time.Now().UTC().Sub(h.startedAt)
}

// SetUp sets a host as up.
func (h *Host) SetUp() {
	if h.downAt != nil {
		h.downtime += time.Now().UTC().Sub(*h.downAt)
	}
	h.downAt = nil
}

// SetDown sets a host as down.
func (h *Host) SetDown(at time.Time) {
	if h.downAt == nil {
		h.downAt = util.OptionalTime(at)
	}
}

// AddTiming adds a timing to the stats collection.
func (h *Host) AddTiming(elapsed time.Duration) {
	if h.stats.Len() >= h.maxStats {
		h.stats.Dequeue()
	}
	h.stats.Enqueue(elapsed)
}

// Ping pings a host and returns the elapsed time and any errors.
func (h *Host) Ping() (time.Duration, error) {
	begin := time.Now()
	req := request.NewHTTPRequest().
		AsGet().
		WithKeepAlives().
		WithURL(h.url.String()).
		WithTimeout(h.timeout.AsDuration())

	if h.transport != nil {
		req = req.WithTransport(h.transport)
	} else {
		req = req.OnCreateTransport(func(_ *url.URL, t *http.Transport) {
			h.transport = t
		})
	}

	meta, err := req.ExecuteWithMeta()

	if err != nil {
		return time.Now().Sub(begin), err
	}

	if meta.StatusCode > http.StatusOK {
		return time.Now().Sub(begin), fmt.Errorf("Non-200 returned from endpoint.")
	}

	return time.Now().Sub(begin), nil
}

// Mean returns the average duration.
func (h Host) Mean() time.Duration {
	// we use a separate sum function because ring buffers
	// are not rangable.
	var accum time.Duration
	h.stats.Each(func(v interface{}) {
		accum += v.(time.Duration)
	})

	return accum / time.Duration(h.stats.Len())
}

// Percentile returns the nth percentile of timing stats.
func (h Host) Percentile(percentile float64) time.Duration {
	var values []time.Duration
	h.stats.Each(func(v interface{}) {
		values = append(values, v.(time.Duration))
	})
	sort.Sort(durations(values))

	index := (percentile / 100.0) * float64(len(values))
	if index == float64(int64(index)) {
		i := RoundToInt(index)

		if i < 1 {
			return time.Duration(0)
		}

		return AverageDuration(values[i-1], values[i])
	}

	i := RoundToInt(index)
	if i < 1 {
		return time.Duration(0)
	}

	return values[i-1]
}

// Status returns the status line for the host.
func (h Host) Status(hostWidth int) string {
	host := util.ColorFixedWidthLeftAligned(h.url.String(), util.ColorReset, hostWidth+2)

	uptimePCT := 1.0
	if h.TotalDowntime() > 0 {
		totalTimeElapsed := time.Now().UTC().Sub(h.startedAt)
		uptimePCT = float64(totalTimeElapsed-h.TotalDowntime()) / float64(totalTimeElapsed)
	}
	uptimeText := fmt.Sprintf("%0.1f", uptimePCT*100)

	if uptimePCT > 0.995 {
		uptimeText = util.Color(uptimeText, util.ColorGreen)
	} else if uptimePCT > 0.990 {
		uptimeText = util.Color(uptimeText, util.ColorLightGreen)
	} else if uptimePCT > 0.95 {
		uptimeText = util.Color(uptimeText, util.ColorYellow)
	} else {
		uptimeText = util.Color(uptimeText, util.ColorRed)
	}
	uptimeText = fmt.Sprintf("(%s)", uptimeText)

	if h.IsUp() && h.stats.Len() > 1 {
		last := h.stats.PeekBack()
		avg := h.Mean()
		p99 := h.Percentile(99.0)
		p90 := h.Percentile(90.0)
		p75 := h.Percentile(75.0)

		return fmt.Sprintf(
			"%s %6s %-6s %s: %-6s %s: %-6s %s: %-7s %s: %-6s %s: %-6s",
			host, statusUP,
			uptimeText,
			labelLast, FormatDuration(RoundDuration(last.(time.Duration), time.Millisecond)),
			labelAverage, FormatDuration(RoundDuration(avg, time.Millisecond)),
			label99th, FormatDuration(RoundDuration(p99, time.Millisecond)),
			label90th, FormatDuration(RoundDuration(p90, time.Millisecond)),
			label75th, FormatDuration(RoundDuration(p75, time.Millisecond)),
		)
	} else if !h.IsUp() {
		downFor := time.Now().Sub(*h.downAt)
		return fmt.Sprintf("%s %6s %-6s Down For: %s", host, statusDOWN, uptimeText, FormatDuration(downFor))
	}
	return fmt.Sprintf("%s %s", host, unknownStatus)
}
