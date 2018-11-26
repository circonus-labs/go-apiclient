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
	testAccount = Account{
		CID: "/account/1234",
		ContactGroups: []string{
			"/contact_group/1701",
			"/contact_group/3141",
		},
		OwnerCID: "/user/42",
		Usage: []AccountLimit{
			{
				Limit: 50,
				Type:  "Host",
				Used:  7,
			},
		},
		Address1:    &[]string{"Hooper's Store"}[0],
		Address2:    &[]string{"Sesame Street"}[0],
		CCEmail:     &[]string{"accounts_payable@yourdomain.com"}[0],
		City:        &[]string{"New York City"}[0],
		Country:     "US",
		Description: &[]string{"Hooper's Store Account"}[0],
		Invites: []AccountInvite{
			{
				Email: "alan@example.com",
				Role:  "Admin",
			},
			{
				Email: "chris.robinson@example.com",
				Role:  "Normal",
			},
		},
		Name:      "hoopers-store",
		StateProv: &[]string{"NY"}[0],
		Timezone:  "America/New_York",
		Users: []AccountUser{
			{
				Role:    "Admin",
				UserCID: "/user/42",
			},
		},
	}
)

func testAccountServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/account/1234" || path == "/account/current" {
			switch r.Method {
			case "GET": // get by id/cid
				ret, err := json.Marshal(testAccount)
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
				fmt.Fprintln(w, "not found")
			}
		} else if path == "/account" {
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []Account
				if reqURL == "/account?f_name_wildcard=%2Aops%2A" {
					c = []Account{testAccount}
				} else if reqURL == "/account" {
					c = []Account{testAccount}
				} else {
					c = []Account{}
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
			default:
				w.WriteHeader(404)
				fmt.Fprintln(w, "not found")
			}
		} else {
			w.WriteHeader(404)
			fmt.Fprintln(w, "not found")
		}
	}

	return httptest.NewServer(http.HandlerFunc(f))
}

func TestFetchAccount(t *testing.T) {
	server := testAccountServer()
	defer server.Close()

	ac := &Config{
		TokenKey: "abc123",
		TokenApp: "test",
		URL:      server.URL,
	}
	apih, err := NewAPI(ac)
	if err != nil {
		t.Errorf("Expected no error, got '%v'", err)
	}

	tests := []struct {
		id           string
		cid          string
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (cid)", "/invalid", "", true, "invalid account CID (" + config.AccountPrefix + "//invalid)"},
		{"valid (default,empty)", "", "*apiclient.Account", false, ""},
		{"valid (short cid)", "1234", "*apiclient.Account", false, ""},
		{"valid (long cid)", "/account/1234", "*apiclient.Account", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			cid := test.cid
			acct, err := apih.FetchAccount(CIDType(&cid))
			if test.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				} else if err.Error() != test.expectedErr {
					t.Fatalf("unexpected error (%s)", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error (%s)", err)
				} else if reflect.TypeOf(acct).String() != test.expectedType {
					t.Fatalf("unexpected type (%s)", reflect.TypeOf(acct))
				}
			}
		})
	}
}

func TestFetchAccounts(t *testing.T) {
	server := testAccountServer()
	defer server.Close()

	ac := &Config{
		TokenKey: "abc123",
		TokenApp: "test",
		URL:      server.URL,
	}
	apih, err := NewAPI(ac)
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	accounts, err := apih.FetchAccounts()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	actualType := reflect.TypeOf(accounts)
	if actualType.String() != "*[]apiclient.Account" {
		t.Fatalf("unexpected type (%s)", actualType.String())
	}

}

func TestUpdateAccount(t *testing.T) {
	server := testAccountServer()
	defer server.Close()

	var apih *API

	ac := &Config{
		TokenKey: "abc123",
		TokenApp: "test",
		URL:      server.URL,
	}
	apih, err := NewAPI(ac)
	if err != nil {
		t.Errorf("Expected no error, got '%v'", err)
	}

	tests := []struct {
		id           string
		cfg          *Account
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid account config (nil)"},
		{"invalid (cid)", &Account{CID: "/invalid"}, "", true, "invalid account CID (/invalid)"},
		{"valid", &testAccount, "*apiclient.Account", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			acct, err := apih.UpdateAccount(test.cfg)
			if test.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				} else if err.Error() != test.expectedErr {
					t.Fatalf("unexpected error (%s)", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error (%s)", err)
				} else if reflect.TypeOf(acct).String() != test.expectedType {
					t.Fatalf("unexpected type (%s)", reflect.TypeOf(acct))
				}
			}
		})
	}
}

func TestSearchAccounts(t *testing.T) {
	server := testAccountServer()
	defer server.Close()

	var apih *API

	ac := &Config{
		TokenKey: "abc123",
		TokenApp: "test",
		URL:      server.URL,
	}
	apih, err := NewAPI(ac)
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	expectedType := "*[]apiclient.Account"

	t.Log("no filter")
	{
		accounts, err := apih.SearchAccounts(nil)
		if err != nil {
			t.Fatalf("unexpected error (%s)", err)
		}

		actualType := reflect.TypeOf(accounts)
		if actualType.String() != expectedType {
			t.Fatalf("unexpected type (%s)", actualType.String())
		}
	}

	t.Log("filter")
	{
		filter := SearchFilterType(map[string][]string{"f_name_wildcard": {"*ops*"}})
		accounts, err := apih.SearchAccounts(&filter)
		if err != nil {
			t.Fatalf("unexpected error (%s)", err)
		}

		actualType := reflect.TypeOf(accounts)
		if actualType.String() != expectedType {
			t.Fatalf("unexpected type (%s)", actualType.String())
		}
	}
}
