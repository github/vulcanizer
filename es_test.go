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

// ServerSetup type contains the Method, Path, Body and Response strings, as well as the HTTP Status code.
type ServerSetup struct {
	Method, Path, Body, Response string
	HTTPStatus                   int
}

// setupTestServer sets up an HTTP test server to serve data to the test that come after it.

func setupTestServers(t *testing.T, setups []*ServerSetup) (string, int, *httptest.Server) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		requestBytes, _ := ioutil.ReadAll(r.Body)
		requestBody := string(requestBytes)

		matched := false
		for _, setup := range setups {
			if r.Method == setup.Method && r.URL.EscapedPath() == setup.Path && requestBody == setup.Body {
				matched = true
				if setup.HTTPStatus == 0 {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(setup.HTTPStatus)
				}
				_, err := w.Write([]byte(setup.Response))
				if err != nil {
					t.Fatalf("Unable to write test server response: %v", err)
				}
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

	client := NewClient(host, port)

	excludeSettings := client.GetClusterExcludeSettings()

	if excludeSettings.Ips[0] != "10.0.0.99" || len(excludeSettings.Ips) != 1 {
		t.Errorf("Expected 10.0.0.99 for excluded ip, got %s", excludeSettings.Ips)
	}

	if excludeSettings.Names[0] != "excluded_name" || len(excludeSettings.Names) != 1 {
		t.Errorf("Expected excluded_name for excluded name, got %s", excludeSettings.Names)
	}

	if excludeSettings.Hosts[0] != "excluded.host" || len(excludeSettings.Hosts) != 1 {
		t.Errorf("Expected excluded.host for excluded host, got %s", excludeSettings.Hosts)
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
	client := NewClient(host, port)

	excludedServers := client.DrainServer("server_to_drain", "None")

	if excludedServers != "server_to_drain" {
		t.Errorf("Expected response server_to_drain, got %s", excludedServers)
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
	client := NewClient(host, port)

	excludedServers := client.DrainServer("server_to_drain", "existing_one,existing_two")

	if excludedServers != "server_to_drain,existing_one,existing_two" {
		t.Errorf("unexpected response, got %s", excludedServers)
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
	client := NewClient(host, port)

	goodServer, excludedServers := client.FillOneServer("good_server")

	if goodServer != "good_server" {
		t.Errorf("unexpected response, got %s", goodServer)
	}

	if excludedServers != "excluded_server1,excluded_server2" {
		t.Errorf("unexpected response, got %s", excludedServers)
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
	client := NewClient(host, port)

	goodServer, excludedServers := client.FillOneServer("good_server")

	if goodServer != "good_server" {
		t.Errorf("unexpected response, got %s", goodServer)
	}

	if excludedServers != "" {
		t.Errorf("unexpected response, got %s", excludedServers)
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
	client := NewClient(host, port)

	excludeSettings := client.FillAll()

	if len(excludeSettings.Ips) != 0 {
		t.Errorf("Expected empty excluded Ips, got %s", excludeSettings.Ips)
	}

	if len(excludeSettings.Names) != 0 {
		t.Errorf("Expected empty excluded Names, got %s", excludeSettings.Names)
	}

	if len(excludeSettings.Hosts) != 0 {
		t.Errorf("Expected empty excluded Hosts, got %s", excludeSettings.Hosts)
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
	client := NewClient(host, port)

	nodes, err := client.GetNodes()

	if err != nil {
		t.Errorf("Unexpected error expected nil, got %s", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Unexpected nodes, got %s", nodes)
	}

	if nodes[0].Name != "foo" {
		t.Errorf("Unexpected node name, expected foo, got %s", nodes[0].Name)
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
	client := NewClient(host, port)

	indices, err := client.GetIndices()

	if err != nil {
		t.Errorf("Unexpected error expected nil, got %s", err)
	}

	if len(indices) != 1 {
		t.Errorf("Unexpected indices, got %v", indices)
	}

	if indices[0].Health != "yellow" || indices[0].ReplicaCount != 1 || indices[0].DocumentCount != 1500 {
		t.Errorf("Unexpected index values, got %v", indices[0])
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
	client := NewClient(host, port)

	caption, health, headers := client.GetHealth()

	if len(caption) < 1 {
		t.Errorf("No caption, got %s", caption)

	}

	if len(headers) != 5 {
		t.Errorf("Unexpected headers, got %s , length: %d", headers, len(headers))
	}

	if health[0][0] != "yellow" {
		t.Errorf("Unexpected cluster status, got %s", health)
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
	client := NewClient(host, port)

	settings, headers := client.GetSettings()

	if len(headers) != 2 {
		t.Errorf("Unexpected headers, got %s", headers)
	}

	if len(settings) != 1 {
		t.Errorf("Unexpected settings, got %s", settings)
	}

	if settings[0][0] != "transient.cluster.routing.allocation.exclude._name" || settings[0][1] != "10.0.0.2" {
		t.Errorf("Unexpected settings, got %s", settings)
	}
}

// TestSetSetting Func is an integration test for all things that use the SetSetting functionality.
func TestSetSettings(t *testing.T) {

	tt := []struct {
		Name        string
		Method      string
		Body        string
		GetResponse string
		PutResponse string
		Setting     string
		SetValue    string
		HTTPStatus  int
		OldValue    string
	}{
		{
			// Tests for behavior with existing transient setting.
			Name:        "Existing Transient Setting",
			GetResponse: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.2"}}}}}}`,
			Body:        `{"transient":{"cluster.routing.allocation.exclude._name":"10.0.0.99"}}`,
			PutResponse: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.99"}}}}}}`,
			Setting:     "cluster.routing.allocation.exclude._name",
			SetValue:    "10.0.0.99",
			OldValue:    "10.0.0.2",
		},

		{
			// Tests for behavior with existing persistent settings.
			Name:        "Existing Persistent Setting",
			GetResponse: `{"transient":{},"persistent":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.2"}}}}}}`,
			Body:        `{"transient":{"cluster.routing.allocation.exclude._name":"10.0.0.99"}}`,
			PutResponse: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.99"}}}}}}`,
			Setting:     "cluster.routing.allocation.exclude._name",
			SetValue:    "10.0.0.99",
			OldValue:    "10.0.0.2",
		},

		{
			// Tests for behavior with NO existing persistent settings.
			Name:        "No existing settings",
			GetResponse: `{"transient":{},"persistent":{}}`,
			Body:        `{"transient":{"cluster.routing.allocation.exclude._name":"10.0.0.99"}}`,
			PutResponse: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.99"}}}}}}`,
			Setting:     "cluster.routing.allocation.exclude._name",
			SetValue:    "10.0.0.99",
			OldValue:    "",
		},
	}

	for _, x := range tt {
		t.Run(x.Name, func(st *testing.T) {

			getSetup := &ServerSetup{
				Method:   "GET",
				Path:     "/_cluster/settings",
				Response: x.GetResponse,
			}
			putSetup := &ServerSetup{
				Method:   "PUT",
				Path:     "/_cluster/settings",
				Body:     x.Body,
				Response: x.PutResponse,
			}

			host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, putSetup})
			defer ts.Close()
			client := NewClient(host, port)

			oldSetting, newSetting, err := client.SetSetting(x.Setting, x.SetValue)

			if err != nil {
				st.Errorf("Expected error to be nil, %s", err)
			}

			if oldSetting != x.OldValue {
				st.Errorf("Unexpected old value, got %s", oldSetting)
			}

			if newSetting != "10.0.0.99" {
				st.Errorf("Unexpected new value, got %s", newSetting)
			}

		})
	}
}

// TestSetSetting Func is an integration test for all things that use the SetAllocation functionality.
func TestAllocationSettings(t *testing.T) {

	tt := []struct {
		Name     string
		Path     string
		Body     string
		Response string
		Setting  string
		Expected string
	}{
		{
			// Allocation Enable.
			Name:     "Allocation Enable",
			Body:     `{"transient":{"cluster.routing.allocation.enable":"all"}}`,
			Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"enable": "all"}}}}}`,
			Setting:  "enable",
			Expected: "all",
		},

		{
			// Allocation Disable.
			Name:     "Allocation Disable",
			Body:     `{"transient":{"cluster.routing.allocation.enable":"none"}}`,
			Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"enable": "none"}}}}}`,
			Setting:  "disable",
			Expected: "none",
		},
	}

	for _, x := range tt {
		t.Run(x.Name, func(st *testing.T) {

			testSetup := &ServerSetup{
				Method:   "PUT",
				Path:     "/_cluster/settings",
				Body:     x.Body,
				Response: x.Response,
			}

			host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
			defer ts.Close()
			client := NewClient(host, port)

			resp := client.SetAllocation(x.Setting)

			if resp != x.Expected {
				st.Errorf("Unexpected response, got %s", resp)
			}

		})
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
		HTTPStatus: http.StatusBadRequest,
		Response:   `{"error":{"root_cause":[{"type":"illegal_argument_exception","reason":"Illegal allocation.enable value [FOO]"}],"type":"illegal_argument_exception","reason":"Illegal allocation.enable value [FOO]"},"status":400}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, putSetup})
	defer ts.Close()
	client := NewClient(host, port)

	_, _, err := client.SetSetting("cluster.routing.allocation.enable", "foo")

	if err == nil {
		t.Errorf("Expected error to not be nil, %s", err)
	}

	if err.Error() != fmt.Sprintf("Bad HTTP Status of 400 from Elasticsearch: %s", putSetup.Response) {
		t.Errorf("Unexpected error message, %s", err)
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
	client := NewClient(host, port)

	snapshots, headers := client.GetSnapshots("octocat")

	if len(headers) != 4 {
		t.Errorf("Unexpected headers, got %s", headers)
	}

	if len(snapshots) != 2 {
		t.Errorf("Unexpected snapshots, got %s", snapshots)
	}

	if snapshots[0][0] != "SUCCESS" || snapshots[0][1] != "snapshot1" ||
		snapshots[0][2] != "2018-04-03T07:41:01.719Z" ||
		snapshots[0][3] != "1s" || snapshots[1][3] != "500ms" {
		t.Errorf("Unexpected snapshots, got %s", snapshots)
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
	client := NewClient(host, port)

	snapshots, headers := client.GetSnapshots("octocat")

	if len(headers) != 4 {
		t.Errorf("Unexpected headers, got %s", headers)
	}

	if len(snapshots) != 1 {
		t.Errorf("Unexpected snapshots, got %s", snapshots)
	}

	if snapshots[0][0] != "IN_PROGRESS" || snapshots[0][1] != "snapshot1" ||
		snapshots[0][2] != "" ||
		snapshots[0][3] != "1h0m0s" {
		t.Errorf("Unexpected snapshots, got %s", snapshots)
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
	client := NewClient(host, port)

	snapshots, headers := client.GetSnapshots("octocat")

	if len(headers) != 4 {
		t.Errorf("Unexpected headers, got %s", headers)
	}

	if len(snapshots) != 10 {
		t.Errorf("Unexpected snapshots, got %s", snapshots)
	}

	if snapshots[0][1] != "snapshot7" ||
		snapshots[9][1] != "snapshot16" {
		t.Errorf("Unexpected snapshots, got %s", snapshots)
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
	client := NewClient(host, port)

	snapshot, headers := client.GetSnapshotStatus("octocat", "snapshot1")

	if len(headers) != 2 {
		t.Errorf("Unexpected headers, got %s", headers)
	}

	if len(snapshot) != 7 {
		t.Errorf("Unexpected snapshots, got %s", snapshot)
	}

	if snapshot[0][0] != "state" || snapshot[0][1] != "SUCCESS" {
		t.Errorf("Unexpected state, got %s", snapshot[0])
	}

	if snapshot[1][0] != "snapshot" || snapshot[1][1] != "snapshot1" {
		t.Errorf("Unexpected snapshot name, got %s", snapshot[1])
	}

	if snapshot[2][0] != "indices" || snapshot[2][1] != "index1\nindex2" {
		t.Errorf("Unexpected indices, got %s", snapshot[2][1])
	}

	if snapshot[5][0] != "duration" || snapshot[5][1] != "1s" {
		t.Errorf("Unexpected indices, got %s", snapshot[5])
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
	client := NewClient(host, port)

	good, bad := client.PerformSnapshotsCheck("octocat")

	if len(good) != 2 {
		t.Errorf("Unexpected good snapshots, got %s", good)
	}

	if len(bad) != 0 {
		t.Errorf("Unexpected bad snapshots, got %s", bad)
	}

	if good[0] != "snapshot1" || good[1] != "snapshot2" {
		t.Errorf("Unexpected snapshots, got %s", good)
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
	client := NewClient(host, port)

	good, bad := client.PerformSnapshotsCheck("octocat")

	if len(good) != 1 {
		t.Errorf("Unexpected good snapshots, got %s", good)
	}

	if len(bad) != 1 {
		t.Errorf("Unexpected bad snapshots, got %s", bad)
	}

	if bad[0] != "snapshot1" || good[0] != "snapshot2" {
		t.Errorf("Unexpected snapshots, good: %s bad: %s", good, bad)
	}
}
