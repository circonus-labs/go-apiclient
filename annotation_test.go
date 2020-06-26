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
	testAnnotation = Annotation{
		CID:            "/annotation/1234",
		Created:        1483033102,
		LastModified:   1483033102,
		LastModifiedBy: "/user/1234",
		Start:          1483033100,
		Stop:           1483033102,
		Category:       "foo",
		Title:          "Foo Bar Baz",
	}
)

func testAnnotationServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/annotation/1234":
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testAnnotation)
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
		case "/annotation":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []Annotation
				switch reqURL {
				case "/annotation?search=%28category%3D%22updates%22%29":
					c = []Annotation{testAnnotation}
				case "/annotation?f__created_gt=1483639916":
					c = []Annotation{testAnnotation}
				case "/annotation?f__created_gt=1483639916&search=%28category%3D%22updates%22%29":
					c = []Annotation{testAnnotation}
				case "/annotation":
					c = []Annotation{testAnnotation}
				default:
					c = []Annotation{}
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
				ret, err := json.Marshal(testAnnotation)
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

func annotationTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testAnnotationServer()

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

func TestNewAnnotation(t *testing.T) {
	annotation := NewAnnotation()
	if reflect.TypeOf(annotation).String() != "*apiclient.Annotation" {
		t.Fatalf("unexpected (%s)", reflect.TypeOf(annotation).String())
	}
}

func TestFetchAnnotation(t *testing.T) {
	apih, server := annotationTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cid          string
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", true, "invalid annotation CID (none)"},
		{"short cid", "1234", "*apiclient.Annotation", false, ""},
		{"long cid", "/annotation/1234", "*apiclient.Annotation", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchAnnotation(CIDType(&test.cid))
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

func TestFetchAnnotations(t *testing.T) {
	apih, server := annotationTestBootstrap(t)
	defer server.Close()

	annotations, err := apih.FetchAnnotations()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(annotations).String() != "*[]apiclient.Annotation" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(annotations).String())
	}
}

func TestUpdateAnnotation(t *testing.T) {
	apih, server := annotationTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id          string
		cfg         *Annotation
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid annotation config (nil)"},
		{"invalid (cid)", &Annotation{CID: "/invalid"}, true, "invalid annotation CID (/invalid)"},
		{"valid", &testAnnotation, false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			_, err := apih.UpdateAnnotation(test.cfg)
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

func TestCreateAnnotation(t *testing.T) {
	apih, server := annotationTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cfg          *Annotation
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid annotation config (nil)"},
		{"valid", &testAnnotation, "*apiclient.Annotation", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.CreateAnnotation(test.cfg)
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

func TestDeleteAnnotation(t *testing.T) {
	apih, server := annotationTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id          string
		cfg         *Annotation
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid annotation config (nil)"},
		{"valid", &testAnnotation, false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteAnnotation(test.cfg)
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

func TestDeleteAnnotationByCID(t *testing.T) {
	apih, server := annotationTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id          string
		cid         string
		shouldFail  bool
		expectedErr string
	}{
		{"empty cid", "", true, "invalid annotation CID (none)"},
		{"short cid", "1234", false, ""},
		{"long cid", "/annotation/1234", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteAnnotationByCID(CIDType(&test.cid))
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

func TestSearchAnnotations(t *testing.T) {
	apih, server := annotationTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.Annotation"
	search := SearchQueryType(`(category="updates")`)
	filter := SearchFilterType(map[string][]string{"f__created_gt": {"1483639916"}})

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
			ack, err := apih.SearchAnnotations(test.search, test.filter)
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
