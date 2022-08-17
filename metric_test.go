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
	testMetric = Metric{
		CID:            "/metric/1234_foo",
		Active:         true,
		CheckCID:       "/check/1234",
		CheckActive:    true,
		CheckBundleCID: "/check_bundle/1234",
		CheckTags:      []string{"cat:tag"},
		CheckUUID:      "",
		Histogram:      "false",
		MetricName:     "foo",
		MetricType:     "numeric",
		// Tags:           []string{"cat1:tag1"},
		// Units:          &[]string{"light years"}[0],
	}
)

func testMetricServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/metric/1234_foo":
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testMetric)
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
		case "/metric":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []Metric
				switch reqURL {
				case "/metric?search=vm%60memory%60used":
					c = []Metric{testMetric}
				case "/metric?f_tags_has=service%3Acache":
					c = []Metric{testMetric}
				case "/metric?f_tags_has=service%3Acache&search=vm%60memory%60used":
					c = []Metric{testMetric}
				case "/metric":
					c = []Metric{testMetric}
				default:
					c = []Metric{}
				}
				if len(c) > 0 {
					ret, err := json.Marshal(c)
					if err != nil {
						panic(err)
					}
					w.WriteHeader(200)
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintln(w, string(ret))
				} else {
					w.WriteHeader(404)
					fmt.Fprintf(w, "not found: %s %s\n", r.Method, reqURL)
				}
			default:
				w.WriteHeader(404)
				fmt.Fprintf(w, "not found: %s %s\n", r.Method, path)
			}
		default:
			w.WriteHeader(404)
			fmt.Fprintf(w, "not found: %s %s\n", r.Method, path)
		}
	}

	return httptest.NewServer(http.HandlerFunc(f))
}

func metricTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testMetricServer()

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

func TestFetchMetric(t *testing.T) {
	apih, server := metricTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cid          string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:          "empty cid",
			shouldFail:  true,
			expectedErr: "invalid metric CID (none)",
		},
		{
			id:           "short cid",
			cid:          "1234_foo",
			expectedType: "*apiclient.Metric",
			shouldFail:   false,
		},
		{
			id:           "long cid",
			cid:          "/metric/1234_foo",
			expectedType: "*apiclient.Metric",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchMetric(CIDType(&test.cid))
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

func TestFetchMetrics(t *testing.T) {
	apih, server := metricTestBootstrap(t)
	defer server.Close()

	metrics, err := apih.FetchMetrics()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(metrics).String() != "*[]apiclient.Metric" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(metrics).String())
	}
}

func TestUpdateMetric(t *testing.T) {
	apih, server := metricTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg          *Metric
		id           string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:          "invalid (nil)",
			shouldFail:  true,
			expectedErr: "invalid metric config (nil)",
		},
		{
			id:          "invalid (cid)",
			cfg:         &Metric{CID: "/invalid"},
			shouldFail:  true,
			expectedErr: "invalid metric CID (/invalid)",
		},
		{
			id:           "valid",
			cfg:          &testMetric,
			expectedType: "*apiclient.Metric",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			maint, err := apih.UpdateMetric(test.cfg)
			if test.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				} else if err.Error() != test.expectedErr {
					t.Fatalf("unexpected error (%s)", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error (%s)", err)
				} else if reflect.TypeOf(maint).String() != test.expectedType {
					t.Fatalf("unexpected type (%s)", reflect.TypeOf(maint).String())
				}
			}
		})
	}
}

func TestSearchMetrics(t *testing.T) {
	apih, server := metricTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.Metric"
	search := SearchQueryType("vm`memory`used")
	filter := SearchFilterType(map[string][]string{"f_tags_has": {"service:cache"}})

	tests := []struct {
		search       *SearchQueryType
		filter       *SearchFilterType
		id           string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:           "no search, no filter",
			expectedType: expectedType,
			shouldFail:   false,
		},
		{
			id:           "search no filter",
			search:       &search,
			expectedType: expectedType,
			shouldFail:   false,
		},
		{
			id:           "filter no search",
			filter:       &filter,
			expectedType: expectedType,
			shouldFail:   false,
		},
		{
			id:           "both filter and search",
			search:       &search,
			filter:       &filter,
			expectedType: expectedType,
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.SearchMetrics(test.search, test.filter)
			if test.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				} else if err.Error() != test.expectedErr {
					t.Fatalf("unexpected error (%s)", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error (%s)", err)
				} else if reflect.TypeOf(ack).String() != test.expectedType {
					t.Fatalf("unexpected type (%s)", reflect.TypeOf(ack).String())
				}
			}
		})
	}
}
