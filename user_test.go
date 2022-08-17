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
		switch path {
		case "/user/current":
			fallthrough
		case "/user/1234":
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
				b, err := io.ReadAll(r.Body)
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
		case "/user":
			switch r.Method {
			case "GET":
				var c []User
				reqURL := r.URL.String()
				switch reqURL {
				case "/user?f_firstname=john&f_lastname=doe":
					c = []User{testUser}
				case "/user":
					c = []User{testUser}
				default:
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
					fmt.Fprintf(w, "not found: %s %s\n", r.Method, reqURL)
				}
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
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:           "valid (default,empty)",
			expectedType: "*apiclient.User",
			shouldFail:   false,
		},
		{
			id:           "valid (short cid)",
			cid:          "1234",
			expectedType: "*apiclient.User",
			shouldFail:   false,
		},
		{
			id:           "valid (long cid)",
			cid:          "/user/1234",
			expectedType: "*apiclient.User",
			shouldFail:   false,
		},
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
		cfg          *User
		id           string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:           "invalid (nil)",
			shouldFail:   true,
			expectedType: "invalid user config (nil)",
		},
		{
			id:           "invalid (cid)",
			cfg:          &User{CID: "/invalid"},
			shouldFail:   true,
			expectedType: "invalid user CID (/invalid)",
		},
		{
			id:           "valid",
			cfg:          &testUser,
			expectedType: "*apiclient.User",
			shouldFail:   false,
		},
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
		filter       *SearchFilterType
		id           string
		expectedType string
		expectedErr  string
		shouldFail   bool
	}{
		{
			id:           "no filter",
			expectedType: expectedType,
			shouldFail:   false,
		},
		{
			id:           "filter",
			filter:       &filter,
			expectedType: expectedType,
			shouldFail:   false,
		},
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
