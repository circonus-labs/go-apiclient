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
		if path == "/acknowledgement/1234" {
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
		} else if path == "/acknowledgement" {
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []Acknowledgement
				if r.URL.String() == "/acknowledgement?search=%28notes%3D%22something%22%29" {
					c = []Acknowledgement{testAcknowledgement}
				} else if r.URL.String() == "/acknowledgement?f__active=true" {
					c = []Acknowledgement{testAcknowledgement}
				} else if r.URL.String() == "/acknowledgement?f__active=true&search=%28notes%3D%22something%22%29" {
					c = []Acknowledgement{testAcknowledgement}
				} else if reqURL == "/acknowledgement" {
					c = []Acknowledgement{testAcknowledgement}
				} else {
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
					fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, reqURL))
				}
			case "POST":
				defer r.Body.Close()
				_, err := ioutil.ReadAll(r.Body)
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
				fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
			}
		} else {
			w.WriteHeader(404)
			fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
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
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (empty cid)", "", "", true, "invalid acknowledgement CID (none)"},
		{"invalid (cid)", "/invalid", "", true, "invalid acknowledgement CID (/acknowledgement//invalid)"},
		{"valid (short cid)", "1234", "*apiclient.Acknowledgement", false, ""},
		{"valid (long cid)", "/acknowledgement/1234", "*apiclient.Acknowledgement", false, ""},
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
		id          string
		cfg         *Acknowledgement
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid acknowledgement config (nil)"},
		{"invalid (cid)", &Acknowledgement{CID: "/invalid"}, true, "invalid acknowledgement CID (/invalid)"},
		{"valid", &testAcknowledgement, false, ""},
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
		id           string
		cfg          *Acknowledgement
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid acknowledgement config (nil)"},
		{"invalid (cid)", &Acknowledgement{CID: "/invalid"}, "", true, "invalid acknowledgement CID (/invalid)"},
		{"valid", &testAcknowledgement, "*apiclient.Acknowledgement", false, ""},
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
