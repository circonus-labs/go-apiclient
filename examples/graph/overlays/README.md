# Test Graph Overlays

This runs a full test on graph overlay types. 

1. Retrieve the specified graph using the Circonus API
1. Modify configuraiton, adding an overlay
1. Update the graph using the modified configuration via the Circonus API

No errors should occur - none from the api, and none parsing the returned graph object

>Note: This is a *LIVE* test. It will contact the Circonus API and the graph will be **modified**, create a temporary graph for the purpose of running these tests.

Example:

```sh
$ go run main.go --key=<circonus_api_key> --app=<circonus_api_key_app_name> --cid=<graph cid, full cid or just uuid portion>
```

Example output:
```
$  go run main.go --key=... --app=... --cid=...
Testing overlay - ID: Kvf1M9 Type: auto_regression -- SUCCESS
Testing overlay - ID: Tvf1M9 Type: graph_comparison -- SUCCESS
Testing overlay - ID: Vvf1M9 Type: autoreg -- SUCCESS
Testing overlay - ID: Wvf1M9 Type: linreg -- SUCCESS
Testing overlay - ID: Rvf1M9 Type: quantileagg -- SUCCESS
Testing overlay - ID: Uvf1M9 Type: expreg -- SUCCESS
Testing overlay - ID: Zvf1M9 Type: sliding_window -- SUCCESS
Testing overlay - ID: Pvf1M9 Type: quantile -- SUCCESS
Testing overlay - ID: Qvf1M9 Type: invquantile -- SUCCESS
Testing overlay - ID: Svf1M9 Type: histogram_agg -- SUCCESS
Testing overlay - ID: Yvf1M9 Type: mvalue -- SUCCESS
Testing overlay - ID: Lvf1M9 Type: linear_regression -- SUCCESS
Testing overlay - ID: Mvf1M9 Type: exponential_regression -- SUCCESS
Testing overlay - ID: Nvf1M9 Type: periodic -- SUCCESS
Testing overlay - ID: Ovf1M9 Type: anomaly_detection -- SUCCESS
Testing overlay - ID: Xvf1M9 Type: SnowthHWFetch -- SUCCESS
```
