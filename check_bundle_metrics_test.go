// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

var (
	testCheckBundleMetrics = CheckBundleMetrics{
		CID: "/check_bundle_metrics/1234",
		Metrics: []CheckBundleMetric{
			{Name: "foo", Type: "numeric", Status: "active"},
			{Name: "bar", Type: "histogram", Status: "active"},
			{Name: "baz", Type: "text", Status: "available"},
			{Name: "fum", Type: "composite", Status: "active", Tags: []string{"cat:tag"}},
			{Name: "zot", Type: "caql", Status: "active", Units: &[]string{"milliseconds"}[0]},
		},
	}
)

func testCheckBundleMetricsServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/check_bundle_metrics/1234" {
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testCheckBundleMetrics)
				if err != nil {
					panic(err)
				}
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, string(ret))
			case "PUT":
				defer r.Body.Close()
				b, err := io.ReadAll(r.Body)
				if err != nil {
					panic(err)
				}
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, string(b))
			default:
				w.WriteHeader(404)
				fmt.Fprintf(w, "not found: %s %s\n", r.Method, path)
			}
		} else {
			w.WriteHeader(404)
			fmt.Fprintf(w, "not found: %s %s\n", r.Method, path)
		}
	}

	return httptest.NewServer(http.HandlerFunc(f))
}

func checkBundleMetricsTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testCheckBundleMetricsServer()

	ac := &Config{
		TokenKey: "abc123",
		TokenApp: "test",
		URL:      server.URL,
	}
	apih, err := NewAPI(ac)
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
		server.Close()
		return nil, nil
	}

	return apih, server
}

func TestFetchCheckBundleMetrics(t *testing.T) {
	apih, server := checkBundleMetricsTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cid          string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:           "empty cid",
			cid:          "",
			expectedType: "",
			shouldFail:   true,
			expectedErr:  "invalid check bundle metrics CID (none)",
		},
		{
			id:           "short cid",
			cid:          "1234",
			expectedType: "*apiclient.CheckBundleMetrics",
			shouldFail:   false,
		},
		{
			id:           "long cid",
			cid:          "/check_bundle_metrics/1234",
			expectedType: "*apiclient.CheckBundleMetrics",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchCheckBundleMetrics(CIDType(&test.cid))
			if test.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				} else if err.Error() != test.expectedErr {
					t.Fatalf("unexpected error (%s)", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error (%s)", err)
				} else if reflect.TypeOf(alert).String() != test.expectedType {
					t.Fatalf("unexpected type (%s)", reflect.TypeOf(alert).String())
				}
			}
		})
	}
}

func TestUpdateCheckBundleMetrics(t *testing.T) {
	apih, server := checkBundleMetricsTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg         *CheckBundleMetrics
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			cfg:         nil,
			shouldFail:  true,
			expectedErr: "invalid check bundle metrics config (nil)",
		},
		{
			id:          "invalid (cid)",
			cfg:         &CheckBundleMetrics{CID: "/invalid"},
			shouldFail:  true,
			expectedErr: "invalid check bundle metrics CID (/invalid)",
		},
		{
			id:         "valid",
			cfg:        &testCheckBundleMetrics,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			_, err := apih.UpdateCheckBundleMetrics(test.cfg)
			if test.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				} else if err.Error() != test.expectedErr {
					t.Fatalf("unexpected error (%s)", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error (%s)", err)
				}
			}
		})
	}
}
