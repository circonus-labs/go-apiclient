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
	testWorksheet = Worksheet{
		CID:         "/worksheet/01234567-89ab-cdef-0123-456789abcdef",
		Description: &[]string{"One graph per active server in our primary data center"}[0],
		Favorite:    true,
		Graphs: []WorksheetGraph{
			{GraphCID: "/graph/aaaaaaaa-0000-1111-2222-0123456789ab"},
			{GraphCID: "/graph/bbbbbbbb-3333-4444-5555-0123456789ab"},
			{GraphCID: "/graph/cccccccc-6666-7777-8888-0123456789ab"},
		},
		Notes: &[]string{"Currently maintained by Oscar"}[0],
		SmartQueries: []WorksheetSmartQuery{
			{
				Name:  "Virtual Machines",
				Order: []string{"/graph/dddddddd-9999-aaaa-bbbb-0123456789ab"},
				Query: "virtual",
			},
		},
		Tags:  []string{"datacenter:primary"},
		Title: "Primary Datacenter Server Graphs",
	}
)

func testWorksheetServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/worksheet/01234567-89ab-cdef-0123-456789abcdef":
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testWorksheet)
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
				fmt.Fprintf(w, "not found: %s %s\n", r.Method, path)
			}
		case "/worksheet":
			switch r.Method {
			case "GET":
				var c []Worksheet
				reqURL := r.URL.String()
				switch reqURL {
				case "/worksheet?search=web+servers":
					c = []Worksheet{testWorksheet}
				case "/worksheet?f_favorite=true":
					c = []Worksheet{testWorksheet}
				case "/worksheet?f_favorite=true&search=web+servers":
					c = []Worksheet{testWorksheet}
				case "/worksheet":
					c = []Worksheet{testWorksheet}
				default:
					c = []Worksheet{}
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
				_, err := ioutil.ReadAll(r.Body)
				if err != nil {
					panic(err)
				}
				ret, err := json.Marshal(testWorksheet)
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

func worksheetTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testWorksheetServer()

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

func TestNewWorksheet(t *testing.T) {
	worksheet := NewWorksheet()
	if reflect.TypeOf(worksheet).String() != "*apiclient.Worksheet" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(worksheet).String())
	}
}

func TestFetchWorksheet(t *testing.T) {
	apih, server := worksheetTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cid          string
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", true, "invalid worksheet CID (none)"},
		{"short cid", "01234567-89ab-cdef-0123-456789abcdef", "*apiclient.Worksheet", false, ""},
		{"long cid", "/worksheet/01234567-89ab-cdef-0123-456789abcdef", "*apiclient.Worksheet", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchWorksheet(CIDType(&test.cid))
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

func TestFetchWorksheets(t *testing.T) {
	apih, server := worksheetTestBootstrap(t)
	defer server.Close()

	worksheets, err := apih.FetchWorksheets()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(worksheets).String() != "*[]apiclient.Worksheet" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(worksheets).String())
	}

}

func TestUpdateWorksheet(t *testing.T) {
	apih, server := worksheetTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cfg          *Worksheet
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid worksheet config (nil)"},
		{"invalid (cid)", &Worksheet{CID: "/invalid"}, "", true, "invalid worksheet CID (/invalid)"},
		{"valid", &testWorksheet, "*apiclient.Worksheet", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			maint, err := apih.UpdateWorksheet(test.cfg)
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

func TestCreateWorksheet(t *testing.T) {
	apih, server := worksheetTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cfg          *Worksheet
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid worksheet config (nil)"},
		{"valid", &testWorksheet, "*apiclient.Worksheet", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.CreateWorksheet(test.cfg)
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

func TestDeleteWorksheet(t *testing.T) {
	apih, server := worksheetTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id          string
		cfg         *Worksheet
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid worksheet config (nil)"},
		{"valid", &testWorksheet, false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteWorksheet(test.cfg)
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

func TestDeleteWorksheetByCID(t *testing.T) {
	apih, server := worksheetTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id          string
		cid         string
		shouldFail  bool
		expectedErr string
	}{
		{"empty cid", "", true, "invalid worksheet CID (none)"},
		{"short cid", "01234567-89ab-cdef-0123-456789abcdef", false, ""},
		{"long cid", "/worksheet/01234567-89ab-cdef-0123-456789abcdef", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteWorksheetByCID(CIDType(&test.cid))
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

func TestSearchWorksheets(t *testing.T) {
	apih, server := worksheetTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.Worksheet"
	search := SearchQueryType("web servers")
	filter := SearchFilterType(map[string][]string{"f_favorite": {"true"}})

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
			ack, err := apih.SearchWorksheets(test.search, test.filter)
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
