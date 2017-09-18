package request

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/blendlabs/go-exception"
	logger "github.com/blendlabs/go-logger"
	"github.com/blendlabs/go-util"
)

// Get returns a new get request.
func Get(url string) *Request {
	return New().AsGet().WithURL(url)
}

// Post returns a new post request with an optional body.
func Post(url string, body []byte) *Request {
	if len(body) > 0 {
		return New().AsPost().WithURL(url).WithPostBody(body)
	}
	return New().AsPost().WithURL(url)
}

// New returns a new HTTPRequest instance.
func New() *Request {
	return &Request{
		Scheme:    "http",
		Verb:      "GET",
		KeepAlive: false,
	}
}

// Request makes http requests.
type Request struct {
	Verb string

	Scheme string
	Host   string
	Path   string

	QueryString url.Values

	Cookies []*http.Cookie

	Header            http.Header
	BasicAuthUsername string
	BasicAuthPassword string

	ContentType string
	PostData    url.Values
	Body        []byte

	Timeout time.Duration

	TLSClientCertPath string
	TLSClientKeyPath  string
	TLSClientCert     []byte
	TLSClientKey      []byte
	TLSSkipVerify     bool

	KeepAlive        bool
	KeepAliveTimeout time.Duration
	Label            string

	logger         *logger.Agent
	state          interface{}
	postedFiles    []PostedFile
	responseBuffer Buffer
	requestStart   time.Time

	err error

	ctx                             context.Context
	transport                       *http.Transport
	createTransportHandler          CreateTransportHandler
	incomingResponseHandler         ResponseHandler
	statefulIncomingResponseHandler StatefulResponseHandler
	outgoingRequestHandler          OutgoingRequestHandler
	mockProvider                    MockedResponseProvider
}

// OnResponse configures an event receiver.
func (hr *Request) OnResponse(hook ResponseHandler) *Request {
	hr.incomingResponseHandler = hook
	return hr
}

// OnResponseStateful configures an event receiver that includes the request state.
func (hr *Request) OnResponseStateful(hook StatefulResponseHandler) *Request {
	hr.statefulIncomingResponseHandler = hook
	return hr
}

// OnCreateTransport configures an event receiver.
func (hr *Request) OnCreateTransport(hook CreateTransportHandler) *Request {
	hr.createTransportHandler = hook
	return hr
}

// OnRequest configures an event receiver.
func (hr *Request) OnRequest(hook OutgoingRequestHandler) *Request {
	hr.outgoingRequestHandler = hook
	return hr
}

// WithContext sets a context for the request.
func (hr *Request) WithContext(ctx context.Context) *Request {
	hr.ctx = ctx
	return hr
}

// WithState adds a state object to the request for later usage.
func (hr *Request) WithState(state interface{}) *Request {
	hr.state = state
	return hr
}

// WithLabel gives the request a logging label.
func (hr *Request) WithLabel(label string) *Request {
	hr.Label = label
	return hr
}

// WithVerifyTLS skips the bad certificate checking on TLS requests.
func (hr *Request) WithVerifyTLS(shouldVerify bool) *Request {
	hr.TLSSkipVerify = !shouldVerify
	return hr
}

// WithMockProvider mocks a request response.
func (hr *Request) WithMockProvider(provider MockedResponseProvider) *Request {
	hr.mockProvider = provider
	return hr
}

// WithLogger enables logging with HTTPRequestLogLevelErrors.
func (hr *Request) WithLogger(agent *logger.Agent) *Request {
	hr.logger = agent
	return hr
}

// Logger returns the request diagnostics agent.
func (hr *Request) Logger() *logger.Agent {
	return hr.logger
}

// WithTransport sets a transport for the request.
func (hr *Request) WithTransport(transport *http.Transport) *Request {
	hr.transport = transport
	return hr
}

// WithKeepAlives sets if the request should use the `Connection=keep-alive` header or not.
func (hr *Request) WithKeepAlives() *Request {
	hr.KeepAlive = true
	hr = hr.WithHeader("Connection", "keep-alive")
	return hr
}

// WithKeepAliveTimeout sets a keep alive timeout for the requests transport.
func (hr *Request) WithKeepAliveTimeout(timeout time.Duration) *Request {
	hr.KeepAliveTimeout = timeout
	return hr
}

// WithContentType sets the `Content-Type` header for the request.
func (hr *Request) WithContentType(contentType string) *Request {
	hr.ContentType = contentType
	return hr
}

// WithScheme sets the scheme, or protocol, of the request.
func (hr *Request) WithScheme(scheme string) *Request {
	hr.Scheme = scheme
	return hr
}

// WithHost sets the target url host for the request.
func (hr *Request) WithHost(host string) *Request {
	hr.Host = host
	return hr
}

// WithPath sets the path component of the host url..
func (hr *Request) WithPath(path string) *Request {
	hr.Path = path
	return hr
}

// WithPathf sets the path component of the host url by the format and arguments.
func (hr *Request) WithPathf(format string, args ...interface{}) *Request {
	hr.Path = fmt.Sprintf(format, args...)
	return hr
}

// WithCombinedPath sets the path component of the host url by combining the input path segments.
func (hr *Request) WithCombinedPath(components ...string) *Request {
	hr.Path = util.String.CombinePathComponents(components...)
	return hr
}

// WithURL sets the request target url whole hog.
func (hr *Request) WithURL(urlString string) *Request {
	workingURL, err := url.Parse(urlString)
	if err != nil {
		hr.err = err
		return hr
	}

	hr.Scheme = workingURL.Scheme
	hr.Host = workingURL.Host
	hr.Path = workingURL.Path
	queryValues, err := url.ParseQuery(workingURL.RawQuery)
	if err != nil {
		hr.err = err
		return hr
	}
	hr.QueryString = queryValues
	return hr
}

// WithHeader sets a header on the request.
func (hr *Request) WithHeader(field string, value string) *Request {
	if hr.Header == nil {
		hr.Header = http.Header{}
	}
	hr.Header.Set(field, value)
	return hr
}

// WithQueryString sets a query string value for the host url of the request.
func (hr *Request) WithQueryString(field string, value string) *Request {
	if hr.QueryString == nil {
		hr.QueryString = url.Values{}
	}
	hr.QueryString.Add(field, value)
	return hr
}

// WithCookie sets a cookie for the request.
func (hr *Request) WithCookie(cookie *http.Cookie) *Request {
	if hr.Cookies == nil {
		hr.Cookies = []*http.Cookie{}
	}
	hr.Cookies = append(hr.Cookies, cookie)
	return hr
}

// WithPostData sets a post data value for the request.
func (hr *Request) WithPostData(field string, value string) *Request {
	if hr.PostData == nil {
		hr.PostData = url.Values{}
	}
	hr.PostData.Add(field, value)
	return hr
}

// WithPostDataFromObject sets the post data for a request as json from a given object.
// Remarks; this differs from `WithJSONBody` in that it sets individual post form fields
// for each member of the object.
func (hr *Request) WithPostDataFromObject(object interface{}) *Request {
	postDatums := util.Reflection.DecomposeToPostDataAsJSON(object)

	for _, item := range postDatums {
		hr.WithPostData(item.Key, item.Value)
	}

	return hr
}

// WithPostedFile adds a posted file to the multipart form elements of the request.
func (hr *Request) WithPostedFile(key, fileName string, fileContents io.Reader) *Request {
	hr.postedFiles = append(hr.postedFiles, PostedFile{Key: key, FileName: fileName, FileContents: fileContents})
	return hr
}

// WithBasicAuth sets the basic auth headers for a request.
func (hr *Request) WithBasicAuth(username, password string) *Request {
	hr.BasicAuthUsername = username
	hr.BasicAuthPassword = password
	return hr
}

// WithTimeout sets a timeout for the request.
// Remarks: This timeout is enforced on client connect, not on request read + response.
func (hr *Request) WithTimeout(timeout time.Duration) *Request {
	hr.Timeout = timeout
	return hr
}

// WithClientTLSCertPath sets a tls cert on the transport for the request.
func (hr *Request) WithClientTLSCertPath(certPath string) *Request {
	hr.TLSClientCertPath = certPath
	return hr
}

// WithClientTLSCert sets a tls cert on the transport for the request.
func (hr *Request) WithClientTLSCert(cert []byte) *Request {
	hr.TLSClientCert = cert
	return hr
}

// WithClientTLSKeyPath sets a tls key on the transport for the request.
func (hr *Request) WithClientTLSKeyPath(keyPath string) *Request {
	hr.TLSClientKeyPath = keyPath
	return hr
}

// WithClientTLSKey sets a tls key on the transport for the request.
func (hr *Request) WithClientTLSKey(key []byte) *Request {
	hr.TLSClientKey = key
	return hr
}

// WithVerb sets the http verb of the request.
func (hr *Request) WithVerb(verb string) *Request {
	hr.Verb = verb
	return hr
}

// AsGet sets the http verb of the request to `GET`.
func (hr *Request) AsGet() *Request {
	hr.Verb = "GET"
	return hr
}

// AsPost sets the http verb of the request to `POST`.
func (hr *Request) AsPost() *Request {
	hr.Verb = "POST"
	return hr
}

// AsPut sets the http verb of the request to `PUT`.
func (hr *Request) AsPut() *Request {
	hr.Verb = "PUT"
	return hr
}

// AsPatch sets the http verb of the request to `PATCH`.
func (hr *Request) AsPatch() *Request {
	hr.Verb = "PATCH"
	return hr
}

// AsDelete sets the http verb of the request to `DELETE`.
func (hr *Request) AsDelete() *Request {
	hr.Verb = "DELETE"
	return hr
}

// AsOptions sets the http verb of the request to `OPTIONS`.
func (hr *Request) AsOptions() *Request {
	hr.Verb = "OPTIONS"
	return hr
}

// WithResponseBuffer sets the response buffer for the request (if you want to re-use one).
// An example is if you're constantly pinging an endpoint with a similarly sized response,
// You can just re-use a buffer for reading the response.
func (hr *Request) WithResponseBuffer(buffer Buffer) *Request {
	hr.responseBuffer = buffer
	return hr
}

// WithPostBodyAsJSON sets the post body raw to be the json representation of an object.
func (hr *Request) WithPostBodyAsJSON(object interface{}) *Request {
	return hr.WithPostBodySerialized(object, serializeJSON).WithContentType("application/json")
}

// WithPostBodyAsXML sets the post body raw to be the xml representation of an object.
func (hr *Request) WithPostBodyAsXML(object interface{}) *Request {
	return hr.WithPostBodySerialized(object, serializeXML).WithContentType("application/xml")
}

// WithPostBodySerialized sets the post body with the results of the given serializer.
func (hr *Request) WithPostBodySerialized(object interface{}, serialize Serializer) *Request {
	body, _ := serialize(object)
	return hr.WithPostBody(body)
}

// WithPostBody sets the post body directly.
func (hr *Request) WithPostBody(body []byte) *Request {
	hr.Body = body
	return hr
}

// URL returns the currently formatted request target url.
func (hr *Request) URL() *url.URL {
	workingURL := &url.URL{Scheme: hr.Scheme, Host: hr.Host, Path: hr.Path}
	workingURL.RawQuery = hr.QueryString.Encode()
	return workingURL
}

// Meta returns the request as a HTTPRequestMeta.
func (hr Request) Meta() *Meta {
	return &Meta{
		StartTime: hr.requestStart,
		Verb:      hr.Verb,
		URL:       hr.URL(),
		Body:      hr.PostBody(),
		Headers:   hr.Headers(),
	}
}

// PostBody returns the current post body.
func (hr Request) PostBody() []byte {
	if len(hr.Body) > 0 {
		return hr.Body
	} else if len(hr.PostData) > 0 {
		return []byte(hr.PostData.Encode())
	}
	return nil
}

// Headers returns the headers on the request.
func (hr Request) Headers() http.Header {
	headers := http.Header{}
	for key, values := range hr.Header {
		for _, value := range values {
			headers.Set(key, value)
		}
	}
	if len(hr.PostData) > 0 {
		headers.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if !isEmpty(hr.ContentType) {
		headers.Set("Content-Type", hr.ContentType)
	}
	return headers
}

// Request returns a http.Request for the HTTPRequest.
func (hr *Request) Request() (*http.Request, error) {
	if hr.err != nil {
		return nil, hr.err
	}

	workingURL := hr.URL()

	if len(hr.Body) > 0 && len(hr.PostData) > 0 {
		return nil, exception.New("Cant set both a body and have post data.")
	}

	req, err := http.NewRequest(hr.Verb, workingURL.String(), bytes.NewBuffer(hr.PostBody()))
	if err != nil {
		return nil, exception.Wrap(err)
	}

	if hr.ctx != nil {
		req = req.WithContext(hr.ctx)
	}

	if !isEmpty(hr.BasicAuthUsername) {
		req.SetBasicAuth(hr.BasicAuthUsername, hr.BasicAuthPassword)
	}

	if hr.Cookies != nil {
		for i := 0; i < len(hr.Cookies); i++ {
			cookie := hr.Cookies[i]
			req.AddCookie(cookie)
		}
	}

	for key, values := range hr.Headers() {
		for _, value := range values {
			req.Header.Set(key, value)
		}
	}

	return req, nil
}

// Response makes the actual request but returns the underlying http.Response object.
func (hr *Request) Response() (*http.Response, error) {
	req, err := hr.Request()
	if err != nil {
		return nil, err
	}

	hr.logRequest()

	if hr.mockProvider != nil {
		mockedRes := hr.mockProvider(hr)
		if mockedRes != nil {
			return mockedRes.Response(), mockedRes.Err
		}
	}

	client := &http.Client{}
	if hr.requiresCustomTransport() {
		transport, transportErr := hr.getTransport()
		if transportErr != nil {
			return nil, exception.Wrap(transportErr)
		}
		client.Transport = transport
	}

	if hr.Timeout != time.Duration(0) {
		client.Timeout = hr.Timeout
	}

	res, resErr := client.Do(req)
	return res, exception.Wrap(resErr)
}

// Execute makes the request but does not read the response.
func (hr *Request) Execute() error {
	_, err := hr.ExecuteWithMeta()
	return exception.Wrap(err)
}

// ExecuteWithMeta makes the request and returns the meta of the response.
func (hr *Request) ExecuteWithMeta() (*ResponseMeta, error) {
	res, err := hr.Response()
	if err != nil {
		return nil, exception.Wrap(err)
	}
	meta := NewResponseMeta(res)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
		if hr.responseBuffer != nil {
			contentLength, err := hr.responseBuffer.ReadFrom(res.Body)
			if err != nil {
				return nil, exception.Wrap(err)
			}
			meta.ContentLength = contentLength
			if hr.incomingResponseHandler != nil {
				hr.logResponse(meta, hr.responseBuffer.Bytes(), hr.state)
			}
		} else {
			contents, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return nil, exception.Wrap(err)
			}
			meta.ContentLength = int64(len(contents))
			hr.logResponse(meta, contents, hr.state)
		}
	}

	return meta, nil
}

// BytesWithMeta fetches the response as bytes with meta.
func (hr *Request) BytesWithMeta() ([]byte, *ResponseMeta, error) {
	res, err := hr.Response()
	resMeta := NewResponseMeta(res)
	if err != nil {
		return nil, resMeta, exception.Wrap(err)
	}
	defer res.Body.Close()

	bytes, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return nil, resMeta, exception.Wrap(readErr)
	}

	resMeta.ContentLength = int64(len(bytes))
	hr.logResponse(resMeta, bytes, hr.state)
	return bytes, resMeta, nil
}

// Bytes fetches the response as bytes.
func (hr *Request) Bytes() ([]byte, error) {
	contents, _, err := hr.BytesWithMeta()
	return contents, err
}

// String returns the body of the response as a string.
func (hr *Request) String() (string, error) {
	responseStr, _, err := hr.StringWithMeta()
	return responseStr, err
}

// StringWithMeta returns the body of the response as a string in addition to the response metadata.
func (hr *Request) StringWithMeta() (string, *ResponseMeta, error) {
	contents, meta, err := hr.BytesWithMeta()
	return string(contents), meta, err
}

// JSON unmarshals the response as json to an object.
func (hr *Request) JSON(destination interface{}) error {
	_, err := hr.deserialize(newJSONDeserializer(destination))
	return err
}

// JSONWithMeta unmarshals the response as json to an object with metadata.
func (hr *Request) JSONWithMeta(destination interface{}) (*ResponseMeta, error) {
	return hr.deserialize(newJSONDeserializer(destination))
}

// JSONWithErrorHandler unmarshals the response as json to an object with metadata or an error object depending on the meta.
func (hr *Request) JSONWithErrorHandler(successObject interface{}, errorObject interface{}) (*ResponseMeta, error) {
	return hr.deserializeWithError(newJSONDeserializer(successObject), newJSONDeserializer(errorObject))
}

// JSONError unmarshals the response as json to an object if the meta indiciates an error.
func (hr *Request) JSONError(errorObject interface{}) (*ResponseMeta, error) {
	return hr.deserializeWithError(nil, newJSONDeserializer(errorObject))
}

// XML unmarshals the response as xml to an object with metadata.
func (hr *Request) XML(destination interface{}) error {
	_, err := hr.deserialize(newXMLDeserializer(destination))
	return err
}

// XMLWithMeta unmarshals the response as xml to an object with metadata.
func (hr *Request) XMLWithMeta(destination interface{}) (*ResponseMeta, error) {
	return hr.deserialize(newXMLDeserializer(destination))
}

// XMLWithErrorHandler unmarshals the response as xml to an object with metadata or an error object depending on the meta.
func (hr *Request) XMLWithErrorHandler(successObject interface{}, errorObject interface{}) (*ResponseMeta, error) {
	return hr.deserializeWithError(newXMLDeserializer(successObject), newXMLDeserializer(errorObject))
}

// Deserialized runs a deserializer with the response.
func (hr *Request) Deserialized(deserialize Deserializer) (*ResponseMeta, error) {
	meta, responseErr := hr.deserialize(func(body []byte) error {
		return deserialize(body)
	})
	return meta, responseErr
}

func (hr *Request) requiresCustomTransport() bool {
	return (!isEmpty(hr.TLSClientCertPath) && !isEmpty(hr.TLSClientKeyPath)) ||
		(!isEmpty(string(hr.TLSClientCert)) && !isEmpty(string(hr.TLSClientKey))) ||
		hr.transport != nil ||
		hr.createTransportHandler != nil ||
		hr.TLSSkipVerify
}

func (hr *Request) getTransport() (*http.Transport, error) {
	if hr.transport != nil {
		return hr.transport, nil
	}
	return hr.Transport()
}

// Transport returns the the custom transport for the request.
func (hr *Request) Transport() (*http.Transport, error) {
	transport := &http.Transport{
		DisableCompression: false,
		DisableKeepAlives:  !hr.KeepAlive,
	}

	dialer := &net.Dialer{}
	if hr.Timeout != time.Duration(0) {
		dialer.Timeout = hr.Timeout
	}

	if hr.KeepAlive {
		if hr.KeepAliveTimeout != time.Duration(0) {
			dialer.KeepAlive = hr.KeepAliveTimeout
		} else {
			dialer.KeepAlive = 30 * time.Second
		}
	}

	transport.Dial = dialer.Dial

	if (!isEmpty(hr.TLSClientCertPath) && !isEmpty(hr.TLSClientKeyPath)) || !isEmpty(string(hr.TLSClientCert)) && !isEmpty(string(hr.TLSClientKey)) {
		var cert tls.Certificate
		var err error

		if !isEmpty(hr.TLSClientCertPath) {
			// If we are using cert paths
			cert, err = tls.LoadX509KeyPair(hr.TLSClientCertPath, hr.TLSClientKeyPath)
		} else {
			// If we are using raw certs
			cert, err = tls.X509KeyPair(hr.TLSClientCert, hr.TLSClientKey)
		}
		if err != nil {
			return nil, exception.Wrap(err)
		}
		tlsConfig := &tls.Config{
			InsecureSkipVerify: hr.TLSSkipVerify,
			Certificates:       []tls.Certificate{cert},
		}
		transport.TLSClientConfig = tlsConfig
	} else {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: hr.TLSSkipVerify,
		}
		transport.TLSClientConfig = tlsConfig
	}

	if hr.createTransportHandler != nil {
		hr.createTransportHandler(hr.URL(), transport)
	}

	return transport, nil
}

func (hr *Request) deserialize(handler Deserializer) (*ResponseMeta, error) {
	res, err := hr.Response()
	meta := NewResponseMeta(res)

	if err != nil {
		return meta, exception.Wrap(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return meta, exception.Wrap(err)
	}

	meta.ContentLength = int64(len(body))
	hr.logResponse(meta, body, hr.state)
	if handler != nil {
		err = handler(body)
	}
	return meta, exception.Wrap(err)
}

func (hr *Request) deserializeWithError(okHandler Deserializer, errorHandler Deserializer) (*ResponseMeta, error) {
	res, err := hr.Response()
	meta := NewResponseMeta(res)

	if err != nil {
		return meta, exception.Wrap(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return meta, exception.Wrap(err)
	}

	meta.ContentLength = int64(len(body))
	hr.logResponse(meta, body, hr.state)
	if res.StatusCode == http.StatusOK {
		if okHandler != nil {
			err = okHandler(body)
		}
	} else if errorHandler != nil {
		err = errorHandler(body)
	}
	return meta, exception.Wrap(err)
}

func (hr *Request) logRequest() {
	hr.requestStart = time.Now().UTC()

	meta := hr.Meta()
	if hr.outgoingRequestHandler != nil {
		hr.outgoingRequestHandler(meta)
	}

	if hr.logger != nil {
		hr.logger.OnEvent(Event, meta)
	}
}

func (hr *Request) logResponse(resMeta *ResponseMeta, responseBody []byte, state interface{}) {
	if hr.statefulIncomingResponseHandler != nil {
		hr.statefulIncomingResponseHandler(hr.Meta(), resMeta, responseBody, state)
	}
	if hr.incomingResponseHandler != nil {
		hr.incomingResponseHandler(hr.Meta(), resMeta, responseBody)
	}

	if hr.logger != nil {
		hr.logger.OnEvent(EventResponse, hr.Meta(), resMeta, responseBody, state)
	}
}

// Hash / Mock Utility Functions

// Hash returns a hashcode for a request.
func (hr *Request) Hash() uint32 {
	if hr == nil {
		return 0
	}

	buffer := bytes.NewBuffer(nil)
	buffer.WriteString(hr.Verb)
	buffer.WriteRune('|')
	buffer.WriteString(hr.URL().String())

	h := fnv.New32a()
	h.Write(buffer.Bytes())
	return h.Sum32()
}

// Equals returns if a request equals another request.
func (hr *Request) Equals(other *Request) bool {
	if other == nil {
		return false
	}

	if hr.Verb != other.Verb {
		return false
	}

	if hr.URL().String() != other.URL().String() {
		return false
	}

	return true
}

//--------------------------------------------------------------------------------
// Unexported Utility Functions
//--------------------------------------------------------------------------------

func newJSONDeserializer(object interface{}) Deserializer {
	return func(body []byte) error {
		return deserializeJSON(object, body)
	}
}

func newXMLDeserializer(object interface{}) Deserializer {
	return func(body []byte) error {
		return deserializeXML(object, body)
	}
}

func deserializeJSON(object interface{}, body []byte) error {
	decoder := json.NewDecoder(bytes.NewBuffer(body))
	decodeErr := decoder.Decode(object)
	return exception.Wrap(decodeErr)
}

func deserializeJSONFromReader(object interface{}, body io.Reader) error {
	decoder := json.NewDecoder(body)
	decodeErr := decoder.Decode(object)
	return exception.Wrap(decodeErr)
}

func serializeJSON(object interface{}) ([]byte, error) {
	return json.Marshal(object)
}

func serializeJSONToReader(object interface{}) (io.Reader, error) {
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	err := encoder.Encode(object)
	return buf, err
}

func deserializeXML(object interface{}, body []byte) error {
	return deserializeXMLFromReader(object, bytes.NewBuffer(body))
}

func deserializeXMLFromReader(object interface{}, reader io.Reader) error {
	decoder := xml.NewDecoder(reader)
	return decoder.Decode(object)
}

func serializeXML(object interface{}) ([]byte, error) {
	return xml.Marshal(object)
}

func serializeXMLToReader(object interface{}) (io.Reader, error) {
	buf := bytes.NewBuffer([]byte{})
	encoder := xml.NewEncoder(buf)
	err := encoder.Encode(object)
	return buf, err
}

func isEmpty(str string) bool {
	return len(str) == 0
}
