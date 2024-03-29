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
	"os"
	"reflect"
	"testing"
)

var (
	testDashboard = Dashboard{}
)

var jsondash = `{
  "_active": true,
  "_cid": "/dashboard/1234",
  "_created": 1483193930,
  "_created_by": "/user/1234",
  "_dashboard_uuid": "01234567-89ab-cdef-0123-456789abcdef",
  "_last_modified": 1483450351,
  "grid_layout": {
    "height": 4,
    "width": 4
  },
  "options": {
    "access_configs": [
    ],
    "fullscreen_hide_title": false,
    "hide_grid": false,
    "linkages": [
    ],
    "scale_text": true,
    "text_size": 16
  },
  "shared": false,
  "title": "foo bar baz",
  "widgets": [
    {
      "active": true,
      "height": 1,
      "name": "Cluster",
      "origin": "d0",
      "settings": {
        "account_id": "1234",
        "algorithm": "cor",
        "cluster_id": 1234,
        "cluster_name": "test",
        "layout": "compact",
        "size": "medium",
        "threshold": 0.7
      },
      "type": "cluster",
      "widget_id": "w4",
      "width": 1
    },
    {
      "active": true,
      "height": 1,
      "name": "HTML",
      "origin": "d1",
      "settings": {
        "markup": "<h1>foo</h1>",
        "title": "html"
      },
      "type": "html",
      "widget_id": "w9",
      "width": 1
    },
    {
      "active": true,
      "height": 1,
      "name": "Chart",
      "origin": "c0",
      "settings": {
        "chart_type": "bar",
        "datapoints": [
          {
            "_check_id": 1234,
            "_metric_type": "numeric",
            "account_id": "1234",
            "label": "Used",
            "metric": "01234567-89ab-cdef-0123-456789abcdef:vm.memory.used"
          },
          {
            "_check_id": 1234,
            "_metric_type": "numeric",
            "account_id": "1234",
            "label": "Free",
            "metric": "01234567-89ab-cdef-0123-456789abcdef:vm.memory.free"
          }
        ],
        "definition": {
          "datasource": "realtime",
          "derive": "gauge",
          "disable_autoformat": false,
          "formula": "",
          "legend": {
            "show": false,
            "type": "html"
          },
          "period": 0,
          "pop_onhover": false,
          "wedge_labels": {
            "on_chart": true,
            "tooltips": false
          },
          "wedge_values": {
            "angle": "0",
            "color": "background",
            "show": true
          }
        },
        "title": "chart graph"
      },
      "type": "chart",
      "widget_id": "w5",
      "width": 1
    },
    {
      "active": true,
      "height": 1,
      "name": "Alerts",
      "origin": "a0",
      "settings": {
        "account_id": "1234",
        "acknowledged": "all",
        "cleared": "all",
        "contact_groups": [
        ],
        "dependents": "all",
        "display": "list",
        "maintenance": "all",
        "min_age": "0",
        "off_hours": [
          17,
          9
        ],
        "search": "",
        "severity": "12345",
        "tag_filter_set": [
        ],
        "time_window": "30M",
        "title": "alerts",
        "week_days": [
          "sun",
          "mon",
          "tue",
          "wed",
          "thu",
          "fri",
          "sat"
        ]
      },
      "type": "alerts",
      "widget_id": "w2",
      "width": 1
    },
    {
      "active": true,
      "height": 1,
      "name": "Graph",
      "origin": "c1",
      "settings": {
        "_graph_title": "foo bar / %Used",
        "account_id": "1234",
        "date_window": "2w",
        "graph_id": "01234567-89ab-cdef-0123-456789abcdef",
        "hide_xaxis": false,
        "hide_yaxis": false,
        "key_inline": false,
        "key_loc": "noop",
        "key_size": 1,
        "key_wrap": false,
        "label": "",
        "overlay_set_id": "",
        "period": 2000,
        "previous_graph_id": "null",
        "realtime": false,
        "show_flags": false
      },
      "type": "graph",
      "widget_id": "w8",
      "width": 1
    },
    {
      "active": true,
      "height": 1,
      "name": "List",
      "origin": "a2",
      "settings": {
        "account_id": "1234",
        "limit": 10,
        "search": "",
        "type": "graph"
      },
      "type": "list",
      "widget_id": "w10",
      "width": 1
    },
    {
      "active": true,
      "height": 1,
      "name": "State",
      "origin": "b2",
      "settings": {
        "caql": "find(\"available\", \"and(dc:atl1,category:management)\") | stats:mean()",
        "good_color": "#6fa428",
        "title": "Test",
        "account_id": "1234",
        "text_align": "left",
        "metric_display_name": "None",
        "display_markup": "Ping Status",
        "show_value": false,
        "layout_style": "inside",
        "bad_rules": [
            {
               "value": "100",
               "color": "#dd9224",
               "criterion": "minimum"
            }
         ],
         "link_url": ""
      },
      "type": "state",
      "widget_id": "w11",
      "width": 1
    },

    {
      "active": true,
      "height": 1,
      "name": "Status",
      "origin": "c2",
      "settings": {
        "account_id": "1234",
        "agent_status_settings": {
          "search": "",
          "show_agent_types": "both",
          "show_contact": false,
          "show_feeds": true,
          "show_setup": false,
          "show_skew": true,
          "show_updates": true
        },
        "content_type": "agent_status",
        "host_status_settings": {
          "layout_style": "grid",
          "search": "",
          "sort_by": "alerts",
          "tag_filter_set": [
          ]
        }
      },
      "type": "status",
      "widget_id": "w12",
      "width": 1
    },
    {
      "active": true,
      "height": 1,
      "name": "Text",
      "origin": "d2",
      "settings": {
        "autoformat": false,
        "body_format": "<p>{metric_name} ({value_type})<br /><strong>{metric_value}</strong><br /><span class=\"date\">{value_date}</span></p>",
        "datapoints": [
          {
            "_cluster_title": "test",
            "_label": "Cluster: test",
            "account_id": "1234",
            "cluster_id": 1234,
            "numeric_only": false
          }
        ],
        "period": 0,
        "title_format": "Metric Status",
        "use_default": true,
        "value_type": "gauge"
      },
      "type": "text",
      "widget_id": "w13",
      "width": 1
    },
    {
      "active": true,
      "height": 1,
      "name": "Chart",
      "origin": "b0",
      "settings": {
        "chart_type": "bar",
        "datapoints": [
          {
            "_cluster_title": "test",
            "_label": "Cluster: test",
            "account_id": "1234",
            "cluster_id": 1234,
            "numeric_only": true
          }
        ],
        "definition": {
          "datasource": "realtime",
          "derive": "gauge",
          "disable_autoformat": false,
          "formula": "",
          "legend": {
            "show": false,
            "type": "html"
          },
          "period": 0,
          "pop_onhover": false,
          "wedge_labels": {
            "on_chart": true,
            "tooltips": false
          },
          "wedge_values": {
            "angle": "0",
            "color": "background",
            "show": true
          }
        },
        "title": "chart metric cluster"
      },
      "type": "chart",
      "widget_id": "w3",
      "width": 1
    },
    {
      "active": true,
      "height": 1,
      "name": "Gauge",
      "origin": "b1",
      "settings": {
        "_check_id": 1234,
        "account_id": "1234",
        "check_uuid": "01234567-89ab-cdef-0123-456789abcdef",
        "disable_autoformat": false,
        "formula": "",
        "metric_display_name": "%Used",
        "metric_name": "fs./foo.df_used_percent",
        "period": 0,
        "range_high": 100,
        "range_low": 0,
        "thresholds": {
          "colors": [
            "#008000",
            "#ffcc00",
            "#ee0000"
          ],
          "flip": false,
          "values": [
            "75%",
            "87.5%"
          ]
        },
        "title": "Metric Gauge",
        "type": "bar",
        "value_type": "gauge"
      },
      "type": "gauge",
      "widget_id": "w7",
      "width": 1
    },
    {
      "active": true,
      "height": 1,
      "name": "Text",
      "origin": "c2",
      "settings": {
        "autoformat": false,
        "body_format": "<p>{metric_name} ({value_type})<br /><strong>{metric_value}</strong><br /><span class=\"date\">{value_date}</span></p>",
        "datapoints": [
          {
            "_check_id": 1234,
            "_metric_type": "numeric",
            "account_id": "1234",
            "label": "cache entries",
            "metric": "01234567-89ab-cdef-0123-456789abcdef:foo.cache_entries"
          },
          {
            "_check_id": 1234,
            "_metric_type": "numeric",
            "account_id": "1234",
            "label": "cache capacity",
            "metric": "01234567-89ab-cdef-0123-456789abcdef:foo.cache_capacity"
          },
          {
            "_check_id": 1234,
            "_metric_type": "numeric",
            "account_id": "1234",
            "label": "cache size",
            "metric": "01234567-89ab-cdef-0123-456789abcdef:foo.cache_size"
          }
        ],
        "period": 0,
        "title_format": "Metric Status",
        "use_default": true,
        "value_type": "gauge"
      },
      "type": "text",
      "widget_id": "w12",
      "width": 1
    },
    {
      "active": true,
      "height": 1,
      "name": "Forecast",
      "origin": "a1",
      "settings": {
        "format": "standard",
        "resource_limit": "0",
        "resource_usage": "metric:average(\"01234567-89ab-cdef-0123-456789abcdef\",p\"fs%60/foo%60df_used_percent\")",
        "thresholds": {
          "colors": [
            "#008000",
            "#ffcc00",
            "#ee0000"
          ],
          "values": [
            "1d",
            "1h"
          ]
        },
        "title": "Resource Forecast",
        "trend": "auto"
      },
      "type": "forecast",
      "widget_id": "w6",
      "width": 1
    }
  ]
}
`

func init() {
	err := json.Unmarshal([]byte(jsondash), &testDashboard)
	if err != nil {
		fmt.Printf("Unable to unmarshal inline json (%s)\n", err)
		os.Exit(1)
	}
}

func testDashboardServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/dashboard/1234":
			switch r.Method {
			case "GET":
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, jsondash)
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
		case "/dashboard":
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []Dashboard
				switch reqURL {
				case "/dashboard?search=my+dashboard":
					c = []Dashboard{testDashboard}
				case "/dashboard?f__created_gt=1483639916":
					c = []Dashboard{testDashboard}
				case "/dashboard?f__created_gt=1483639916&search=my+dashboard":
					c = []Dashboard{testDashboard}
				case "/dashboard":
					c = []Dashboard{testDashboard}
				default:
					c = []Dashboard{}
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
				ret, err := json.Marshal(testDashboard)
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

func dashboardTestBootstrap(t *testing.T) (*API, *httptest.Server) {
	server := testDashboardServer()

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

func TestNewDashboard(t *testing.T) {
	dashboard := NewDashboard()
	if reflect.TypeOf(dashboard).String() != "*apiclient.Dashboard" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(dashboard).String())
	}
}

func TestFetchDashboard(t *testing.T) {
	apih, server := dashboardTestBootstrap(t)
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
			expectedErr:  "invalid dashboard CID (none)",
		},
		{
			id:           "short cid",
			cid:          "1234",
			expectedType: "*apiclient.Dashboard",
			shouldFail:   false,
		},
		{
			id:           "long cid",
			cid:          "/dashboard/1234",
			expectedType: "*apiclient.Dashboard",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			dash, err := apih.FetchDashboard(CIDType(&test.cid))
			if test.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				} else if err.Error() != test.expectedErr {
					t.Fatalf("unexpected error (%s)", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error (%s)", err)
				} else if reflect.TypeOf(dash).String() != test.expectedType {
					t.Fatalf("unexpected type (%s)", reflect.TypeOf(dash).String())
				} /* else {
					data, err := json.MarshalIndent(dash, "", "  ")
					if err != nil {
						t.Fatalf("unexpected error marshing (%s)", err)
					}
					fmt.Printf("%s\n", string(data))
				}*/
			}
		})
	}
}

func TestFetchDashboards(t *testing.T) {
	apih, server := dashboardTestBootstrap(t)
	defer server.Close()

	dashboards, err := apih.FetchDashboards()
	if err != nil {
		t.Fatalf("unexpected error (%s)", err)
	}

	if reflect.TypeOf(dashboards).String() != "*[]apiclient.Dashboard" {
		t.Fatalf("unexpected type (%s)", reflect.TypeOf(dashboards).String())
	}
}

func TestUpdateDashboard(t *testing.T) {
	apih, server := dashboardTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg         *Dashboard
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			cfg:         nil,
			shouldFail:  true,
			expectedErr: "invalid dashboard config (nil)",
		},
		{
			id:          "invalid (cid)",
			cfg:         &Dashboard{CID: "/invalid"},
			shouldFail:  true,
			expectedErr: "invalid dashboard CID (/invalid)",
		},
		{
			id:         "valid",
			cfg:        &testDashboard,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			_, err := apih.UpdateDashboard(test.cfg)
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

func TestCreateDashboard(t *testing.T) {
	apih, server := dashboardTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg          *Dashboard
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
			expectedErr:  "invalid dashboard config (nil)",
		},
		{
			id:           "valid",
			cfg:          &testDashboard,
			expectedType: "*apiclient.Dashboard",
			shouldFail:   false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.CreateDashboard(test.cfg)
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

func TestDeleteDashboard(t *testing.T) {
	apih, server := dashboardTestBootstrap(t)
	defer server.Close()

	tests := []struct {
		cfg         *Dashboard
		id          string
		expectedErr string
		shouldFail  bool
	}{
		{
			id:          "invalid (nil)",
			cfg:         nil,
			shouldFail:  true,
			expectedErr: "invalid dashboard config (nil)",
		},
		{
			id:         "valid",
			cfg:        &testDashboard,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteDashboard(test.cfg)
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

func TestDeleteDashboardByCID(t *testing.T) {
	apih, server := dashboardTestBootstrap(t)
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
			expectedErr: "invalid dashboard CID (none)",
		},
		{
			id:         "short cid",
			cid:        "1234",
			shouldFail: false,
		},
		{
			id:         "long cid",
			cid:        "/dashboard/1234",
			shouldFail: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			wasDeleted, err := apih.DeleteDashboardByCID(CIDType(&test.cid))
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

func TestSearchDashboards(t *testing.T) {
	apih, server := dashboardTestBootstrap(t)
	defer server.Close()

	expectedType := "*[]apiclient.Dashboard"
	search := SearchQueryType("my dashboard")
	filter := SearchFilterType(map[string][]string{"f__created_gt": {"1483639916"}})

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
			ack, err := apih.SearchDashboards(test.search, test.filter)
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
