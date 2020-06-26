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
	testOutlierReport = OutlierReport{
		CID:              "/outlier_report/1234",
		Created:          1483033102,
		CreatedBy:        "/user/1234",
		LastModified:     1483033102,
		LastModifiedBy:   "/user/1234",
		Config:           "",
		MetricClusterCID: "/metric_cluster/1234",
		Tags:             []string{"cat:tag"},
		Title:            "foo bar",
	}
)

func testOutlierReportServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/outlier_report/1234":
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testOutlierReport)
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
		case "/outlier_report":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []OutlierReport
				switch reqURL {
				case "/outlier_report?search=requests+per+second":
					c = []OutlierReport{testOutlierReport}
				case "/outlier_report?f_tags_has=service%3Aweb":
					c = []OutlierReport{testOutlierReport}
				case "/outlier_report?f_tags_has=service%3Aweb&search=requests+per+second":
					c = []OutlierReport{testOutlierReport}
				case "/outlier_report":
					c = []OutlierReport{testOutlierReport}
				default:
					c = []OutlierReport{}
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
				ret, err := json.Marshal(testOutlierReport)
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

func outlierReportTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testOutlierReportServer()

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

func TestNewOutlierReport(t *testing.T) {
	outlierReport := NewOutlierReport()
	if reflect.TypeOf(outlierReport).String() != "*apiclient.OutlierReport" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(outlierReport).String())
	}
}

func TestFetchOutlierReport(t *testing.T) {
	apih, server := outlierReportTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cid          string
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", true, "invalid outlier report CID (none)"},
		{"short cid", "1234", "*apiclient.OutlierReport", false, ""},
		{"long cid", "/outlier_report/1234", "*apiclient.OutlierReport", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchOutlierReport(CIDType(&test.cid))
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

func TestFetchOutlierReports(t *testing.T) {
	apih, server := outlierReportTestBootstrap(t)
	defer server.Close()

	reports, err := apih.FetchOutlierReports()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(reports).String() != "*[]apiclient.OutlierReport" {
		t.Fatalf("unexpected tyep (%s)", reflect.TypeOf(reports).String())
	}

}

func TestUpdateOutlierReport(t *testing.T) {
	apih, server := outlierReportTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cfg          *OutlierReport
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid outlier report config (nil)"},
		{"invalid (cid)", &OutlierReport{CID: "/invalid"}, "", true, "invalid outlier report CID (/invalid)"},
		{"valid", &testOutlierReport, "*apiclient.OutlierReport", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			maint, err := apih.UpdateOutlierReport(test.cfg)
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

func TestCreateOutlierReport(t *testing.T) {
	apih, server := outlierReportTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id           string
		cfg          *OutlierReport
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"invalid (nil)", nil, "", true, "invalid outlier report config (nil)"},
		{"valid", &testOutlierReport, "*apiclient.OutlierReport", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.CreateOutlierReport(test.cfg)
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

func TestDeleteOutlierReport(t *testing.T) {
	apih, server := outlierReportTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id          string
		cfg         *OutlierReport
		shouldFail  bool
		expectedErr string
	}{
		{"invalid (nil)", nil, true, "invalid outlier report config (nil)"},
		{"valid", &testOutlierReport, false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteOutlierReport(test.cfg)
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

func TestDeleteOutlierReportByCID(t *testing.T) {
	apih, server := outlierReportTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		id          string
		cid         string
		shouldFail  bool
		expectedErr string
	}{
		{"empty cid", "", true, "invalid outlier report CID (none)"},
		{"short cid", "1234", false, ""},
		{"long cid", "/outlier_report/1234", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteOutlierReportByCID(CIDType(&test.cid))
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

func TestSearchOutlierReports(t *testing.T) {
	apih, server := outlierReportTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.OutlierReport"
	search := SearchQueryType("requests per second")
	filter := SearchFilterType(map[string][]string{"f_tags_has": {"service:web"}})

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
			ack, err := apih.SearchOutlierReports(test.search, test.filter)
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
