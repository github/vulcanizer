package vulcanizer

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jeremywohl/flatten"
	"github.com/parnurzeal/gorequest"
	"github.com/tidwall/gjson"
)

// Hold the values for what values are in the cluster.allocation.exclude settings.
// Relevant Elasticsearch documentation: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/allocation-filtering.html
type ExcludeSettings struct {
	Ips, Hosts, Names []string
}

type Auth struct {
	User     string
	Password string
}

// Hold connection information to a Elasticsearch cluster.
type Client struct {
	Host      string
	Port      int
	Secure    bool
	Path      string
	TLSConfig *tls.Config
	Timeout   time.Duration
	*Auth
}

// Holds information about an Elasticsearch node, based on a combination of the
// _cat/nodes and _cat/allocationAPI: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cat-nodes.html,
// https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cat-allocation.html
type Node struct {
	Name        string `json:"name"`
	IP          string `json:"ip"`
	ID          string `json:"id"`
	Role        string `json:"role"`
	Master      string `json:"master"`
	Jdk         string `json:"jdk"`
	Version     string `json:"version"`
	Shards      string
	DiskIndices string
	DiskUsed    string
	DiskAvail   string
	DiskTotal   string
	DiskPercent string
}

// Holds a subset of information from the _nodes/stats endpoint:
// https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-nodes-stats.html
type NodeStats struct {
	Name     string
	Role     string
	JVMStats NodeJVM
}

// Holds information about an Elasticsearch node's JVM settings.
// From _nodes/stats/jvm: https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-nodes-stats.html
type NodeJVM struct {
	HeapUsedBytes         int `json:"heap_used_in_bytes"`
	HeapUsedPercentage    int `json:"heap_used_percent"`
	HeapMaxBytes          int `json:"heap_max_in_bytes"`
	NonHeapUsedBytes      int `json:"non_heap_used_in_bytes"`
	NonHeapCommittedBytes int `json:"non_heap_committed_in_bytes"`
}

// DiskAllocation holds disk allocation information per node, based on _cat/allocation
// API: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cat-allocation.html
type DiskAllocation struct {
	Name        string `json:"name"`
	IP          string `json:"ip"`
	Node        string `json:"node"`
	Shards      string `json:"shards"`
	DiskIndices string `json:"disk.indices"`
	DiskUsed    string `json:"disk.used"`
	DiskAvail   string `json:"disk.avail"`
	DiskTotal   string `json:"disk.total"`
	DiskPercent string `json:"disk.percent"`
}

// Holds information about an Elasticsearch index, based on the _cat/indices
// API: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cat-indices.html
type Index struct {
	Health        string `json:"health"`
	Status        string `json:"status"`
	Name          string `json:"index"`
	PrimaryShards int    `json:"pri,string"`
	ReplicaCount  int    `json:"rep,string"`
	IndexSize     string `json:"store.size"`
	DocumentCount int    `json:"docs.count,string"`
}

// Holds information about an index shard, based on the _cat/shards
// API: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cat-shards.html
type Shard struct {
	Index string `json:"index"`
	Shard string `json:"shard"`
	Type  string `json:"prirep"`
	State string `json:"state"`
	Docs  string `json:"docs"`
	Store string `json:"store"`
	IP    string `json:"ip"`
	Node  string `json:"node"`
}

// Holds information about overlapping shards for a given set of cluster nodes
type ShardOverlap struct {
	Index         string
	Shard         string
	PrimaryFound  bool
	ReplicasFound int
	ReplicasTotal int
}

// Holds information about shard recovery based on the _cat/recovery
// API: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cat-recovery.html
type ShardRecovery struct {
	Index                string `json:"index"`
	Shard                string `json:"shard"`
	Time                 string `json:"time"`
	Type                 string `json:"type"`
	Stage                string `json:"stage"`
	SourceHost           string `json:"source_host"`
	SourceNode           string `json:"source_node"`
	TargetHost           string `json:"target_host"`
	TargetNode           string `json:"target_node"`
	Repository           string `json:"repository"`
	Snapshot             string `json:"snapshot"`
	Files                int    `json:"files,string"`
	FilesRecovered       int    `json:"files_recovered,string"`
	FilesPercent         string `json:"files_percent"`
	FilesTotal           int    `json:"files_total,string"`
	Bytes                int    `json:"bytes,string"`
	BytesRecovered       int    `json:"bytes_recovered,string"`
	BytesPercent         string `json:"bytes_percent"`
	BytesTotal           int    `json:"bytes_total,string"`
	TranslogOps          int    `json:"translog_ops,string"`
	TranslogOpsRecovered int    `json:"translog_ops_recovered,string"`
	TranslogOpsPercent   string `json:"translog_ops_percent"`
}

// Holds information about an Elasticsearch alias, based on the _cat/aliases
// API: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cat-alias.html
type Alias struct {
	Name          string `json:"alias"`
	IndexName     string `json:"index"`
	Filter        string `json:"filter"`
	RoutingIndex  string `json:"routing.index"`
	RoutingSearch string `json:"routing.search"`
}

// Represent the two possible aliases actions: add or remove
type AliasActionType string

const (
	AddAlias    AliasActionType = "add"
	RemoveAlias AliasActionType = "remove"
)

// Holds information needed to perform an alias modification, based on the aliases
// API: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/indices-aliases.html
type AliasAction struct {
	ActionType AliasActionType
	IndexName  string `json:"index"`
	AliasName  string `json:"alias"`
}

func (ac *AliasAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		&map[AliasActionType]struct {
			IndexName string `json:"index"`
			AliasName string `json:"alias"`
		}{
			ac.ActionType: {
				IndexName: ac.IndexName,
				AliasName: ac.AliasName,
			},
		},
	)
}

// Holds information about the health of an Elasticsearch cluster, based on the
// cluster health API: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cluster-health.html
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

// Holds information about the health of an Elasticsearch index, based on the index
// level of the cluster health API: https://www.elastic.co/guide/en/elasticsearch/reference/5.6/cluster-health.html
type IndexHealth struct {
	Name               string
	Status             string `json:"status"`
	ActiveShards       int    `json:"active_shards"`
	RelocatingShards   int    `json:"relocating_shards"`
	InitializingShards int    `json:"initializing_shards"`
	UnassignedShards   int    `json:"unassigned_shards"`
}

// Holds slices for persistent and transient cluster settings.
type ClusterSettings struct {
	PersistentSettings []Setting
	TransientSettings  []Setting
}

// A setting name and value with the setting name to be a "collapsed" version of
// the setting. A setting of:
//
//	{ "indices": { "recovery" : { "max_bytes_per_sec": "10mb" } } }
//
// would be represented by:
//
//	ClusterSetting{ Setting: "indices.recovery.max_bytes_per_sec", Value: "10mb" }
type Setting struct {
	Setting string
	Value   string
}

type snapshotWrapper struct {
	Snapshots []Snapshot `json:"snapshots"`
}

type acknowledgedResponse struct {
	Acknowledged bool `json:"acknowledged"`
}

// Holds information about an Elasticsearch snapshot, based on the snapshot
// API: https://www.elastic.co/guide/en/elasticsearch/reference/current/modules-snapshots.html
type Snapshot struct {
	State          string    `json:"state"`
	Name           string    `json:"snapshot"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
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

// Holds information about an Elasticsearch snapshot repository.
type Repository struct {
	Name     string
	Type     string
	Settings map[string]interface{}
}

// Internal struct for repository requests since Name is part of URL path
type repo struct {
	Type     string                 `json:"type"`
	Settings map[string]interface{} `json:"settings"`
}

// Holds information about the tokens that Elasticsearch analyzes
type Token struct {
	Text        string `json:"token"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
	Type        string `json:"type"`
	Position    int    `json:"position"`
}

type ReloadSecureSettingsResponse struct {
	Summary struct {
		Total      int `json:"total"`
		Failed     int `json:"failed"`
		Successful int `json:"successful"`
	} `json:"_nodes"`
	ClusterName string `json:"cluster_name"`
	Nodes       map[string]struct {
		Name            string `json:"name"`
		ReloadException *struct {
			Type   string `json:"type"`
			Reason string `json:"reason"`
		} `json:"reload_exception"`
	} `json:"nodes"`
}

// Initialize a new vulcanizer client to use.
// Deprecated: NewClient has been deprecated in favor of using struct initialization.
func NewClient(host string, port int) *Client {
	if port > 0 {
		return &Client{Host: host, Port: port}
	}
	return &Client{Host: host}
}

const clusterSettingsPath = "_cluster/settings"

func settingsToStructs(rawJSON string) ([]Setting, error) {
	flatSettings, err := flatten.FlattenString(rawJSON, "", flatten.DotStyle)
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

	clusterSettings := make([]Setting, 0, len(keys))
	for _, k := range keys {
		setting := Setting{
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

// Estimate time remaining for recovery
func (s ShardRecovery) TimeRemaining() (time.Duration, error) {
	elapsed, err := time.ParseDuration(s.Time)
	if err != nil {
		return time.Second, err
	}
	// Calculate the rate of change
	rate := float64(s.BytesRecovered) / elapsed.Seconds()
	bytesLeft := s.BytesTotal - s.BytesRecovered
	// Divide the remaining bytes to recover by the rate of change
	return time.Duration(float64(bytesLeft)/rate) * time.Second, nil
}

// Can we safely remove nodes without data loss?
func (s ShardOverlap) SafeToRemove() bool {
	return !(s.PrimaryFound && s.ReplicasFound >= s.ReplicasTotal)
}

// Check if we should consider shard as a primary shard
func (s ShardOverlap) isPrimaryShard(shard Shard) bool {
	return shard.Type == "p" && (shard.State == "STARTED" || shard.State == "RELOCATING")
}

// Check if we should consider shard as a replica shard
func (s ShardOverlap) isReplicaShard(shard Shard) bool {
	return shard.Type == "r" && (shard.State == "STARTED" || shard.State == "RELOCATING")
}

func (c *Client) getAgent(method, path string) *gorequest.SuperAgent {
	agent := gorequest.New().Set("Accept", "application/json").Set("Content-Type", "application/json")
	agent.Method = method

	var protocol string
	if c.Secure {
		protocol = "https"
	} else {
		protocol = "http"
	}

	if c.Path != "" {
		path = fmt.Sprintf("%s/%s", c.Path, path)
	}

	if c.Port > 0 {
		agent.Url = fmt.Sprintf("%s://%s:%v/%s", protocol, c.Host, c.Port, path)
	} else {
		agent.Url = fmt.Sprintf("%s://%s/%s", protocol, c.Host, path)
	}

	if c.Auth != nil {
		agent.SetBasicAuth(c.Auth.User, c.Auth.Password)
	}

	if c.TLSConfig != nil {
		agent.TLSClientConfig(c.TLSConfig)
	}

	if c.Timeout != 0 {
		agent.Timeout(c.Timeout)
	} else {
		agent.Timeout(1 * time.Minute)
	}

	return agent
}

func (c *Client) buildGetRequest(path string) *gorequest.SuperAgent {
	return c.getAgent(gorequest.GET, path)
}

func (c *Client) buildPutRequest(path string) *gorequest.SuperAgent {
	return c.getAgent(gorequest.PUT, path)
}

func (c *Client) buildDeleteRequest(path string) *gorequest.SuperAgent {
	return c.getAgent(gorequest.DELETE, path)
}

func (c *Client) buildPostRequest(path string) *gorequest.SuperAgent {
	return c.getAgent(gorequest.POST, path)
}

// Get current cluster settings for shard allocation exclusion rules.
func (c *Client) GetClusterExcludeSettings() (ExcludeSettings, error) {
	body, err := handleErrWithBytes(c.buildGetRequest(clusterSettingsPath))

	if err != nil {
		return ExcludeSettings{}, err
	}

	excludedArray := gjson.GetManyBytes(body, "transient.cluster.routing.allocation.exclude._ip", "transient.cluster.routing.allocation.exclude._name", "transient.cluster.routing.allocation.exclude._host")

	excludeSettings := excludeSettingsFromJSON(excludedArray)
	return excludeSettings, nil
}

// Set shard allocation exclusion rules such that the Elasticsearch node with
// the name `serverToDrain` is excluded. This should cause Elasticsearch to
// migrate shards away from that node.
//
// Use case: You need to restart an Elasticsearch node. In order to do so safely,
// you should migrate data away from it. Calling `DrainServer` with the node name
// will move data off of the specified node.
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

// Set shard allocation exclusion rules such that the Elasticsearch node with
// the name `serverToFill` is no longer being excluded. This should cause
// Elasticsearch to migrate shards to that node.
//
// Use case: You have completed maintenance on an Elasticsearch node and it's
// ready to hold data again. Calling `FillOneServer` with the node name will
// remove that node name from the shard exclusion rules and allow data to be
// relocated onto the node.
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

// Removes all shard allocation exclusion rules.
//
// Use case: You had been performing maintenance on a number of Elasticsearch
// nodes. They are all ready to receive data again. Calling `FillAll` will
// remove all the allocation exclusion rules on the cluster, allowing
// Elasticsearch to freely allocate shards on the previously excluded nodes.
func (c *Client) FillAll() (ExcludeSettings, error) {

	agent := c.buildPutRequest(clusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(`{"transient" : { "cluster.routing.allocation.exclude" : { "_name" :  "", "_ip" : "", "_host" : ""}}}`)

	body, err := handleErrWithBytes(agent)

	if err != nil {
		return ExcludeSettings{}, err
	}

	excludedArray := gjson.GetManyBytes(body, "transient.cluster.routing.allocation.exclude._ip", "transient.cluster.routing.allocation.exclude._name", "transient.cluster.routing.allocation.exclude._host")

	return excludeSettingsFromJSON(excludedArray), nil
}

// Get all the nodes in the cluster.
//
// Use case: You want to see what nodes Elasticsearch considers part of the cluster.
func (c *Client) GetNodes() ([]Node, error) {
	var nodes []Node

	agent := c.buildGetRequest("_cat/nodes?h=master,role,name,ip,id,jdk,version")
	err := handleErrWithStruct(agent, &nodes)

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// Get all the nodes and their allocation/disk usage in the cluster.
//
// Use case: You want to see how much disk is being used by the nodes in the cluster.
func (c *Client) GetNodeAllocations() ([]Node, error) {
	var nodes []Node
	var nodeErr error
	// Get the node info first
	nodes, nodeErr = c.GetNodes()

	if nodeErr != nil {
		return nil, nodeErr
	}

	// Now get the allocation info and decorate the existing nodes
	var allocations []DiskAllocation
	agent := c.buildGetRequest("_cat/allocation?v&h=shards,disk.indices,disk.used,disk.avail,disk.total,disk.percent,ip,name,node")
	err := handleErrWithStruct(agent, &allocations)

	if err != nil {
		return nil, err
	}

	nodes = enrichNodesWithAllocations(nodes, allocations)

	return nodes, nil
}

// Get all the nodes' JVM Heap statistics.
//
// Use case: You want to see how much heap each node is using and their max heap size.

func (c *Client) GetNodeJVMStats() ([]NodeStats, error) {

	// NodeStats is not the top level of "nodes" as the individual node name
	// is the key of each node. Eg. "nodes.H1iBOLqqToyT8CHF9C0W0w.name = es-node-1".
	// This is tricky to unmarshal to struct, so let gjson deal with it.

	var nodesStats []NodeStats
	// Get node stats/jvm
	agent := c.buildGetRequest("_nodes/stats/jvm")
	bytes, err := handleErrWithBytes(agent)
	if err != nil {
		return nil, err
	}

	nodesRes := gjson.GetBytes(bytes, "nodes")

	var itErr error
	// Iterate over each node.
	nodesRes.ForEach(func(key, value gjson.Result) bool {
		var jvmStats NodeJVM
		memString := value.Get("jvm.mem").String()
		err = json.Unmarshal([]byte(memString), &jvmStats)
		if err != nil {
			itErr = fmt.Errorf("failed to unmarshal mem stats: %w", err)
			return false
		}

		// Let's grab the nodes role(s). Different format depending on version
		var role string

		if value.Get("attributes.master").Exists() {
			// Probably Elasticsearch 1.7
			masterRole := value.Get("attributes.master").String()
			dataRole := value.Get("attributes.data").String()

			if dataRole != "false" {
				role = "d"
			}
			if masterRole == "true" {
				role = "M" + role
			}
		}

		if value.Get("roles").Exists() {
			// Probably Elasticsearch 5+

			// Elasticsearch 5,6 and 7 has quite a few roles, let's collect them
			roleRes := value.Get("roles").Array()
			for _, res := range roleRes {
				sr := res.String()
				if sr == "master" {
					role = "M" + role
					continue
				}
				role += sr[:1]
			}
		}
		nodeStat := NodeStats{
			Name:     value.Get("name").String(),
			Role:     role,
			JVMStats: jvmStats,
		}

		nodesStats = append(nodesStats, nodeStat)

		return true
	})

	if itErr != nil {
		return nil, itErr
	}

	return nodesStats, nil

}

// Get all the indices in the cluster.
//
// Use case: You want to see some basic info on all the indices of the cluster.
func (c *Client) GetAllIndices() ([]Index, error) {
	var indices []Index
	err := handleErrWithStruct(c.buildGetRequest("_cat/indices?h=health,status,index,pri,rep,store.size,docs.count"), &indices)

	if err != nil {
		return nil, err
	}

	return indices, nil
}

// Get a subset of indices
func (c *Client) GetIndices(index string) ([]Index, error) {
	var indices []Index
	err := handleErrWithStruct(c.buildGetRequest(fmt.Sprintf("_cat/indices/%s?h=health,status,index,pri,rep,store.size,docs.count", index)), &indices)

	if err != nil {
		return nil, err
	}

	return indices, nil
}

// Get a subset of indices including hidden ones
func (c *Client) GetHiddenIndices(index string) ([]Index, error) {
	var indices []Index
	err := handleErrWithStruct(c.buildGetRequest(fmt.Sprintf("_cat/indices/%s?h=health,status,index,pri,rep,store.size,docs.count&expand_wildcards=open,closed,hidden", index)), &indices)

	if err != nil {
		return nil, err
	}

	return indices, nil
}

// Get all the aliases in the cluster.
//
// Use case: You want to see some basic info on all the aliases of the cluster
func (c *Client) GetAllAliases() ([]Alias, error) {
	var aliases []Alias

	err := handleErrWithStruct(c.buildGetRequest("_cat/aliases?h=alias,index,filter,routing.index,routing.search"), &aliases)

	if err != nil {
		return nil, err
	}

	return aliases, nil
}

// Get a subset the aliases in the cluster.
//
// Use case: You want to see some basic info on a subset of the aliases of the cluster
func (c *Client) GetAliases(alias string) ([]Alias, error) {
	var aliases []Alias

	path := fmt.Sprintf("_cat/aliases/%s?h=alias,index,filter,routing.index,routing.search", alias)
	err := handleErrWithStruct(c.buildGetRequest(path), &aliases)

	if err != nil {
		return nil, err
	}

	return aliases, nil
}

// Interact with aliases in the cluster.
//
// Use case: You want to add, delete or update an index alias
func (c *Client) ModifyAliases(actions []AliasAction) error {
	request := map[string][]AliasAction{"actions": actions}

	agent := c.buildPostRequest("_aliases").
		Set("Content-Type", "application/json").
		Send(request)

	var response struct {
		Acknowledged bool `json:"acknowledged"`
	}
	err := handleErrWithStruct(agent, &response)

	if err != nil {
		return err
	}

	return nil
}

// Delete an index in the cluster.
//
// Use case: You want to remove an index and all of its data.
func (c *Client) DeleteIndex(indexName string) error {
	return c.DeleteIndexWithQueryParameters(indexName, nil)
}

// Delete an index in the cluster with query parameters.
//
// Use case: You want to remove an index and all of its data. You also want to
// specify query parameters such as timeout.
func (c *Client) DeleteIndexWithQueryParameters(indexName string, queryParamMap map[string][]string) error {
	queryParams := make([]string, 0, len(queryParamMap))
	for key, value := range queryParamMap {
		queryParams = append(queryParams, fmt.Sprintf("%s=%s", key,
			strings.Join(value, ",")))
	}
	queryString := strings.Join(queryParams, "&")

	agent := c.buildDeleteRequest(fmt.Sprintf("%s?%s", indexName, queryString))
	var response acknowledgedResponse

	err := handleErrWithStruct(agent, &response)

	if err != nil {
		return err
	}

	if !response.Acknowledged {
		return fmt.Errorf(`Request to delete index "%s" was not acknowledged. %+v`, indexName, response)
	}

	return nil
}

// Open an index on the cluster
//
// Use case: You want to open a closed index
func (c *Client) OpenIndex(indexName string) error {
	// var response acknowledgedResponse

	var response struct {
		Acknowledged bool `json:"acknowledged"`
	}
	err := handleErrWithStruct(c.buildPostRequest(fmt.Sprintf("%s/_open", indexName)), &response)

	if err != nil {
		return err
	}

	if !response.Acknowledged {
		return fmt.Errorf(`Request to open index "%s" was not acknowledged. %+v`, indexName, response)
	}

	return nil
}

// Close an index on the cluster
//
// Use case: You want to close an opened index
func (c *Client) CloseIndex(indexName string) error {
	// var response acknowledgedResponse

	var response struct {
		Acknowledged bool `json:"acknowledged"`
	}
	err := handleErrWithStruct(c.buildPostRequest(fmt.Sprintf("%s/_close", indexName)), &response)

	if err != nil {
		return err
	}

	if !response.Acknowledged {
		return fmt.Errorf(`Request to close index "%s" was not acknowledged. %+v`, indexName, response)
	}

	return nil
}

// Get the health of the cluster.
//
// Use case: You want to see information needed to determine if the Elasticsearch cluster is healthy (green) or not (yellow/red).
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

// Get all the persistent and transient cluster settings.
//
// Use case: You want to see the current settings in the cluster.
func (c *Client) GetClusterSettings() (ClusterSettings, error) {
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

// Enables or disables allocation for the cluster.
//
// Use case: You are performing an operation the cluster where nodes may be dropping in and out. Elasticsearch will typically try to rebalance immediately but you want the cluster to hold off rebalancing until you complete your task. Calling `SetAllocation("disable")` will disable allocation so Elasticsearch won't move/relocate any shards. Once you complete your task, calling `SetAllocation("enable")` will allow Elasticsearch to relocate shards again.
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

// Set a new value for a cluster setting. Returns existing value and new value as well as error, in that order
// If the setting is not set in Elasticsearch (it's falling back to default configuration) SetClusterSetting's existingValue will be nil.
// If the value provided is nil, SetClusterSetting will remove the setting so that Elasticsearch falls back on default configuration for that setting.
//
// Use case: You've doubled the number of nodes in your cluster and you want to increase the number of shards the cluster can relocate at one time. Calling `SetClusterSetting("cluster.routing.allocation.cluster_concurrent_rebalance", "100")` will update that value with the cluster. Once data relocation is complete you can decrease the setting by calling `SetClusterSetting("cluster.routing.allocation.cluster_concurrent_rebalance", "20")`.
func (c *Client) SetClusterSetting(setting string, value *string) (*string, *string, error) {
	var existingValue *string
	var newValue *string
	settingsBody, err := handleErrWithBytes(c.buildGetRequest(clusterSettingsPath))

	if err != nil {
		return existingValue, newValue, err
	}

	existingResults := gjson.GetManyBytes(settingsBody, fmt.Sprintf("transient.%s", setting), fmt.Sprintf("persistent.%s", setting))

	var newSettingBody string

	if value == nil {
		newSettingBody = fmt.Sprintf(`{"transient" : { "%s" : null}}`, setting)
	} else {
		newSettingBody = fmt.Sprintf(`{"transient" : { "%s" : "%s"}}`, setting, *value)
	}

	agent := c.buildPutRequest(clusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(newSettingBody)

	body, err := handleErrWithBytes(agent)

	if err != nil {
		return existingValue, newValue, err
	}

	newResults := gjson.GetBytes(body, fmt.Sprintf("transient.%s", setting)).String()
	if newResults != "" {
		newValue = &newResults
	}

	if existingResults[0].String() == "" {
		if existingResults[1].String() != "" {
			value := existingResults[1].String()
			existingValue = &value
		}
	} else {
		value := existingResults[0].String()
		existingValue = &value
	}

	return existingValue, newValue, nil
}

// List the snapshots of the given repository.
//
// Use case: You want to see information on snapshots in a repository.
func (c *Client) GetSnapshots(repository string) ([]Snapshot, error) {

	var snapshotWrapper snapshotWrapper

	err := handleErrWithStruct(c.buildGetRequest(fmt.Sprintf("_snapshot/%s/_all", repository)), &snapshotWrapper)

	if err != nil {
		return nil, err
	}

	return snapshotWrapper.Snapshots, nil
}

// Get detailed information about a particular snapshot.
//
// Use case: You had a snapshot fail and you want to see the reason why and what shards/nodes the error occurred on.
func (c *Client) GetSnapshotStatus(repository string, snapshot string) (Snapshot, error) {

	var snapshotWrapper snapshotWrapper

	err := handleErrWithStruct(c.buildGetRequest(fmt.Sprintf("_snapshot/%s/%s", repository, snapshot)), &snapshotWrapper)

	if err != nil {
		return Snapshot{}, err
	}

	return snapshotWrapper.Snapshots[0], nil
}

// Delete a snapshot
//
// Use case: You want to delete older snapshots so that they don't take up extra space.
func (c *Client) DeleteSnapshot(repository string, snapshot string) error {
	var response acknowledgedResponse

	err := handleErrWithStruct(c.buildDeleteRequest(fmt.Sprintf("_snapshot/%s/%s", repository, snapshot)).Timeout(10*time.Minute), &response)

	if err != nil {
		return err
	}

	if !response.Acknowledged {
		return fmt.Errorf(`Request to delete snapshot "%s" on repository "%s" was not acknowledged. %+v`, snapshot, repository, response)
	}

	return nil
}

// Verify a snapshot repository
//
// Use case: Have Elasticsearch verify a repository to make sure that all nodes can access the snapshot location correctly.
func (c *Client) VerifyRepository(repository string) (bool, error) {

	_, err := handleErrWithBytes(c.buildPostRequest(fmt.Sprintf("_snapshot/%s/_verify", repository)))

	if err != nil {
		return false, err
	}

	return true, nil
}

var (
	ErrRepositoryNameRequired = errors.New("Repository Name is required")
	ErrRepositoryTypeRequired = errors.New("Repository Type is required")
)

// Register a snapshot repository
//
// Use case: Register a snapshot repository in Elasticsearch
func (c *Client) RegisterRepository(repository Repository) error {

	if repository.Name == "" {
		return ErrRepositoryNameRequired
	}

	if repository.Type == "" {
		return ErrRepositoryTypeRequired
	}

	repo := repo{Type: repository.Type, Settings: repository.Settings}

	agent := c.buildPutRequest(fmt.Sprintf("_snapshot/%s", repository.Name)).
		Set("Content-Type", "application/json").
		Send(repo)

	_, err := handleErrWithBytes(agent)

	if err != nil {
		return err
	}

	return nil
}

// Remove a snapshot repository
//
// Use case: Remove a snapshot repository in Elasticsearch
func (c *Client) RemoveRepository(name string) error {

	if name == "" {
		return ErrRepositoryNameRequired
	}

	_, err := handleErrWithBytes(c.buildDeleteRequest(fmt.Sprintf("_snapshot/%s", name)))

	if err != nil {
		return err
	}

	return nil
}

// List snapshot repositories on the cluster
//
// Use case: You want to see all of the configured backup repositories on the given cluster, what types they are and if they are verified.
func (c *Client) GetRepositories() ([]Repository, error) {
	var repos map[string]repo

	err := handleErrWithStruct(c.buildGetRequest("_snapshot/_all"), &repos)
	if err != nil {
		return nil, err
	}

	repositories := make([]Repository, 0, len(repos))
	for name, r := range repos {
		// Sanitize AWS secrets if they exist in the settings
		delete(r.Settings, "access_key")
		delete(r.Settings, "secret_key")
		repositories = append(repositories, Repository{
			Name:     name,
			Type:     r.Type,
			Settings: r.Settings,
		})
	}

	return repositories, nil
}

// Take a snapshot of specific indices on the cluster to the given repository
//
// Use case: You want to backup certain indices on the cluster to the given repository.
func (c *Client) SnapshotIndices(repository string, snapshot string, indices []string) error {
	if repository == "" {
		return errors.New("Empty string for repository is not allowed")
	}

	if snapshot == "" {
		return errors.New("Empty string for snapshot is not allowed")
	}

	if len(indices) == 0 {
		return errors.New("No indices provided to snapshot")
	}

	agent := c.buildPutRequest(fmt.Sprintf("_snapshot/%s/%s", repository, snapshot)).
		Set("Content-Type", "application/json").
		Send(fmt.Sprintf(`{"indices" : "%s"}`, strings.Join(indices, ",")))

	_, err := handleErrWithBytes(agent)

	return err
}

// Take a snapshot of all indices on the cluster to the given repository
//
// Use case: You want to backup all of the indices on the cluster to the given repository.
func (c *Client) SnapshotAllIndices(repository string, snapshot string) error {
	if repository == "" {
		return errors.New("Empty string for repository is not allowed")
	}

	if snapshot == "" {
		return errors.New("Empty string for snapshot is not allowed")
	}

	agent := c.buildPutRequest(fmt.Sprintf("_snapshot/%s/%s", repository, snapshot))
	_, err := handleErrWithBytes(agent)

	return err
}

func (c *Client) SnapshotAllIndicesWithBodyParams(repository string, snapshot string, bodyParams map[string]interface{}) error {
	if repository == "" {
		return errors.New("empty string for repository is not allowed")
	}

	if snapshot == "" {
		return errors.New("empty string for snapshot is not allowed")
	}

	if bodyParams == nil {
		return errors.New("no body params provided, please use SnapshotAllIndices Function instead")
	}

	parsedJson, parsingErr := json.Marshal(bodyParams)

	if parsingErr != nil {
		return parsingErr
	}

	agent := c.buildPutRequest(fmt.Sprintf("_snapshot/%s/%s", repository, snapshot)).
		Set("Content-Type", "application/json").
		Send(string(parsedJson))

	_, err := handleErrWithBytes(agent)

	return err
}

// Restore an index or indices on the cluster
//
// Use case: You want to restore a particular index or indices onto your cluster with a new name.
func (c *Client) RestoreSnapshotIndices(repository string, snapshot string, indices []string, restoredIndexPrefix string, indexSettings map[string]interface{}) error {
	if repository == "" {
		return errors.New("Empty string for repository is not allowed")
	}

	if snapshot == "" {
		return errors.New("Empty string for snapshot is not allowed")
	}

	request := struct {
		Indices           string                 `json:"indices"`
		RenamePattern     string                 `json:"rename_pattern"`
		RenameReplacement string                 `json:"rename_replacement"`
		IndexSettings     map[string]interface{} `json:"index_settings,omitempty"`
	}{
		Indices:           strings.Join(indices, ","),
		RenamePattern:     "(.+)",
		RenameReplacement: fmt.Sprintf("%s$1", restoredIndexPrefix),
		IndexSettings:     indexSettings,
	}

	agent := c.buildPostRequest(fmt.Sprintf("_snapshot/%s/%s/_restore", repository, snapshot)).
		Set("Content-Type", "application/json").
		Send(request)

	_, err := handleErrWithBytes(agent)

	return err
}

// Call the analyze API with sample text and an analyzer. https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-analyze.html
//
// Use case: You want to see how Elasticsearch will break up sample text given a specific analyzer.
func (c *Client) AnalyzeText(analyzer, text string) ([]Token, error) {
	request := struct {
		Analyzer string `json:"analyzer"`
		Text     string `json:"text"`
	}{
		analyzer,
		text,
	}

	agent := c.buildPostRequest("_analyze").
		Set("Content-Type", "application/json").
		Send(request)

	var tokenWrapper struct {
		Tokens []Token `json:"tokens"`
	}

	err := handleErrWithStruct(agent, &tokenWrapper)
	if err != nil {
		return nil, err
	}

	return tokenWrapper.Tokens, nil
}

// Call the analyze API with sample text on an index and a specific field . https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-analyze.html
//
// Use case: You have a particular field that might have custom analyzers and you want to see how this field will tokenize some particular text.
func (c *Client) AnalyzeTextWithField(index, field, text string) ([]Token, error) {
	request := struct {
		Field string `json:"field"`
		Text  string `json:"text"`
	}{
		field,
		text,
	}

	agent := c.buildPostRequest(fmt.Sprintf("%s/_analyze", index)).
		Set("Content-Type", "application/json").
		Send(request)

	var tokenWrapper struct {
		Tokens []Token `json:"tokens"`
	}

	err := handleErrWithStruct(agent, &tokenWrapper)
	if err != nil {
		return nil, err
	}

	return tokenWrapper.Tokens, nil
}

// Get the settings of an index in a pretty-printed format.
//
// Use case: You can view the custom settings that are set on a particular index.
func (c *Client) GetPrettyIndexSettings(index string) (string, error) {
	body, err := handleErrWithBytes(c.buildGetRequest(fmt.Sprintf("%s/_settings", index)))

	if err != nil {
		return "", err
	}

	rawSettings := gjson.GetBytes(body, fmt.Sprintf("%s.settings.index", escapeIndexName(index))).Raw

	var prettyPrinted bytes.Buffer
	err = json.Indent(&prettyPrinted, []byte(rawSettings), "", "  ")
	if err != nil {
		return "", err
	}

	return prettyPrinted.String(), nil
}

// Get the settings of an index in a machine-oriented format.
//
// Use case: You can view the custom settings that are set on a particular index.
func (c *Client) GetIndexSettings(index string) ([]Setting, error) {
	body, err := handleErrWithBytes(c.buildGetRequest(fmt.Sprintf("%s/_settings", index)))

	if err != nil {
		return nil, err
	}

	rawSettings := gjson.GetBytes(body, fmt.Sprintf("%s.settings.index", escapeIndexName(index))).Raw

	settings, err := settingsToStructs(rawSettings)

	return settings, err
}

// Set a setting on an index.
//
// Use case: Set or update an index setting for a particular index.
func (c *Client) SetIndexSetting(index, setting, value string) (string, string, error) {
	settingsPath := fmt.Sprintf("%s/_settings", index)
	body, err := handleErrWithBytes(c.buildGetRequest(settingsPath))
	if err != nil {
		return "", "", err
	}

	currentValue := gjson.GetBytes(body, fmt.Sprintf("%s.settings.index.%s", escapeIndexName(index), setting)).Str

	agent := c.buildPutRequest(settingsPath).Set("Content-Type", "application/json").
		Send(fmt.Sprintf(`{"index" : { "%s" : "%s"}}`, setting, value))

	_, err = handleErrWithBytes(agent)
	if err != nil {
		return "", "", err
	}

	return currentValue, value, nil
}

// Get the mappings of an index in a pretty-printed format.
//
// Use case: You can view the custom mappings that are set on a particular index.
func (c *Client) GetPrettyIndexMappings(index string) (string, error) {
	body, err := handleErrWithBytes(c.buildGetRequest(fmt.Sprintf("%s/_mappings", index)))

	if err != nil {
		return "", err
	}

	var prettyPrinted bytes.Buffer
	err = json.Indent(&prettyPrinted, body, "", "  ")
	if err != nil {
		return "", err
	}

	return prettyPrinted.String(), nil
}

// Get the segments of an index in a pretty-printed format
//
// Use case: you can view the segments of a particular index
func (c *Client) GetPrettyIndexSegments(index string) (string, error) {
	body, err := handleErrWithBytes(c.buildGetRequest(fmt.Sprintf("%s/_segments", index)))

	if err != nil {
		return "", err
	}

	var prettyPrinted bytes.Buffer
	err = json.Indent(&prettyPrinted, body, "", "  ")
	if err != nil {
		return "", err
	}

	return prettyPrinted.String(), nil
}

// Get shard data for all or a subset of nodes
//
// Use case: You can view shard information on all nodes or a subset.
func (c *Client) GetShards(nodes []string) ([]Shard, error) {
	var allShards []Shard
	req := c.buildGetRequest("_cat/shards")
	err := handleErrWithStruct(req, &allShards)

	if err != nil {
		return nil, err
	}

	// No nodes passed, so return all shards
	if len(nodes) == 0 {
		return allShards, nil
	}

	var filteredShards []Shard
	nodeRegexps := make([]*regexp.Regexp, 0, len(nodes))

	for _, node := range nodes {
		nodeRegexp, err := regexp.Compile(node)
		if err != nil {
			return nil, err
		}
		nodeRegexps = append(nodeRegexps, nodeRegexp)
	}

	for _, shard := range allShards {
		for _, nodeRegexp := range nodeRegexps {
			// Support regexp matching of node name
			matches := nodeRegexp.MatchString(shard.Node)

			if matches {
				filteredShards = append(filteredShards, shard)
			}
		}
	}

	return filteredShards, nil
}

// Get details regarding shard distribution across a given set of cluster nodes.
//
// Use case: You can leverage this information to determine if it's safe to remove cluster nodes without losing data.
func (c *Client) GetShardOverlap(nodes []string) (map[string]ShardOverlap, error) {
	shards, err := c.GetShards(nodes)
	overlap := map[string]ShardOverlap{}

	if err != nil {
		fmt.Printf("Error getting shards: %s", err)
		return nil, err
	}

	_indices, err := c.GetAllIndices()

	if err != nil {
		fmt.Printf("Error getting indices: %s", err)
		return nil, err
	}

	// Map-ify this slice of indices for easy lookup
	indices := map[string]Index{}
	for _, index := range _indices {
		indices[index.Name] = index
	}

	for _, shard := range shards {
		// Map key is the concatenation of the index name + "_" + shard number
		name := fmt.Sprintf("%s_%s", shard.Index, shard.Shard)

		val, ok := overlap[name]
		if ok {
			// We've already seen this index/shard combo
			if val.isPrimaryShard(shard) {
				val.PrimaryFound = true
			} else if val.isReplicaShard(shard) {
				val.ReplicasFound++
			}
			overlap[name] = val
		} else {
			// First occurrence of index/shard combo
			val := ShardOverlap{
				Index:         shard.Index,
				Shard:         shard.Shard,
				PrimaryFound:  val.isPrimaryShard(shard),
				ReplicasFound: 0,
				ReplicasTotal: indices[shard.Index].ReplicaCount,
			}
			// Ensure we're only counting fully started replica shards
			if val.isReplicaShard(shard) {
				val.ReplicasFound = 1
			}
			overlap[name] = val
		}
	}
	return overlap, nil
}

// Get details regarding shard recovery operations across a set of cluster nodes.
//
// Use case: You can view the shard recovery progress of the cluster.
func (c *Client) GetShardRecovery(nodes []string, onlyActive bool) ([]ShardRecovery, error) {
	var allRecoveries []ShardRecovery
	uri := "_cat/recovery"

	if onlyActive {
		uri = fmt.Sprintf("%s?active_only=true", uri)
	}

	req := c.buildGetRequest(uri)
	err := handleErrWithStruct(req, &allRecoveries)

	if err != nil {
		return nil, err
	}

	// No nodes passed, so return all shards
	if len(nodes) == 0 {
		return allRecoveries, nil
	}

	var filteredRecoveries []ShardRecovery
	nodeRegexps := make([]*regexp.Regexp, 0, len(nodes))

	for _, node := range nodes {
		nodeRegexp, err := regexp.Compile(node)
		if err != nil {
			return nil, err
		}
		nodeRegexps = append(nodeRegexps, nodeRegexp)
	}

	for _, shard := range allRecoveries {
		for _, nodeRegexp := range nodeRegexps {
			// Support regexp matching of node name
			matchesSource := nodeRegexp.MatchString(shard.SourceNode)
			matchesTarget := nodeRegexp.MatchString(shard.TargetNode)

			// Return if either source node or target node matches
			if matchesSource || matchesTarget {
				filteredRecoveries = append(filteredRecoveries, shard)
			}
		}
	}

	return filteredRecoveries, nil
}

// Get details regarding shard recovery operations across a set of cluster nodes sending the desired query parameters
//
// Use case: You can view the shard recovery progress of the cluster with the bytes=b parameter.
func (c *Client) GetShardRecoveryWithQueryParams(nodes []string, params map[string]string) ([]ShardRecovery, error) {
	var allRecoveries []ShardRecovery
	uri := "_cat/recovery"

	queryStrings := []string{}
	for param, val := range params {
		queryStrings = append(queryStrings, fmt.Sprintf("%s=%s", param, val))
	}

	uri = fmt.Sprintf("%s?%s", uri, strings.Join(queryStrings, "&"))

	req := c.buildGetRequest(uri)
	err := handleErrWithStruct(req, &allRecoveries)

	if err != nil {
		return nil, err
	}

	// No nodes passed, so return all shards
	if len(nodes) == 0 {
		return allRecoveries, nil
	}

	var filteredRecoveries []ShardRecovery
	nodeRegexps := make([]*regexp.Regexp, 0, len(nodes))

	for _, node := range nodes {
		nodeRegexp, err := regexp.Compile(node)
		if err != nil {
			return nil, err
		}
		nodeRegexps = append(nodeRegexps, nodeRegexp)
	}

	for _, shard := range allRecoveries {
		for _, nodeRegexp := range nodeRegexps {
			// Support regexp matching of node name
			matchesSource := nodeRegexp.MatchString(shard.SourceNode)
			matchesTarget := nodeRegexp.MatchString(shard.TargetNode)

			// Return if either source node or target node matches
			if matchesSource || matchesTarget {
				filteredRecoveries = append(filteredRecoveries, shard)
			}
		}
	}

	return filteredRecoveries, nil
}

// GetDuration gets the total duration of a snapshot
func (s *Snapshot) GetDuration() int {
	if s.DurationMillis > 0 {
		return s.DurationMillis
	}
	// This will avoid returning incorrect values like "-1554114611822"
	return int(time.Since(s.StartTime) / time.Millisecond)

}

// GetEndTime gets the time when a snapshot ended
func (s *Snapshot) GetEndTime() string {
	if s.DurationMillis > 0 {
		return s.EndTime.Format(time.RFC3339)
	}
	// This will avoid returning incorrect values like "1970-01-01T00:00:00.000Z"
	return ""
}

// Reload secure node settings
//
// Use case: Call the reload secure settings API https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-nodes-reload-secure-settings.html
func (c *Client) ReloadSecureSettings() (ReloadSecureSettingsResponse, error) {
	var response ReloadSecureSettingsResponse
	err := handleErrWithStruct(c.buildPostRequest("_nodes/reload_secure_settings"), &response)

	if err != nil {
		return ReloadSecureSettingsResponse{}, err
	}

	return response, nil
}

// Reload secure node settings with password
//
// Use case: Call the reload secure settings API with a supplied password https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-nodes-reload-secure-settings.html
func (c *Client) ReloadSecureSettingsWithPassword(password string) (ReloadSecureSettingsResponse, error) {

	if password == "" {
		return ReloadSecureSettingsResponse{}, errors.New("Keystore password is required")
	}

	requestBody := struct {
		Password string `json:"secure_settings_password"`
	}{
		Password: password,
	}

	agent := c.buildPostRequest("_nodes/reload_secure_settings").
		Set("Content-Type", "application/json").
		Send(requestBody)

	var response ReloadSecureSettingsResponse

	err := handleErrWithStruct(agent, &response)

	if err != nil {
		return ReloadSecureSettingsResponse{}, err
	}

	return response, nil
}

// GetHotThreads allows to get the current hot threads on each node on the cluster
func (c *Client) GetHotThreads() (string, error) {
	body, err := handleErrWithBytes(c.buildGetRequest("_nodes/hot_threads"))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// GetNodesHotThreads allows to get the current hot threads on given nodes on the cluster
func (c *Client) GetNodesHotThreads(nodesIDs []string) (string, error) {
	joinedNodesIDs := strings.Join(nodesIDs, ",")
	url := fmt.Sprintf("_nodes/%s/hot_threads", strings.ReplaceAll(joinedNodesIDs, " ", ""))
	body, err := handleErrWithBytes(c.buildGetRequest(url))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// ClusterAllocationExplainRequest represents request data that can be sent to
// `_cluster/allocation/explain` calls
type ClusterAllocationExplainRequest struct {
	// Specifies the node ID or the name of the node to only explain a shard that
	// is currently located on the specified node.
	CurrentNode string `json:"current_node,omitempty"`

	// Specifies the name of the index that you would like an explanation for.
	Index string `json:"index,omitempty"`

	// If true, returns explanation for the primary shard for the given shard ID.
	Primary bool `json:"primary,omitempty"`

	// Specifies the ID of the shard that you would like an explanation for.
	Shard *int `json:"shard,omitempty"`
}

// ClusterAllocationExplain provides an explanation for a shard’s current allocation.
// For more info, please check https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-allocation-explain.html
func (c *Client) ClusterAllocationExplain(req *ClusterAllocationExplainRequest, prettyOutput bool) (string, error) {
	var urlBuilder strings.Builder
	urlBuilder.WriteString("_cluster/allocation/explain")
	if prettyOutput {
		urlBuilder.WriteString("?pretty")
	}

	agent := c.buildGetRequest(urlBuilder.String())
	if req != nil {
		agent.Set("Content-Type", "application/json").Send(req)
	}

	body, err := handleErrWithBytes(agent)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// ClusterAllocationExplainWithQueryParams provides an explanation for a shard’s current allocation with optional query parameters.
// For more info, please check https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-allocation-explain.html
func (c *Client) ClusterAllocationExplainWithQueryParams(req *ClusterAllocationExplainRequest, params map[string]string) (string, error) {
	uri := "_cluster/allocation/explain"
	queryStrings := []string{}
	for param, val := range params {
		queryStrings = append(queryStrings, fmt.Sprintf("%s=%s", param, val))
	}

	uri = fmt.Sprintf("%s?%s", uri, strings.Join(queryStrings, "&"))

	agent := c.buildGetRequest(uri)
	if req != nil {
		agent.Set("Content-Type", "application/json").Send(req)
	}

	body, err := handleErrWithBytes(agent)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

type RerouteRequest struct {
	// The commands to perform (move, cancel, allocate, etc)
	Commands []RerouteCommand `json:"commands,omitempty"`
}

type RerouteCommand struct {
	AllocateStalePrimary AllocateStalePrimary `json:"allocate_stale_primary,omitempty"`
}

type AllocateStalePrimary struct {
	// The node ID or node name of the node to assign the shard to.
	Node string `json:"node,omitempty"`

	// The name of the index containing the shard to be assigned.
	Index string `json:"index,omitempty"`

	// The shard ID of the shard to be assigned.
	Shard *int `json:"shard,omitempty"`

	// If a node which has the good copy of the data rejoins the cluster later on, that data will be deleted or overwritten with the data of the stale copy that was forcefully allocated with this command.
	AcceptDataLoss bool `json:"accept_data_loss,omitempty"`
}

// RerouteWithRetryFailed retries allocation of shards that are blocked due to too many subsequent allocation failures.
func (c *Client) RerouteWithRetryFailed() error {
	var urlBuilder strings.Builder
	urlBuilder.WriteString("_cluster/reroute?retry_failed=true")

	agent := c.buildPostRequest(urlBuilder.String())

	_, err := handleErrWithBytes(agent)
	if err != nil {
		return err
	}

	return nil
}

// AllocateStalePrimary allows to manually allocate a stale primary shard to a specific node
func (c *Client) AllocateStalePrimaryShard(node, index string, shard int) error {
	var urlBuilder strings.Builder
	urlBuilder.WriteString("_cluster/reroute")

	agent := c.buildPostRequest(urlBuilder.String())

	req := RerouteRequest{
		Commands: []RerouteCommand{
			{
				AllocateStalePrimary: AllocateStalePrimary{
					Node:           node,
					Index:          index,
					Shard:          &shard,
					AcceptDataLoss: true,
				},
			},
		},
	}
	agent.Set("Content-Type", "application/json").Send(req)

	_, err := handleErrWithBytes(agent)
	if err != nil {
		return err
	}

	return nil
}

// RemoveIndexILMPolicy removes the ILM policy from the index
func (c *Client) RemoveIndexILMPolicy(index string) error {
	agent := c.buildPostRequest(fmt.Sprintf("%s/_ilm/remove", index))

	_, err := handleErrWithBytes(agent)
	if err != nil {
		return err
	}

	ilmHistoryIndices, err := c.GetHiddenIndices(fmt.Sprintf("%s*.ds-ilm-history-*", index))
	if err != nil {
		return err
	}

	for _, ilmHistoryIndex := range ilmHistoryIndices {
		err = c.DeleteIndex(ilmHistoryIndex.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

// LicenseCluster takes in the Elasticsearch license encoded as a string
func (c *Client) LicenseCluster(license string) error {
	// If the license is empty, return an error
	if license == "" {
		return errors.New("license is required")
	}

	// Build the request to apply the license to the cluster
	agent := c.buildPutRequest("_license").
		Set("Content-Type", "application/json").
		Send(license)

	// Execute the request
	_, err := handleErrWithBytes(agent)
	if err != nil {
		return err
	}

	return nil
}
