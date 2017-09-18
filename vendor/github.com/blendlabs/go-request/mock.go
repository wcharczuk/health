package request

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

var (
	isMocked bool
	mocks    = map[uint32]MockedResponseGenerator{}
	catchAll MockedResponseGenerator
)

// MockedResponse is the metadata and response body for a response
type MockedResponse struct {
	Meta ResponseMeta
	Res  []byte
	Err  error
}

// Response returns a response object for the mock response.
func (mr MockedResponse) Response() *http.Response {
	buff := bytes.NewBuffer(mr.Res)
	res := http.Response{}
	buffLen := buff.Len()
	res.Body = ioutil.NopCloser(buff)
	res.ContentLength = int64(buffLen)
	res.Header = mr.Meta.Headers
	res.StatusCode = mr.Meta.StatusCode
	return &res
}

// MockedResponseGenerator is a function that returns a mocked response.
type MockedResponseGenerator func(*Request) MockedResponse

// MockedResponseInjector injects the mocked response into the request response.
func MockedResponseInjector(req *Request) *MockedResponse {
	if !isMocked {
		return nil
	}

	if gen, hasGen := mocks[req.Hash()]; hasGen {
		return ref(gen(req))
	}
	if catchAll != nil {
		return ref(catchAll(req))
	}
	panic(fmt.Sprintf("no mock registered for %s %s", req.Verb, req.URL().String()))
}

// MockCatchAll sets a "catch all" mock generator.
func MockCatchAll(generator MockedResponseGenerator) {
	isMocked = true
	catchAll = generator
}

// MockResponse mocks are response with a given generator.
func MockResponse(req *Request, generator MockedResponseGenerator) {
	isMocked = true
	reqHashCode := req.Hash()
	mocks[reqHashCode] = generator
}

// MockResponseFromBinary mocks a service request response from a set of binary responses.
func MockResponseFromBinary(req *Request, statusCode int, responseBody []byte) {
	MockResponse(req, func(_ *Request) MockedResponse {
		return MockedResponse{
			Meta: ResponseMeta{
				StatusCode:    statusCode,
				ContentLength: int64(len(responseBody)),
				CompleteTime:  time.Now().UTC(),
			},
			Res: responseBody,
		}
	})
}

// MockResponseFromString mocks a service request response from a string responseBody.
func MockResponseFromString(verb string, url string, statusCode int, responseBody string) {
	MockResponseFromBinary(New().WithVerb(verb).WithURL(url), statusCode, []byte(responseBody))
}

// MockResponseFromFile mocks a service request response from a set of file paths.
func MockResponseFromFile(verb string, url string, statusCode int, responseFilePath string) {
	MockResponse(New().WithVerb(verb).WithURL(url), readFile(statusCode, responseFilePath))
}

// ClearMockedResponses clears any mocked responses that have been set up for the test.
func ClearMockedResponses() {
	isMocked = false
	catchAll = nil
	mocks = map[uint32]MockedResponseGenerator{}
}

func readFile(statusCode int, filePath string) MockedResponseGenerator {
	return func(_ *Request) MockedResponse {
		f, err := os.Open(filePath)
		if err != nil {
			return MockedResponse{
				Meta: ResponseMeta{
					StatusCode: http.StatusInternalServerError,
				},
				Err: err,
			}
		}
		defer f.Close()

		contents, err := ioutil.ReadAll(f)
		if err != nil {
			return MockedResponse{
				Meta: ResponseMeta{
					StatusCode: http.StatusInternalServerError,
				},
				Err: err,
			}
		}

		return MockedResponse{
			Meta: ResponseMeta{
				StatusCode:    statusCode,
				ContentLength: int64(len(contents)),
			},
			Res: contents,
		}
	}
}

func ref(res MockedResponse) *MockedResponse {
	return &res
}
