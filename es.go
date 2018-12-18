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

//Hold the values for what values are in the cluster.allocation.exclude settings.
//Relevant Elasticsearch documentation: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/allocation-filtering.html
type ExcludeSettings struct {
	Ips, Hosts, Names []string
}

//Hold connection information to a Elasticsearch cluster.
type Client struct {
	Host string
	Port int
}

//Holds information about an Elasticsearch node, based on the _cat/nodes API: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cat-nodes.html
type Node struct {
	Name   string `json:"name"`
	Ip     string `json:"ip"`
	Id     string `json:"id"`
	Role   string `json:"role"`
	Master string `json:"master"`
	Jdk    string `json:"jdk"`
}

//Holds information about an Elasticsearch index, based on the _cat/indices API: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cat-indices.html
type Index struct {
	Health        string `json:"health"`
	Status        string `json:"status"`
	Name          string `json:"index"`
	PrimaryShards int    `json:"pri,string"`
	ReplicaCount  int    `json:"rep,string"`
	IndexSize     string `json:"store.size"`
	DocumentCount int    `json:"docs.count,string"`
}

//Holds information about the health of an Elasticsearch cluster, based on the cluster health API: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cluster-health.html
type ClusterHealth struct {
	Cluster                string  `json:"cluster_name"`
	Status                 string  `json:"status"`
	ActiveShards           int     `json:"active_shards"`
	RelocatingShards       int     `json:"relocating_shards"`
	InitializingShards     int     `json:"initializing_shards"`
	UnassignedShards       int     `json:"unassigned_shards"`
	ActiveShardsPercentage float64 `json:"active_shards_percent_as_number"`
	Message                string
	RawIndices             map[string]IndexHealth `json:"indices"`
	HealthyIndices         []IndexHealth
	UnhealthyIndices       []IndexHealth
}

//Holds information about the health of an Elasticsearch index, based on the index level of the cluster health API: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cluster-health.html
type IndexHealth struct {
	Name               string
	Status             string `json:"status"`
	ActiveShards       int    `json:"active_shards"`
	RelocatingShards   int    `json:"relocating_shards"`
	InitializingShards int    `json:"initializing_shards"`
	UnassignedShards   int    `json:"unassigned_shards"`
}

//Holds slices for persistent and transient cluster settings.
type ClusterSettings struct {
	PersistentSettings []ClusterSetting
	TransientSettings  []ClusterSetting
}

//A setting name and value with the setting name to be a "collapsed" version of the setting. A setting of:
//  { "indices": { "recovery" : { "max_bytes_per_sec": "10mb" } } }
//would be represented by:
//  ClusterSetting{ Setting: "indices.recovery.max_bytes_per_sec", Value: "10mb" }
type ClusterSetting struct {
	Setting, Value string
}

type snapshotWrapper struct {
	Snapshots []Snapshot `json:"snapshots"`
}

type acknowledgedResponse struct {
	Acknowledged bool `json:"acknowledged"`
}

//Holds information about an Elasticsearch snapshot, based on the snapshot API: https://www.elastic.co/guide/en/elasticsearch/reference/current/modules-snapshots.html
type Snapshot struct {
	State          string    `json:"state"`
	Name           string    `json:"snapshot"`
	StartTime      time.Time `json:"start_time,string"`
	EndTime        time.Time `json:"end_time,string"`
	DurationMillis int       `json:"duration_in_millis"`
	Indices        []string  `json:"indices"`
	Shards         struct {
		Total      int `json:"total"`
		Failed     int `json:"failed"`
		Successful int `json:"successful"`
	} `json:"shards"`
	Failures []struct {
		Index   string `json:"index"`
		ShardID int    `json:"shard_id"`
		Reason  string `json:"reason"`
		NodeID  string `json:"node_id"`
		Status  string `json:"status"`
	} `json:"failures"`
}

//Initialize a new vulcanizer client to use.
func NewClient(host string, port int) *Client {
	return &Client{host, port}
}

const clusterSettingsPath = "_cluster/settings"

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

func handleErrWithBytes(s *gorequest.SuperAgent) ([]byte, error) {
	response, body, errs := s.EndBytes()

	if len(errs) > 0 {
		return nil, combineErrors(errs)
	}

	if response.StatusCode != http.StatusOK {
		errorMessage := fmt.Sprintf("Bad HTTP Status from Elasticsearch: %v, %s", response.StatusCode, body)
		return nil, errors.New(errorMessage)
	}
	return body, nil
}

func handleErrWithStruct(s *gorequest.SuperAgent, v interface{}) error {
	response, body, errs := s.EndStruct(v)

	if len(errs) > 0 {
		return combineErrors(errs)
	}

	if response.StatusCode != http.StatusOK {
		errorMessage := fmt.Sprintf("Bad HTTP Status from Elasticsearch: %v, %s", response.StatusCode, body)
		return errors.New(errorMessage)
	}
	return nil
}

func (c *Client) buildGetRequest(path string) *gorequest.SuperAgent {
	return gorequest.New().Get(fmt.Sprintf("http://%s:%v/%s", c.Host, c.Port, path)).Set("Accept", "application/json")
}

func (c *Client) buildPutRequest(path string) *gorequest.SuperAgent {
	return gorequest.New().Put(fmt.Sprintf("http://%s:%v/%s", c.Host, c.Port, path))
}

func (c *Client) buildDeleteRequest(path string) *gorequest.SuperAgent {
	return gorequest.New().Delete(fmt.Sprintf("http://%s:%v/%s", c.Host, c.Port, path))
}

func (c *Client) buildPostRequest(path string) *gorequest.SuperAgent {
	return gorequest.New().Post(fmt.Sprintf("http://%s:%v/%s", c.Host, c.Port, path))
}

// Get current cluster settings for shard allocation exclusion rules.
func (c *Client) GetClusterExcludeSettings() (ExcludeSettings, error) {
	body, err := handleErrWithBytes(c.buildGetRequest(clusterSettingsPath))

	if err != nil {
		return ExcludeSettings{}, err
	}

	excludedArray := gjson.GetManyBytes(body, "transient.cluster.routing.allocation.exclude._ip", "transient.cluster.routing.allocation.exclude._name", "transient.cluster.routing.allocation.exclude._host")

	excludeSettings := excludeSettingsFromJson(excludedArray)
	return excludeSettings, nil
}

//Set shard allocation exclusion rules such that the Elasticsearch node with the name `serverToDrain` is excluded. This should cause Elasticsearch to migrate shards away from that node.
//
//Use case: You need to restart an Elasticsearch node. In order to do so safely, you should migrate data away from it. Calling `DrainServer` with the node name will move data off of the specified node.
func (c *Client) DrainServer(serverToDrain string) (ExcludeSettings, error) {
	excludeSettings, err := c.GetClusterExcludeSettings()

	if err != nil {
		return ExcludeSettings{}, err
	}

	excludeSettings.Names = append(excludeSettings.Names, serverToDrain)

	agent := c.buildPutRequest(clusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(fmt.Sprintf(`{"transient" : { "cluster.routing.allocation.exclude._name" : "%s"}}`, strings.Join(excludeSettings.Names, ",")))

	_, err = handleErrWithBytes(agent)

	if err != nil {
		return ExcludeSettings{}, err
	}

	return excludeSettings, nil
}

//Set shard allocation exclusion rules such that the Elasticsearch node with the name `serverToFill` is no longer being excluded. This should cause Elasticsearch to migrate shards to that node.
//
//Use case: You have completed maintenance on an Elasticsearch node and it's ready to hold data again. Calling `FillOneServer` with the node name will remove that node name from the shard exclusion rules and allow data to be relocated onto the node.
func (c *Client) FillOneServer(serverToFill string) (ExcludeSettings, error) {

	// Get the current list of strings
	excludeSettings, err := c.GetClusterExcludeSettings()
	if err != nil {
		return ExcludeSettings{}, err
	}

	serverToFill = strings.TrimSpace(serverToFill)

	newNamesDrained := []string{}
	for _, s := range excludeSettings.Names {
		if s != serverToFill {
			newNamesDrained = append(newNamesDrained, s)
		}
	}

	agent := c.buildPutRequest(clusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(fmt.Sprintf(`{"transient" : { "cluster.routing.allocation.exclude._name" : "%s"}}`, strings.Join(newNamesDrained, ",")))

	_, err = handleErrWithBytes(agent)

	if err != nil {
		return ExcludeSettings{}, err
	}

	return c.GetClusterExcludeSettings()
}

//Removes all shard allocation exclusion rules.
//
//Use case: You had been performing maintenance on a number of Elasticsearch nodes. They are all ready to receive data again. Calling `FillAll` will remove all the allocation exclusion rules on the cluster, allowing Elasticsearch to freely allocate shards on the previously excluded nodes.
func (c *Client) FillAll() (ExcludeSettings, error) {

	agent := c.buildPutRequest(clusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(`{"transient" : { "cluster.routing.allocation.exclude" : { "_name" :  "", "_ip" : "", "_host" : ""}}}`)

	body, err := handleErrWithBytes(agent)

	if err != nil {
		return ExcludeSettings{}, err
	}

	excludedArray := gjson.GetManyBytes(body, "transient.cluster.routing.allocation.exclude._ip", "transient.cluster.routing.allocation.exclude._name", "transient.cluster.routing.allocation.exclude._host")

	return excludeSettingsFromJson(excludedArray), nil
}

//Get all the nodes in the cluster.
//
//Use case: You want to see what nodes Elasticsearch considers part of the cluster.
func (c *Client) GetNodes() ([]Node, error) {
	var nodes []Node

	agent := c.buildGetRequest("_cat/nodes?h=master,role,name,ip,id,jdk")
	err := handleErrWithStruct(agent, &nodes)

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

//Get all the indices in the cluster.
//
//Use case: You want to see some basic info on all the indices of the cluster.
func (c *Client) GetIndices() ([]Index, error) {
	var indices []Index
	err := handleErrWithStruct(c.buildGetRequest("_cat/indices?h=health,status,index,pri,rep,store.size,docs.count"), &indices)

	if err != nil {
		return nil, err
	}

	return indices, nil
}

//Get the health of the cluster.
//
//Use case: You want to see information needed to determine if the Elasticsearch cluster is healthy (green) or not (yellow/red).
func (c *Client) GetHealth() (ClusterHealth, error) {
	var health ClusterHealth
	err := handleErrWithStruct(c.buildGetRequest("_cluster/health?level=indices"), &health)
	if err != nil {
		return ClusterHealth{}, err
	}

	for indexName, index := range health.RawIndices {
		index.Name = indexName

		if index.Status == "green" {
			health.HealthyIndices = append(health.HealthyIndices, index)
		} else {
			health.UnhealthyIndices = append(health.UnhealthyIndices, index)
		}
	}

	health.Message = captionHealth(health)

	return health, nil
}

//Get all the persistent and transient cluster settings.
//
//Use case: You want to see the current settings in the cluster.
func (c *Client) GetSettings() (ClusterSettings, error) {
	clusterSettings := ClusterSettings{}
	body, err := handleErrWithBytes(c.buildGetRequest(clusterSettingsPath))

	if err != nil {
		return clusterSettings, err
	}

	rawPersistentSettings := gjson.GetBytes(body, "persistent").Raw
	rawTransientSettings := gjson.GetBytes(body, "transient").Raw

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

//Enables or disables allocation for the cluster.
//
//Use case: You are performing an operation the cluster where nodes may be dropping in and out. Elasticsearch will typically try to rebalance immediately but you want the cluster to hold off rebalancing until you complete your task. Calling `SetAllocation("disable")` will disable allocation so Elasticsearch won't move/relocate any shards. Once you complete your task, calling `SetAllocation("enable")` will allow Elasticsearch to relocate shards again.
func (c *Client) SetAllocation(allocation string) (string, error) {

	var allocationSetting string

	if allocation == "enable" {
		allocationSetting = "all"
	} else {
		allocationSetting = "none"
	}

	agent := c.buildPutRequest(clusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(fmt.Sprintf(`{"transient" : { "cluster.routing.allocation.enable" : "%s"}}`, allocationSetting))

	body, err := handleErrWithBytes(agent)

	if err != nil {
		return "", err
	}

	allocationVal := gjson.GetBytes(body, "transient.cluster.routing.allocation.enable")

	return allocationVal.String(), nil
}

//Set a new value for a cluster setting
//
//Use case: You've doubled the number of nodes in your cluster and you want to increase the number of shards the cluster can relocate at one time. Calling `SetSetting("cluster.routing.allocation.cluster_concurrent_rebalance", "100")` will update that value with the cluster. Once data relocation is complete you can decrease the setting by calling `SetSetting("cluster.routing.allocation.cluster_concurrent_rebalance", "20")`.
func (c *Client) SetSetting(setting string, value string) (string, string, error) {

	settingsBody, err := handleErrWithBytes(c.buildGetRequest(clusterSettingsPath))

	if err != nil {
		return "", "", err
	}

	existingValues := gjson.GetManyBytes(settingsBody, fmt.Sprintf("transient.%s", setting), fmt.Sprintf("persistent.%s", setting))

	agent := c.buildPutRequest(clusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(fmt.Sprintf(`{"transient" : { "%s" : "%s"}}`, setting, value))

	body, err := handleErrWithBytes(agent)

	if err != nil {
		return "", "", err
	}

	newValue := gjson.GetBytes(body, fmt.Sprintf("transient.%s", setting)).String()

	var existingValue string

	if existingValues[0].String() == "" {
		existingValue = existingValues[1].String()
	} else {
		existingValue = existingValues[0].String()
	}

	return existingValue, newValue, nil
}

//List the snapshots of the given repository.
//
//Use case: You want to see information on snapshots in a repository.
func (c *Client) GetSnapshots(repository string) ([]Snapshot, error) {

	var snapshotWrapper snapshotWrapper

	err := handleErrWithStruct(c.buildGetRequest(fmt.Sprintf("_snapshot/%s/_all", repository)), &snapshotWrapper)

	if err != nil {
		return nil, err
	}

	return snapshotWrapper.Snapshots, nil
}

//Get detailed information about a particular snapshot.
//
//Use case: You had a snapshot fail and you want to see the reason why and what shards/nodes the error occurred on.
func (c *Client) GetSnapshotStatus(repository string, snapshot string) (Snapshot, error) {

	var snapshotWrapper snapshotWrapper

	err := handleErrWithStruct(c.buildGetRequest(fmt.Sprintf("_snapshot/%s/%s", repository, snapshot)), &snapshotWrapper)

	if err != nil {
		return Snapshot{}, err
	}

	return snapshotWrapper.Snapshots[0], nil
}

//Delete a snapshot
//
//Use case: You want to delete older snapshots so that they don't take up extra space.
func (c *Client) DeleteSnapshot(repository string, snapshot string) error {
	var response acknowledgedResponse

	err := handleErrWithStruct(c.buildDeleteRequest(fmt.Sprintf("_snapshot/%s/%s", repository, snapshot)).Timeout(10*time.Minute), &response)

	if err != nil {
		return err
	}

	if !response.Acknowledged {
		return fmt.Errorf(`Request to delete snapshot "%s" on respository "%s" was not acknowledged. %+v`, snapshot, repository, response)
	}

	return nil
}

//Verify a snapshot repository
//
//Use case: Have Elasticsearch verify a repository to make sure that all nodes can access the snapshot location correctly.
func (c *Client) VerifyRepository(repository string) (bool, error) {

	_, err := handleErrWithBytes(c.buildPostRequest(fmt.Sprintf("_snapshot/%s/_verify", repository)))

	if err != nil {
		return false, err
	}

	return true, nil
}
