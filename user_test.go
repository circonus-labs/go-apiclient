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
	testUser = User{
		CID:       "/user/1234",
		Email:     "john@example.com",
		Firstname: "John",
		Lastname:  "Doe",
		ContactInfo: UserContactInfo{
			SMS:  "123-456-7890",
			XMPP: "foobar",
		},
	}
)

func testUserServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/user/1234" || path == "/user/current" {
			switch r.Method {
			case "GET": // get by id/cid
				ret, err := json.Marshal(testUser)
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
		} else if path == "/user" {
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []User
				if reqURL == "/user?f_firstname=john&f_lastname=doe" {
					c = []User{testUser}
				} else if reqURL == "/user" {
					c = []User{testUser}
				} else {
					c = []User{}
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

func userTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testUserServer()

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

func TestFetchUser(t *testing.T) {
	apih, server := userTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cid          string
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (cid)", "/invalid", "", true, "invalid user CID (" + config.UserPrefix + "//invalid)"},
		{"valid (default,empty)", "", "*apiclient.User", false, ""},
		{"valid (short cid)", "1234", "*apiclient.User", false, ""},
		{"valid (long cid)", "/user/1234", "*apiclient.User", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			cid := test.cid
			acct, err := apih.FetchUser(CIDType(&cid))
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

func TestFetchUsers(t *testing.T) {
	apih, server := userTestBootstrap(t)
	defer server.Close()

	users, err := apih.FetchUsers()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(users).String() != "*[]apiclient.User" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(users).String())
	}

}

func TestUpdateUser(t *testing.T) {
	apih, server := userTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cfg          *User
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid user config (nil)"},
		{"invalid (cid)", &User{CID: "/invalid"}, "", true, "invalid user CID (/invalid)"},
		{"valid", &testUser, "*apiclient.User", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			acct, err := apih.UpdateUser(test.cfg)
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

func TestSearchUsers(t *testing.T) {
	apih, server := userTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.User"
	filter := SearchFilterType(map[string][]string{"f_firstname": {"john"}, "f_lastname": {"doe"}})

	tests := []struct {
		id           string
		filter       *SearchFilterType
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"no filter", nil, expectedType, false, ""},
		{"filter", &filter, expectedType, false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.SearchUsers(test.filter)
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
