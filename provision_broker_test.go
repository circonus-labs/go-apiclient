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
	testProvisionBroker = ProvisionBroker{
		Cert: "...",
		Stratcons: []BrokerStratcon{
			{CN: "foobar", Host: "foobar.example.com", Port: "12345"},
		},
		CSR:                     "...",
		ExternalHost:            "abc-123.example.com",
		ExternalPort:            "443",
		IPAddress:               "192.168.1.10",
		Latitude:                "",
		Longitude:               "",
		Name:                    "abc123",
		Port:                    "43191",
		PreferReverseConnection: true,
		Rebuild:                 false,
		Tags:                    []string{"cat:tag"},
	}
)

func testProvisionBrokerServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/provision_broker/abc-1234" {
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testProvisionBroker)
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
		} else if path == "/provision_broker" {
			switch r.Method {
			case "POST":
				defer r.Body.Close()
				_, err := ioutil.ReadAll(r.Body)
				if err != nil {
					panic(err)
				}
				ret, err := json.Marshal(testProvisionBroker)
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

func provisionBrokerTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testProvisionBrokerServer()

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

func TestNewProvisionBroker(t *testing.T) {
	provisionBroker := NewProvisionBroker()
	if reflect.TypeOf(provisionBroker).String() != "*apiclient.ProvisionBroker" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(provisionBroker).String())
	}
}

func TestFetchProvisionBroker(t *testing.T) {
	apih, server := provisionBrokerTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cid          string
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", true, "invalid provision broker CID (none)"},
		{"short cid", "abc-1234", "*apiclient.ProvisionBroker", false, ""},
		{"long cid", "/provision_broker/abc-1234", "*apiclient.ProvisionBroker", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchProvisionBroker(CIDType(&test.cid))
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

func TestUpdateProvisionBroker(t *testing.T) {
	apih, server := provisionBrokerTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cid          string
		cfg          *ProvisionBroker
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (cid)", "", nil, "", true, "invalid provision broker CID (none)"},
		{"invalid (cfg)", "abc", nil, "", true, "invalid provision broker config (nil)"},
		{"invalid (cid)", "/invalid", &ProvisionBroker{}, "", true, "invalid provision broker CID (/invalid)"},
		{"valid", "/provision_broker/abc-1234", &testProvisionBroker, "*apiclient.ProvisionBroker", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			maint, err := apih.UpdateProvisionBroker(CIDType(&test.cid), test.cfg)
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

func TestCreateProvisionBroker(t *testing.T) {
	apih, server := provisionBrokerTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cfg          *ProvisionBroker
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid provision broker config (nil)"},
		{"valid", &testProvisionBroker, "*apiclient.ProvisionBroker", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.CreateProvisionBroker(test.cfg)
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
