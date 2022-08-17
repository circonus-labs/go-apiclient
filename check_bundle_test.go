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
				b, err := io.ReadAll(r.Body)
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
				fmt.Fprintf(w, "not found: %s %s\n", r.Method, path)
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
					fmt.Fprintf(w, "not found: %s %s\n", r.Method, reqURL)
				}
			case "POST": // create
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
				fmt.Fprintf(w, "not found: %s %s\n", r.Method, path)
			}
		default:
			w.WriteHeader(404)
			fmt.Fprintf(w, "not found: %s %s\n", r.Method, path)
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
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:           "empty cid",
			cid:          "",
			expectedType: "",
			shouldFail:   true,
			expectedErr:  "invalid check bundle CID (none)",
		},
		{
			id:           "short cid",
			cid:          "1234",
			expectedType: "*apiclient.CheckBundle",
			shouldFail:   false,
		},
		{
			id:           "long cid",
			cid:          "/check_bundle/1234",
			expectedType: "*apiclient.CheckBundle",
			shouldFail:   false,
		},
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
		cfg         *CheckBundle
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			cfg:         nil,
			shouldFail:  true,
			expectedErr: "invalid check bundle config (nil)",
		},
		{
			id:          "invalid (cid)",
			cfg:         &CheckBundle{CID: "/invalid"},
			shouldFail:  true,
			expectedErr: "invalid check bundle CID (/invalid)",
		},
		{
			id:         "valid",
			cfg:        &testCheckBundle,
			shouldFail: false,
		},
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
		cfg          *CheckBundle
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
			expectedErr:  "invalid check bundle config (nil)",
		},
		{
			id:           "valid",
			cfg:          &testCheckBundle,
			expectedType: "*apiclient.CheckBundle",
			shouldFail:   false,
		},
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
		cfg         *CheckBundle
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			cfg:         nil,
			shouldFail:  true,
			expectedErr: "invalid check bundle config (nil)",
		},
		{
			id:         "valid",
			cfg:        &testCheckBundle,
			shouldFail: false,
		},
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
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "empty cid",
			cid:         "",
			shouldFail:  true,
			expectedErr: "invalid check bundle CID (none)",
		},
		{
			id:         "short cid",
			cid:        "1234",
			shouldFail: false,
		},
		{
			id:         "long cid",
			cid:        "/check_bundle/1234",
			shouldFail: false,
		},
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

func Test_fixTags(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want []string
	}{
		{
			name: "blank",
			tags: []string{"foo", ""},
			want: []string{"foo"},
		},
		{
			name: "duplicate",
			tags: []string{"foo", "foo"},
			want: []string{"foo"},
		},
		{
			name: "lowercase",
			tags: []string{"Foo"},
			want: []string{"foo"},
		},
		{
			name: "combo1",
			tags: []string{"FOO", "foo", ""},
			want: []string{"foo"},
		},
		{
			name: "combo2",
			tags: []string{"FOO:BAR", "foo", ""},
			want: []string{"foo", "foo:bar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fixTags(tt.tags); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fixTags() = %v, want %v", got, tt.want)
			}
		})
	}
}
