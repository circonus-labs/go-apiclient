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
	testMetricCluster = MetricCluster{
		Name: "test",
		CID:  "/metric_cluster/1234",
		Queries: []MetricQuery{
			{
				Query: "*Req*",
				Type:  "average",
			},
		},
		Description: "",
		Tags:        []string{},
	}
)

func testMetricClusterServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metric_cluster/1234": // handle GET/PUT/DELETE
			switch r.Method {
			case "PUT": // update
				defer r.Body.Close()
				b, err := io.ReadAll(r.Body)
				if err != nil {
					panic(err)
				}
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, string(b))
			case "GET": // get by id/cid
				ret, err := json.Marshal(testMetricCluster)
				if err != nil {
					panic(err)
				}
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, string(ret))
			case "DELETE": // delete
				w.WriteHeader(200)
				fmt.Fprintln(w, "")
			default:
				w.WriteHeader(404)
				fmt.Fprintf(w, "not found: %s %s\n", r.Method, r.URL.Path)
			}
		case "/metric_cluster":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []MetricCluster
				switch reqURL {
				case "/metric_cluster?search=web+servers":
					c = []MetricCluster{testMetricCluster}
				case "/metric_cluster?f_tags_has=dc%3Asfo1":
					c = []MetricCluster{testMetricCluster}
				case "/metric_cluster?f_tags_has=dc%3Asfo1&search=web+servers":
					c = []MetricCluster{testMetricCluster}
				case "/metric_cluster":
					c = []MetricCluster{testMetricCluster}
				case "/metric_cluster?extra=_matching_metrics":
					c = []MetricCluster{testMetricCluster}
				case "/metric_cluster?extra=_matching_uuid_metrics":
					c = []MetricCluster{testMetricCluster}
				default:
					c = []MetricCluster{}
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
					fmt.Fprintf(w, "not found: %s %s\n", r.Method, r.URL.Path)
				}
			case "POST": // create
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
				fmt.Fprintf(w, "not found: %s %s\n", r.Method, r.URL.Path)
			}
		default:
			w.WriteHeader(404)
			fmt.Fprintf(w, "not found: %s %s\n", r.Method, r.URL.Path)
		}
	}

	return httptest.NewServer(http.HandlerFunc(f))
}

func metricClusterTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testMetricClusterServer()

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

func TestNewMetricCluster(t *testing.T) {
	metricCluster := NewMetricCluster()
	if reflect.TypeOf(metricCluster).String() != "*apiclient.MetricCluster" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(metricCluster).String())
	}
}

func TestFetchMetricCluster(t *testing.T) {
	apih, server := metricClusterTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cid          string
		extras       string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:          "empty cid",
			shouldFail:  true,
			expectedErr: "invalid metric cluster CID (none)",
		},
		{
			id:           "short cid",
			cid:          "1234",
			expectedType: "*apiclient.MetricCluster",
			shouldFail:   false,
		},
		{
			id:           "long cid",
			cid:          "/metric_cluster/1234",
			expectedType: "*apiclient.MetricCluster",
			shouldFail:   false,
		},
		{
			id:           "cid xtra/metrics",
			cid:          "/metric_cluster/1234",
			extras:       "metrics",
			expectedType: "*apiclient.MetricCluster",
			shouldFail:   false,
		},
		{
			id:           "cid xtra/uuids",
			cid:          "/metric_cluster/1234",
			extras:       "uuids",
			expectedType: "*apiclient.MetricCluster",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchMetricCluster(CIDType(&test.cid), test.extras)
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

func TestFetchMetricClusters(t *testing.T) {
	apih, server := metricClusterTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		extras       string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:           "no extras",
			expectedType: "*[]apiclient.MetricCluster",
			shouldFail:   false,
		},
		{
			id:           "xtra/metrics",
			extras:       "metrics",
			expectedType: "*[]apiclient.MetricCluster",
			shouldFail:   false,
		},
		{
			id:           "xtra/uuids",
			extras:       "uuids",
			expectedType: "*[]apiclient.MetricCluster",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchMetricClusters(test.extras)
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

func TestUpdateMetricCluster(t *testing.T) {
	apih, server := metricClusterTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg          *MetricCluster
		id           string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:          "invalid (nil)",
			shouldFail:  true,
			expectedErr: "invalid metric cluster config (nil)",
		},
		{
			id:          "invalid (cid)",
			cfg:         &MetricCluster{CID: "/invalid"},
			shouldFail:  true,
			expectedErr: "invalid metric cluster CID (/invalid)",
		},
		{
			id:           "valid",
			cfg:          &testMetricCluster,
			expectedType: "*apiclient.MetricCluster",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			maint, err := apih.UpdateMetricCluster(test.cfg)
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

func TestCreateMetricCluster(t *testing.T) {
	apih, server := metricClusterTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg          *MetricCluster
		id           string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:          "invalid (nil)",
			shouldFail:  true,
			expectedErr: "invalid metric cluster config (nil)",
		},
		{
			id:           "valid",
			cfg:          &testMetricCluster,
			expectedType: "*apiclient.MetricCluster",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.CreateMetricCluster(test.cfg)
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

func TestDeleteMetricCluster(t *testing.T) {
	apih, server := metricClusterTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg         *MetricCluster
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			shouldFail:  true,
			expectedErr: "invalid metric cluster config (nil)",
		},
		{
			id:         "valid",
			cfg:        &testMetricCluster,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteMetricCluster(test.cfg)
			if test.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				} else if err.Error() != test.expectedErr {
					t.Fatalf("unexpected error (%s)", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error (%s)", err)
				} else if !wasDeleted {
					t.Fatal("expected true (deleted)")
				}
			}
		})
	}
}

func TestDeleteMetricClusterByCID(t *testing.T) {
	apih, server := metricClusterTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id          string
		cid         string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "empty cid",
			shouldFail:  true,
			expectedErr: "invalid metric cluster CID (none)",
		},
		{
			id:         "short cid",
			cid:        "1234",
			shouldFail: false,
		},
		{
			id:         "long cid",
			cid:        "/metric_cluster/1234",
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteMetricClusterByCID(CIDType(&test.cid))
			if test.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				} else if err.Error() != test.expectedErr {
					t.Fatalf("unexpected error (%s)", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error (%s)", err)
				} else if !wasDeleted {
					t.Fatal("expected true (deleted)")
				}
			}
		})
	}
}

func TestSearchMetricClusters(t *testing.T) {
	apih, server := metricClusterTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.MetricCluster"
	search := SearchQueryType("web servers")
	filter := SearchFilterType(map[string][]string{"f_tags_has": {"dc:sfo1"}})

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
			ack, err := apih.SearchMetricClusters(test.search, test.filter)
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
