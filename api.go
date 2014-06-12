package consulapi

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// QueryOptions are used to parameterize a query
type QueryOptions struct {
	// Providing a datacenter overwrites the DC provided
	// by the Config
	Datacenter string

	// AllowStale allows any Consul server (non-leader) to service
	// a read. This allows for lower latency and higher throughput
	AllowStale bool

	// RequireConsistent forces the read to be fully consistent.
	// This is more expensive but prevents ever performing a stale
	// read.
	RequireConsistent bool

	// WaitIndex is used to enable a blocking query. Waits
	// until the timeout or the next index is reached
	WaitIndex uint64

	// WaitTime is used to bound the duration of a wait.
	// Defaults to that of the Config, but can be overriden.
	WaitTime time.Duration
}

// WriteOptions are used to parameterize a write
type WriteOptions struct {
	// Providing a datacenter overwrites the DC provided
	// by the Config
	Datacenter string
}

// QueryMeta is used to return meta data about a query
type QueryMeta struct {
	// LastIndex. This can be used as a WaitIndex to perform
	// a blocking query
	LastIndex uint64

	// Time of last contact from the leader for the
	// server servicing the request
	LastContact time.Duration

	// Is there a known leader
	KnownLeader bool

	// How long did the request take
	RequestTime time.Duration
}

// WriteMeta is used to return meta data about a write
type WriteMeta struct {
	// How long did the request take
	RequestTime time.Duration
}

// Config is used to configure the creation of a client
type Config struct {
	// Address is the address of the Consul server
	Address string

	// Datacenter to use. If not provided, the default agent datacenter is used.
	Datacenter string

	// HttpClient is the client to use. Default will be
	// used if not provided.
	HttpClient *http.Client

	// WaitTime limits how long a Watch will block. If not provided,
	// the agent default values will be used.
	WaitTime time.Duration
}

// DefaultConfig returns a default configuration for the client
func DefaultConfig() *Config {
	return &Config{
		Address:    "127.0.0.1:8500",
		HttpClient: http.DefaultClient,
	}
}

// Client provides a client to the Consul API
type Client struct {
	config Config
}

// NewClient returns a new client
func NewClient(config *Config) (*Client, error) {
	client := &Client{
		config: *config,
	}
	return client, nil
}

// request is used to help build up a request
type request struct {
	config *Config
	method string
	url    *url.URL
	params url.Values
	body   io.Reader
}

// toHTTP converts the request to an HTTP request
func (r *request) toHTTP() (*http.Request, error) {
	// Encode the query parameters
	r.url.RawQuery = r.params.Encode()

	// Get the url sring
	urlRaw := r.url.String()

	// Create the HTTP request
	return http.NewRequest(r.method, urlRaw, r.body)
}

// setQueryOptions is used to annotate the request with
// additional query options
func (r *request) setQueryOptions(q *QueryOptions) {
	if q.Datacenter != "" {
		r.params.Set("dc", q.Datacenter)
	}
	if q.AllowStale {
		r.params.Set("stale", "")
	}
	if q.RequireConsistent {
		r.params.Set("consistent", "")
	}
	if q.WaitIndex != 0 {
		r.params.Set("index", strconv.FormatUint(q.WaitIndex, 10))
	}
	if q.WaitTime != 0 {
		waitMsec := fmt.Sprintf("%dms", q.WaitTime/time.Millisecond)
		r.params.Set("wait", waitMsec)
	} else if r.config.WaitTime != 0 {
		waitMsec := fmt.Sprintf("%dms", r.config.WaitTime/time.Millisecond)
		r.params.Set("wait", waitMsec)
	}
}

// setWriteOptions is used to annotate the request with
// additional write options
func (r *request) setWriteOptions(q *WriteOptions) {
	if q.Datacenter != "" {
		r.params.Set("dc", q.Datacenter)
	}
}

// newRequest is used to create a new request
func (c *Client) newRequest(method, path string) *request {
	r := &request{
		config: &c.config,
		method: "GET",
		url: &url.URL{
			Scheme: "http",
			Host:   c.config.Address,
			Path:   path,
		},
		params: make(map[string][]string),
	}
	if c.config.Datacenter != "" {
		r.params.Set("dc", c.config.Datacenter)
	}
	return r
}

// doRequest runs a request with our client
func (c *Client) doRequest(req *http.Request) (time.Duration, *http.Response, error) {
	start := time.Now()
	resp, err := c.config.HttpClient.Do(req)
	diff := time.Now().Sub(start)
	return diff, resp, err
}
