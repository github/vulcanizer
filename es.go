package vulcanizer

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jeremywohl/flatten"
	"github.com/parnurzeal/gorequest"
	"github.com/tidwall/gjson"
)

type ExcludeSettings struct {
	Ips, Hosts, Names []string
}

type Client struct {
	Host string
	Port int
}

type Node struct {
	Name   string `json:"name"`
	Ip     string `json:"ip"`
	Id     string `json:"id"`
	Role   string `json:"role"`
	Master string `json:"master"`
}

type Index struct {
	Health        string `json:"health"`
	Status        string `json:"status"`
	Name          string `json:"index"`
	PrimaryShards int    `json:"pri,string"`
	ReplicaCount  int    `json:"rep,string"`
	IndexSize     string `json:"store.size"`
	DocumentCount int    `json:"docs.count,string"`
}

type ClusterHealth struct {
	Cluster                string `json:"cluster"`
	Status                 string `json:"status"`
	RelocatingShards       int    `json:"relo,string"`
	InitializingShards     int    `json:"init,string"`
	UnassignedShards       int    `json:"unassign,string"`
	ActiveShardsPercentage string `json:"active_shards_percent"`
	Message                string
}

type ClusterSettings struct {
	PersistentSettings []ClusterSetting
	TransientSettings  []ClusterSetting
}

type ClusterSetting struct {
	Setting, Value string
}

func NewClient(host string, port int) *Client {
	return &Client{host, port}
}

const clusterSettingsPath = "_cluster/settings"

func (c *Client) buildGetRequest(path string) *gorequest.SuperAgent {
	return gorequest.New().Get(fmt.Sprintf("http://%s:%v/%s", c.Host, c.Port, path)).Set("Accept", "application/json")
}

func (c *Client) buildPutRequest(path string) *gorequest.SuperAgent {
	return gorequest.New().Put(fmt.Sprintf("http://%s:%v/%s", c.Host, c.Port, path))
}

// Get current cluster settings for exclusion
func (c *Client) GetClusterExcludeSettings() *ExcludeSettings {
	_, body, _ := c.buildGetRequest(clusterSettingsPath).End()

	excludedArray := gjson.GetMany(body, "transient.cluster.routing.allocation.exclude._ip", "transient.cluster.routing.allocation.exclude._name", "transient.cluster.routing.allocation.exclude._host")

	excludeSettings := ExcludeSettingsFromJson(excludedArray)
	return excludeSettings
}

func (c *Client) DrainServer(serverToDrain string, namesExcluded string) (excludedServers string) {

	var drainList string

	if namesExcluded != "None" {
		drainList = serverToDrain + "," + namesExcluded

	} else {
		drainList = serverToDrain
	}

	_, body, _ := c.buildPutRequest(clusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(`{"transient" : { "cluster.routing.allocation.exclude._name" : "` + drainList + `"}}`).
		End()

	drainingServers := gjson.Get(body, "transient.cluster.routing.allocation.exclude._name")

	return drainingServers.String()
}

func (c *Client) FillOneServer(serverToFill string) (serverFilling, excludedServers string) {

	// Get the current list of strings
	excludeSettings := c.GetClusterExcludeSettings()

	serverToFill = strings.TrimSpace(serverToFill)

	newNamesDrained := []string{}
	for _, s := range excludeSettings.Names {
		if s != serverToFill {
			newNamesDrained = append(newNamesDrained, s)
		}
	}

	_, body, _ := c.buildPutRequest(clusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(`{"transient" : { "cluster.routing.allocation.exclude._name" : "` + strings.Join(newNamesDrained, ",") + `"}}`).
		End()

	drainingServers := gjson.Get(body, "transient.cluster.routing.allocation.exclude._name")

	return serverToFill, drainingServers.String()
}

func (c *Client) FillAll() *ExcludeSettings {

	_, body, _ := c.buildPutRequest(clusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(`{"transient" : { "cluster.routing.allocation.exclude" : { "_name" :  "", "_ip" : "", "_host" : ""}}}`).
		End()

	excludedArray := gjson.GetMany(body, "transient.cluster.routing.allocation.exclude._ip", "transient.cluster.routing.allocation.exclude._name", "transient.cluster.routing.allocation.exclude._host")

	return ExcludeSettingsFromJson(excludedArray)
}

func (c *Client) GetNodes() ([]Node, error) {
	var nodes []Node
	_, _, errs := c.buildGetRequest("_cat/nodes?h=master,role,name,ip,id").EndStruct(&nodes)

	if len(errs) > 0 {
		return nil, combineErrors(errs)
	}

	return nodes, nil
}

func (c *Client) GetIndices() ([]Index, error) {
	var indices []Index
	_, _, errs := c.buildGetRequest("_cat/indices?h=health,status,index,pri,rep,store.size,docs.count").EndStruct(&indices)

	if len(errs) > 0 {
		return nil, combineErrors(errs)
	}

	return indices, nil
}

func (c *Client) GetHealth() ([]ClusterHealth, error) {
	var health []ClusterHealth
	_, _, errs := c.buildGetRequest("_cat/health?h=cluster,status,relo,init,unassign,pending_tasks,active_shards_percent").EndStruct(&health)

	if len(errs) > 0 {
		return nil, combineErrors(errs)
	}

	for i := range health {
		health[i].Message = captionHealth(health[i].Status)
	}

	return health, nil
}

func settingsToStructs(rawJson string) ([]ClusterSetting, error) {
	flatSettings, err := flatten.FlattenString(rawJson, "", flatten.DotStyle)
	if err != nil {
		return nil, err
	}

	settingsMap, _ := gjson.Parse(flatSettings).Value().(map[string]interface{})
	keys := []string{}

	for k, v := range settingsMap {
		strValue := v.(string)
		if strValue != "" {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)

	var clusterSettings []ClusterSetting
	for _, k := range keys {
		setting := ClusterSetting{
			Setting: k,
			Value:   settingsMap[k].(string),
		}

		clusterSettings = append(clusterSettings, setting)
	}
	return clusterSettings, nil
}

func (c *Client) GetSettings() (ClusterSettings, error) {
	_, body, errs := c.buildGetRequest(clusterSettingsPath).End()

	clusterSettings := ClusterSettings{}

	if len(errs) > 0 {
		return clusterSettings, combineErrors(errs)
	}

	rawPersistentSettings := gjson.Get(body, "persistent").Raw
	rawTransientSettings := gjson.Get(body, "transient").Raw

	persisentSettings, err := settingsToStructs(rawPersistentSettings)
	if err != nil {
		return clusterSettings, err
	}

	transientSetting, err := settingsToStructs(rawTransientSettings)
	if err != nil {
		return clusterSettings, err
	}

	clusterSettings.PersistentSettings = persisentSettings
	clusterSettings.TransientSettings = transientSetting

	return clusterSettings, nil
}

func (c *Client) SetAllocation(allocation string) string {

	var allocationSetting string

	if allocation == "enable" {
		allocationSetting = "all"
	} else {
		allocationSetting = "none"
	}

	_, body, _ := c.buildPutRequest(clusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(fmt.Sprintf(`{"transient" : { "cluster.routing.allocation.enable" : "%s"}}`, allocationSetting)).
		End()

	allocationVal := gjson.Get(body, "transient.cluster.routing.allocation.enable")

	return allocationVal.String()
}

func (c *Client) SetSetting(setting string, value string) (string, string, error) {

	_, settingsBody, _ := c.buildGetRequest(clusterSettingsPath).End()

	existingValues := gjson.GetMany(settingsBody, fmt.Sprintf("transient.%s", setting), fmt.Sprintf("persistent.%s", setting))

	response, body, errs := c.buildPutRequest(clusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(fmt.Sprintf(`{"transient" : { "%s" : "%s"}}`, setting, value)).
		End()

	if len(errs) > 0 {
		return "", "", combineErrors(errs)
	}

	if response.StatusCode != http.StatusOK {
		errorMessage := fmt.Sprintf("Bad HTTP Status of %v from Elasticsearch: %s", response.StatusCode, body)
		return "", "", errors.New(errorMessage)
	}

	newValue := gjson.Get(body, fmt.Sprintf("transient.%s", setting)).String()

	var existingValue string

	if existingValues[0].String() == "" {
		existingValue = existingValues[1].String()
	} else {
		existingValue = existingValues[0].String()
	}

	return existingValue, newValue, nil
}

func (c *Client) GetSnapshots(repository string) ([][]string, []string) {

	_, body, _ := c.buildGetRequest(fmt.Sprintf("_snapshot/%s/_all", repository)).End()

	results := [][]string{}
	headers := []string{"state", "snapshot", "finished", "duration"}

	snapshotResults := gjson.Get(body, "snapshots")
	snapshotResults.ForEach(func(key, value gjson.Result) bool {

		millis := value.Get("duration_in_millis").String()
		duration, _ := time.ParseDuration(fmt.Sprintf("%sms", millis))

		result := []string{
			value.Get("state").String(),
			value.Get("snapshot").String(),
			value.Get("end_time").String(),
			fmt.Sprintf("%v", duration),
		}

		results = append(results, result)
		return true // keep iterating
	})

	if len(results) > 10 {
		results = results[len(results)-10:]
	}

	return results, headers
}

func (c *Client) GetSnapshotStatus(repository string, snapshot string) ([][]string, []string) {
	_, body, _ := c.buildGetRequest(fmt.Sprintf("_snapshot/%s/%s", repository, snapshot)).End()

	headers := []string{"metric", "value"}

	snapshotResult := gjson.Get(body, "snapshots.0")

	millis := snapshotResult.Get("duration_in_millis").String()
	duration, _ := time.ParseDuration(fmt.Sprintf("%sms", millis))

	indices := snapshotResult.Get("indices").Array()

	display_indices := []string{}

	for _, index := range indices {
		display_indices = append(display_indices, index.String())
	}

	results := [][]string{
		[]string{"state", snapshotResult.Get("state").String()},
		[]string{"snapshot", snapshotResult.Get("snapshot").String()},
		[]string{"indices", strings.Join(display_indices, "\n")},
		[]string{"started", snapshotResult.Get("start_time").String()},
		[]string{"finished", snapshotResult.Get("end_time").String()},
		[]string{"duration", fmt.Sprintf("%v", duration)},
		[]string{"shards", fmt.Sprintf("Successful shards %s, failed shards %s", snapshotResult.Get("shards.successful").String(), snapshotResult.Get("shards.failed").String())},
	}

	return results, headers
}

func (c *Client) PerformSnapshotsCheck(cluster string) ([]string, []string) {
	_, body, _ := c.buildGetRequest(fmt.Sprintf("_snapshot/%s/_all", cluster)).End()

	results := []map[string]interface{}{}

	snapshotResults := gjson.Get(body, "snapshots")
	snapshotResults.ForEach(func(key, value gjson.Result) bool {

		snapshot := value.Value().(map[string]interface{})
		results = append(results, snapshot)
		return true // keep iterating
	})

	if len(results) > 5 {
		results = results[len(results)-5:]
	}

	goodSnapshots := []string{}
	badSnapshots := []string{}

	for _, snapshot := range results {
		name := snapshot["snapshot"].(string)

		if snapshot["state"].(string) == "SUCCESS" {
			goodSnapshots = append(goodSnapshots, name)
		} else {
			badSnapshots = append(badSnapshots, name)
		}
	}

	return goodSnapshots, badSnapshots
}
