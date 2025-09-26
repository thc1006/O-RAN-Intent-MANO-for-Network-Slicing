package mocks

import (
	"bytes"
	"io"
	"net/http"
	"strings"
)

// MockHTTPClient provides a mock implementation of http.Client for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)

	// Call tracking
	Requests []*http.Request
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.Requests = append(m.Requests, req)
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}

// Helper function to create HTTP responses
func CreateHTTPResponse(statusCode int, body string, headers map[string]string) *http.Response {
	header := make(http.Header)
	for k, v := range headers {
		header.Set(k, v)
	}

	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		Header:     header,
	}
}

// MockRoundTripper implements http.RoundTripper for more advanced mocking
type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
	Requests      []*http.Request
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.Requests = append(m.Requests, req)
	if m.RoundTripFunc != nil {
		return m.RoundTripFunc(req)
	}
	return CreateHTTPResponse(200, "{}", nil), nil
}