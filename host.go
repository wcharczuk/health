package health

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/blendlabs/go-request"
	"github.com/blendlabs/go-util"
	"github.com/blendlabs/go-util/collections"
)

var (
	label99th     = util.ColorLightBlack.Apply("99th")
	label90th     = util.ColorLightBlack.Apply("90th")
	label75th     = util.ColorLightBlack.Apply("75th")
	labelAverage  = util.ColorLightBlack.Apply("Average")
	labelLast     = util.ColorLightBlack.Apply("Last")
	labelUptime   = util.ColorLightBlack.Apply("Uptime")
	unknownStatus = util.ColorLightBlack.Apply("UNKNOWN")
	statusUP      = util.ColorGreen.Apply("UP")
	statusDOWN    = util.ColorRed.Apply("DOWN")
)

// NewHost returns a new host.
func NewHost(host string, timeout time.Duration, maxStats int) (*Host, error) {
	hostURL, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	return &Host{
		url:          hostURL,
		maxStats:     maxStats,
		timeout:      timeout,
		startedAtUTC: time.Now().UTC(),
		transport:    &http.Transport{},
		stats:        collections.NewRingBufferWithCapacity(maxStats),
		errs:         collections.NewRingBuffer(),
	}, nil
}

// Host is a server to ping
type Host struct {
	url          *url.URL
	startedAtUTC time.Time
	downAt       *time.Time
	downtime     time.Duration
	stats        collections.Queue
	transport    *http.Transport
	req          *request.Request
	timeout      time.Duration
	errs         collections.Queue
	maxStats     int
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
	return time.Now().UTC().Sub(h.startedAtUTC)
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

func (h *Host) ensureRequest() *request.Request {
	if h.req != nil {
		return h.req
	}

	req := request.New().
		AsGet().
		WithKeepAlives().
		WithURL(h.url.String()).
		WithTimeout(h.timeout)

	h.req = req
	return req
}

// Ping pings a host and returns the elapsed time and any errors.
func (h *Host) Ping() (time.Duration, error) {
	req := h.ensureRequest()

	begin := time.Now()
	meta, err := req.ExecuteWithMeta()
	elapsed := time.Now().Sub(begin)
	if err != nil {
		return elapsed, err
	}

	if meta.StatusCode > http.StatusOK {
		return elapsed, fmt.Errorf("non-200 returned from endpoint")
	}

	return elapsed, nil
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

// WriteStatus writes the status line for the host.
func (h Host) WriteStatus(hostWidth int, maxElapsed time.Duration, writer io.Writer) error {
	host := util.ColorReset.Apply(util.String.FixedWidthLeftAligned(h.url.String(), hostWidth+2))

	uptimePCT := 1.0
	if h.TotalDowntime() > 0 {
		totalTime := h.TotalTime() / time.Millisecond
		downTime := h.TotalDowntime() / time.Millisecond
		uptimePCT = float64(totalTime-downTime) / float64(totalTime)
	}
	var uptimeText string
	if uptimePCT < 1.0 {
		uptimeText = fmt.Sprintf("%0.3f", uptimePCT*100)
	} else {
		uptimeText = fmt.Sprintf("%d", int(uptimePCT*100))
	}

	if uptimePCT > 0.995 {
		uptimeText = util.ColorGreen.Apply(uptimeText)
	} else if uptimePCT > 0.990 {
		uptimeText = util.ColorLightGreen.Apply(uptimeText)
	} else if uptimePCT > 0.95 {
		uptimeText = util.ColorYellow.Apply(uptimeText)
	} else {
		uptimeText = util.ColorRed.Apply(uptimeText)
	}
	uptimeText = fmt.Sprintf("%s%%%%", uptimeText)

	if !h.IsUp() {
		downFor := time.Now().Sub(*h.downAt)
		_, err := fmt.Fprintf(writer, "%s %6s %-6s Down For: %s\r\n", host, statusDOWN, uptimeText, FormatDuration(downFor))
		return err
	}

	if h.stats.Len() == 0 {
		_, err := fmt.Fprintf(writer, "%s %s\r\n", host, unknownStatus)
		return err
	}

	avg := h.Mean()
	p99 := h.Percentile(99.0)
	p90 := h.Percentile(90.0)

	var last5 []time.Duration
	var last5Floats []float64
	h.stats.ReverseEachUntil(func(v interface{}) bool {
		tv := v.(time.Duration)
		last5 = append(last5, tv)
		last5Floats = append(last5Floats, float64(tv))
		return len(last5) < 5
	})

	last := last5[0]

	buf := bytes.NewBuffer(nil)
	buf.WriteString(host)
	buf.WriteRune(rune(' '))
	buf.WriteString(fmt.Sprintf("%6s", statusUP))
	buf.WriteRune(rune(' '))
	buf.WriteString(fmt.Sprintf("%-6s", uptimeText))
	buf.WriteRune(rune(' '))
	buf.WriteString(fmt.Sprintf("%-5s", FormatSparklines(last5Floats, float64(maxElapsed))))
	buf.WriteRune(rune(' '))
	buf.WriteString(fmt.Sprintf("%s: %-6s", labelLast, FormatDuration(RoundDuration(last, time.Millisecond))))
	buf.WriteString(fmt.Sprintf("%s: %-6s", labelAverage, FormatDuration(RoundDuration(avg, time.Millisecond))))
	buf.WriteString(fmt.Sprintf("%s: %-6s", label99th, FormatDuration(RoundDuration(p99, time.Millisecond))))
	buf.WriteString(fmt.Sprintf("%s: %-6s", label90th, FormatDuration(RoundDuration(p90, time.Millisecond))))
	buf.WriteRune(rune('\r'))
	buf.WriteRune(rune('\n'))
	_, err := writer.Write(buf.Bytes())
	return err
}

// WriteDowntimeStatus writes downtime status if any is present.
func (h Host) WriteDowntimeStatus(hostWidth int, writer io.Writer) error {
	host := util.ColorReset.Apply(util.String.FixedWidthLeftAligned(h.url.String(), hostWidth+2))

	if h.TotalDowntime() > 0 {
		totalTime := h.TotalTime()
		downTime := h.TotalDowntime()
		uptimePCT := float64((totalTime-downTime)/time.Millisecond) / float64(totalTime/time.Millisecond)
		fmt.Fprintf(writer, "%s total: %v down: %v Δ: %0.3f%%\r\n", host, totalTime, downTime, uptimePCT*100)
	}

	return nil
}

// WriteErrorStatus writes the error status.
func (h Host) WriteErrorStatus(hostWidth int, writer io.Writer) error {
	host := util.ColorReset.Apply(util.String.FixedWidthLeftAligned(h.url.String(), hostWidth+2))

	buf := bytes.NewBuffer(nil)

	var index int
	h.errs.EachUntil(func(err interface{}) bool {
		buf.WriteString(host)
		buf.WriteRune(rune(' '))
		buf.WriteString(fmt.Sprintf("%v", err))
		buf.WriteRune(rune('\r'))
		buf.WriteRune(rune('\n'))
		index++
		return index < 5
	})

	_, writeErr := writer.Write(buf.Bytes())
	return writeErr
}
