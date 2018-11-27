// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
		Tags:           []string{"cat1:tag1"},
		Units:          &[]string{"light years"}[0],
	}
)

func testMetricServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/metric/1234_foo" {
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
				b, err := ioutil.ReadAll(r.Body)
				if err != nil {
					panic(err)
				}
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, string(b))
			default:
				w.WriteHeader(404)
				fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
			}
		} else if path == "/metric" {
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []Metric
				if reqURL == "/metric?search=vm%60memory%60used" {
					c = []Metric{testMetric}
				} else if reqURL == "/metric?f_tags_has=service%3Acache" {
					c = []Metric{testMetric}
				} else if reqURL == "/metric?f_tags_has=service%3Acache&search=vm%60memory%60used" {
					c = []Metric{testMetric}
				} else if reqURL == "/metric" {
					c = []Metric{testMetric}
				} else {
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
					fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, reqURL))
				}
			default:
				w.WriteHeader(404)
				fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
			}
		} else {
			w.WriteHeader(404)
			fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
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
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", true, "invalid metric CID (none)"},
		{"invalid cid", "/invalid", "", true, "invalid metric CID (/metric//invalid)"},
		{"short cid", "1234_foo", "*apiclient.Metric", false, ""},
		{"long cid", "/metric/1234_foo", "*apiclient.Metric", false, ""},
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
		id           string
		cfg          *Metric
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid metric config (nil)"},
		{"invalid (cid)", &Metric{CID: "/invalid"}, "", true, "invalid metric CID (/invalid)"},
		{"valid", &testMetric, "*apiclient.Metric", false, ""},
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
		id           string
		search       *SearchQueryType
		filter       *SearchFilterType
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"no search, no filter", nil, nil, expectedType, false, ""},
		{"search no filter", &search, nil, expectedType, false, ""},
		{"filter no search", nil, &filter, expectedType, false, ""},
		{"both filter and search", &search, &filter, expectedType, false, ""},
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
