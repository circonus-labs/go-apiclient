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
	"os"
	"reflect"
	"testing"
)

var (
	testFormula1 = "=A-B"
	testFormula2 = "=VAL/1000"
	testGraph    = Graph{
		CID:        "/graph/01234567-89ab-cdef-0123-456789abcdef",
		AccessKeys: []GraphAccessKey{},
		Composites: []GraphComposite{
			{
				Axis:        "l",
				Color:       "#000000",
				DataFormula: &testFormula1,
				Hidden:      false,
				Name:        "Time After First Byte",
			},
		},
		Datapoints: []GraphDatapoint{
			{
				Axis:        "l",
				CheckID:     1234,
				Color:       &[]string{"#ff0000"}[0],
				DataFormula: &testFormula2,
				Derive:      "gauge",
				Hidden:      false,
				MetricName:  "duration",
				MetricType:  "numeric",
				Name:        "Total Request Time",
			},
			{
				Axis:        "l",
				CheckID:     2345,
				Color:       &[]string{"#00ff00"}[0],
				DataFormula: &testFormula2,
				Derive:      "gauge",
				Hidden:      false,
				MetricName:  "tt_firstbyte",
				MetricType:  "numeric",
				Name:        "Time Till First Byte",
			},
		},
		Description: "Time to first byte verses time to whole thing",
		LineStyle:   &[]string{"interpolated"}[0],
		LogLeftY:    &[]int{10}[0],
		Notes:       &[]string{"This graph shows just the main webserver"}[0],
		Style:       &[]string{"line"}[0],
		Tags:        []string{"datacenter:primary"},
		Title:       "Slow Webserver",
	}
)

func testGraphServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/graph/01234567-89ab-cdef-0123-456789abcdef":
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testGraph)
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
			case "DELETE":
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
			default:
				w.WriteHeader(404)
				fmt.Fprintf(w, "not found: %s %s\n", r.Method, path)
			}
		case "/graph":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []Graph
				switch reqURL {
				case "/graph?search=CPU+Utilization":
					c = []Graph{testGraph}
				case "/graph?f__tags_has=os%3Arhel7":
					c = []Graph{testGraph}
				case "/graph?f__tags_has=os%3Arhel7&search=CPU+Utilization":
					c = []Graph{testGraph}
				case "/graph":
					c = []Graph{testGraph}
				default:
					c = []Graph{}
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
			case "POST":
				defer r.Body.Close()
				_, err := io.ReadAll(r.Body)
				if err != nil {
					panic(err)
				}
				ret, err := json.Marshal(testGraph)
				if err != nil {
					panic(err)
				}
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, string(ret))
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

func graphTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testGraphServer()

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

func TestNewGraph(t *testing.T) {
	graph := NewGraph()
	if reflect.TypeOf(graph).String() != "*apiclient.Graph" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(graph).String())
	}
}

func TestFetchGraph(t *testing.T) {
	apih, server := graphTestBootstrap(t)
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
			expectedErr:  "invalid graph CID (none)",
		},
		{
			id:           "short cid",
			cid:          "01234567-89ab-cdef-0123-456789abcdef",
			expectedType: "*apiclient.Graph",
			shouldFail:   false,
		},
		{
			id:           "long cid",
			cid:          "/graph/01234567-89ab-cdef-0123-456789abcdef",
			expectedType: "*apiclient.Graph",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchGraph(CIDType(&test.cid))
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

func TestFetchGraphs(t *testing.T) {
	apih, server := graphTestBootstrap(t)
	defer server.Close()

	graphs, err := apih.FetchGraphs()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(graphs).String() != "*[]apiclient.Graph" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(graphs).String())
	}
}

func TestUpdateGraph(t *testing.T) {
	apih, server := graphTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg         *Graph
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			cfg:         nil,
			shouldFail:  true,
			expectedErr: "invalid graph config (nil)",
		},
		{
			id:          "invalid (cid)",
			cfg:         &Graph{CID: "/invalid"},
			shouldFail:  true,
			expectedErr: "invalid graph CID (/invalid)",
		},
		{
			id:         "valid",
			cfg:        &testGraph,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			_, err := apih.UpdateGraph(test.cfg)
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

func TestCreateGraph(t *testing.T) {
	apih, server := graphTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg          *Graph
		id           string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:           "invalid (nil)",
			cfg:          nil,
			expectedType: "",
			shouldFail:   true,
			expectedErr:  "invalid graph config (nil)",
		},
		{
			id:           "valid",
			cfg:          &testGraph,
			expectedType: "*apiclient.Graph",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.CreateGraph(test.cfg)
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

func TestDeleteGraph(t *testing.T) {
	apih, server := graphTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg         *Graph
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			cfg:         nil,
			shouldFail:  true,
			expectedErr: "invalid graph config (nil)",
		},
		{
			id:         "valid",
			cfg:        &testGraph,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteGraph(test.cfg)
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

func TestDeleteGraphByCID(t *testing.T) {
	apih, server := graphTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id          string
		cid         string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "empty cid",
			cid:         "",
			shouldFail:  true,
			expectedErr: "invalid graph CID (none)",
		},
		{
			id:         "short cid",
			cid:        "01234567-89ab-cdef-0123-456789abcdef",
			shouldFail: false,
		},
		{
			id:         "long cid",
			cid:        "/graph/01234567-89ab-cdef-0123-456789abcdef",
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteGraphByCID(CIDType(&test.cid))
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

func TestSearchGraphs(t *testing.T) {
	apih, server := graphTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.Graph"
	search := SearchQueryType("CPU Utilization")
	filter := SearchFilterType(map[string][]string{"f__tags_has": {"os:rhel7"}})

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
			search:       nil,
			filter:       nil,
			expectedType: expectedType,
			shouldFail:   false,
		},
		{
			id:           "search no filter",
			search:       &search,
			filter:       nil,
			expectedType: expectedType,
			shouldFail:   false,
		},
		{
			id:           "filter no search",
			search:       nil,
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
			ack, err := apih.SearchGraphs(test.search, test.filter)
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

func TestGraphOverlaySet(t *testing.T) {
	t.Log("testing graph overlay set struct")

	testJSON, err := os.ReadFile("testdata/graph_overlayset.json")
	if err != nil {
		t.Fatal(err)
	}

	var g Graph
	if err = json.Unmarshal(testJSON, &g); err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}

	var g2 Graph
	if err := json.Unmarshal(data, &g2); err != nil {
		t.Fatal(err)
	}

}
