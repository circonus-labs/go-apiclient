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
	u2          = []byte(`{"foo":"bar"}`)
	testRuleSet = RuleSet{
		CID:      "/rule_set/1234_tt_firstbyte",
		CheckCID: "/check/1234",
		ContactGroups: map[uint8][]string{
			1: {"/contact_group/1234", "/contact_group/5678"},
			2: {"/contact_group/1234"},
			3: {"/contact_group/1234"},
			4: {},
			5: {},
		},
		Link:       &[]string{"http://example.com/how2fix/webserver_down/"}[0],
		MetricName: "tt_firstbyte",
		MetricType: "numeric",
		Notes:      &[]string{"Determine if the HTTP request is taking too long to start (or is down.)  Don't fire if ping is already alerting"}[0],
		UserJSON:   json.RawMessage(`{"foo":"bar","b2": {"bar":1},"b3":[1,2,3]}`),
		Parent:     &[]string{"1233_ping"}[0],
		Rules: []RuleSetRule{
			{
				Criteria:          "on absence",
				Severity:          1,
				Value:             "300",
				Wait:              5,
				WindowingDuration: 300,
				WindowingFunction: nil,
			},
			{
				Criteria: "max value",
				Severity: 2,
				Value:    "1000",
				Wait:     5,
			},
		},
	}

	// rule_set CID format changed 2019-06-05...
	// original cid format /rule_set/checkid_metricname
	// new cid format /rule_set/[0-9]+
	testRuleSetNewCID = RuleSet{ //nolint:gochecknoglobals
		CID:      "/rule_set/1234",
		CheckCID: "/check/1234",
		ContactGroups: map[uint8][]string{
			1: {"/contact_group/1234", "/contact_group/5678"},
			2: {"/contact_group/1234"},
			3: {"/contact_group/1234"},
			4: {},
			5: {},
		},
		Link:       &[]string{"http://example.com/how2fix/webserver_down/"}[0],
		MetricName: "tt_firstbyte",
		MetricType: "numeric",
		Notes:      &[]string{"Determine if the HTTP request is taking too long to start (or is down.)  Don't fire if ping is already alerting"}[0],
		UserJSON:   json.RawMessage(u2),
		Parent:     &[]string{"1233_ping"}[0],
		Rules: []RuleSetRule{
			{
				Criteria:          "on absence",
				Severity:          1,
				Value:             "300",
				Wait:              5,
				WindowingDuration: 300,
				WindowingFunction: nil,
			},
			{
				Criteria: "max value",
				Severity: 2,
				Value:    "1000",
				Wait:     5,
			},
		},
	}
)

func testRuleSetServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/rule_set/1234_tt_firstbyte":
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testRuleSet)
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
		case "/rule_set/1234": //nolint:dupl
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testRuleSetNewCID)
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
		case "/rule_set":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []RuleSet
				switch reqURL {
				case "/rule_set?search=request%60latency_ms":
					c = []RuleSet{testRuleSet}
				case "/rule_set?f_tags_has=service%3Aweb":
					c = []RuleSet{testRuleSet}
				case "/rule_set?f_tags_has=service%3Aweb&search=request%60latency_ms":
					c = []RuleSet{testRuleSet}
				case "/rule_set":
					c = []RuleSet{testRuleSet}
				default:
					c = []RuleSet{}
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
				ret, err := json.Marshal(testRuleSet)
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

func ruleSetTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testRuleSetServer()

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

func TestNewRuleSet(t *testing.T) {
	ruleSet := NewRuleSet()
	if reflect.TypeOf(ruleSet).String() != "*apiclient.RuleSet" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(ruleSet).String())
	}
}

func TestFetchRuleSet(t *testing.T) {
	apih, server := ruleSetTestBootstrap(t)
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
			expectedErr: "invalid rule set CID (none)",
		},
		{
			id:           "short (old) cid",
			cid:          "1234_tt_firstbyte",
			expectedType: "*apiclient.RuleSet",
			shouldFail:   false,
		},
		{
			id:           "long (old) cid",
			cid:          "/rule_set/1234_tt_firstbyte",
			expectedType: "*apiclient.RuleSet",
			shouldFail:   false,
		},
		{
			id:           "short (new) cid",
			cid:          "1234",
			expectedType: "*apiclient.RuleSet",
			shouldFail:   false,
		},
		{
			id:           "long (new) cid",
			cid:          "/rule_set/1234",
			expectedType: "*apiclient.RuleSet",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchRuleSet(CIDType(&test.cid))
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

func TestFetchRuleSets(t *testing.T) {
	apih, server := ruleSetTestBootstrap(t)
	defer server.Close()

	ruleSets, err := apih.FetchRuleSets()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(ruleSets).String() != "*[]apiclient.RuleSet" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(ruleSets).String())
	}
}

func TestUpdateRuleSet(t *testing.T) {
	apih, server := ruleSetTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg          *RuleSet
		id           string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:          "invalid (nil)",
			shouldFail:  true,
			expectedErr: "invalid rule set config (nil)",
		},
		{
			id:          "invalid (cid)",
			cfg:         &RuleSet{CID: "/invalid"},
			shouldFail:  true,
			expectedErr: "invalid rule set CID (/invalid)",
		},
		{
			id:           "valid",
			cfg:          &testRuleSet,
			expectedType: "*apiclient.RuleSet",
			shouldFail:   false,
		},
		{
			id:           "valid",
			cfg:          &testRuleSetNewCID,
			expectedType: "*apiclient.RuleSet",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			maint, err := apih.UpdateRuleSet(test.cfg)
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

func TestCreateRuleSet(t *testing.T) {
	apih, server := ruleSetTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg          *RuleSet
		id           string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:          "invalid (nil)",
			shouldFail:  true,
			expectedErr: "invalid rule set config (nil)",
		},
		{
			id:           "valid",
			cfg:          &testRuleSet,
			expectedType: "*apiclient.RuleSet",
			shouldFail:   false,
		},
		{
			id:           "valid",
			cfg:          &testRuleSetNewCID,
			expectedType: "*apiclient.RuleSet",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.CreateRuleSet(test.cfg)
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

func TestDeleteRuleSet(t *testing.T) {
	apih, server := ruleSetTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg         *RuleSet
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			shouldFail:  true,
			expectedErr: "invalid rule set config (nil)",
		},
		{
			id:         "valid",
			cfg:        &testRuleSet,
			shouldFail: false,
		},
		{
			id:         "valid",
			cfg:        &testRuleSetNewCID,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteRuleSet(test.cfg)
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

func TestDeleteRuleSetByCID(t *testing.T) {
	apih, server := ruleSetTestBootstrap(t)
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
			expectedErr: "invalid rule set CID (none)",
		},
		{
			id:         "short (old) cid",
			cid:        "1234_tt_firstbyte",
			shouldFail: false,
		},
		{
			id:         "long (old) cid",
			cid:        "/rule_set/1234_tt_firstbyte",
			shouldFail: false,
		},
		{
			id:         "short (new) cid",
			cid:        "1234",
			shouldFail: false,
		},
		{
			id:         "long (new) cid",
			cid:        "/rule_set/1234",
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteRuleSetByCID(CIDType(&test.cid))
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

func TestSearchRuleSets(t *testing.T) {
	apih, server := ruleSetTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.RuleSet"
	search := SearchQueryType("request`latency_ms")
	filter := SearchFilterType(map[string][]string{"f_tags_has": {"service:web"}})

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
			ack, err := apih.SearchRuleSets(test.search, test.filter)
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
