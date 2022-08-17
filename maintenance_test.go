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
	testMaintenance = Maintenance{
		CID:        "/maintenance/1234",
		Item:       "/check/1234",
		Notes:      "upgrading blah",
		Severities: []string{"1", "2", "3", "4", "5"},
		Start:      1483033100,
		Stop:       1483033102,
		Type:       "check",
		Tags:       []string{"cat:tag"},
	}
)

func testMaintenanceServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/maintenance/1234":
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testMaintenance)
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
		case "/maintenance":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []Maintenance
				switch reqURL {
				case "/maintenance?search=%2Fcheck_bundle%2F1234":
					c = []Maintenance{testMaintenance}
				case "/maintenance?f_start_gt=1483639916":
					c = []Maintenance{testMaintenance}
				case "/maintenance?f_start_gt=1483639916&search=%2Fcheck_bundle%2F1234":
					c = []Maintenance{testMaintenance}
				case "/maintenance":
					c = []Maintenance{testMaintenance}
				default:
					c = []Maintenance{}
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
				ret, err := json.Marshal(testMaintenance)
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

func maintenanceTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testMaintenanceServer()

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

func TestNewMaintenanceWindow(t *testing.T) {
	maintenance := NewMaintenanceWindow()
	if reflect.TypeOf(maintenance).String() != "*apiclient.Maintenance" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(maintenance).String())
	}
}

func TestFetchMaintenanceWindow(t *testing.T) {
	apih, server := maintenanceTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cid          string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:          "empty cid",
			shouldFail:  true,
			expectedErr: "invalid maintenance window CID (none)",
		},
		{
			id:           "short cid",
			cid:          "1234",
			expectedType: "*apiclient.Maintenance",
			shouldFail:   false,
		},
		{
			id:          "long cid",
			cid:         "/maintenance/1234",
			expectedErr: "*apiclient.Maintenance",
			shouldFail:  false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchMaintenanceWindow(CIDType(&test.cid))
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

func TestFetchMaintenanceWindows(t *testing.T) {
	apih, server := maintenanceTestBootstrap(t)
	defer server.Close()

	maintenances, err := apih.FetchMaintenanceWindows()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(maintenances).String() != "*[]apiclient.Maintenance" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(maintenances).String())
	}
}

func TestUpdateMaintenanceWindow(t *testing.T) {
	apih, server := maintenanceTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg          *Maintenance
		id           string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:          "invalid (nil)",
			shouldFail:  true,
			expectedErr: "invalid maintenance window config (nil)",
		},
		{
			id:          "invalid (cid)",
			cfg:         &Maintenance{CID: "/invalid"},
			shouldFail:  true,
			expectedErr: "invalid maintenance window CID (/invalid)",
		},
		{
			id:           "valid",
			cfg:          &testMaintenance,
			expectedType: "*apiclient.Maintenance",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			maint, err := apih.UpdateMaintenanceWindow(test.cfg)
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

func TestCreateMaintenanceWindow(t *testing.T) {
	apih, server := maintenanceTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg          *Maintenance
		id           string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:          "invalid (nil)",
			shouldFail:  true,
			expectedErr: "invalid maintenance window config (nil)",
		},
		{
			id:           "valid",
			cfg:          &testMaintenance,
			expectedType: "*apiclient.Maintenance",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.CreateMaintenanceWindow(test.cfg)
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

func TestDeleteMaintenanceWindow(t *testing.T) {
	apih, server := maintenanceTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg         *Maintenance
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			shouldFail:  true,
			expectedErr: "invalid maintenance window config (nil)",
		},
		{
			id:         "valid",
			cfg:        &testMaintenance,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteMaintenanceWindow(test.cfg)
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

func TestDeleteMaintenanceWindowByCID(t *testing.T) {
	apih, server := maintenanceTestBootstrap(t)
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
			expectedErr: "invalid maintenance window CID (none)",
		},
		{
			id:         "short cid",
			cid:        "1234",
			shouldFail: false,
		},
		{
			id:         "long cid",
			cid:        "/maintenance/1234",
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteMaintenanceWindowByCID(CIDType(&test.cid))
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

func TestSearchMaintenances(t *testing.T) {
	apih, server := maintenanceTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.Maintenance"
	search := SearchQueryType("/check_bundle/1234")
	filter := SearchFilterType(map[string][]string{"f_start_gt": {"1483639916"}})

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
			ack, err := apih.SearchMaintenanceWindows(test.search, test.filter)
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
