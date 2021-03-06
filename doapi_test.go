package godo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

var (
	mux *http.ServeMux

	client *Client

	server *httptest.Server
)

func setup() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	client = NewClient(nil)
	url, _ := url.Parse(server.URL)
	client.BaseURL = url
}

func teardown() {
	server.Close()
}

func testMethod(t *testing.T, r *http.Request, expected string) {
	if expected != r.Method {
		t.Errorf("Request method = %v, expected %v", r.Method, expected)
	}
}

type values map[string]string

func testFormValues(t *testing.T, r *http.Request, values values) {
	expected := url.Values{}
	for k, v := range values {
		expected.Add(k, v)
	}

	r.ParseForm()
	if !reflect.DeepEqual(expected, r.Form) {
		t.Errorf("Request parameters = %v, expected %v", r.Form, expected)
	}
}

func testURLParseError(t *testing.T, err error) {
	if err == nil {
		t.Errorf("Expected error to be returned")
	}
	if err, ok := err.(*url.Error); !ok || err.Op != "parse" {
		t.Errorf("Expected URL parse error, got %+v", err)
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient(nil)
	if c.BaseURL.String() != defaultBaseURL {
		t.Errorf("NewClient BaseURL = %v, expected %v", c.BaseURL.String(), defaultBaseURL)
	}

	if c.UserAgent != userAgent {
		t.Errorf("NewClick UserAgent = %v, expected %v", c.UserAgent, userAgent)
	}
}

func TestNewRequest(t *testing.T) {
	c := NewClient(nil)

	inURL, outURL := "/foo", defaultBaseURL+"foo"
	inBody, outBody := &DropletCreateRequest{Name: "l"}, `{"name":"l","region":"","size":"","image":"","ssh_keys":null}`+"\n"
	req, _ := c.NewRequest("GET", inURL, inBody)

	// test relative URL was expanded
	if req.URL.String() != outURL {
		t.Errorf("NewRequest(%v) URL = %v, expected %v", inURL, req.URL, outURL)
	}

	// test body was JSON encoded
	body, _ := ioutil.ReadAll(req.Body)
	if string(body) != outBody {
		t.Errorf("NewRequest(%v)Body = %v, expected %v", inBody, string(body), outBody)
	}

	// test default user-agent is attached to the request
	userAgent := req.Header.Get("User-Agent")
	if c.UserAgent != userAgent {
		t.Errorf("NewRequest() User-Agent = %v, expected %v", userAgent, c.UserAgent)
	}
}

func TestNewRequest_invalidJSON(t *testing.T) {
	c := NewClient(nil)

	type T struct {
		A map[int]interface{}
	}
	_, err := c.NewRequest("GET", "/", &T{})

	if err == nil {
		t.Error("Expected error to be returned.")
	}
	if err, ok := err.(*json.UnsupportedTypeError); !ok {
		t.Errorf("Expected a JSON error; got %#v.", err)
	}
}

func TestNewRequest_badURL(t *testing.T) {
	c := NewClient(nil)
	_, err := c.NewRequest("GET", ":", nil)
	testURLParseError(t, err)
}

func TestDo(t *testing.T) {
	setup()
	defer teardown()

	type foo struct {
		A string
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if m := "GET"; m != r.Method {
			t.Errorf("Request method = %v, expected %v", r.Method, m)
		}
		fmt.Fprint(w, `{"A":"a"}`)
	})

	req, _ := client.NewRequest("GET", "/", nil)
	body := new(foo)
	client.Do(req, body)

	expected := &foo{"a"}
	if !reflect.DeepEqual(body, expected) {
		t.Errorf("Response body = %v, expected %v", body, expected)
	}
}

func TestDo_httpError(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", 400)
	})

	req, _ := client.NewRequest("GET", "/", nil)
	_, err := client.Do(req, nil)

	if err == nil {
		t.Error("Expected HTTP 400 error.")
	}
}

// Test handling of an error caused by the internal http client's Do()
// function.
func TestDo_redirectLoop(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusFound)
	})

	req, _ := client.NewRequest("GET", "/", nil)
	_, err := client.Do(req, nil)

	if err == nil {
		t.Error("Expected error to be returned.")
	}
	if err, ok := err.(*url.Error); !ok {
		t.Errorf("Expected a URL error; got %#v.", err)
	}
}

func TestCheckResponse(t *testing.T) {
	res := &http.Response{
		Request:    &http.Request{},
		StatusCode: http.StatusBadRequest,
		Body: ioutil.NopCloser(strings.NewReader(`{"message":"m",
			"errors": [{"resource": "r", "field": "f", "code": "c"}]}`)),
	}
	err := CheckResponse(res).(*ErrorResponse)

	if err == nil {
		t.Fatalf("Expected error response.")
	}

	expected := &ErrorResponse{
		Response: res,
		Message:  "m",
	}
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Error = %#v, expected %#v", err, expected)
	}
}

// ensure that we properly handle API errors that do not contain a response
// body
func TestCheckResponse_noBody(t *testing.T) {
	res := &http.Response{
		Request:    &http.Request{},
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(strings.NewReader("")),
	}
	err := CheckResponse(res).(*ErrorResponse)

	if err == nil {
		t.Errorf("Expected error response.")
	}

	expected := &ErrorResponse{
		Response: res,
	}
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Error = %#v, expected %#v", err, expected)
	}
}

func TestErrorResponse_Error(t *testing.T) {
	res := &http.Response{Request: &http.Request{}}
	err := ErrorResponse{Message: "m", Response: res}
	if err.Error() == "" {
		t.Errorf("Expected non-empty ErrorResponse.Error()")
	}
}

func TestDo_rateLimit(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add(headerRateLimit, "60")
		w.Header().Add(headerRateRemaining, "59")
		w.Header().Add(headerRateReset, "1372700873")
	})

	var expected int

	if expected = 0; client.Rate.Limit != expected {
		t.Errorf("Client rate limit = %v, expected %v", client.Rate.Limit, expected)
	}
	if expected = 0; client.Rate.Remaining != expected {
		t.Errorf("Client rate remaining = %v, got %v", client.Rate.Remaining, expected)
	}
	if !client.Rate.Reset.IsZero() {
		t.Errorf("Client rate reset not initialized to zero value")
	}

	req, _ := client.NewRequest("GET", "/", nil)
	client.Do(req, nil)

	if expected = 60; client.Rate.Limit != expected {
		t.Errorf("Client rate limit = %v, expected %v", client.Rate.Limit, expected)
	}
	if expected = 59; client.Rate.Remaining != expected {
		t.Errorf("Client rate remaining = %v, expected %v", client.Rate.Remaining, expected)
	}
	reset := time.Date(2013, 7, 1, 17, 47, 53, 0, time.UTC)
	if client.Rate.Reset.UTC() != reset {
		t.Errorf("Client rate reset = %v, expected %v", client.Rate.Reset, reset)
	}
}

func TestDo_rateLimit_errorResponse(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add(headerRateLimit, "60")
		w.Header().Add(headerRateRemaining, "59")
		w.Header().Add(headerRateReset, "1372700873")
		http.Error(w, "Bad Request", 400)
	})

	var expected int

	req, _ := client.NewRequest("GET", "/", nil)
	client.Do(req, nil)

	if expected = 60; client.Rate.Limit != expected {
		t.Errorf("Client rate limit = %v, expected %v", client.Rate.Limit, expected)
	}
	if expected = 59; client.Rate.Remaining != expected {
		t.Errorf("Client rate remaining = %v, expected %v", client.Rate.Remaining, expected)
	}
	reset := time.Date(2013, 7, 1, 17, 47, 53, 0, time.UTC)
	if client.Rate.Reset.UTC() != reset {
		t.Errorf("Client rate reset = %v, expected %v", client.Rate.Reset, reset)
	}
}

func TestResponse_populatePageValues(t *testing.T) {
	r := http.Response{
		Header: http.Header{
			"Link": {`<https://api.digitalocean.com/?page=1>; rel="first",` +
				` <https://api.digitalocean.com/?page=2>; rel="prev",` +
				` <https://api.digitalocean.com/?page=4>; rel="next",` +
				` <https://api.digitalocean.com/?page=5>; rel="last"`,
			},
		},
	}

	response := newResponse(&r)

	links := map[string]string{
		"first": "https://api.digitalocean.com/?page=1",
		"prev":  "https://api.digitalocean.com/?page=2",
		"next":  "https://api.digitalocean.com/?page=4",
		"last":  "https://api.digitalocean.com/?page=5",
	}

	if expected, got := links["first"], response.FirstPage; expected != got {
		t.Errorf("response.FirstPage: %v, expected %v", got, expected)
	}
	if expected, got := links["prev"], response.PrevPage; expected != got {
		t.Errorf("response.PrevPage: %v, expected %v", got, expected)
	}
	if expected, got := links["next"], response.NextPage; expected != got {
		t.Errorf("response.NextPage: %v, expected %v", got, expected)
	}
	if expected, got := links["last"], response.LastPage; expected != got {
		t.Errorf("response.LastPage: %v, expected %v", got, expected)
	}
}

func TestResponse_populatePageValues_invalid(t *testing.T) {
	r := http.Response{
		Header: http.Header{
			"Link": {`<https://api.digitalocean.com/?page=1>,` +
				`<https://api.digitalocean.com/?page=abc>; rel="first",` +
				`https://api.digitalocean.com/?page=2; rel="prev",` +
				`<https://api.digitalocean.com/>; rel="next",` +
				`<https://api.digitalocean.com/?page=>; rel="last"`,
			},
		},
	}

	response := newResponse(&r)
	if expected, got := "", response.FirstPage; expected != got {
		t.Errorf("response.FirstPage: %v, expected %v", expected, got)
	}
	if expected, got := "", response.PrevPage; expected != got {
		t.Errorf("response.PrevPage: %v, expected %v", expected, got)
	}
	if expected, got := "", response.NextPage; expected != got {
		t.Errorf("response.NextPage: %v, expected %v", expected, got)
	}
	if expected, got := "", response.LastPage; expected != got {
		t.Errorf("response.LastPage: %v, expected %v", expected, got)
	}

	// more invalid URLs
	r = http.Response{
		Header: http.Header{
			"Link": {`<https://api.digitalocean.com/%?page=2>; rel="first"`},
		},
	}
}
