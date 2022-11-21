package vulcanizer

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"gotest.tools/assert"
)

// ServerSetup type contains the Method, Path, Body and Response strings, as well as the HTTP Status code.
type ServerSetup struct {
	Method, Path, Body, Response string
	HTTPStatus                   int
	extraChecksFn                func(t *testing.T, r *http.Request)
}

func buildTestServer(t *testing.T, setups []*ServerSetup, tls bool) *httptest.Server {
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestBytes, _ := ioutil.ReadAll(r.Body)
		requestBody := string(requestBytes)

		matched := false
		for _, setup := range setups {
			if setup.extraChecksFn != nil {
				setup.extraChecksFn(t, r)
			}
			// Extra piece of debug incase there's a typo in your test's response, like a rogue space somewhere
			if r.Method == setup.Method && r.URL.EscapedPath() == setup.Path && requestBody != setup.Body {
				t.Fatalf("request body not matching: %s != %s", requestBody, setup.Body)
			}

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
	})

	var ts *httptest.Server

	if tls {
		ts = httptest.NewTLSServer(handlerFunc)
	} else {
		ts = httptest.NewServer(handlerFunc)
	}

	return ts
}

// setupTestServer sets up an HTTP test server to serve data to the test that come after it.
func setupTestServers(t *testing.T, setups []*ServerSetup) (string, int, *httptest.Server) {

	ts := buildTestServer(t, setups, false)

	url, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(url.Port())
	return url.Hostname(), port, ts
}

func setupTestTLSServers(t *testing.T, setups []*ServerSetup) (string, int, *httptest.Server) {

	ts := buildTestServer(t, setups, true)

	url, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(url.Port())
	return url.Hostname(), port, ts
}

func stringToPointer(v string) *string { return &v }

func TestGetClusterExcludeSettings(t *testing.T) {

	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/settings",
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_host":"excluded.host","_name":"excluded_name","_ip":"10.0.0.99"}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()

	client := NewClient(host, port)

	excludeSettings, err := client.GetClusterExcludeSettings()

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

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

	getSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/settings",
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":""}}}}}}`,
	}

	putSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_cluster/settings",
		Body:     `{"transient":{"cluster.routing.allocation.exclude._name":"server_to_drain"}}`,
		Response: `{"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"server_to_drain"}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, putSetup})
	defer ts.Close()
	client := NewClient(host, port)

	excludeSettings, err := client.DrainServer("server_to_drain")

	if err != nil {
		t.Errorf("Unexpected error, %s", err)
	}

	if excludeSettings.Names[0] != "server_to_drain" {
		t.Errorf("Expected response, got %+v", excludeSettings)
	}
}

func TestDrainServer_ExistingValues(t *testing.T) {

	getSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/settings",
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"existing_one,existing_two"}}}}}}`,
	}

	putSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_cluster/settings",
		Body:     `{"transient":{"cluster.routing.allocation.exclude._name":"existing_one,existing_two,server_to_drain"}}`,
		Response: `{"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"server_to_drain,existing_one,existing_two"}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, putSetup})
	defer ts.Close()
	client := NewClient(host, port)

	excludeSettings, err := client.DrainServer("server_to_drain")
	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if len(excludeSettings.Names) != 3 || excludeSettings.Names[2] != "server_to_drain" {
		t.Errorf("unexpected response, got %+v", excludeSettings)
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

	excludeSettings, err := client.FillOneServer("good_server")

	if err != nil {
		t.Errorf("Unexpected error expected nil, got %s", err)
	}

	if excludeSettings.Names[0] != "excluded_server1" {
		t.Errorf("unexpected response, got %+v", excludeSettings)
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

	_, err := client.FillOneServer("good_server")

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
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

	excludeSettings, err := client.FillAll()
	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

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
		Response: `[{"master": "*", "role": "d", "name": "foo", "ip": "127.0.0.1", "id": "abc", "jdk": "1.8", "version": "6.4.0"}]`,
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

	if nodes[0].Version != "6.4.0" {
		t.Errorf("Unexpected version, expected 6.4.0, got %s", nodes[0].Version)
	}
}

func TestGetNodeAllocations(t *testing.T) {
	nodeSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/nodes",
		Response: `[{"master": "*", "role": "d", "name": "foo", "ip": "127.0.0.1", "id": "abc", "jdk": "1.8", "version": "6.4.0"}]`,
	}

	allocationSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/allocation",
		Response: `[{"shards": "108", "disk.indices": "683.8gb", "disk.used": "735.2gb", "disk.avail": "248gb", "disk.total": "983.3gb", "disk.percent": "74", "ip": "127.0.0.1", "node": "foo"}]`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{nodeSetup, allocationSetup})
	defer ts.Close()
	client := NewClient(host, port)

	nodes, err := client.GetNodeAllocations()

	if err != nil {
		t.Errorf("Unexpected error expected nil, got %s", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Unexpected nodes, got %s", nodes)
	}

	if nodes[0].Name != "foo" {
		t.Errorf("Unexpected node name, expected foo, got %s", nodes[0].Name)
	}

	if nodes[0].Version != "6.4.0" {
		t.Errorf("Unexpected version, expected 6.4.0, got %s", nodes[0].Version)
	}

	if nodes[0].Shards != "108" {
		t.Errorf("Unexpected Shards, expected 108, got %s", nodes[0].Shards)
	}

	if nodes[0].DiskPercent != "74" {
		t.Errorf("Unexpected DiskPercent, expected 74, got %s", nodes[0].DiskPercent)
	}

	if nodes[0].DiskUsed != "735.2gb" {
		t.Errorf("Unexpected DiskUsed, expected 735.2gb, got %s", nodes[0].DiskUsed)
	}

	if nodes[0].IP != "127.0.0.1" {
		t.Errorf("Unexpected node IP, expected 127.0.0.1, got %s", nodes[0].IP)
	}
}

func TestGetAllIndices(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/indices",
		Response: `[{"health":"yellow","status":"open","index":"index1","pri":"5","rep":"1","store.size":"3.6kb", "docs.count":"1500"}]`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	indices, err := client.GetAllIndices()

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

func TestGetIndices(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/indices/test*",
		Response: `[{"health":"yellow","status":"open","index":"test_one","pri":"5","rep":"1","store.size":"3.6kb", "docs.count":"1500"}]`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	indices, err := client.GetIndices("test*")

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

func TestGetAllAliases(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/aliases",
		Response: `[{"alias": "test","index": "test_v1","filter": "-filter","routing.index": "-routing.index","routing.search": "-routing.search"},{"alias": "test_again","index": "test_again_v1","filter": "--filter","routing.index": "--routing.index","routing.search": "--routing.search"}]`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	aliases, err := client.GetAllAliases()

	if err != nil {
		t.Errorf("Unexpected error expected nil, got %s", err)
	}

	if len(aliases) != 2 {
		t.Errorf("Unexpected aliases, got %v", aliases)
	}

	if aliases[0].Name != "test" || aliases[0].IndexName != "test_v1" || aliases[0].Filter != "-filter" || aliases[0].RoutingIndex != "-routing.index" || aliases[0].RoutingSearch != "-routing.search" {
		t.Errorf("Unexpected index values, got %v", aliases[0])
	}

	if aliases[1].Name != "test_again" || aliases[1].IndexName != "test_again_v1" || aliases[1].Filter != "--filter" || aliases[1].RoutingIndex != "--routing.index" || aliases[1].RoutingSearch != "--routing.search" {
		t.Errorf("Unexpected index values, got %v", aliases[1])
	}
}

func TestGetAliases(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/aliases/test*",
		Response: `[{"alias": "test","index": "test_v1","filter": "-filter","routing.index": "-routing.index","routing.search": "-routing.search"},{"alias": "test_again","index": "test_again_v1","filter": "--filter","routing.index": "--routing.index","routing.search": "--routing.search"}]`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	aliases, err := client.GetAliases("test*")

	if err != nil {
		t.Errorf("Unexpected error expected nil, got %s", err)
	}

	if len(aliases) != 2 {
		t.Errorf("Unexpected aliases, got %v", aliases)
	}

	if aliases[0].Name != "test" || aliases[0].IndexName != "test_v1" || aliases[0].Filter != "-filter" || aliases[0].RoutingIndex != "-routing.index" || aliases[0].RoutingSearch != "-routing.search" {
		t.Errorf("Unexpected index values, got %v", aliases[0])
	}

	if aliases[1].Name != "test_again" || aliases[1].IndexName != "test_again_v1" || aliases[1].Filter != "--filter" || aliases[1].RoutingIndex != "--routing.index" || aliases[1].RoutingSearch != "--routing.search" {
		t.Errorf("Unexpected index values, got %v", aliases[1])
	}
}

func TestModifyAliases(t *testing.T) {
	tt := []struct {
		Name     string
		Actions  []AliasAction
		Body     string
		Response string
	}{
		{
			Name: "add alias",
			Actions: []AliasAction{
				{
					ActionType: AddAlias,
					IndexName:  "test",
					AliasName:  "test_alias",
				},
			},
			Body:     `{"actions":[{"add":{"alias":"test_alias","index":"test"}}]}`,
			Response: `{"acknowledged": true}`,
		},
		{
			Name: "update alias",
			Actions: []AliasAction{
				{
					ActionType: AddAlias,
					IndexName:  "test",
					AliasName:  "test_alias",
				},
				{
					ActionType: RemoveAlias,
					IndexName:  "test",
					AliasName:  "test_alias",
				},
			},
			Body:     `{"actions":[{"add":{"alias":"test_alias","index":"test"}},{"remove":{"alias":"test_alias","index":"test"}}]}`,
			Response: `{"acknowledged": true}`,
		},
		{
			Name: "delete alias",
			Actions: []AliasAction{
				{
					ActionType: RemoveAlias,
					IndexName:  "test",
					AliasName:  "test_alias",
				},
			},
			Body:     `{"actions":[{"remove":{"alias":"test_alias","index":"test"}}]}`,
			Response: `{"acknowledged": true}`,
		},
	}

	for _, x := range tt {
		t.Run(x.Name, func(t *testing.T) {
			testSetup := &ServerSetup{
				Method:   "POST",
				Path:     "/_aliases",
				Body:     x.Body,
				Response: x.Response,
			}

			host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
			defer ts.Close()
			client := NewClient(host, port)

			err := client.ModifyAliases(x.Actions)
			if err != nil {
				t.Errorf("Unexpected error expected nil, got %s", err)
			}
		})
	}
}

func TestDeleteIndex(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "DELETE",
		Path:     "/badindex",
		Response: `{"acknowledged": true}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	err := client.DeleteIndex("badindex")

	if err != nil {
		t.Errorf("Unexpected error expected nil, got %s", err)
	}
}

func TestOpenIndex(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "POST",
		Path:     "/openindex*/_open",
		Response: `{"acknowledged": true}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	err := client.OpenIndex("openindex*")

	if err != nil {
		t.Errorf("Unexpected error expected nil, got %s", err)
	}
}

func TestCloseIndex(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "POST",
		Path:     "/closeindex*/_close",
		Response: `{"acknowledged": true}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	err := client.CloseIndex("closeindex*")

	if err != nil {
		t.Errorf("Unexpected error expected nil, got %s", err)
	}
}

func TestGetHealth(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/health",
		Response: `{"cluster_name":"mycluster","status":"green","timed_out":false,"number_of_nodes":1,"number_of_data_nodes":1,"active_primary_shards":5,"active_shards":5,"relocating_shards":0,"initializing_shards":0,"unassigned_shards":0,"delayed_unassigned_shards":0,"number_of_pending_tasks":0,"number_of_in_flight_fetch":0,"task_max_waiting_in_queue_millis":0,"active_shards_percent_as_number":100.0,"indices":{"unhealthyIndex":{"status":"yellow"},"healthyIndex":{"status":"green","number_of_shards":5,"number_of_replicas":0,"active_primary_shards":5,"active_shards":5,"relocating_shards":0,"initializing_shards":0,"unassigned_shards":0}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	health, err := client.GetHealth()

	if err != nil {
		t.Errorf("Unexpected error expected nil, got %s", err)
	}

	if health.ActiveShards != 5 {
		t.Errorf("Unexpected active shards, expected 5, got %d", health.ActiveShards)
	}

	if len(health.HealthyIndices) != 1 || health.HealthyIndices[0].Name != "healthyIndex" {
		t.Errorf("Unexpected values in healthy indices, got %+v", health)
	}

	if len(health.UnhealthyIndices) != 1 || health.UnhealthyIndices[0].Name != "unhealthyIndex" {
		t.Errorf("Unexpected values in unhealthy indices, got %+v", health)
	}
}

func TestGetHealth_TLS(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/health",
		Response: `{"cluster_name":"mycluster","status":"green","timed_out":false,"number_of_nodes":1,"number_of_data_nodes":1,"active_primary_shards":5,"active_shards":5,"relocating_shards":0,"initializing_shards":0,"unassigned_shards":0,"delayed_unassigned_shards":0,"number_of_pending_tasks":0,"number_of_in_flight_fetch":0,"task_max_waiting_in_queue_millis":0,"active_shards_percent_as_number":100.0,"indices":{"unhealthyIndex":{"status":"yellow"},"healthyIndex":{"status":"green","number_of_shards":5,"number_of_replicas":0,"active_primary_shards":5,"active_shards":5,"relocating_shards":0,"initializing_shards":0,"unassigned_shards":0}}}`,
	}

	host, port, ts := setupTestTLSServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)
	client.Secure = true
	// nolint:gosec
	// G402: TLS InsecureSkipVerify set true. (gosec)
	client.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	health, err := client.GetHealth()

	if err != nil {
		t.Errorf("Unexpected error expected nil, got %s", err)
	}

	if health.ActiveShards != 5 {
		t.Errorf("Unexpected active shards, expected 5, got %d", health.ActiveShards)
	}

	if len(health.HealthyIndices) != 1 || health.HealthyIndices[0].Name != "healthyIndex" {
		t.Errorf("Unexpected values in healthy indices, got %+v", health)
	}

	if len(health.UnhealthyIndices) != 1 || health.UnhealthyIndices[0].Name != "unhealthyIndex" {
		t.Errorf("Unexpected values in unhealthy indices, got %+v", health)
	}

}

func TestGetClusterSettings(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cluster/settings",
		Response: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_host":"","_name":"10.0.0.2","_ip":""}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	clusterSettings, err := client.GetClusterSettings()

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if len(clusterSettings.PersistentSettings) != 0 {
		t.Errorf("Unexpected persistent settings, got %v", clusterSettings.PersistentSettings)
	}

	if len(clusterSettings.TransientSettings) != 1 {
		t.Errorf("Unexpected transient settings, got %v", clusterSettings.TransientSettings)
	}

	if clusterSettings.TransientSettings[0].Setting != "cluster.routing.allocation.exclude._name" {
		t.Errorf("Unexpected setting name, got %s", clusterSettings.TransientSettings[0].Setting)
	}

	if clusterSettings.TransientSettings[0].Value != "10.0.0.2" {
		t.Errorf("Unexpected setting value, got %s", clusterSettings.TransientSettings[0].Value)
	}
}

// TestSetClusterSetting Func is an integration test for all things that use the SetClusterSetting functionality.
func TestSetClusterSettings(t *testing.T) {

	tt := []struct {
		Name        string
		Method      string
		Body        string
		GetResponse string
		PutResponse string
		Setting     string
		SetValue    *string
		HTTPStatus  int
		OldValue    *string
	}{
		{
			// Tests for behavior with existing transient setting.
			Name:        "Existing Transient Setting",
			GetResponse: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.2"}}}}}}`,
			Body:        `{"transient":{"cluster.routing.allocation.exclude._name":"10.0.0.99"}}`,
			PutResponse: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.99"}}}}}}`,
			Setting:     "cluster.routing.allocation.exclude._name",
			SetValue:    stringToPointer("10.0.0.99"),
			OldValue:    stringToPointer("10.0.0.2"),
		},

		{
			// Tests for behavior with existing persistent settings.
			Name:        "Existing Persistent Setting",
			GetResponse: `{"transient":{},"persistent":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.2"}}}}}}`,
			Body:        `{"transient":{"cluster.routing.allocation.exclude._name":"10.0.0.99"}}`,
			PutResponse: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.99"}}}}}}`,
			Setting:     "cluster.routing.allocation.exclude._name",
			SetValue:    stringToPointer("10.0.0.99"),
			OldValue:    stringToPointer("10.0.0.2"),
		},

		{
			// Tests for behavior with NO existing persistent settings.
			Name:        "No existing settings",
			GetResponse: `{"transient":{},"persistent":{}}`,
			Body:        `{"transient":{"cluster.routing.allocation.exclude._name":"10.0.0.99"}}`,
			PutResponse: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.99"}}}}}}`,
			Setting:     "cluster.routing.allocation.exclude._name",
			SetValue:    stringToPointer("10.0.0.99"),
			OldValue:    nil,
		},

		{
			// Tests for behavior when removing setting (null'ing it).
			Name:        "Removing Existing Persistent Setting",
			GetResponse: `{"transient":{},"persistent":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.2"}}}}}}`,
			Body:        `{"transient":{"cluster.routing.allocation.exclude._name":null}}`,
			PutResponse: `{"transient":{},"persistent":{}}`,
			Setting:     "cluster.routing.allocation.exclude._name",
			SetValue:    nil,
			OldValue:    stringToPointer("10.0.0.2"),
		},

		{
			// Tests for behavior when removing setting (null'ing it) that is already null.
			Name:        "Removing Null Persistent Setting",
			GetResponse: `{"transient":{},"persistent":{}}`,
			Body:        `{"transient":{"cluster.routing.allocation.exclude._name":null}}`,
			PutResponse: `{"transient":{},"persistent":{}}`,
			Setting:     "cluster.routing.allocation.exclude._name",
			SetValue:    nil,
			OldValue:    nil,
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

			oldSetting, newSetting, err := client.SetClusterSetting(x.Setting, x.SetValue)

			if err != nil {
				st.Errorf("Expected error to be nil, %s", err)
			}

			if oldSetting == nil {
				if x.OldValue != nil {
					st.Fatalf("Unexpected old value: expected old value to be %s, got nil", *x.OldValue)
				}
			}

			if oldSetting != nil {
				if x.OldValue == nil {
					st.Fatalf("Unexpected old value: expected old value to be nil, got %v", *oldSetting)
				}
				if *oldSetting != *x.OldValue {
					st.Errorf("Unexpected old value: expected %s, got %s", *x.OldValue, *oldSetting)
				}
			}

			if newSetting == nil {
				if x.SetValue != nil {
					st.Fatalf("Unexpected new value, got nil, expected %s", *x.SetValue)
				}
			}

			if newSetting != nil {
				if x.SetValue == nil {
					st.Errorf("Unexpected new value, got %s, expected nil", *newSetting)
				}
				if *newSetting != *x.SetValue {
					st.Errorf("Unexpected new value, got %v, expected %v", newSetting, x.SetValue)
				}
			}
		})
	}
}

// TestAllocationSettings Func is an integration test for all things that use the SetAllocation functionality.
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

			resp, err := client.SetAllocation(x.Setting)

			if err != nil {
				st.Errorf("Unexpected error, got %s", err)
			}

			if resp != x.Expected {
				st.Errorf("Unexpected response, got %s", resp)
			}

		})
	}
}

func TestSetClusterSetting_BadRequest(t *testing.T) {
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

	_, _, err := client.SetClusterSetting("cluster.routing.allocation.enable", stringToPointer("foo"))

	if err == nil {
		t.Errorf("Expected error to not be nil, %s", err)
	}

	if err.Error() != fmt.Sprintf("Bad HTTP Status from Elasticsearch: 400, %s", putSetup.Response) {
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

	snapshots, err := client.GetSnapshots("octocat")

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if len(snapshots) != 2 {
		t.Errorf("Unexpected snapshots, got %v", snapshots)
	}

	if snapshots[0].State != "SUCCESS" || snapshots[0].Name != "snapshot1" ||
		snapshots[0].Shards.Successful != 93 {
		t.Errorf("Unexpected snapshots, got %v", snapshots)
	}

	if snapshots[0].Indices[0] != "index1" || snapshots[0].Indices[1] != "index2" ||
		len(snapshots[0].Indices) != 2 {
		t.Errorf("Unexpected snapshots, got %v", snapshots)
	}
}

func TestGetSnapshots_Inprogress(t *testing.T) {
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

	snapshots, err := client.GetSnapshots("octocat")
	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if len(snapshots) != 1 {
		t.Errorf("Unexpected snapshots, got %v", snapshots)
	}

	if snapshots[0].State != "IN_PROGRESS" {
		t.Errorf("Unexpected snapshots, got %v", snapshots)
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

	snapshot, err := client.GetSnapshotStatus("octocat", "snapshot1")
	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if snapshot.State != "SUCCESS" {
		t.Errorf("Unexpected state, got %+v", snapshot)
	}

	if snapshot.Name != "snapshot1" {
		t.Errorf("Unexpected name, got %+v", snapshot)
	}
}

func TestDeleteSnapshot(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "DELETE",
		Path:     "/_snapshot/octocat/snapshot1",
		Response: `{"acknowledged": true}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	err := client.DeleteSnapshot("octocat", "snapshot1")
	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}
}

func TestRegisterRepository(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_snapshot/mysnapshotrepo",
		Body:     `{"settings":{"location":"/backups"},"type":"fs"}`,
		Response: `{"acknowledged":true}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	repo := Repository{
		Name: "mysnapshotrepo",
		Type: "fs",
		Settings: map[string]interface{}{
			"location": "/backups",
		},
	}

	err := client.RegisterRepository(repo)
	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}
}

func TestRegisterRepository_MissingName(t *testing.T) {
	host, port, ts := setupTestServers(t, []*ServerSetup{})
	defer ts.Close()
	client := NewClient(host, port)

	repo := Repository{
		Type: "fs",
		Settings: map[string]interface{}{
			"location": "/backups",
		},
	}

	err := client.RegisterRepository(repo)
	if !errors.Is(err, ErrRepositoryNameRequired) {
		t.Error("Expected validation for missing repository name.")
	}
}

func TestRegisterRepository_MissingType(t *testing.T) {
	host, port, ts := setupTestServers(t, []*ServerSetup{})
	defer ts.Close()
	client := NewClient(host, port)

	repo := Repository{
		Name: "myrepo",
		Settings: map[string]interface{}{
			"location": "/backups",
		},
	}

	err := client.RegisterRepository(repo)
	if !errors.Is(err, ErrRepositoryTypeRequired) {
		t.Error("Expected validation for missing repository type.")
	}
}

func TestRemoveRepository(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "DELETE",
		Path:     "/_snapshot/mysnapshotrepo",
		Response: `{"acknowledged":true}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	err := client.RemoveRepository("mysnapshotrepo")
	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}
}

func TestRemoveRepository_MissingName(t *testing.T) {
	host, port, ts := setupTestServers(t, []*ServerSetup{})
	defer ts.Close()
	client := NewClient(host, port)

	err := client.RemoveRepository("")
	if !errors.Is(err, ErrRepositoryNameRequired) {
		t.Error("Expected validation for missing repository name.")
	}
}

func TestVerifyRepository(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "POST",
		Path:     "/_snapshot/octocat/_verify",
		Response: `{"nodes":{"YaTBa_BtRmOoz1bHKJeQ8w":{"name":"YaTBa_B"}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	verified, err := client.VerifyRepository("octocat")
	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if !verified {
		t.Errorf("Expected repository to be verified, got %v", verified)
	}
}

func TestGetRepositories(t *testing.T) {
	testSetup := &ServerSetup{
		Method: "GET",
		Path:   "/_snapshot/_all",
		Response: `{
			"fileSystemRepo": { "type": "fs", "settings": { "location": "/foo/bar" } },
			"s3Repo": { "type": "s3", "settings": { "bucket": "myBucket", "base_path": "foo", "access_key": "access", "secret_key": "secret" } }
}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	repos, err := client.GetRepositories()

	if err != nil {
		t.Fatalf("Got error getting repositories: %s", err)
	}

	if len(repos) != 2 {
		t.Fatalf("Expected two repositories, got %d", len(repos))
	}

	var fsRepo, s3Repo Repository

	for _, r := range repos {
		if r.Type == "fs" {
			fsRepo = r
		} else if r.Type == "s3" {
			s3Repo = r
		}
	}

	if fsRepo.Name != "fileSystemRepo" || fsRepo.Type != "fs" || fsRepo.Settings["location"] != "/foo/bar" {
		t.Fatalf("Unexpected fs repo settings, got: %+v", fsRepo)
	}

	if s3Repo.Name != "s3Repo" || s3Repo.Type != "s3" || s3Repo.Settings["bucket"] != "myBucket" {
		t.Fatalf("Unexpected s3 repo settings, got: %+v", s3Repo)
	}

	if _, exists := s3Repo.Settings["access_key"]; exists {
		t.Fatalf("Expected access_key to be scrubbed from s3Repo.")
	}

	if _, exists := s3Repo.Settings["secret_key"]; exists {
		t.Fatalf("Expected secret_key to be scrubbed from s3Repo.")
	}
}

func TestSnapshotIndices_ErrorConditions(t *testing.T) {

	tt := []struct {
		Name        string
		Repository  string
		Snapshot    string
		Indices     []string
		ExpectError bool
	}{
		{
			Name:        "Do not allow blank repository",
			Repository:  "",
			Snapshot:    "snapshot1",
			Indices:     []string{"index1"},
			ExpectError: true,
		},
		{
			Name:        "Do not allow blank snapshot name",
			Repository:  "backup",
			Snapshot:    "",
			Indices:     []string{"index1"},
			ExpectError: true,
		},
		{
			Name:        "Do not allow empty indices",
			Repository:  "backup",
			Snapshot:    "snapshot1",
			Indices:     []string{},
			ExpectError: true,
		},
	}
	client := &Client{}

	for _, test := range tt {
		t.Run(test.Name, func(st *testing.T) {
			err := client.SnapshotIndices(test.Repository, test.Snapshot, test.Indices)

			if err == nil && test.ExpectError {
				st.Errorf("Expected error for test values %+v", test)
			}

			if err != nil && !test.ExpectError {
				st.Errorf("Expected no error for test values. Got error %s for test  %+v", err, test)
			}
		})
	}
}

func TestSnapshotAllIndices_ErrorConditions(t *testing.T) {

	tt := []struct {
		Name        string
		Repository  string
		Snapshot    string
		ExpectError bool
	}{
		{
			Name:        "Do not allow blank repository",
			Repository:  "",
			Snapshot:    "snapshot1",
			ExpectError: true,
		},
		{
			Name:        "Do not allow blank snapshot name",
			Repository:  "backup",
			Snapshot:    "",
			ExpectError: true,
		},
	}
	client := &Client{}

	for _, test := range tt {
		t.Run(test.Name, func(st *testing.T) {
			err := client.SnapshotAllIndices(test.Repository, test.Snapshot)

			if err == nil && test.ExpectError {
				st.Errorf("Expected error for test values %+v", test)
			}

			if err != nil && !test.ExpectError {
				st.Errorf("Expected no error for test values. Got error %s for test  %+v", err, test)
			}
		})
	}
}

func TestSnapshotIndices(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_snapshot/backup-repo/snapshot1",
		Body:     `{"indices":"index1,index2"}`,
		Response: `{"acknowledged": true }`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	err := client.SnapshotIndices("backup-repo", "snapshot1", []string{"index1", "index2"})

	if err != nil {
		t.Fatalf("Got error taking snapshot: %s", err)
	}
}

func TestSnapshotAllIndices(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/_snapshot/backup-repo/snapshot1",
		Response: `{"acknowledged": true }`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	err := client.SnapshotAllIndices("backup-repo", "snapshot1")

	if err != nil {
		t.Fatalf("Got error taking snapshot: %s", err)
	}
}

func TestRestoreSnapshotIndices_ErrorConditions(t *testing.T) {
	tt := []struct {
		Name        string
		Repository  string
		Snapshot    string
		ExpectError bool
	}{
		{
			Name:        "Do not allow blank repository",
			Repository:  "",
			Snapshot:    "snapshot1",
			ExpectError: true,
		},
		{
			Name:        "Do not allow blank snapshot name",
			Repository:  "backup",
			Snapshot:    "",
			ExpectError: true,
		},
	}
	client := &Client{}

	for _, test := range tt {
		t.Run(test.Name, func(st *testing.T) {
			err := client.RestoreSnapshotIndices(test.Repository, test.Snapshot, []string{}, "", nil)

			if err == nil && test.ExpectError {
				st.Errorf("Expected error for test values %+v", test)
			}

			if err != nil && !test.ExpectError {
				st.Errorf("Expected no error for test values. Got error %s for test  %+v", err, test)
			}
		})
	}
}

func TestRestoreSnapshotIndices(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "POST",
		Path:     "/_snapshot/backup-repo/snapshot1/_restore",
		Body:     `{"indices":"index1,index2","rename_pattern":"(.+)","rename_replacement":"restored_$1"}`,
		Response: `{"acknowledged": true }`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	err := client.RestoreSnapshotIndices("backup-repo", "snapshot1", []string{"index1", "index2"}, "restored_", nil)

	if err != nil {
		t.Fatalf("Got error taking snapshot: %s", err)
	}
}

func TestRestoreSnapshotIndicesWithSettings(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "POST",
		Path:     "/_snapshot/backup-repo/snapshot1/_restore",
		Body:     `{"index_settings":{"index.number_of_replicas":0,"index.refresh_interval":"-1"},"indices":"index1,index2","rename_pattern":"(.+)","rename_replacement":"restored_$1"}`,
		Response: `{"acknowledged": true }`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	indexSettings := map[string]interface{}{
		"index.number_of_replicas": 0,
		"index.refresh_interval":   "-1",
	}

	err := client.RestoreSnapshotIndices("backup-repo", "snapshot1", []string{"index1", "index2"}, "restored_", indexSettings)

	if err != nil {
		t.Fatalf("Got error taking snapshot: %s", err)
	}
}

func TestAnalyzeText(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "POST",
		Path:     "/_analyze",
		Body:     `{"analyzer":"stop","text":"This is a great test."}`,
		Response: `{"tokens":[{"token":"great","start_offset":10,"end_offset":15,"type":"word","position":3},{"token":"test","start_offset":16,"end_offset":20,"type":"word","position":4}]}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	tokens, err := client.AnalyzeText("stop", "This is a great test.")
	if err != nil {
		t.Fatalf("Got error getting analyzing text: %s", err)
	}

	if len(tokens) != 2 {
		t.Fatalf("Got wrong number of tokens, expected 2 got %d", len(tokens))
	}

	if tokens[0].Text != "great" || tokens[1].Text != "test" {
		t.Fatalf("Unexpected token text, got: %+v", tokens)
	}

	if tokens[0].Type != "word" || tokens[1].Type != "word" {
		t.Fatalf("Unexpected token type, got: %+v", tokens)
	}
}

func TestAnalyzeTextWithField(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "POST",
		Path:     "/myindex/_analyze",
		Body:     `{"field":"user_email","text":"user@example.com"}`,
		Response: `{"tokens":[{"token":"user@example.com","start_offset":0,"end_offset":16,"type":"<EMAIL>","position":1}]}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	tokens, err := client.AnalyzeTextWithField("myindex", "user_email", "user@example.com")
	if err != nil {
		t.Fatalf("Got error getting analyzing text: %s", err)
	}

	if len(tokens) != 1 {
		t.Fatalf("Got wrong number of tokens, expected 1 got %d", len(tokens))
	}

	if tokens[0].Text != "user@example.com" || tokens[0].Type != "<EMAIL>" {
		t.Fatalf("Unexpected token got: %+v", tokens)
	}
}

func TestGetPrettyIndexSettings(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/octocat/_settings",
		Response: `{"octocat":{"settings":{"index":{"number_of_shards":"5","provided_name":"octocat","creation_date":"1535035072757","analysis":{"analyzer":{"my_custom_analyzer":{"filter":["lowercase","asciifolding"],"char_filter":["html_strip"],"type":"custom","tokenizer":"standard"}}},"number_of_replicas":"0","uuid":"Q_Jm1mD2Syy8JgMUiicqcw","version":{"created":"5061099"}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	indexSettings, err := client.GetPrettyIndexSettings("octocat")

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if indexSettings == "" {
		t.Error("Unexpected index settings, got empty string")
	}
}

func TestGetPrettyIndexSettings_SpecialName(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/.special/_settings",
		Response: `{".special":{"settings":{"index":{"number_of_shards":"5","provided_name":"octocat","creation_date":"1535035072757","analysis":{"analyzer":{"my_custom_analyzer":{"filter":["lowercase","asciifolding"],"char_filter":["html_strip"],"type":"custom","tokenizer":"standard"}}},"number_of_replicas":"0","uuid":"Q_Jm1mD2Syy8JgMUiicqcw","version":{"created":"5061099"}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	indexSettings, err := client.GetPrettyIndexSettings(".special")

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if indexSettings == "" {
		t.Error("Unexpected index settings, got empty string")
	}
}

func TestGetIndexSettings(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/octocat/_settings",
		Response: `{"octocat":{"settings":{"index":{"number_of_shards":"5","provided_name":"octocat","creation_date":"1535035072757","analysis":{"analyzer":{"my_custom_analyzer":{"filter":["lowercase","asciifolding"],"char_filter":["html_strip"],"type":"custom","tokenizer":"standard"}}},"number_of_replicas":"0","uuid":"Q_Jm1mD2Syy8JgMUiicqcw","version":{"created":"5061099"}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	settings, err := client.GetIndexSettings("octocat")

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if len(settings) != 11 {
		t.Errorf("Unexpected number of settings, got %d", len(settings))
	}

	for i := range settings {
		s := settings[i]
		if s.Setting == "number_of_shards" && s.Value != "5" {
			t.Errorf("Unexpected shards value, expected 5, got %s", s.Value)
		}

		if s.Setting == "number_of_replicas" && s.Value != "0" {
			t.Errorf("Unexpected replicas value, expected 0, got %s", s.Value)
		}
	}
}

func TestGetIndexSettings_SpecialName(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/.special/_settings",
		Response: `{".special":{"settings":{"index":{"number_of_shards":"5","provided_name":"octocat","creation_date":"1535035072757","analysis":{"analyzer":{"my_custom_analyzer":{"filter":["lowercase","asciifolding"],"char_filter":["html_strip"],"type":"custom","tokenizer":"standard"}}},"number_of_replicas":"0","uuid":"Q_Jm1mD2Syy8JgMUiicqcw","version":{"created":"5061099"}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	settings, err := client.GetIndexSettings(".special")

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if len(settings) != 11 {
		t.Errorf("Unexpected number of settings, got %d", len(settings))
	}

	for i := range settings {
		s := settings[i]
		if s.Setting == "number_of_shards" && s.Value != "5" {
			t.Errorf("Unexpected shards value, expected 5, got %s", s.Value)
		}

		if s.Setting == "number_of_replicas" && s.Value != "0" {
			t.Errorf("Unexpected replicas value, expected 0, got %s", s.Value)
		}
	}
}

func TestSetIndexSetting(t *testing.T) {
	getSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/octocat/_settings",
		Response: `{"octocat":{"settings":{"index":{"number_of_shards":"5","provided_name":"octocat","creation_date":"1535035072757","analysis":{"analyzer":{"my_custom_analyzer":{"filter":["lowercase","asciifolding"],"char_filter":["html_strip"],"type":"custom","tokenizer":"standard"}}},"number_of_replicas":"0","uuid":"Q_Jm1mD2Syy8JgMUiicqcw","version":{"created":"5061099"}}}}}`,
	}

	updateSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/octocat/_settings",
		Body:     `{"index":{"number_of_replicas":"2"}}`,
		Response: `{"accepted": true}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, updateSetup})
	defer ts.Close()
	client := NewClient(host, port)

	previous, current, err := client.SetIndexSetting("octocat", "number_of_replicas", "2")

	if err != nil {
		t.Errorf("Error setting index setting: %s", err)
	}

	if previous != "0" {
		t.Errorf("Unexpected previous setting value, expected 0, got %s", previous)
	}

	if current != "2" {
		t.Errorf("Unexpected current setting value, expected 2, got %s", current)
	}
}

func TestSetIndexSetting_SpecialName(t *testing.T) {
	getSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/.special/_settings",
		Response: `{".special":{"settings":{"index":{"number_of_shards":"5","provided_name":"octocat","creation_date":"1535035072757","analysis":{"analyzer":{"my_custom_analyzer":{"filter":["lowercase","asciifolding"],"char_filter":["html_strip"],"type":"custom","tokenizer":"standard"}}},"number_of_replicas":"0","uuid":"Q_Jm1mD2Syy8JgMUiicqcw","version":{"created":"5061099"}}}}}`,
	}

	updateSetup := &ServerSetup{
		Method:   "PUT",
		Path:     "/.special/_settings",
		Body:     `{"index":{"number_of_replicas":"2"}}`,
		Response: `{"accepted": true}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, updateSetup})
	defer ts.Close()
	client := NewClient(host, port)

	previous, current, err := client.SetIndexSetting(".special", "number_of_replicas", "2")

	if err != nil {
		t.Errorf("Error setting index setting: %s", err)
	}

	if previous != "0" {
		t.Errorf("Unexpected previous setting value, expected 0, got %s", previous)
	}

	if current != "2" {
		t.Errorf("Unexpected current setting value, expected 2, got %s", current)
	}
}

func TestGetPrettyIndexMappings(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/octocat/_mappings",
		Response: `{"octocat":{"mappings":{"doc":{"properties":{"created_at":{"type":"date"}}}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	mappings, err := client.GetPrettyIndexMappings("octocat")

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if mappings == "" {
		t.Error("Unexpected index mappings, got empty string")
	}

	lineCount := strings.Count(mappings, "\n")
	if lineCount != 12 {
		t.Errorf("Unexpected line count on mappings, expected 12, got %d", lineCount)
	}
}

func TestGetPrettyIndexSegments(t *testing.T) {
	testSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/octocat/_segments",
		Response: `{"_shards":{"total":2,"successful":1,"failed":0},"indices":{"octocat":{"shards":{"0":[{"routing":{"state":"STARTED","primary":true,"node":"Qwlz-A-6TyqJ-uLGaf8_0w"},"num_committed_segments":0,"num_search_segments":0,"segments":{}}]}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	mappings, err := client.GetPrettyIndexSegments("octocat")

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if mappings == "" {
		t.Error("Unexpected index segments, got empty string")
	}

	lineCount := strings.Count(mappings, "\n")
	if lineCount != 24 {
		t.Errorf("Unexpected line count on segments, expected 24, got %d", lineCount)
	}
}

// Shared server setup for GetShards tests
var getShardsTestSetup = &ServerSetup{
	Method:   "GET",
	Path:     "/_cat/shards",
	Response: `[{"index":"test_index","shard":"1","prirep":"p","state":"STARTED","docs":"0","store":"162b","ip":"123.123.123.123","node":"node-abc123"},{"index":"test_index","shard":"1","prirep":"r","state":"STARTED","docs":"0","store":"162b","ip":"123.123.123.123","node":"node-def456"}]`,
}

func TestGetShards_OneNode(t *testing.T) {
	host, port, ts := setupTestServers(t, []*ServerSetup{getShardsTestSetup})
	defer ts.Close()
	client := NewClient(host, port)

	shards, err := client.GetShards([]string{"node-abc123"})

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if shards == nil {
		t.Error("Expected slice of shards, got nil instead")
	}

	if len(shards) > 1 {
		t.Errorf("Expected slice of 1 shard, got %d instead", len(shards))
	}

}

func TestGetShards_Regexp(t *testing.T) {
	host, port, ts := setupTestServers(t, []*ServerSetup{getShardsTestSetup})
	defer ts.Close()
	client := NewClient(host, port)

	shards, err := client.GetShards([]string{"node-[a-c]{3}\\d+$"})

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if shards == nil {
		t.Error("Expected slice of shards, got nil instead")
	}

	if len(shards) > 1 {
		t.Errorf("Expected slice of 1 shard, got %d instead", len(shards))
	}

	if shards[0].Node != "node-abc123" {
		t.Errorf("Expected to find Node name node-abc123, found %s instead", shards[0].Node)
	}

}

func TestGetShards_MultiNode(t *testing.T) {
	host, port, ts := setupTestServers(t, []*ServerSetup{getShardsTestSetup})
	defer ts.Close()
	client := NewClient(host, port)

	shards, err := client.GetShards([]string{"node-abc123", "node-def456"})

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if shards == nil {
		t.Error("Expected slice of shards, got nil instead")
	}

	if len(shards) != 2 {
		t.Errorf("Expected slice of 2 shards, got %d instead", len(shards))
	}

}

func TestGetShards_NoNodes(t *testing.T) {
	host, port, ts := setupTestServers(t, []*ServerSetup{getShardsTestSetup})
	defer ts.Close()
	client := NewClient(host, port)

	shards, err := client.GetShards(nil)

	if err != nil {
		t.Errorf("Unexpected error, got %s", err)
	}

	if shards == nil {
		t.Error("Expected slice of shards, got nil instead")
	}

	if len(shards) != 2 {
		t.Errorf("Expected slice of 2 shards, got %d instead", len(shards))
	}

}

func TestGetShardOverlap_Safe(t *testing.T) {
	getShardsTestSetup = &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/shards",
		Response: `[{"index":"test_index","shard":"1","prirep":"p","state":"STARTED","docs":"0","store":"162b","ip":"123.123.123.123","node":"node-abc123"},{"index":"test_index","shard":"1","prirep":"r","state":"UNASSIGNED","docs":"0","store":"162b","ip":"123.123.123.123","node":"node-abc123"}]`,
	}

	getIndicesTestSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/indices",
		Response: `[{"health":"green","status":"open","index":"test_index","pri":"5","rep":"1","store.size":"3.6kb", "docs.count":"1500"}]`,
	}
	host, port, ts := setupTestServers(t, []*ServerSetup{getShardsTestSetup, getIndicesTestSetup})
	defer ts.Close()
	client := NewClient(host, port)

	overlap, err := client.GetShardOverlap([]string{"node-abc123"})

	if err != nil {
		t.Errorf("Unexpected error, get %s", err)
	}

	if overlap == nil {
		t.Errorf("Expected slice of shards, got nil instead")
	}

	if val, ok := overlap["test_index_1"]; ok {

		if !val.SafeToRemove() {
			t.Error("Expected SafeToRemove=true, got false instead")
		}
	} else {
		t.Errorf("Expected overlap data, got nil instead")
	}
}

func TestGetShardOverlap_UnSafe(t *testing.T) {
	getShardOverlapTestSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/shards",
		Response: `[{"index":"test_index","shard":"1","prirep":"p","state":"STARTED","docs":"0","store":"162b","ip":"123.123.123.123","node":"node-abc123"},{"index":"test_index","shard":"1","prirep":"r","state":"STARTED","docs":"0","store":"162b","ip":"123.123.123.123","node":"node-abc123"}]`,
	}

	getIndicesTestSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/indices",
		Response: `[{"health":"green","status":"open","index":"test_index","pri":"5","rep":"1","store.size":"3.6kb", "docs.count":"1500"}]`,
	}
	host, port, ts := setupTestServers(t, []*ServerSetup{getShardOverlapTestSetup, getIndicesTestSetup})
	defer ts.Close()
	client := NewClient(host, port)

	overlap, err := client.GetShardOverlap([]string{"node-abc123"})

	if err != nil {
		t.Errorf("Unexpected error, get %s", err)
	}

	if overlap == nil {
		t.Errorf("Expected slice of shards, got nil instead")
	}

	if val, ok := overlap["test_index_1"]; ok {
		if val.SafeToRemove() {
			t.Error("Expected SafeToRemove=false, got true instead")
		}
	} else {
		t.Errorf("Expected overlap data, got nil instead")
	}
}

func TestGetShardOverlap_UnSafeRelocating(t *testing.T) {
	getShardsTestSetup = &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/shards",
		Response: `[{"index":"test_index","shard":"1","prirep":"p","state":"RELOCATING","docs":"0","store":"162b","ip":"123.123.123.123","node":"node-def456 -> 123.123.123.123 bDke_wlKn4Lk node-abc123"},{"index":"test_index","shard":"1","prirep":"r","state":"STARTED","docs":"0","store":"162b","ip":"123.123.123.123","node":"node-abc123"}]`,
	}

	getIndicesTestSetup := &ServerSetup{
		Method:   "GET",
		Path:     "/_cat/indices",
		Response: `[{"health":"green","status":"open","index":"test_index","pri":"5","rep":"1","store.size":"3.6kb", "docs.count":"1500"}]`,
	}
	host, port, ts := setupTestServers(t, []*ServerSetup{getShardsTestSetup, getIndicesTestSetup})
	defer ts.Close()
	client := NewClient(host, port)

	overlap, err := client.GetShardOverlap([]string{"node-abc123"})

	if err != nil {
		t.Errorf("Unexpected error, get %s", err)
	}

	if overlap == nil {
		t.Errorf("Expected slice of shards, got nil instead")
	}

	if val, ok := overlap["test_index_1"]; ok {
		if val.SafeToRemove() {
			t.Error("Expected SafeToRemove=false, got true instead")
		}
	} else {
		t.Errorf("Expected overlap data, got nil instead")
	}
}

var getRecoveryTestSetup = &ServerSetup{
	Method:   "GET",
	Path:     "/_cat/recovery",
	Response: `[{"index":"test_index","shard":"0","time":"2h","type":"peer","stage":"index","source_host":"123.123.123.123","source_node":"node-0","target_host":"124.124.124.124","target_node":"node-1","repository":"n/a","snapshot":"n/a","files":"400","files_recovered":"100","files_percent":"25%","files_total":"400","bytes":"400","bytes_recovered":"100","bytes_percent":"25%","bytes_total":"400","translog_ops":"400","translog_ops_recovered":"0","translog_ops_percent":"0.0%"}]`,
}

func TestGetShardRecovery(t *testing.T) {
	host, port, ts := setupTestServers(t, []*ServerSetup{getRecoveryTestSetup})
	defer ts.Close()
	client := NewClient(host, port)

	recoveries, err := client.GetShardRecovery([]string{}, false)

	if err != nil {
		t.Errorf("Unexpected error getting shard recoveries: %s", err)
	}

	assert.Equal(t, len(recoveries), 1)

	assert.DeepEqual(t, recoveries[0], ShardRecovery{
		Index:                "test_index",
		Shard:                "0",
		Time:                 "2h",
		Type:                 "peer",
		Stage:                "index",
		SourceHost:           "123.123.123.123",
		SourceNode:           "node-0",
		TargetHost:           "124.124.124.124",
		TargetNode:           "node-1",
		Repository:           "n/a",
		Snapshot:             "n/a",
		Bytes:                400,
		BytesTotal:           400,
		BytesRecovered:       100,
		BytesPercent:         "25%",
		Files:                400,
		FilesTotal:           400,
		FilesRecovered:       100,
		FilesPercent:         "25%",
		TranslogOps:          400,
		TranslogOpsRecovered: 0,
		TranslogOpsPercent:   "0.0%",
	})

}

func TestGetShardRecoveryRemaining(t *testing.T) {
	host, port, ts := setupTestServers(t, []*ServerSetup{getRecoveryTestSetup})
	defer ts.Close()
	client := NewClient(host, port)

	recoveries, err := client.GetShardRecovery([]string{}, false)
	if err != nil {
		t.Errorf("Unexpected error getting shard recoveries: %s", err)
	}

	estRemaining, err := recoveries[0].TimeRemaining()
	if err != nil {
		t.Errorf("Error getting estimated time remaining: %s", err)
	}

	assert.Equal(t, estRemaining, time.Hour*6)
}

func TestReloadSecureSettings(t *testing.T) {
	serverSetup := &ServerSetup{
		Method:   "POST",
		Path:     "/_nodes/reload_secure_settings",
		Response: `{"_nodes":{"total":2,"successful":2,"failed":0},"cluster_name":"vulcanizer-elasticsearch-v7","nodes":{"iJeJx6ydSbKf_cvzDt1_gg":{"name":"vulcanizer-elasticsearch-v7"},"GXtqL0WdSguHQdo2xHNX_A":{"name":"vulcanizer-elasticsearch-v7-2","reload_exception":{"type":"illegal_state_exception","reason":"Keystore is missing"}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{serverSetup})
	defer ts.Close()
	client := NewClient(host, port)

	response, err := client.ReloadSecureSettings()

	if err != nil {
		t.Errorf("Unexpected error, get %s", err)
	}

	if response.Summary.Successful != 2 {
		t.Errorf("Expected response to parse 2 successful nodes from summary, got %#v", response)
	}

	goodNode := response.Nodes["iJeJx6ydSbKf_cvzDt1_gg"]
	badNode := response.Nodes["GXtqL0WdSguHQdo2xHNX_A"]

	if goodNode.Name != "vulcanizer-elasticsearch-v7" && goodNode.ReloadException != nil {
		t.Errorf("Expected to parse good node response correctly, got %#v", goodNode)
	}

	if badNode.Name != "vulcanizer-elasticsearch-v7-2" && badNode.ReloadException.Reason != "Keystore is missing" {
		t.Errorf("Expected to parse bad node response correctly, got %#v", goodNode)
	}
}

func TestReloadSecureSettingsWithPassword(t *testing.T) {
	serverSetup := &ServerSetup{
		Method:   "POST",
		Path:     "/_nodes/reload_secure_settings",
		Body:     `{"secure_settings_password":"123456"}`,
		Response: `{"_nodes":{"total":2,"successful":2,"failed":0},"cluster_name":"vulcanizer-elasticsearch-v7","nodes":{"iJeJx6ydSbKf_cvzDt1_gg":{"name":"vulcanizer-elasticsearch-v7"},"GXtqL0WdSguHQdo2xHNX_A":{"name":"vulcanizer-elasticsearch-v7-2","reload_exception":{"type":"illegal_state_exception","reason":"Keystore is missing"}}}}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{serverSetup})
	defer ts.Close()
	client := NewClient(host, port)

	response, err := client.ReloadSecureSettingsWithPassword("123456")

	if err != nil {
		t.Errorf("Unexpected error, get %s", err)
	}

	if response.Summary.Successful != 2 {
		t.Errorf("Expected response to parse 2 successful nodes from summary, got %#v", response)
	}

	goodNode := response.Nodes["iJeJx6ydSbKf_cvzDt1_gg"]
	badNode := response.Nodes["GXtqL0WdSguHQdo2xHNX_A"]

	if goodNode.Name != "vulcanizer-elasticsearch-v7" && goodNode.ReloadException != nil {
		t.Errorf("Expected to parse good node response correctly, got %#v", goodNode)
	}

	if badNode.Name != "vulcanizer-elasticsearch-v7-2" && badNode.ReloadException.Reason != "Keystore is missing" {
		t.Errorf("Expected to parse bad node response correctly, got %#v", goodNode)
	}
}

func TestGetHotThreads(t *testing.T) {
	testSetup := &ServerSetup{
		Method: "GET",
		Path:   "/_nodes/hot_threads",
		Response: `
::: {Mister Sinister}{c0k7r8tKS0CGObWF7yzgQQ}{127.0.0.1}{127.0.0.1:9300}
   Hot threads at 2022-08-04T20:30:34.357Z, interval=500ms, busiestThreads=3, ignoreIdleThreads=true:

    0.0% (172.8micros out of 500ms) cpu usage by thread 'elasticsearch[Mister Sinister][transport_client_timer][T#1]{Hashed wheel timer #1}'
     10/10 snapshots sharing following 5 elements
       java.base@11.0.16/java.lang.Thread.sleep(Native Method)
       app//org.jboss.netty.util.HashedWheelTimer$Worker.waitForNextTick(HashedWheelTimer.java:445)
       app//org.jboss.netty.util.HashedWheelTimer$Worker.run(HashedWheelTimer.java:364)
       app//org.jboss.netty.util.ThreadRenamingRunnable.run(ThreadRenamingRunnable.java:108)
       java.base@11.0.16/java.lang.Thread.run(Thread.java:829)`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	hotThreads, err := client.GetHotThreads()
	if err != nil {
		t.Fatalf("Unexpected error expected nil, got %s", err)
	}

	if hotThreads != testSetup.Response {
		t.Errorf("Unexpected response. got %v want %v", hotThreads, testSetup.Response)
	}
}

func TestGetNodesHotThreads(t *testing.T) {
	testSetup := &ServerSetup{
		Method: "GET",
		Path:   "/_nodes/nodeid1,nodeid2/hot_threads",
		Response: `
::: {Mister Sinister}{nodeid1}{127.0.0.1}{127.0.0.1:9300}
   Hot threads at 2022-08-04T20:30:34.357Z, interval=500ms, busiestThreads=3, ignoreIdleThreads=true:`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	hotThreads, err := client.GetNodesHotThreads([]string{"nodeid1  ", "  nodeid2"})
	if err != nil {
		t.Fatalf("Unexpected error expected nil, got %s", err)
	}

	if hotThreads != testSetup.Response {
		t.Errorf("Unexpected response. got %v want %v", hotThreads, testSetup.Response)
	}
}

func TestClusterAllocationExplain(t *testing.T) {
	shardID := 0
	tests := []struct {
		name         string
		request      *ClusterAllocationExplainRequest
		prettyOutput bool
		expectedBody string
	}{
		{
			name:         "with nil request",
			request:      nil,
			expectedBody: "",
		},
		{
			name: "with current_node set",
			request: &ClusterAllocationExplainRequest{
				CurrentNode: "test-node",
			},
			expectedBody: `{"current_node":"test-node"}`,
		},
		{
			name: "with index set",
			request: &ClusterAllocationExplainRequest{
				Index: "test-index",
			},
			expectedBody: `{"index":"test-index"}`,
		},
		{
			name: "with primary set",
			request: &ClusterAllocationExplainRequest{
				Primary: true,
			},
			expectedBody: `{"primary":true}`,
		},
		{
			name: "with shard set",
			request: &ClusterAllocationExplainRequest{
				Shard: &shardID,
			},
			expectedBody: `{"shard":0}`,
		},
		{
			name:         "with pretty output",
			request:      nil,
			prettyOutput: true,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			testSetup := &ServerSetup{
				Method: "GET",
				Path:   "/_cluster/allocation/explain",
				Body:   tc.expectedBody,
			}

			if tc.prettyOutput {
				testSetup.extraChecksFn = func(t *testing.T, r *http.Request) {
					expectedURL := "/_cluster/allocation/explain?pretty="
					if r.URL.String() != expectedURL {
						t.Errorf("Unexpected url query. Want %s, got %s", expectedURL, r.URL.String())
					}
				}
			}

			host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
			defer ts.Close()
			client := NewClient(host, port)

			_, err := client.ClusterAllocationExplain(tc.request, tc.prettyOutput)
			if err != nil {
				t.Fatalf("Unexpected error. expected nil, got %s", err)
			}
		})
	}
}

func TestReroute(t *testing.T) {
	testSetup := &ServerSetup{
		Method: "POST",
		Path:   "/_cluster/reroute",
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	err := client.RerouteWithRetryFailed()
	if err != nil {
		t.Fatalf("Unexpected error expected nil, got %s", err)
	}
}

func TestAllocateStalePrimaryShard(t *testing.T) {
	testSetup := &ServerSetup{
		Method: "POST",
		Path:   "/_cluster/reroute",
		Body:   `{"commands":[{"allocate_stale_primary":{"accept_data_loss":true,"index":"test-index","node":"test-node","shard":0}}]}`,
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	err := client.AllocateStalePrimaryShard("test-node", "test-index", 0)
	if err != nil {
		t.Fatalf("Unexpected error. expected nil, got %s", err)
	}
}

func TestRemoveIndexILMPolicy(t *testing.T) {
	testSetup := &ServerSetup{
		Method: "POST",
		Path:   "/test-index/_ilm/remove",
	}

	host, port, ts := setupTestServers(t, []*ServerSetup{testSetup})
	defer ts.Close()
	client := NewClient(host, port)

	err := client.RemoveIndexILMPolicy("test-index")
	if err != nil {
		t.Fatalf("Unexpected error. expected nil, got %s", err)
	}
}
