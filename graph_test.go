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
				b, err := ioutil.ReadAll(r.Body)
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
				fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
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
					fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, reqURL))
				}
			case "POST":
				defer r.Body.Close()
				_, err := ioutil.ReadAll(r.Body)
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
				fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
			}
		default:
			w.WriteHeader(404)
			fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
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
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", true, "invalid graph CID (none)"},
		{"short cid", "01234567-89ab-cdef-0123-456789abcdef", "*apiclient.Graph", false, ""},
		{"long cid", "/graph/01234567-89ab-cdef-0123-456789abcdef", "*apiclient.Graph", false, ""},
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
		id          string
		cfg         *Graph
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid graph config (nil)"},
		{"invalid (cid)", &Graph{CID: "/invalid"}, true, "invalid graph CID (/invalid)"},
		{"valid", &testGraph, false, ""},
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
		id           string
		cfg          *Graph
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid graph config (nil)"},
		{"valid", &testGraph, "*apiclient.Graph", false, ""},
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
		id          string
		cfg         *Graph
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid graph config (nil)"},
		{"valid", &testGraph, false, ""},
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
		shouldFail  bool
		expectedErr string
	}{
		{"empty cid", "", true, "invalid graph CID (none)"},
		{"short cid", "01234567-89ab-cdef-0123-456789abcdef", false, ""},
		{"long cid", "/graph/01234567-89ab-cdef-0123-456789abcdef", false, ""},
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

	testJSON, err := ioutil.ReadFile("testdata/graph_overlayset.json")
	if err != nil {
		t.Fatal(err)
	}

	var g Graph
	if err := json.Unmarshal(testJSON, &g); err != nil {
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
