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
	testContactGroup = ContactGroup{
		CID:               "/contact_group/1234",
		LastModifiedBy:    "/user/1234",
		LastModified:      1483041636,
		AggregationWindow: 300,
		Contacts: ContactGroupContacts{
			External: []ContactGroupContactsExternal{
				{
					Info:   "12125550100",
					Method: "sms",
				},
				{
					Info:   "bert@example.com",
					Method: "xmpp",
				},
				{
					Info:   "ernie@example.com",
					Method: "email",
				},
			},
			Users: []ContactGroupContactsUser{
				{
					Info:    "snuffy@example.com",
					Method:  "email",
					UserCID: "/user/1234",
				},
				{
					Info:    "12125550199",
					Method:  "sms",
					UserCID: "/user/4567",
				},
			},
		},
		Escalations: []*ContactGroupEscalation{
			{
				After:           900,
				ContactGroupCID: "/contact_group/4567",
			},
			nil,
			nil,
			nil,
			nil,
		},
		Name:      "FooBar",
		Reminders: []uint{10, 0, 0, 15, 30},
	}
)

func testContactGroupServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/contact_group/1234":
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testContactGroup)
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
		case "/contact_group":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []ContactGroup
				switch reqURL {
				case "/contact_group?search=%28name%3D%22ops%22%29":
					c = []ContactGroup{testContactGroup}
				case "/contact_group?f__last_modified_gt=1483639916":
					c = []ContactGroup{testContactGroup}
				case "/contact_group?f__last_modified_gt=1483639916&search=%28name%3D%22ops%22%29":
					c = []ContactGroup{testContactGroup}
				case "/contact_group":
					c = []ContactGroup{testContactGroup}
				default:
					c = []ContactGroup{}
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
				ret, err := json.Marshal(testContactGroup)
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

func contactGroupTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testContactGroupServer()

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

func TestNewContactGroup(t *testing.T) {
	contactGroup := NewContactGroup()
	if reflect.TypeOf(contactGroup).String() != "*apiclient.ContactGroup" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(contactGroup).String())
	}
}

func TestFetchContactGroup(t *testing.T) {
	apih, server := contactGroupTestBootstrap(t)
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
			expectedErr:  "invalid contact group CID (none)",
		},
		{
			id:           "short cid",
			cid:          "1234",
			expectedType: "*apiclient.ContactGroup",
			shouldFail:   false,
		},
		{
			id:           "long cid",
			cid:          "/contact_group/1234",
			expectedType: "*apiclient.ContactGroup",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchContactGroup(CIDType(&test.cid))
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

func TestFetchContactGroups(t *testing.T) {
	apih, server := contactGroupTestBootstrap(t)
	defer server.Close()

	contactGroups, err := apih.FetchContactGroups()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(contactGroups).String() != "*[]apiclient.ContactGroup" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(contactGroups).String())
	}

}

func TestUpdateContactGroup(t *testing.T) {
	apih, server := contactGroupTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg         *ContactGroup
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			cfg:         nil,
			shouldFail:  true,
			expectedErr: "invalid contact group config (nil)",
		},
		{
			id:          "invalid (cid)",
			cfg:         &ContactGroup{CID: "/invalid"},
			shouldFail:  true,
			expectedErr: "invalid contact group CID (/invalid)",
		},
		{
			id:         "valid",
			cfg:        &testContactGroup,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			_, err := apih.UpdateContactGroup(test.cfg)
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

func TestCreateContactGroup(t *testing.T) {
	apih, server := contactGroupTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg          *ContactGroup
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
			expectedErr:  "invalid contact group config (nil)",
		},
		{
			id:           "valid",
			cfg:          &testContactGroup,
			expectedType: "*apiclient.ContactGroup",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.CreateContactGroup(test.cfg)
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

func TestDeleteContactGroup(t *testing.T) {
	apih, server := contactGroupTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg         *ContactGroup
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			cfg:         nil,
			shouldFail:  true,
			expectedErr: "invalid contact group config (nil)",
		},
		{
			id:         "valid",
			cfg:        &testContactGroup,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteContactGroup(test.cfg)
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

func TestDeleteContactGroupByCID(t *testing.T) {
	apih, server := contactGroupTestBootstrap(t)
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
			expectedErr: "invalid contact group CID (none)",
		},
		{
			id:         "short cid",
			cid:        "1234",
			shouldFail: false,
		},
		{
			id:         "long cid",
			cid:        "/contact_group/1234",
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteContactGroupByCID(CIDType(&test.cid))
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

func TestSearchContactGroups(t *testing.T) {
	apih, server := contactGroupTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.ContactGroup"
	search := SearchQueryType(`(name="ops")`)
	filter := SearchFilterType(map[string][]string{"f__last_modified_gt": {"1483639916"}})

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
			id:          "search no filter",
			search:      &search,
			filter:      nil,
			expectedErr: expectedType,
			shouldFail:  false,
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
			ack, err := apih.SearchContactGroups(test.search, test.filter)
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
