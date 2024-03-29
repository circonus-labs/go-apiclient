// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.17
// +build go1.17

package apiclient

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
)

var rnd *rand.Rand

func init() {
	n, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		rnd = rand.New(rand.NewSource(time.Now().UTC().UnixNano())) //nolint:gosec //G404
		return
	}
	rand.New(rand.NewSource(n.Int64())) //nolint:gosec //G404
}

const (
	// a few sensible defaults
	defaultAPIURL = "https://api.circonus.com/v2"
	defaultAPIApp = "circonus-goapiclient"
	minRetryWait  = 1 * time.Second
	maxRetryWait  = 15 * time.Second
	maxRetries    = 4 // equating to 1 + maxRetries total attempts
)

// Logger facilitates use of any logger supporting the required methods
// rather than just standard log package log.Logger
type Logger interface {
	Printf(string, ...interface{})
}

// TokenKeyType - Circonus API Token key
type TokenKeyType string

// TokenAppType - Circonus API Token app name
type TokenAppType string

// TokenAccountIDType - Circonus API Token account id
type TokenAccountIDType string

// CIDType Circonus object cid
type CIDType *string

// IDType Circonus object id
type IDType int

// URLType submission url type
type URLType string

// SearchQueryType search query (see: https://login.circonus.com/resources/api#searching)
type SearchQueryType string

// SearchFilterType search filter (see: https://login.circonus.com/resources/api#filtering)
type SearchFilterType map[string][]string

// TagType search/select/custom tag(s) type
type TagType []string

// Config options for Circonus API
type Config struct {
	Log Logger
	// TLSConfig defines a custom tls configuration to use when communicating with the API
	TLSConfig *tls.Config
	// CACert deprecating, use TLSConfig instead
	CACert *x509.CertPool
	// URL defines the API URL - default https://api.circonus.com/v2/
	URL string
	// TokenKey defines the key to use when communicating with the API
	TokenKey string
	// TokenApp defines the app to use when communicating with the API
	TokenApp       string
	TokenAccountID string
	MinRetryDelay  string
	MaxRetryDelay  string
	MaxRetries     uint
	DisableRetries bool
	Debug          bool
}

// API Circonus API
type API struct {
	Log                     Logger
	caCert                  *x509.CertPool
	tlsConfig               *tls.Config
	apiURL                  *url.URL
	key                     TokenKeyType
	app                     TokenAppType
	accountID               TokenAccountIDType
	minRetryDelay           time.Duration
	maxRetryDelay           time.Duration
	maxRetries              uint
	useExponentialBackoff   bool
	Debug                   bool
	useExponentialBackoffmu sync.Mutex
}

// NewClient returns a new Circonus API (alias for New)
func NewClient(ac *Config) (*API, error) {
	return New(ac)
}

// NewAPI returns a new Circonus API (alias for New)
func NewAPI(ac *Config) (*API, error) {
	return New(ac)
}

// New returns a new Circonus API
func New(ac *Config) (*API, error) {

	if ac == nil {
		return nil, errors.New("invalid Circonus API configuration (nil)")
	}

	key := TokenKeyType(ac.TokenKey)
	if key == "" {
		return nil, errors.New("Circonus API Token is required")
	}

	app := TokenAppType(ac.TokenApp)
	if app == "" {
		app = defaultAPIApp
	}

	acctID := TokenAccountIDType(ac.TokenAccountID)

	au := ac.URL
	if au == "" {
		au = defaultAPIURL
	}
	if !strings.Contains(au, "/") {
		// if just a hostname is passed, ASSume "https" and a path prefix of "/v2"
		au = fmt.Sprintf("https://%s/v2", ac.URL)
	}
	if last := len(au) - 1; last >= 0 && au[last] == '/' {
		// strip off trailing '/'
		au = au[:last]
	}
	apiURL, err := url.Parse(au)
	if err != nil {
		return nil, errors.Wrap(err, "parsing Circonus API URL")
	}

	a := &API{
		apiURL:                apiURL,
		key:                   key,
		app:                   app,
		accountID:             acctID,
		caCert:                ac.CACert,
		tlsConfig:             ac.TLSConfig,
		Debug:                 ac.Debug,
		Log:                   ac.Log,
		useExponentialBackoff: false,
	}

	a.Debug = ac.Debug
	a.Log = ac.Log
	if a.Debug && a.Log == nil {
		a.Log = log.New(os.Stdout, "", log.LstdFlags)
	}
	if a.Log == nil {
		a.Log = log.New(io.Discard, "", log.LstdFlags)
	}

	a.maxRetries = maxRetries
	if ac.MaxRetries > 0 {
		a.maxRetries = ac.MaxRetries
	}

	if ac.DisableRetries {
		a.maxRetries = 0
	}

	a.minRetryDelay = minRetryWait
	if ac.MinRetryDelay != "" {
		mr, err := time.ParseDuration(ac.MinRetryDelay)
		if err != nil {
			a.Log.Printf("[ERR] min retry delay (%s): %s", ac.MinRetryDelay, err)
		}
		a.minRetryDelay = mr
	}

	a.maxRetryDelay = maxRetryWait
	if ac.MaxRetryDelay != "" {
		mr, err := time.ParseDuration(ac.MaxRetryDelay)
		if err != nil {
			a.Log.Printf("[ERR] max retry delay (%s): %s", ac.MaxRetryDelay, err)
		}
		a.maxRetryDelay = mr
	}

	return a, nil
}

// EnableExponentialBackoff enables use of exponential backoff for next API call(s)
// and use exponential backoff for all API calls until exponential backoff is disabled.
func (a *API) EnableExponentialBackoff() {
	a.useExponentialBackoffmu.Lock()
	a.useExponentialBackoff = true
	a.useExponentialBackoffmu.Unlock()
}

// DisableExponentialBackoff disables use of exponential backoff. If a request using
// exponential backoff is currently running, it will stop using exponential backoff
// on its next iteration (if needed).
func (a *API) DisableExponentialBackoff() {
	a.useExponentialBackoffmu.Lock()
	a.useExponentialBackoff = false
	a.useExponentialBackoffmu.Unlock()
}

// Get API request
func (a *API) Get(reqPath string) ([]byte, error) {
	return a.apiRequest("GET", reqPath, nil)
}

// Delete API request
func (a *API) Delete(reqPath string) ([]byte, error) {
	return a.apiRequest("DELETE", reqPath, nil)
}

// Post API request
func (a *API) Post(reqPath string, data []byte) ([]byte, error) {
	return a.apiRequest("POST", reqPath, data)
}

// Put API request
func (a *API) Put(reqPath string, data []byte) ([]byte, error) {
	return a.apiRequest("PUT", reqPath, data)
}

func backoff(interval uint) float64 {
	return math.Floor(((float64(interval) * (1 + rnd.Float64())) / 2) + .5) //nolint:gosec
}

// apiRequest manages retry strategy for exponential backoffs
func (a *API) apiRequest(reqMethod string, reqPath string, data []byte) ([]byte, error) {
	backoffs := []uint{2, 4, 8, 16, 32}
	attempts := 0
	success := false

	var result []byte
	var err error

	for !success {
		result, err = a.apiCall(reqMethod, reqPath, data)
		if err == nil {
			success = true
		}

		// break and return error if not using exponential backoff
		if err != nil {
			if !a.useExponentialBackoff {
				break
			}
			if strings.Contains(err.Error(), "code 400") {
				break
			}
			if strings.Contains(err.Error(), "code 403") {
				break
			}
			if strings.Contains(err.Error(), "code 404") {
				break
			}
		}

		if !success {
			var wait float64
			if attempts >= len(backoffs) {
				wait = backoff(backoffs[len(backoffs)-1])
			} else {
				wait = backoff(backoffs[attempts])
			}
			attempts++
			a.Log.Printf("Circonus API call failed %s, retrying in %d seconds.\n", err.Error(), uint(wait))
			time.Sleep(time.Duration(wait) * time.Second)
		}
	}

	return result, err
}

// apiCall call Circonus API
func (a *API) apiCall(reqMethod string, reqPath string, data []byte) ([]byte, error) {
	reqURL := a.apiURL.String()

	if reqPath == "" {
		return nil, errors.New("invalid Circonus API URL path (empty)")
	}
	if reqPath[:1] != "/" {
		reqURL += "/"
	}
	if len(reqPath) >= 3 && reqPath[:3] == "/v2" {
		reqURL += reqPath[3:]
	} else {
		reqURL += reqPath
	}

	// keep last HTTP error in the event of retry failure
	var lastHTTPError error
	retryPolicy := func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return false, errors.Wrap(ctxErr, "Circonus API call")
		}

		if err != nil {
			lastHTTPError = err
			return true, errors.Wrap(err, "Circonus API call")
		}
		// Check the response code. We retry on 500-range responses to allow
		// the server time to recover, as 500's are typically not permanent
		// errors and may relate to outages on the server side. This will catch
		// invalid response codes as well, like 0 and 999.
		// Retry on 429 (rate limit) as well.
		if resp.StatusCode == 0 || // wtf?!
			resp.StatusCode >= 500 || // rutroh
			resp.StatusCode == 429 { // rate limit
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				lastHTTPError = errors.Errorf("- response: %d %s", resp.StatusCode, readErr.Error())
			} else {
				lastHTTPError = errors.Errorf("- response: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
			}
			return true, nil
		}
		return false, nil
	}

	if len(data) > 0 {
		a.Log.Printf("[DEBUG] sending json (%s)\n", string(data))
	}

	dataReader := bytes.NewReader(data)

	req, err := retryablehttp.NewRequest(reqMethod, reqURL, dataReader)
	if err != nil {
		return nil, errors.Errorf("creating Circonus API request: %s %+v", reqURL, err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-Circonus-Auth-Token", string(a.key))
	req.Header.Add("X-Circonus-App-Name", string(a.app))
	if string(a.accountID) != "" {
		req.Header.Add("X-Circonus-Account-ID", string(a.accountID))
	}
	req.Header.Add("Cache-Control", "no-store")

	client := retryablehttp.NewClient()
	if a.apiURL.Scheme == "https" {
		var tlscfg *tls.Config
		if a.tlsConfig != nil { // preference full custom tls config
			tlscfg = a.tlsConfig
		} else if a.caCert != nil {
			tlscfg = &tls.Config{
				RootCAs:    a.caCert,
				MinVersion: tls.VersionTLS12,
			}
		}
		client.HTTPClient.Transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     tlscfg,
			DisableKeepAlives:   true,
			MaxIdleConnsPerHost: -1,
			DisableCompression:  true,
		}
	} else {
		client.HTTPClient.Transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableKeepAlives:   true,
			MaxIdleConnsPerHost: -1,
			DisableCompression:  true,
		}
	}

	a.useExponentialBackoffmu.Lock()
	eb := a.useExponentialBackoff
	a.useExponentialBackoffmu.Unlock()

	if eb {
		// limit to one request if using exponential backoff
		client.RetryWaitMin = 1 * time.Second
		client.RetryWaitMax = 60 * time.Second
		client.RetryMax = 0
	} else {
		client.RetryWaitMin = a.minRetryDelay
		client.RetryWaitMax = a.maxRetryDelay
		client.RetryMax = int(a.maxRetries)
	}

	// retryablehttp only groks log or no log
	if a.Debug {
		client.Logger = a.Log
	} else {
		client.Logger = log.New(io.Discard, "", log.LstdFlags)
	}

	client.CheckRetry = retryPolicy

	resp, err := client.Do(req)
	if err != nil {
		if lastHTTPError != nil {
			return nil, lastHTTPError
		}
		return nil, errors.Errorf("Circonus API call - %s: %+v", reqURL, err)
	}

	defer resp.Body.Close() // nolint: errcheck
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading Circonus API response")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := fmt.Sprintf("API response code %d: %s", resp.StatusCode, string(body))
		if a.Debug {
			a.Log.Printf("%s\n", msg)
		}

		return nil, errors.New(msg)
	}

	return body, nil
}
