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
	testAcknowledgement = Acknowledgement{
		CID:               "/acknowledgement/1234",
		AcknowledgedBy:    "/user/1234",
		AcknowledgedOn:    1483033102,
		Active:            true,
		LastModified:      1483033102,
		LastModifiedBy:    "/user/1234",
		AcknowledgedUntil: "1d",
		Notes:             "blah blah blah",
	}
)

func testAcknowledgementServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/acknowledgement/1234":
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testAcknowledgement)
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
				fmt.Fprintln(w, "not found: "+r.Method+" "+path)
			}
		case "/acknowledgement":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []Acknowledgement
				switch r.URL.String() {
				case "/acknowledgement?search=%28notes%3D%22something%22%29":
					c = []Acknowledgement{testAcknowledgement}
				case "/acknowledgement?f__active=true":
					c = []Acknowledgement{testAcknowledgement}
				case "/acknowledgement?f__active=true&search=%28notes%3D%22something%22%29":
					c = []Acknowledgement{testAcknowledgement}
				case "/acknowledgement":
					c = []Acknowledgement{testAcknowledgement}
				default:
					c = []Acknowledgement{}
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
				ret, err := json.Marshal(testAcknowledgement)
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

func acknowledgementTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testAcknowledgementServer()

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

func TestNewAcknowledgement(t *testing.T) {
	ack := NewAcknowledgement()
	if reflect.TypeOf(ack).String() != "*apiclient.Acknowledgement" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(ack).String())
	}
}

func TestFetchAcknowledgement(t *testing.T) {
	apih, server := acknowledgementTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cid          string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:           "invalid (empty cid)",
			cid:          "",
			expectedType: "",
			shouldFail:   true,
			expectedErr:  "invalid acknowledgement CID (none)",
		},
		{
			id:           "valid (short cid)",
			cid:          "1234",
			expectedType: "*apiclient.Acknowledgement",
			shouldFail:   false,
		},
		{
			id:           "valid (long cid)",
			cid:          "/acknowledgement/1234",
			expectedType: "*apiclient.Acknowledgement",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.FetchAcknowledgement(CIDType(&test.cid))
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

func TestFetchAcknowledgements(t *testing.T) {
	apih, server := acknowledgementTestBootstrap(t)
	defer server.Close()

	acks, err := apih.FetchAcknowledgements()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(acks).String() != "*[]apiclient.Acknowledgement" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(acks).String())
	}
}

func TestUpdateAcknowledgement(t *testing.T) {
	apih, server := acknowledgementTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg         *Acknowledgement
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			cfg:         nil,
			shouldFail:  true,
			expectedErr: "invalid acknowledgement config (nil)",
		},
		{
			id:          "invalid (cid)",
			cfg:         &Acknowledgement{CID: "/invalid"},
			shouldFail:  true,
			expectedErr: "invalid acknowledgement CID (/invalid)",
		},
		{
			id:         "valid",
			cfg:        &testAcknowledgement,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			_, err := apih.UpdateAcknowledgement(test.cfg)
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

func TestCreateAcknowledgement(t *testing.T) {
	apih, server := acknowledgementTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg          *Acknowledgement
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
			expectedErr:  "invalid acknowledgement config (nil)",
		},
		{
			id:           "invalid (cid)",
			cfg:          &Acknowledgement{CID: "/invalid"},
			expectedType: "",
			shouldFail:   true,
			expectedErr:  "invalid acknowledgement CID (/invalid)",
		},
		{
			id:           "valid",
			cfg:          &testAcknowledgement,
			expectedType: "*apiclient.Acknowledgement",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.UpdateAcknowledgement(test.cfg)
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

func TestSearchAcknowledgement(t *testing.T) {
	apih, server := acknowledgementTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.Acknowledgement"
	search := SearchQueryType(`(notes="something")`)
	filter := SearchFilterType(map[string][]string{"f__active": {"true"}})

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
			ack, err := apih.SearchAcknowledgements(test.search, test.filter)
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
