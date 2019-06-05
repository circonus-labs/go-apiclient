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
				b, err := ioutil.ReadAll(r.Body)
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
				fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, r.URL.Path))
			}
		case "/metric_cluster":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []MetricCluster
				if reqURL == "/metric_cluster?search=web+servers" {
					c = []MetricCluster{testMetricCluster}
				} else if reqURL == "/metric_cluster?f_tags_has=dc%3Asfo1" {
					c = []MetricCluster{testMetricCluster}
				} else if reqURL == "/metric_cluster?f_tags_has=dc%3Asfo1&search=web+servers" {
					c = []MetricCluster{testMetricCluster}
				} else if reqURL == "/metric_cluster" {
					c = []MetricCluster{testMetricCluster}
				} else if reqURL == "/metric_cluster?extra=_matching_metrics" {
					c = []MetricCluster{testMetricCluster}
				} else if reqURL == "/metric_cluster?extra=_matching_uuid_metrics" {
					c = []MetricCluster{testMetricCluster}
				} else {
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
					fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, r.URL.Path))
				}
			case "POST": // create
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
				fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, r.URL.Path))
			}
		default:
			w.WriteHeader(404)
			fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, r.URL.Path))
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
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", "", true, "invalid metric cluster CID (none)"},
		{"short cid", "1234", "", "*apiclient.MetricCluster", false, ""},
		{"long cid", "/metric_cluster/1234", "", "*apiclient.MetricCluster", false, ""},
		{"cid xtra/metrics", "/metric_cluster/1234", "metrics", "*apiclient.MetricCluster", false, ""},
		{"cid xtra/uuids", "/metric_cluster/1234", "uuids", "*apiclient.MetricCluster", false, ""},
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
		shouldFail   bool
		expectedErr  string
	}{
		{"no extras", "", "*[]apiclient.MetricCluster", false, ""},
		{"xtra/metrics", "metrics", "*[]apiclient.MetricCluster", false, ""},
		{"xtra/uuids", "uuids", "*[]apiclient.MetricCluster", false, ""},
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
		id           string
		cfg          *MetricCluster
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid metric cluster config (nil)"},
		{"invalid (cid)", &MetricCluster{CID: "/invalid"}, "", true, "invalid metric cluster CID (/invalid)"},
		{"valid", &testMetricCluster, "*apiclient.MetricCluster", false, ""},
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
		id           string
		cfg          *MetricCluster
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid metric cluster config (nil)"},
		{"valid", &testMetricCluster, "*apiclient.MetricCluster", false, ""},
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
		id          string
		cfg         *MetricCluster
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid metric cluster config (nil)"},
		{"valid", &testMetricCluster, false, ""},
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
		shouldFail  bool
		expectedErr string
	}{
		{"empty cid", "", true, "invalid metric cluster CID (none)"},
		{"short cid", "1234", false, ""},
		{"long cid", "/metric_cluster/1234", false, ""},
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
