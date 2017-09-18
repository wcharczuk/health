package request

import (
	"fmt"
	"strconv"

	logger "github.com/blendlabs/go-logger"
)

const (
	// Event is a diagnostics agent event flag.
	Event logger.Event = "request"
	// EventResponse is a diagnostics agent event flag.
	EventResponse logger.Event = "request.response"
)

// NewOutgoingListener creates a new logger handler for `EventFlagOutgoingResponse` events.
func NewOutgoingListener(handler func(writer *logger.Writer, ts logger.TimeSource, req *Meta)) logger.EventListener {
	return func(writer *logger.Writer, ts logger.TimeSource, eventFlag logger.Event, state ...interface{}) {
		handler(writer, ts, state[0].(*Meta))
	}
}

// WriteOutgoingRequest is a helper method to write outgoing request events to a logger writer.
func WriteOutgoingRequest(writer *logger.Writer, ts logger.TimeSource, req *Meta) {
	buffer := writer.GetBuffer()
	defer writer.PutBuffer(buffer)
	buffer.WriteString("[" + writer.Colorize(string(Event), logger.ColorGreen) + "]")
	buffer.WriteRune(logger.RuneSpace)
	buffer.WriteString(fmt.Sprintf("%s %s", req.Verb, req.URL.String()))
	writer.WriteWithTimeSource(ts, buffer.Bytes())
}

// WriteOutgoingRequestBody is a helper method to write outgoing request bodies to a logger writer.
func WriteOutgoingRequestBody(writer *logger.Writer, ts logger.TimeSource, req *Meta) {
	buffer := writer.GetBuffer()
	defer writer.PutBuffer(buffer)
	buffer.WriteString("[" + writer.Colorize(string(Event), logger.ColorGreen) + "]")
	buffer.WriteRune(logger.RuneSpace)
	buffer.WriteString("request body")
	buffer.WriteRune(logger.RuneNewline)
	buffer.Write(req.Body)
	writer.WriteWithTimeSource(ts, buffer.Bytes())
}

// NewOutgoingResponseListener creates a new logger handler for `EventFlagOutgoingResponse` events.
func NewOutgoingResponseListener(handler func(writer *logger.Writer, ts logger.TimeSource, req *Meta, res *ResponseMeta, body []byte)) logger.EventListener {
	return func(writer *logger.Writer, ts logger.TimeSource, eventFlag logger.Event, state ...interface{}) {
		handler(writer, ts, state[0].(*Meta), state[1].(*ResponseMeta), state[2].([]byte))
	}
}

// WriteOutgoingRequestResponse is a helper method to write outgoing request response events to a logger writer.
func WriteOutgoingRequestResponse(writer *logger.Writer, ts logger.TimeSource, req *Meta, res *ResponseMeta, body []byte) {
	buffer := writer.GetBuffer()
	defer writer.PutBuffer(buffer)
	buffer.WriteString("[" + writer.Colorize(string(EventResponse), logger.ColorGreen) + "]")
	buffer.WriteRune(logger.RuneSpace)
	buffer.WriteString(fmt.Sprintf("%s %s %s", writer.ColorizeByStatusCode(res.StatusCode, strconv.Itoa(res.StatusCode)), req.Verb, req.URL.String()))
	buffer.WriteRune(logger.RuneNewline)
	buffer.Write(body)
	writer.WriteWithTimeSource(ts, buffer.Bytes())
}
