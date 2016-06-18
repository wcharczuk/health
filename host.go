package health

import (
	"fmt"
	"io"
	"io/ioutil"
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
	unknownStatus = util.Color("UNKNOWN", util.ColorLightBlack)
	statusUP      = util.Color("UP", util.ColorGreen)
	statusDOWN    = util.Color("DOWN", util.ColorRed)
)

// NewHost returns a new host.
func NewHost(host string, timeout time.Duration, maxStats int) *Host {
	hostURL, _ := url.Parse(host)
	return &Host{
		url:      hostURL,
		maxStats: maxStats,
		timeout:  timeout,
		stats:    collections.NewRingBufferWithCapacity(maxStats),
	}
}

// Host is a server to ping
type Host struct {
	url       *url.URL
	downAt    *time.Time
	stats     collections.Queue
	transport *http.Transport
	timeout   time.Duration
	maxStats  int
}

// SetTimeout sets the timeout used by `ping`.
func (h *Host) SetTimeout(timeout time.Duration) {
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

// SetUp sets a host as up.
func (h *Host) SetUp() {
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
		WithTimeout(h.timeout)

	if h.transport != nil {
		req = req.WithTransport(h.transport)
	} else {
		req = req.OnCreateTransport(func(_ *url.URL, t *http.Transport) {
			h.transport = t
		})
	}

	res, err := req.FetchRawResponse()

	if err != nil {
		return time.Now().Sub(begin), err
	}

	if res != nil && res.Body != nil {
		io.Copy(ioutil.Discard, res.Body)
		res.Body.Close()
	}

	return time.Now().Sub(begin), nil
}

// Mean returns the average duration.
func (h Host) Mean() time.Duration {
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
	if h.IsUp() && h.stats.Len() > 1 {
		last := h.stats.PeekBack()
		avg := h.Mean()
		p99 := h.Percentile(99.0)
		p90 := h.Percentile(90.0)
		p75 := h.Percentile(75.0)
		return fmt.Sprintf(
			"%s %6s %s: %-6s %s: %-6s %s: %-7s %s: %-6s %s: %-6s",
			host, statusUP,
			labelLast, FormatDuration(last.(time.Duration)),
			labelAverage, FormatDuration(avg),
			label99th, FormatDuration(p99),
			label90th, FormatDuration(p90),
			label75th, FormatDuration(p75),
		)
	} else if !h.IsUp() {
		downFor := time.Now().Sub(*h.downAt)
		return fmt.Sprintf("%s %6s Down For: %s", host, statusDOWN, FormatDuration(downFor))
	}
	return fmt.Sprintf("%s %s", host, unknownStatus)
}
