package vulcanizer

import (
	"bytes"
  "crypto/tls"
	"encoding/json"
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

type Auth struct {
	User     string
	Password string
}

//Hold connection information to a Elasticsearch cluster.
type Client struct {
	Host      string
	Port      int
	Secure    bool
	TLSConfig *tls.Config
	Timeout   time.Duration
	*Auth
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
	PersistentSettings []Setting
	TransientSettings  []Setting
}

//A setting name and value with the setting name to be a "collapsed" version of the setting. A setting of:
//  { "indices": { "recovery" : { "max_bytes_per_sec": "10mb" } } }
//would be represented by:
//  ClusterSetting{ Setting: "indices.recovery.max_bytes_per_sec", Value: "10mb" }
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

//Holds information about an Elasticsearch snapshot repository.
type Repository struct {
	Name     string
	Type     string
	Settings map[string]interface{}
}

//Holds information about the tokens that Elasticsearch analyzes
type Token struct {
	Text        string `json:"token"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
	Type        string `json:"type"`
	Position    int    `json:"position"`
}

//Initialize a new vulcanizer client to use.
func NewClient(host string, port int) *Client {
	return &Client{Host: host, Port: port}
}

const clusterSettingsPath = "_cluster/settings"

func settingsToStructs(rawJson string) ([]Setting, error) {
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

	var clusterSettings []Setting
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

func (c *Client) getAgent(method, path string) *gorequest.SuperAgent {
	agent := gorequest.New().Set("Accept", "application/json")
	agent.Method = method

	var protocol string
	if c.Secure {
		protocol = "https"
	} else {
		protocol = "http"
	}

	agent.Url = fmt.Sprintf("%s://%s:%v/%s", protocol, c.Host, c.Port, path)

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

//Delete an index in the cluster.
//
//Use case: You want to remove an index and all of its data.
func (c *Client) DeleteIndex(indexName string) error {
	var response acknowledgedResponse

	err := handleErrWithStruct(c.buildDeleteRequest(indexName), &response)

	if err != nil {
		return err
	}

	if !response.Acknowledged {
		return fmt.Errorf(`Request to delete index "%s" was not acknowledged. %+v`, indexName, response)
	}

	return nil
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
//Use case: You've doubled the number of nodes in your cluster and you want to increase the number of shards the cluster can relocate at one time. Calling `SetClusterSetting("cluster.routing.allocation.cluster_concurrent_rebalance", "100")` will update that value with the cluster. Once data relocation is complete you can decrease the setting by calling `SetClusterSetting("cluster.routing.allocation.cluster_concurrent_rebalance", "20")`.
func (c *Client) SetClusterSetting(setting string, value string) (string, string, error) {

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
		return fmt.Errorf(`Request to delete snapshot "%s" on repository "%s" was not acknowledged. %+v`, snapshot, repository, response)
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

type repo struct {
	Type     string                 `json:"type"`
	Settings map[string]interface{} `json:"settings"`
}

//List snapshot respositories on the cluster
//
//Use case: You want to see all of the configured backup repositories on the given cluster, what types they are and if they are verified.
func (c *Client) GetRepositories() ([]Repository, error) {

	var repos map[string]repo
	var repositories []Repository

	err := handleErrWithStruct(c.buildGetRequest("_snapshot/_all"), &repos)

	if err != nil {
		return nil, err
	}

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

//Take a snapshot of specific indices on the cluster to the given repository
//
//Use case: You want to backup certain indices on the cluster to the given repository.
func (c *Client) SnapshotIndices(repository string, snapshot string, indices []string) error {
	if repository == "" {
		return fmt.Errorf("Empty string for repository is not allowed.")
	}

	if snapshot == "" {
		return fmt.Errorf("Empty string for snapshot is not allowed.")
	}

	if len(indices) == 0 {
		return fmt.Errorf("No indices provided to snapshot.")
	}

	agent := c.buildPutRequest(fmt.Sprintf("_snapshot/%s/%s", repository, snapshot)).
		Set("Content-Type", "application/json").
		Send(fmt.Sprintf(`{"indices" : "%s"}`, strings.Join(indices, ",")))

	_, err := handleErrWithBytes(agent)

	return err
}

//Take a snapshot of all indices on the cluster to the given repository
//
//Use case: You want to backup all of the indices on the cluster to the given repository.
func (c *Client) SnapshotAllIndices(repository string, snapshot string) error {
	if repository == "" {
		return fmt.Errorf("Empty string for repository is not allowed.")
	}

	if snapshot == "" {
		return fmt.Errorf("Empty string for snapshot is not allowed.")
	}

	agent := c.buildPutRequest(fmt.Sprintf("_snapshot/%s/%s", repository, snapshot))
	_, err := handleErrWithBytes(agent)

	return err
}

//Restore an index or indices on the cluster
//
//Use case: You want to restore a particular index or indices onto your cluster with a new name.
func (c *Client) RestoreSnapshotIndices(repository string, snapshot string, indices []string, restoredIndexPrefix string, indexSettings map[string]interface{}) error {
	if repository == "" {
		return fmt.Errorf("Empty string for repository is not allowed.")
	}

	if snapshot == "" {
		return fmt.Errorf("Empty string for snapshot is not allowed.")
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

//Call the analyze API with sample text and an analyzer. https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-analyze.html
//
//Use case: You want to see how Elasticsearch will break up sample text given a specific analyzer.
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

//Call the analyze API with sample text on an index and a specific field . https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-analyze.html
//
//Use case: You have a particular field that might have custom analyzers and you want to see how this field will tokenize some particular text.
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

//Get the settings of an index in a pretty-printed format.
//
//Use case: You can view the custom settings that are set on a particular index.
func (c *Client) GetPrettyIndexSettings(index string) (string, error) {
	body, err := handleErrWithBytes(c.buildGetRequest(fmt.Sprintf("%s/_settings", index)))

	if err != nil {
		return "", err
	}

	rawSettings := gjson.GetBytes(body, fmt.Sprintf("%s.settings.index", index)).Raw

	var prettyPrinted bytes.Buffer
	err = json.Indent(&prettyPrinted, []byte(rawSettings), "", "  ")
	if err != nil {
		return "", err
	}

	return prettyPrinted.String(), nil
}

//Get the settings of an index in a machine-oriented format.
//
//Use case: You can view the custom settings that are set on a particular index.
func (c *Client) GetIndexSettings(index string) ([]Setting, error) {
	body, err := handleErrWithBytes(c.buildGetRequest(fmt.Sprintf("%s/_settings", index)))

	if err != nil {
		return nil, err
	}

	rawSettings := gjson.GetBytes(body, fmt.Sprintf("%s.settings.index", index)).Raw

	settings, err := settingsToStructs(rawSettings)

	return settings, err
}

//Set a setting on an index.
//
//Use case: Set or update an index setting for a particular index.
func (c *Client) SetIndexSetting(index, setting, value string) (string, string, error) {
	settingsPath := fmt.Sprintf("%s/_settings", index)
	body, err := handleErrWithBytes(c.buildGetRequest(settingsPath))
	if err != nil {
		return "", "", err
	}

	currentValue := gjson.GetBytes(body, fmt.Sprintf("%s.settings.index.%s", index, setting)).Str

	agent := c.buildPutRequest(settingsPath).Set("Content-Type", "application/json").
		Send(fmt.Sprintf(`{"index" : { "%s" : "%s"}}`, setting, value))

	_, err = handleErrWithBytes(agent)
	if err != nil {
		return "", "", err
	}

	return currentValue, value, nil
}
