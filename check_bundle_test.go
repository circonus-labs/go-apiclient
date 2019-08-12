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

	"github.com/circonus-labs/go-apiclient/config"
)

var (
	testCheckBundle = CheckBundle{
		CheckUUIDs:         []string{"abc123-a1b2-c3d4-e5f6-123abc"},
		Checks:             []string{"/check/1234"},
		CID:                "/check_bundle/1234",
		Created:            0,
		LastModified:       0,
		LastModifedBy:      "",
		ReverseConnectURLs: []string{""},
		Config:             map[config.Key]string{},
		Brokers:            []string{"/broker/1234"},
		DisplayName:        "test check",
		Metrics:            []CheckBundleMetric{},
		MetricLimit:        0,
		Notes:              nil,
		Period:             60,
		Status:             "active",
		Target:             "127.0.0.1",
		Timeout:            10,
		Type:               "httptrap",
		Tags:               []string{},
	}
)

func testCheckBundleServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/check_bundle/1234":
			switch r.Method {
			case "PUT": // update
				defer r.Body.Close()
				b, err := ioutil.ReadAll(r.Body)
				if err != nil {
					panic(err)
				}
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, string(b))
			case "GET": // get by id/cid
				ret, err := json.Marshal(testCheckBundle)
				if err != nil {
					panic(err)
				}
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, string(ret))
			case "DELETE": // delete
				w.WriteHeader(200)
				fmt.Fprintln(w, "")
			default:
				w.WriteHeader(404)
				fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
			}
		case "/check_bundle":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []CheckBundle
				switch reqURL {
				case "/check_bundle?search=test":
					c = []CheckBundle{testCheckBundle}
				case "/check_bundle?f__tags_has=cat%3Atag":
					c = []CheckBundle{testCheckBundle}
				case "/check_bundle?f__tags_has=cat%3Atag&search=test":
					c = []CheckBundle{testCheckBundle}
				case "/check_bundle":
					c = []CheckBundle{testCheckBundle}
				default:
					c = []CheckBundle{}
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
			case "POST": // create
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
		default:
			w.WriteHeader(404)
			fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
		}
	}

	return httptest.NewServer(http.HandlerFunc(f))
}

func checkBundleTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testCheckBundleServer()

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

func TestNewCheckBundle(t *testing.T) {
	bundle := NewCheckBundle()
	if reflect.TypeOf(bundle).String() != "*apiclient.CheckBundle" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(bundle).String())
	}
}

func TestFetchCheckBundle(t *testing.T) {
	apih, server := checkBundleTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cid          string
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", true, "invalid check bundle CID (none)"},
		{"short cid", "1234", "*apiclient.CheckBundle", false, ""},
		{"long cid", "/check_bundle/1234", "*apiclient.CheckBundle", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchCheckBundle(CIDType(&test.cid))
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

func TestFetchCheckBundles(t *testing.T) {
	apih, server := checkBundleTestBootstrap(t)
	defer server.Close()

	bundles, err := apih.FetchCheckBundles()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(bundles).String() != "*[]apiclient.CheckBundle" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(bundles).String())
	}
}

func TestUpdateCheckBundle(t *testing.T) {
	apih, server := checkBundleTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id          string
		cfg         *CheckBundle
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid check bundle config (nil)"},
		{"invalid (cid)", &CheckBundle{CID: "/invalid"}, true, "invalid check bundle CID (/invalid)"},
		{"valid", &testCheckBundle, false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			_, err := apih.UpdateCheckBundle(test.cfg)
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

func TestCreateCheckBundle(t *testing.T) {
	apih, server := checkBundleTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cfg          *CheckBundle
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid check bundle config (nil)"},
		{"valid", &testCheckBundle, "*apiclient.CheckBundle", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.CreateCheckBundle(test.cfg)
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

func TestDeleteCheckBundle(t *testing.T) {
	apih, server := checkBundleTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id          string
		cfg         *CheckBundle
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid check bundle config (nil)"},
		{"valid", &testCheckBundle, false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteCheckBundle(test.cfg)
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

func TestDeleteCheckBundleByCID(t *testing.T) {
	apih, server := checkBundleTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id          string
		cid         string
		shouldFail  bool
		expectedErr string
	}{
		{"empty cid", "", true, "invalid check bundle CID (none)"},
		{"short cid", "1234", false, ""},
		{"long cid", "/check_bundle/1234", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteCheckBundleByCID(CIDType(&test.cid))
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

func TestSearchCheckBundles(t *testing.T) {
	apih, server := checkBundleTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.CheckBundle"
	search := SearchQueryType("test")
	filter := SearchFilterType(map[string][]string{"f__tags_has": {"cat:tag"}})

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
			ack, err := apih.SearchCheckBundles(test.search, test.filter)
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
