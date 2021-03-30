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
				_, err := ioutil.ReadAll(r.Body)
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

	tests := []struct { //nolint:govet
		id           string
		cid          string
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", true, "invalid rule set CID (none)"},
		{"short (old) cid", "1234_tt_firstbyte", "*apiclient.RuleSet", false, ""},
		{"long (old) cid", "/rule_set/1234_tt_firstbyte", "*apiclient.RuleSet", false, ""},
		{"short (new) cid", "1234", "*apiclient.RuleSet", false, ""},
		{"long (new) cid", "/rule_set/1234", "*apiclient.RuleSet", false, ""},
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

	tests := []struct { //nolint:govet
		id           string
		cfg          *RuleSet
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid rule set config (nil)"},
		{"invalid (cid)", &RuleSet{CID: "/invalid"}, "", true, "invalid rule set CID (/invalid)"},
		{"valid", &testRuleSet, "*apiclient.RuleSet", false, ""},
		{"valid", &testRuleSetNewCID, "*apiclient.RuleSet", false, ""},
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

	tests := []struct { //nolint:govet
		id           string
		cfg          *RuleSet
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid rule set config (nil)"},
		{"valid", &testRuleSet, "*apiclient.RuleSet", false, ""},
		{"valid", &testRuleSetNewCID, "*apiclient.RuleSet", false, ""},
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

	tests := []struct { //nolint:govet
		id          string
		cfg         *RuleSet
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid rule set config (nil)"},
		{"valid", &testRuleSet, false, ""},
		{"valid", &testRuleSetNewCID, false, ""},
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

	tests := []struct { //nolint:govet
		id          string
		cid         string
		shouldFail  bool
		expectedErr string
	}{
		{"empty cid", "", true, "invalid rule set CID (none)"},
		{"short (old) cid", "1234_tt_firstbyte", false, ""},
		{"long (old) cid", "/rule_set/1234_tt_firstbyte", false, ""},
		{"short (new) cid", "1234", false, ""},
		{"long (new) cid", "/rule_set/1234", false, ""},
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
