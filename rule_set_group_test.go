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
	testRuleSetGroup = RuleSetGroup{
		CID: "/rule_set_group/1234",
		ContactGroups: map[uint8][]string{
			1: {"/contact_group/1234", "/contact_group/5678"},
			2: {"/contact_group/1234"},
			3: {"/contact_group/1234"},
			4: {},
			5: {},
		},
		Formulas: []RuleSetGroupFormula{
			{
				Expression:    "(A and B) and not C",
				RaiseSeverity: 2,
				Wait:          0,
			},
			{
				Expression:    "3",
				RaiseSeverity: 1,
				Wait:          5,
			},
		},
		Name: "Multiple webservers gone bad",
		RuleSetConditions: []RuleSetGroupCondition{
			{
				MatchingSeverities: []string{"1", "2"},
				RuleSetCID:         "/rule_set/1234_tt_firstbyte",
			},
			{
				MatchingSeverities: []string{"1", "2"},
				RuleSetCID:         "/rule_set/5678_tt_firstbyte",
			},
			{
				MatchingSeverities: []string{"1", "2"},
				RuleSetCID:         "/rule_set/9012_tt_firstbyte",
			},
		},
	}
)

func testRuleSetGroupServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/rule_set_group/1234":
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testRuleSetGroup)
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
		case "/rule_set_group":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []RuleSetGroup
				switch reqURL {
				case "/rule_set_group?search=web+requests":
					c = []RuleSetGroup{testRuleSetGroup}
				case "/rule_set_group?f_tags_has=location%3Aconus":
					c = []RuleSetGroup{testRuleSetGroup}
				case "/rule_set_group?f_tags_has=location%3Aconus&search=web+requests":
					c = []RuleSetGroup{testRuleSetGroup}
				case "/rule_set_group":
					c = []RuleSetGroup{testRuleSetGroup}
				default:
					c = []RuleSetGroup{}
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
				ret, err := json.Marshal(testRuleSetGroup)
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

func ruleSetGroupTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testRuleSetGroupServer()

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

func TestNewRuleSetGroup(t *testing.T) {
	ruleSetGroup := NewRuleSetGroup()
	if reflect.TypeOf(ruleSetGroup).String() != "*apiclient.RuleSetGroup" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(ruleSetGroup).String())
	}
}

func TestFetchRuleSetGroup(t *testing.T) {
	apih, server := ruleSetGroupTestBootstrap(t)
	defer server.Close()

	tests := []struct { //nolint:govet
		id           string
		cid          string
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", true, "invalid rule set group CID (none)"},
		{"short cid", "1234", "*apiclient.RuleSetGroup", false, ""},
		{"long cid", "/rule_set_group/1234", "*apiclient.RuleSetGroup", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchRuleSetGroup(CIDType(&test.cid))
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

func TestFetchRuleSetGroups(t *testing.T) {
	apih, server := ruleSetGroupTestBootstrap(t)
	defer server.Close()

	ruleSetGroups, err := apih.FetchRuleSetGroups()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(ruleSetGroups).String() != "*[]apiclient.RuleSetGroup" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(ruleSetGroups).String())
	}
}

func TestUpdateRuleSetGroup(t *testing.T) {
	apih, server := ruleSetGroupTestBootstrap(t)
	defer server.Close()

	tests := []struct { //nolint:govet
		id           string
		cfg          *RuleSetGroup
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid rule set group config (nil)"},
		{"invalid (cid)", &RuleSetGroup{CID: "/invalid"}, "", true, "invalid rule set group CID (/invalid)"},
		{"valid", &testRuleSetGroup, "*apiclient.RuleSetGroup", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			maint, err := apih.UpdateRuleSetGroup(test.cfg)
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

func TestCreateRuleSetGroup(t *testing.T) {
	apih, server := ruleSetGroupTestBootstrap(t)
	defer server.Close()

	tests := []struct { //nolint:govet
		id           string
		cfg          *RuleSetGroup
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid rule set group config (nil)"},
		{"valid", &testRuleSetGroup, "*apiclient.RuleSetGroup", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.CreateRuleSetGroup(test.cfg)
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

func TestDeleteRuleSetGroup(t *testing.T) {
	apih, server := ruleSetGroupTestBootstrap(t)
	defer server.Close()

	tests := []struct { //nolint:govet
		id          string
		cfg         *RuleSetGroup
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid rule set group config (nil)"},
		{"valid", &testRuleSetGroup, false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteRuleSetGroup(test.cfg)
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

func TestDeleteRuleSetGroupByCID(t *testing.T) {
	apih, server := ruleSetGroupTestBootstrap(t)
	defer server.Close()

	tests := []struct { //nolint:govet
		id          string
		cid         string
		shouldFail  bool
		expectedErr string
	}{
		{"empty cid", "", true, "invalid rule set group CID (none)"},
		{"short cid", "1234", false, ""},
		{"long cid", "/rule_set_group/1234", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteRuleSetGroupByCID(CIDType(&test.cid))
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

func TestSearchRuleSetGroups(t *testing.T) {
	apih, server := ruleSetGroupTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.RuleSetGroup"
	search := SearchQueryType("web requests")
	filter := SearchFilterType(map[string][]string{"f_tags_has": {"location:conus"}})

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
			ack, err := apih.SearchRuleSetGroups(test.search, test.filter)
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
