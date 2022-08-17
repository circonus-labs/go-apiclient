// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func callServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, r.Method)
	}

	return httptest.NewServer(http.HandlerFunc(f))
}

func sslCallServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, r.Method)
	}

	return httptest.NewTLSServer(http.HandlerFunc(f))
}

var (
	numReq = 0
	maxReq = 2
)

func retryCallServer() *httptest.Server {
	gets := 0
	puts := 0
	posts := 0
	deletes := 0
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/auth_error_token":
			w.WriteHeader(403)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"reference":"abc123","explanation":"The authentication token you supplied is invalid","server":"foo","tag":"bar","message":"The password doesn't match the right format.  Are you passing the app name as the password and the token as the password?","code":"Forbidden.BadToken"}`)
		case "/auth_error_app":
			w.WriteHeader(403)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"reference":"abc123","explanation":"There is a problem with the application string you are trying to access the API with","server":"foo","tag":"bar","message":"App 'foobar' not allowed","code":"Forbidden.BadApp"}`)
		case "/rate_limit":
			ok := false
			switch r.Method {
			case "GET":
				if gets > 0 {
					ok = true
				}
				gets++
			case "PUT":
				if puts > 0 {
					ok = true
				}
				puts++
			case "POST":
				if posts > 0 {
					ok = true
				}
				posts++
			case "DELETE":
				if deletes > 0 {
					ok = true
				}
				deletes++
			}
			if ok {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "ok")
				return
			}
			w.Header().Set("Retry-After", "11")
			w.WriteHeader(429)
			fmt.Fprintln(w, "rate limit")
		default:
			numReq++
			if numReq > maxReq {
				w.WriteHeader(200)
			} else {
				w.WriteHeader(500)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, "blah blah blah, error...")
		}
	}

	return httptest.NewServer(http.HandlerFunc(f))
}

func TestNew(t *testing.T) {

	tests := []struct {
		cfg        *Config
		id         string
		shouldFail bool
	}{
		{
			id:         "invalid config (nil)",
			cfg:        nil,
			shouldFail: true,
		},
		{
			id:         "invalid config (blank)",
			cfg:        &Config{},
			shouldFail: true,
		},
		{
			id: "token - default app/url",
			cfg: &Config{
				TokenKey: "foo",
			},
			shouldFail: false,
		},
		{
			id: "token,app - default url",
			cfg: &Config{
				TokenKey: "foo",
				TokenApp: "bar",
			},
			shouldFail: false,
		},
		{
			id: "token,app,acctid - default url",
			cfg: &Config{
				TokenKey:       "foo",
				TokenApp:       "bar",
				TokenAccountID: "0",
			},
			shouldFail: false,
		},
		{
			id: "token,app,url(host)",
			cfg: &Config{
				TokenKey: "foo",
				TokenApp: "bar",
				URL:      "foo.example.com",
			},
			shouldFail: false,
		},
		{
			id: "token,app,url(trailing /)",
			cfg: &Config{
				TokenKey: "foo",
				TokenApp: "bar",
				URL:      "foo.example.com/path/",
			},
			shouldFail: false,
		},
		{
			id: "token,app,url(w/o trailing /)",
			cfg: &Config{
				TokenKey: "foo",
				TokenApp: "bar",
				URL:      "foo.example.com/path",
			},
			shouldFail: false,
		},
		{
			id: "invalid (url)",
			cfg: &Config{
				TokenKey: "foo",
				TokenApp: "bar",
				URL:      `http://foo.example.com\path`,
			},
			shouldFail: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			t.Parallel()
			_, err := New(test.cfg)
			if test.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error (%s)", err)
				}
			}
		})
	}
}

func TestEnableExponentialBackoff(t *testing.T) {
	ac := &Config{
		TokenKey: "foo",
		TokenApp: "bar",
	}

	apih, err := NewAPI(ac)
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	apih.EnableExponentialBackoff()
	if !apih.useExponentialBackoff {
		t.Fatal("expected true (enabled)")
	}
}

func TestDisableExponentialBackoff(t *testing.T) {
	ac := &Config{
		TokenKey: "foo",
		TokenApp: "bar",
	}

	apih, err := NewAPI(ac)
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	apih.DisableExponentialBackoff()
	if apih.useExponentialBackoff {
		t.Fatal("expected false (disabled)")
	}
}

func TestApiCall(t *testing.T) {
	server := callServer()
	defer server.Close()

	ac := &Config{
		TokenKey:       "foo",
		TokenApp:       "bar",
		TokenAccountID: "0",
		URL:            server.URL,
	}

	apih, err := NewAPI(ac)
	if err != nil {
		t.Errorf("Expected no error, got '%+v'", err)
	}

	t.Log("invalid URL path")
	{
		_, err := apih.apiCall("GET", "", nil)
		expectedError := errors.New("invalid Circonus API URL path (empty)")
		if err == nil {
			t.Errorf("Expected error")
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected %+v go '%+v'", expectedError, err)
		}
	}

	t.Log("URL path fixup, prefix '/'")
	{
		call := "GET"
		resp, err := apih.apiCall(call, "nothing", nil)
		if err != nil {
			t.Errorf("Expected no error, got '%+v'", resp)
		}
		expected := fmt.Sprintf("%s\n", call)
		if string(resp) != expected {
			t.Errorf("Expected\n'%s'\ngot\n'%s'\n", expected, resp)
		}
	}

	t.Log("URL path fixup, remove '/v2' prefix")
	{
		call := "GET"
		resp, err := apih.apiCall(call, "/v2/nothing", nil)
		if err != nil {
			t.Errorf("Expected no error, got '%+v'", resp)
		}
		expected := fmt.Sprintf("%s\n", call)
		if string(resp) != expected {
			t.Errorf("Expected\n'%s'\ngot\n'%s'\n", expected, resp)
		}
	}

	calls := []string{"GET", "PUT", "POST", "DELETE"}
	for _, call := range calls {
		t.Logf("Testing %s call", call)
		resp, err := apih.apiCall(call, "/", nil)
		if err != nil {
			t.Errorf("Expected no error, got '%+v'", resp)
		}

		expected := fmt.Sprintf("%s\n", call)
		if string(resp) != expected {
			t.Errorf("Expected\n'%s'\ngot\n'%s'\n", expected, resp)
		}
	}
}

func TestSSLApiCall(t *testing.T) {
	server := sslCallServer()
	defer server.Close()

	t.Log("using TLSConfig")
	{
		c := server.Certificate()
		cp := x509.NewCertPool()
		cp.AddCert(c)

		ac := &Config{
			TokenKey:       "foo",
			TokenApp:       "bar",
			TokenAccountID: "0",
			TLSConfig:      &tls.Config{RootCAs: cp},
			URL:            server.URL,
		}

		apih, err := NewAPI(ac)
		if err != nil {
			t.Errorf("Expected no error, got '%+v'", err)
		}

		calls := []string{"GET", "PUT", "POST", "DELETE"}
		for _, call := range calls {
			t.Logf("Testing %s call", call)
			resp, err := apih.apiCall(call, "/", nil)
			if err != nil {
				t.Errorf("Expected no error, got '%+v'", resp)
			}

			expected := fmt.Sprintf("%s\n", call)
			if string(resp) != expected {
				t.Errorf("Expected\n'%s'\ngot\n'%s'\n", expected, resp)
			}
		}
	}

	t.Log("using CACert - deprecated, use TLSConfig")
	{
		c := server.Certificate()
		cp := x509.NewCertPool()
		cp.AddCert(c)

		ac := &Config{
			TokenKey:       "foo",
			TokenApp:       "bar",
			TokenAccountID: "0",
			CACert:         cp,
			URL:            server.URL,
		}

		apih, err := NewAPI(ac)
		if err != nil {
			t.Errorf("Expected no error, got '%+v'", err)
		}

		calls := []string{"GET", "PUT", "POST", "DELETE"}
		for _, call := range calls {
			t.Logf("Testing %s call", call)
			resp, err := apih.apiCall(call, "/", nil)
			if err != nil {
				t.Errorf("Expected no error, got '%+v'", resp)
			}

			expected := fmt.Sprintf("%s\n", call)
			if string(resp) != expected {
				t.Errorf("Expected\n'%s'\ngot\n'%s'\n", expected, resp)
			}
		}
	}
}

func TestApiGet(t *testing.T) {
	server := callServer()
	defer server.Close()

	ac := &Config{
		TokenKey: "foo",
		TokenApp: "bar",
		URL:      server.URL,
	}

	client, err := NewClient(ac)
	if err != nil {
		t.Errorf("Expected no error, got '%+v'", err)
	}

	resp, err := client.Get("/")

	if err != nil {
		t.Errorf("Expected no error, got '%+v'", resp)
	}

	expected := "GET\n"
	if string(resp) != expected {
		t.Errorf("Expected\n'%s'\ngot\n'%s'\n", expected, resp)
	}

}

func TestApiPut(t *testing.T) {
	server := callServer()
	defer server.Close()

	ac := &Config{
		TokenKey: "foo",
		TokenApp: "bar",
		URL:      server.URL,
	}

	client, err := NewClient(ac)
	if err != nil {
		t.Errorf("Expected no error, got '%+v'", err)
	}

	resp, err := client.Put("/", nil)

	if err != nil {
		t.Errorf("Expected no error, got '%+v'", resp)
	}

	expected := "PUT\n"
	if string(resp) != expected {
		t.Errorf("Expected\n'%s'\ngot\n'%s'\n", expected, resp)
	}

}

func TestApiPost(t *testing.T) {
	server := callServer()
	defer server.Close()

	ac := &Config{
		TokenKey: "foo",
		TokenApp: "bar",
		URL:      server.URL,
	}

	client, err := NewClient(ac)
	if err != nil {
		t.Errorf("Expected no error, got '%+v'", err)
	}

	resp, err := client.Post("/", nil)

	if err != nil {
		t.Errorf("Expected no error, got '%+v'", resp)
	}

	expected := "POST\n"
	if string(resp) != expected {
		t.Errorf("Expected\n'%s'\ngot\n'%s'\n", expected, resp)
	}

}

func TestApiDelete(t *testing.T) {
	server := callServer()
	defer server.Close()

	ac := &Config{
		TokenKey: "foo",
		TokenApp: "bar",
		URL:      server.URL,
	}

	client, err := NewClient(ac)
	if err != nil {
		t.Errorf("Expected no error, got '%+v'", err)
	}

	resp, err := client.Delete("/")

	if err != nil {
		t.Errorf("Expected no error, got '%+v'", resp)
	}

	expected := "DELETE\n"
	if string(resp) != expected {
		t.Errorf("Expected\n'%s'\ngot\n'%s'\n", expected, resp)
	}

}

func TestApiRequest(t *testing.T) {
	if os.Getenv("DO_RETRY_TESTS") == "" {
		t.Skip("Skipping retry tests - DO_RETRY_TESTS environment var not set")
	}

	server := retryCallServer()
	defer server.Close()

	ac := &Config{
		TokenKey: "foo",
		TokenApp: "bar",
		URL:      server.URL,
	}

	apih, err := NewAPI(ac)
	if err != nil {
		t.Errorf("Expected no error, got '%+v'", err)
	}

	t.Log("Testing api request retries, this may take a few...")

	apih.DisableExponentialBackoff()

	t.Log("drift retry")
	{
		calls := []string{"GET", "PUT", "POST", "DELETE"}
		for _, call := range calls {
			t.Logf("\tTesting %d %s call(s)", maxReq, call)
			numReq = 0
			start := time.Now()
			resp, err := apih.apiRequest(call, "/", nil)
			if err != nil {
				t.Errorf("Expected no error, got '%+v'", resp)
			}
			elapsed := time.Since(start)
			t.Log("\tTime: ", elapsed)

			expected := "blah blah blah, error...\n"
			if string(resp) != expected {
				t.Errorf("Expected\n'%s'\ngot\n'%s'\n", expected, resp)
			}
		}
	}

	apih.EnableExponentialBackoff()

	t.Log("exponential backoff")
	{
		calls := []string{"GET", "PUT", "POST", "DELETE"}
		for _, call := range calls {
			t.Logf("\tTesting %d %s call(s)", maxReq, call)
			numReq = 0
			start := time.Now()
			resp, err := apih.apiRequest(call, "/", nil)
			if err != nil {
				t.Errorf("Expected no error, got '%+v'", resp)
			}
			elapsed := time.Since(start)
			t.Log("\tTime: ", elapsed)

			expected := "blah blah blah, error...\n"
			if string(resp) != expected {
				t.Errorf("Expected\n'%s'\ngot\n'%s'\n", expected, resp)
			}
		}
	}

	apih.DisableExponentialBackoff()

	t.Log("rate limit")
	{
		calls := []string{"GET", "PUT", "POST", "DELETE"}
		for _, call := range calls {
			t.Logf("\tTesting %d %s call(s)", 1, call)
			numReq = 0
			start := time.Now()
			resp, err := apih.apiRequest(call, "/rate_limit", nil)
			if err != nil {
				t.Errorf("Expected no error, got '%+v'", resp)
			}
			elapsed := time.Since(start)
			t.Log("\tTime: ", elapsed)

			expected := "ok"
			if string(resp) != expected {
				t.Errorf("Expected\n'%s'\ngot\n'%s'\n", expected, resp)
			}
		}
	}

	t.Log("drift retry - bad token")
	{
		_, err := apih.apiRequest("GET", "/auth_error_token", nil)
		if err == nil {
			t.Fatal("expected error")
		}
	}

	t.Log("drift retry - bad app")
	{
		_, err := apih.apiRequest("GET", "/auth_error_app", nil)
		if err == nil {
			t.Fatal("expected error")
		}
	}

	apih.EnableExponentialBackoff()

	t.Log("exponential backoff - bad token")
	{
		_, err := apih.apiRequest("GET", "/auth_error_token", nil)
		if err == nil {
			t.Fatal("expected error")
		}
	}

	t.Log("exponential backoff - bad app")
	{
		_, err := apih.apiRequest("GET", "/auth_error_app", nil)
		if err == nil {
			t.Fatal("expected error")
		}
	}

}
