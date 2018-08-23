package vulcanizer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

type ServerSetup struct {
	Method, Path, Body, Response string
	HttpStatus                   int
}

func setupTestServers(t *testing.T, setups []*ServerSetup) (string, int, *httptest.Server) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		requestBytes, _ := ioutil.ReadAll(r.Body)
		requestBody := string(requestBytes)

		matched := false
		for _, setup := range setups {
			if r.Method == setup.Method && r.URL.EscapedPath() == setup.Path && requestBody == setup.Body {
				matched = true
				if setup.HttpStatus == 0 {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(setup.HttpStatus)
				}
				w.Write([]byte(setup.Response))
			}
		}

		if !matched {
			t.Fatalf("No requests matched setup. Got method %s, Path %s, body %s", r.Method, r.URL.EscapedPath(), requestBody)
		}
	}))
	url, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(url.Port())
	return url.Hostname(), port, ts
}

func TestGetClusterExcludeSettings(t *testing.T) {

	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/settings",
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_host":"excluded.host","_name":"excluded_name","_ip":"10.0.0.99"}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	excludeSettings := GetClusterExcludeSettings(host, port)

	if excludeSettings.Ips[0] != "10.0.0.99" || len(excludeSettings.Ips) != 1 {
		t.Fatalf("Expected 10.0.0.99 for excluded ip, got %s", excludeSettings.Ips)
	}

	if excludeSettings.Names[0] != "excluded_name" || len(excludeSettings.Names) != 1 {
		t.Fatalf("Expected excluded_name for excluded name, got %s", excludeSettings.Names)
	}

	if excludeSettings.Hosts[0] != "excluded.host" || len(excludeSettings.Hosts) != 1 {
		t.Fatalf("Expected excluded.host for excluded host, got %s", excludeSettings.Hosts)
	}
}

func TestDrainServer_OneValue(t *testing.T) {

	testSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_cluster/settings",
		Body:     `{"transient":{"cluster.routing.allocation.exclude._name":"server_to_drain"}}`,
		Response: `{"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"server_to_drain"}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	excludedServers := DrainServer(host, port, "server_to_drain", "None")

	if excludedServers != "server_to_drain" {
		t.Fatalf("Expected response server_to_drain, got %s", excludedServers)
	}
}

func TestDrainServer_ExistingValues(t *testing.T) {

	testSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_cluster/settings",
		Body:     `{"transient":{"cluster.routing.allocation.exclude._name":"server_to_drain,existing_one,existing_two"}}`,
		Response: `{"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"server_to_drain,existing_one,existing_two"}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	excludedServers := DrainServer(host, port, "server_to_drain", "existing_one,existing_two")

	if excludedServers != "server_to_drain,existing_one,existing_two" {
		t.Fatalf("unexpected response, got %s", excludedServers)
	}
}

func TestFillOneServer_ExistingServers(t *testing.T) {

	getSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/settings",
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"excluded_server1,good_server,excluded_server2"}}}}}}`,
	}

	putSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_cluster/settings",
		Body:     `{"transient":{"cluster.routing.allocation.exclude._name":"excluded_server1,excluded_server2"}}`,
		Response: `{"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"excluded_server1,excluded_server2"}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, putSetup})
	defer ts.Close()

	goodServer, excludedServers := FillOneServer(host, port, "good_server")

	if goodServer != "good_server" {
		t.Fatalf("unexpected response, got %s", goodServer)
	}

	if excludedServers != "excluded_server1,excluded_server2" {
		t.Fatalf("unexpected response, got %s", excludedServers)
	}
}

func TestFillOneServer_OneServer(t *testing.T) {

	getSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/settings",
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"good_server"}}}}}}`,
	}

	putSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_cluster/settings",
		Body:     `{"transient":{"cluster.routing.allocation.exclude._name":""}}`,
		Response: `{"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":""}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, putSetup})
	defer ts.Close()

	goodServer, excludedServers := FillOneServer(host, port, "good_server")

	if goodServer != "good_server" {
		t.Fatalf("unexpected response, got %s", goodServer)
	}

	if excludedServers != "" {
		t.Fatalf("unexpected response, got %s", excludedServers)
	}
}

func TestFillAll(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_cluster/settings",
		Body:     `{"transient":{"cluster.routing.allocation.exclude":{"_host":"","_ip":"","_name":""}}}`,
		Response: `{"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"", "_ip": "", "_host": ""}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	excludeSettings := FillAll(host, port)

	if len(excludeSettings.Ips) != 0 {
		t.Fatalf("Expected empty excluded Ips, got %s", excludeSettings.Ips)
	}

	if len(excludeSettings.Names) != 0 {
		t.Fatalf("Expected empty excluded Names, got %s", excludeSettings.Names)
	}

	if len(excludeSettings.Hosts) != 0 {
		t.Fatalf("Expected empty excluded Hosts, got %s", excludeSettings.Hosts)
	}
}

func TestGetNodes(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/nodes",
		Response: `[{"master": "*", "role": "d", "name": "foo", "ip": "127.0.0.1", "id": "abc"}]`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	nodes, headers := GetNodes(host, port)

	if len(headers) != 5 {
		t.Fatalf("Unexpected headers, got %s", headers)
	}

	if nodes[0][2] != "foo" && nodes[0][1] == "d" {
		t.Fatalf("Unexpected node name, got %s", nodes)
	}
}

func TestGetIndices(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/indices",
		Response: `[{"health":"yellow","status":"open","index":"index1","pri":"5","rep":"1","store.size":"3.6kb", "docs.count":"1500"}]`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	indices, headers := GetIndices(host, port)

	if len(headers) != 7 {
		t.Fatalf("Unexpected headers, got %s", headers)
	}

	if indices[0][2] != "index1" || indices[0][5] != "3.6kb" || indices[0][6] != "1500" {
		t.Fatalf("Unexpected index name, got %s", indices)
	}
}

func TestGetHealth(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/health",
		Response: `[{"cluster":"elasticsearch_nickcanz","status":"yellow","relo":"0","init":"0","unassign":"5","pending_tasks":"0","active_shards_percent":"50.0%"}]`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	caption, health, headers := GetHealth(host, port)

	if len(caption) < 1 {
		t.Fatalf("No caption, got %s", caption)

	}

	if len(headers) != 5 {
		t.Fatalf("Unexpected headers, got %s , length: %d", headers, len(headers))
	}

	if health[0][0] != "yellow" {
		t.Fatalf("Unexpected cluster status, got %s", health)
	}
}

func TestGetSettings(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/settings",
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_host":"","_name":"10.0.0.2","_ip":""}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	settings, headers := GetSettings(host, port)

	if len(headers) != 2 {
		t.Fatalf("Unexpected headers, got %s", headers)
	}

	if len(settings) != 1 {
		t.Fatalf("Unexpected settings, got %s", settings)
	}

	if settings[0][0] != "transient.cluster.routing.allocation.exclude._name" || settings[0][1] != "10.0.0.2" {
		t.Fatalf("Unexpected settings, got %s", settings)
	}
}

func TestSetSetting_ExistingTransientSetting(t *testing.T) {
	getSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/settings",
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.2"}}}}}}`,
	}
	putSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_cluster/settings",
		Body:     `{"transient":{"cluster.routing.allocation.exclude._name":"10.0.0.99"}}`,
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.99"}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, putSetup})
	defer ts.Close()

	oldValue, newValue, err := SetSetting(host, port, "cluster.routing.allocation.exclude._name", "10.0.0.99")

	if err != nil {
		t.Fatalf("Expected error to be nil, %s", err)
	}

	if oldValue != "10.0.0.2" {
		t.Fatalf("Unexpected old value, got %s", oldValue)
	}

	if newValue != "10.0.0.99" {
		t.Fatalf("Unexpected new value, got %s", newValue)
	}
}

func TestSetSetting_ExistingPersistentSetting(t *testing.T) {
	getSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/settings",
		Response: `{"transient":{},"persistent":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.2"}}}}}}`,
	}
	putSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_cluster/settings",
		Body:     `{"transient":{"cluster.routing.allocation.exclude._name":"10.0.0.99"}}`,
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.99"}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, putSetup})
	defer ts.Close()

	oldValue, newValue, err := SetSetting(host, port, "cluster.routing.allocation.exclude._name", "10.0.0.99")

	if err != nil {
		t.Fatalf("Expected error to be nil, %s", err)
	}

	if oldValue != "10.0.0.2" {
		t.Fatalf("Unexpected old value, got %s", oldValue)
	}

	if newValue != "10.0.0.99" {
		t.Fatalf("Unexpected new value, got %s", newValue)
	}
}

func TestSetSetting_NoExistingSetting(t *testing.T) {
	getSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/settings",
		Response: `{"transient":{},"persistent":{}}`,
	}
	putSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_cluster/settings",
		Body:     `{"transient":{"cluster.routing.allocation.exclude._name":"10.0.0.99"}}`,
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.99"}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, putSetup})
	defer ts.Close()

	oldValue, newValue, err := SetSetting(host, port, "cluster.routing.allocation.exclude._name", "10.0.0.99")

	if err != nil {
		t.Fatalf("Expected error to be nil, %s", err)
	}

	if oldValue != "" {
		t.Fatalf("Unexpected old value, got %s", oldValue)
	}

	if newValue != "10.0.0.99" {
		t.Fatalf("Unexpected new value, got %s", newValue)
	}
}

func TestSetSetting_BadRequest(t *testing.T) {
	getSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/settings",
		Response: `{"transient":{},"persistent":{}}`,
	}
	putSetup := &ServerSetup{
		Method:     "PUT",
		Path:       "/_cluster/settings",
		Body:       `{"transient":{"cluster.routing.allocation.enable":"foo"}}`,
		HttpStatus: http.StatusBadRequest,
		Response:   `{"error":{"root_cause":[{"type":"illegal_argument_exception","reason":"Illegal allocation.enable value [FOO]"}],"type":"illegal_argument_exception","reason":"Illegal allocation.enable value [FOO]"},"status":400}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, putSetup})
	defer ts.Close()

	_, _, err := SetSetting(host, port, "cluster.routing.allocation.enable", "foo")

	if err == nil {
		t.Fatalf("Expected error to not be nil, %s", err)
	}

	if err.Error() != fmt.Sprintf("Bad HTTP Status of 400 from Elasticsearch: %s", putSetup.Response) {
		t.Fatalf("Unexpected error message, %s", err)
	}
}

func TestSetAllocation_Enable(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_cluster/settings",
		Body:     `{"transient":{"cluster.routing.allocation.enable":"all"}}`,
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"enable": "all"}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	resp := SetAllocation(host, port, "enable")

	if resp != "all" {
		t.Fatalf("Unexpected response, got %s", resp)
	}
}

func TestSetAllocation_Disable(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_cluster/settings",
		Body:     `{"transient":{"cluster.routing.allocation.enable":"none"}}`,
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"enable": "none"}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	resp := SetAllocation(host, port, "disable")

	if resp != "none" {
		t.Fatalf("Unexpected response, got %s", resp)
	}
}

func TestGetSnapshots(t *testing.T) {
	testSetup := &ServerSetup{
		Method: "GET",
		Path:   "/_snapshot/octocat/_all",
		Response: `{
  "snapshots": [
    {
      "snapshot": "snapshot1",
      "uuid": "kXx-r58tSOeVvDbvCC1IsQ",
      "version_id": 5060699,
      "version": "5.6.6",
      "indices": [ "index1", "index2" ],
      "state": "SUCCESS",
      "start_time": "2018-04-03T06:06:24.837Z",
      "start_time_in_millis": 1522735584837,
      "end_time": "2018-04-03T07:41:01.719Z",
      "end_time_in_millis": 1522741261719,
      "duration_in_millis": 1000,
      "failures": [],
      "shards": { "total": 93, "failed": 0, "successful": 93 }
    },
    {
      "snapshot": "snapshot2",
      "uuid": "ReLFDkUfQcysi6HG2y40uw",
      "version_id": 5060699,
      "version": "5.6.6",
      "indices": [ "index1", "index2" ],
      "state": "SUCCESS",
      "start_time": "2018-04-03T18:13:11.012Z",
      "start_time_in_millis": 1522779191012,
      "end_time": "2018-04-03T18:25:58.440Z",
      "end_time_in_millis": 1522779958440,
      "duration_in_millis": 500,
      "failures": [],
      "shards": { "total": 93, "failed": 0, "successful": 93 }
    }
  ]
}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	snapshots, headers := GetSnapshots(host, port, "octocat")

	if len(headers) != 4 {
		t.Fatalf("Unexpected headers, got %s", headers)
	}

	if len(snapshots) != 2 {
		t.Fatalf("Unexpected snapshots, got %s", snapshots)
	}

	if snapshots[0][0] != "SUCCESS" || snapshots[0][1] != "snapshot1" ||
		snapshots[0][2] != "2018-04-03T07:41:01.719Z" ||
		snapshots[0][3] != "1s" || snapshots[1][3] != "500ms" {
		t.Fatalf("Unexpected snapshots, got %s", snapshots)
	}
}

func TestGetSnapshots_PartialSnapshot(t *testing.T) {
	testSetup := &ServerSetup{
		Method: "GET",
		Path:   "/_snapshot/octocat/_all",
		Response: `{
  "snapshots": [
    {
      "snapshot": "snapshot1",
      "uuid": "kXx-r58tSOeVvDbvCC1IsQ",
      "version_id": 5060699,
      "version": "5.6.6",
      "indices": [ "index1", "index2" ],
      "state": "IN_PROGRESS",
      "start_time": "2018-04-03T06:06:24.837Z",
      "start_time_in_millis": 1522735584837,
      "duration_in_millis": 3600000,
      "failures": [],
      "shards": { "total": 93, "failed": 0, "successful": 93 }
    }
  ]
}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	snapshots, headers := GetSnapshots(host, port, "octocat")

	if len(headers) != 4 {
		t.Fatalf("Unexpected headers, got %s", headers)
	}

	if len(snapshots) != 1 {
		t.Fatalf("Unexpected snapshots, got %s", snapshots)
	}

	if snapshots[0][0] != "IN_PROGRESS" || snapshots[0][1] != "snapshot1" ||
		snapshots[0][2] != "" ||
		snapshots[0][3] != "1h0m0s" {
		t.Fatalf("Unexpected snapshots, got %s", snapshots)
	}
}

func TestGetSnapshots_Last10(t *testing.T) {
	testSetup := &ServerSetup{
		Method: "GET",
		Path:   "/_snapshot/octocat/_all",
		Response: `{

  "snapshots": [
    { "snapshot": "snapshot1", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot2", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot3", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot4", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot5", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot6", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot7", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot8", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot9", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot10", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot11", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot12", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot13", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot14", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot15", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
    { "snapshot": "snapshot16", "state": "SUCCESS", "end_time": "2018-04-03T18:25:58.440Z", "end_time_in_millis": 1522779958440, "duration_in_millis": 500 },
  ]
}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	snapshots, headers := GetSnapshots(host, port, "octocat")

	if len(headers) != 4 {
		t.Fatalf("Unexpected headers, got %s", headers)
	}

	if len(snapshots) != 10 {
		t.Fatalf("Unexpected snapshots, got %s", snapshots)
	}

	if snapshots[0][1] != "snapshot7" ||
		snapshots[9][1] != "snapshot16" {
		t.Fatalf("Unexpected snapshots, got %s", snapshots)
	}
}

func TestGetSnapshotStatus(t *testing.T) {
	testSetup := &ServerSetup{
		Method: "GET",
		Path:   "/_snapshot/octocat/snapshot1",
		Response: `{
  "snapshots": [
    {
      "snapshot": "snapshot1",
      "uuid": "kXx-r58tSOeVvDbvCC1IsQ",
      "version_id": 5060699,
      "version": "5.6.6",
      "indices": [ "index1", "index2" ],
      "state": "SUCCESS",
      "start_time": "2018-04-03T06:06:24.837Z",
      "start_time_in_millis": 1522735584837,
      "end_time": "2018-04-03T07:41:01.719Z",
      "end_time_in_millis": 1522741261719,
      "duration_in_millis": 1000,
      "failures": [],
      "shards": { "total": 93, "failed": 0, "successful": 93 }
    }
  ]
}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	snapshot, headers := GetSnapshotStatus(host, port, "octocat", "snapshot1")

	if len(headers) != 2 {
		t.Fatalf("Unexpected headers, got %s", headers)
	}

	if len(snapshot) != 7 {
		t.Fatalf("Unexpected snapshots, got %s", snapshot)
	}

	if snapshot[0][0] != "state" || snapshot[0][1] != "SUCCESS" {
		t.Fatalf("Unexpected state, got %s", snapshot[0])
	}

	if snapshot[1][0] != "snapshot" || snapshot[1][1] != "snapshot1" {
		t.Fatalf("Unexpected snapshot name, got %s", snapshot[1])
	}

	if snapshot[2][0] != "indices" || snapshot[2][1] != "index1\nindex2" {
		t.Fatalf("Unexpected indices, got %s", snapshot[2][1])
	}

	if snapshot[5][0] != "duration" || snapshot[5][1] != "1s" {
		t.Fatalf("Unexpected indices, got %s", snapshot[5])
	}
}

func TestPerformSnapshotCheck(t *testing.T) {
	testSetup := &ServerSetup{
		Method: "GET",
		Path:   "/_snapshot/octocat/_all",
		Response: `{
  "snapshots": [
    {
      "snapshot": "snapshot1",
      "uuid": "kXx-r58tSOeVvDbvCC1IsQ",
      "version_id": 5060699,
      "version": "5.6.6",
      "indices": [ "index1", "index2" ],
      "state": "SUCCESS",
      "start_time": "2018-04-03T06:06:24.837Z",
      "start_time_in_millis": 1522735584837,
      "end_time": "2018-04-03T07:41:01.719Z",
      "end_time_in_millis": 1522741261719,
      "duration_in_millis": 1000,
      "failures": [],
      "shards": { "total": 93, "failed": 0, "successful": 93 }
    },
    {
      "snapshot": "snapshot2",
      "uuid": "ReLFDkUfQcysi6HG2y40uw",
      "version_id": 5060699,
      "version": "5.6.6",
      "indices": [ "index1", "index2" ],
      "state": "SUCCESS",
      "start_time": "2018-04-03T18:13:11.012Z",
      "start_time_in_millis": 1522779191012,
      "end_time": "2018-04-03T18:25:58.440Z",
      "end_time_in_millis": 1522779958440,
      "duration_in_millis": 500,
      "failures": [],
      "shards": { "total": 93, "failed": 0, "successful": 93 }
    }
  ]
}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	good, bad := PerformSnapshotsCheck(host, port, "octocat")

	if len(good) != 2 {
		t.Fatalf("Unexpected good snapshots, got %s", good)
	}

	if len(bad) != 0 {
		t.Fatalf("Unexpected bad snapshots, got %s", bad)
	}

	if good[0] != "snapshot1" || good[1] != "snapshot2" {
		t.Fatalf("Unexpected snapshots, got %s", good)
	}
}

func TestPerformSnapshotCheck_SomeFailing(t *testing.T) {
	testSetup := &ServerSetup{
		Method: "GET",
		Path:   "/_snapshot/octocat/_all",
		Response: `{
  "snapshots": [
    {
      "snapshot": "snapshot1",
      "uuid": "kXx-r58tSOeVvDbvCC1IsQ",
      "version_id": 5060699,
      "version": "5.6.6",
      "indices": [ "index1", "index2" ],
      "state": "FAILED",
      "start_time": "2018-04-03T06:06:24.837Z",
      "start_time_in_millis": 1522735584837,
      "end_time": "2018-04-03T07:41:01.719Z",
      "end_time_in_millis": 1522741261719,
      "duration_in_millis": 1000,
      "failures": [],
      "shards": { "total": 93, "failed": 0, "successful": 93 }
    },
    {
      "snapshot": "snapshot2",
      "uuid": "ReLFDkUfQcysi6HG2y40uw",
      "version_id": 5060699,
      "version": "5.6.6",
      "indices": [ "index1", "index2" ],
      "state": "SUCCESS",
      "start_time": "2018-04-03T18:13:11.012Z",
      "start_time_in_millis": 1522779191012,
      "end_time": "2018-04-03T18:25:58.440Z",
      "end_time_in_millis": 1522779958440,
      "duration_in_millis": 500,
      "failures": [],
      "shards": { "total": 93, "failed": 0, "successful": 93 }
    }
  ]
}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	good, bad := PerformSnapshotsCheck(host, port, "octocat")

	if len(good) != 1 {
		t.Fatalf("Unexpected good snapshots, got %s", good)
	}

	if len(bad) != 1 {
		t.Fatalf("Unexpected bad snapshots, got %s", bad)
	}

	if bad[0] != "snapshot1" || good[0] != "snapshot2" {
		t.Fatalf("Unexpected snapshots, good: %s bad: %s", good, bad)
	}
}