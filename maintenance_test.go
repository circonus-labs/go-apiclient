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
		if path == "/maintenance/1234" {
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
		} else if path == "/maintenance" {
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []Maintenance
				if reqURL == "/maintenance?search=%2Fcheck_bundle%2F1234" {
					c = []Maintenance{testMaintenance}
				} else if reqURL == "/maintenance?f_start_gt=1483639916" {
					c = []Maintenance{testMaintenance}
				} else if reqURL == "/maintenance?f_start_gt=1483639916&search=%2Fcheck_bundle%2F1234" {
					c = []Maintenance{testMaintenance}
				} else if reqURL == "/maintenance" {
					c = []Maintenance{testMaintenance}
				} else {
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
					fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, reqURL))
				}
			case "POST":
				defer r.Body.Close()
				_, err := ioutil.ReadAll(r.Body)
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
				fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
			}
		} else {
			w.WriteHeader(404)
			fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
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
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", true, "invalid maintenance window CID (none)"},
		{"invalid cid", "/invalid", "", true, "invalid maintenance window CID (/maintenance//invalid)"},
		{"short cid", "1234", "*apiclient.Maintenance", false, ""},
		{"long cid", "/maintenance/1234", "*apiclient.Maintenance", false, ""},
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
		id           string
		cfg          *Maintenance
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid maintenance window config (nil)"},
		{"invalid (cid)", &Maintenance{CID: "/invalid"}, "", true, "invalid maintenance window CID (/invalid)"},
		{"valid", &testMaintenance, "*apiclient.Maintenance", false, ""},
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
		id           string
		cfg          *Maintenance
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid maintenance window config (nil)"},
		{"valid", &testMaintenance, "*apiclient.Maintenance", false, ""},
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
		id          string
		cfg         *Maintenance
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid maintenance window config (nil)"},
		{"invalid (cid)", &Maintenance{CID: "/invalid"}, true, "invalid maintenance window CID (/maintenance//invalid)"},
		{"valid", &testMaintenance, false, ""},
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
		shouldFail  bool
		expectedErr string
	}{
		{"empty cid", "", true, "invalid maintenance window CID (none)"},
		{"invalid cid", "/invalid", true, "invalid maintenance window CID (/maintenance//invalid)"},
		{"short cid", "1234", false, ""},
		{"long cid", "/maintenance/1234", false, ""},
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
