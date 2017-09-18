go-request
==========

[![Build Status](https://travis-ci.org/blendlabs/go-request.svg?branch=master)](https://travis-ci.org/blendlabs/go-request)

This is a simple convenience library for making service requests and deserializing the results to objects either from JSON or from XML.

## Usage

Here is an exmple of fetching an object:

```go
myObject := MyObject{}
reqErr := request.NewRequest().AsGet().WithUrl("http://myservice.com/api/foo").JSON(&myObject)
```

Here is an example of fetching a raw response:

```go
res, res_err := request.NewRequest().AsGet().WithUrl(host).WithTimeout(5000).FetchRawResponse()
defer res.Body.Close()
//... do things with the raw body ...
```
