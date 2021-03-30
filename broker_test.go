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
)

var (
	testBroker = Broker{
		CID:       "/broker/1234",
		Longitude: nil,
		Latitude:  nil,
		Name:      "test broker",
		Tags:      []string{},
		Type:      "enterprise",
		Details: []BrokerDetail{
			{
				CN:           "testbroker.example.com",
				ExternalHost: &[]string{"testbroker.example.com"}[0],
				ExternalPort: 43191,
				IP:           &[]string{"127.0.0.1"}[0],
				MinVer:       0,
				Modules:      []string{"a", "b", "c"},
				Port:         &[]uint16{43191}[0],
				Skew:         nil,
				Status:       "active",
				Version:      nil,
			},
		},
	}
)

func testBrokerServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/broker/1234":
			switch r.Method {
			case "GET": // get by id/cid
				ret, err := json.Marshal(testBroker)
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
		case "/broker":
			switch r.Method {
			case "GET": // search or filter
				reqURL := r.URL.String()
				var c []Broker
				switch r.URL.String() {
				case "/broker?search=httptrap":
					c = []Broker{testBroker}
				case "/broker?f__type=enterprise":
					c = []Broker{testBroker}
				case "/broker?f__type=enterprise&search=httptrap":
					c = []Broker{testBroker}
				case "/broker":
					c = []Broker{testBroker}
				default:
					c = []Broker{}
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

func brokerTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testBrokerServer()

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

func TestFetchBroker(t *testing.T) {
	apih, server := brokerTestBootstrap(t)
	defer server.Close()

	tests := []struct { //nolint:govet
		id           string
		cid          string
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", true, "invalid broker CID (none)"},
		{"short cid", "1234", "*apiclient.Broker", false, ""},
		{"long cid", "/broker/1234", "*apiclient.Broker", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchBroker(CIDType(&test.cid))
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

func TestFetchBrokers(t *testing.T) {
	apih, server := brokerTestBootstrap(t)
	defer server.Close()

	brokers, err := apih.FetchBrokers()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(brokers).String() != "*[]apiclient.Broker" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(brokers).String())
	}
}

func TestSearchBrokers(t *testing.T) {
	apih, server := brokerTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.Broker"
	search := SearchQueryType("httptrap")
	filter := SearchFilterType{"f__type": []string{"enterprise"}}

	tests := []struct { //nolint:govet
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
			ack, err := apih.SearchBrokers(test.search, test.filter)
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
