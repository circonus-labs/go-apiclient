// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/circonus-labs/go-apiclient/config"
)

var (
	testCheck = Check{
		CID:            "/check/1234",
		Active:         true,
		BrokerCID:      "/broker/1234",
		CheckBundleCID: "/check_bundle/1234",
		CheckUUID:      "abc123-a1b2-c3d4-e5f6-123abc",
		Details:        map[config.Key]string{config.SubmissionURL: "http://example.com/"},
	}
)

func testCheckServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/check/1234":
			switch r.Method {
			case "GET": // get by id/cid
				ret, err := json.Marshal(testCheck)
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
		case "/check":
			switch r.Method {
			case "GET": // search or filter
				reqURL := r.URL.String()
				var c []Check
				switch reqURL {

				case "/check?search=test":
					c = []Check{testCheck}
				case "/check?f__tags_has=cat%3Atag":
					c = []Check{testCheck}
				case "/check?f__tags_has=cat%3Atag&search=test":
					c = []Check{testCheck}
				case "/check":
					c = []Check{testCheck}
				default:
					c = []Check{}
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

func checkTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testCheckServer()

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

func TestFetchCheck(t *testing.T) {
	apih, server := checkTestBootstrap(t)
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
			expectedErr:  "invalid check CID (none)",
		},
		{
			id:           "short cid",
			cid:          "1234",
			expectedType: "*apiclient.Check",
			shouldFail:   false,
		},
		{
			id:           "long cid",
			cid:          "/check/1234",
			expectedType: "*apiclient.Check",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchCheck(CIDType(&test.cid))
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

func TestFetchChecks(t *testing.T) {
	apih, server := checkTestBootstrap(t)
	defer server.Close()

	checks, err := apih.FetchChecks()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(checks).String() != "*[]apiclient.Check" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(checks).String())
	}
}

func TestSearchChecks(t *testing.T) {
	apih, server := checkTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.Check"
	search := SearchQueryType("test")
	filter := SearchFilterType{"f__tags_has": []string{"cat:tag"}}

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
			ack, err := apih.SearchChecks(test.search, test.filter)
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
