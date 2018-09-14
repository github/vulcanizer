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

type snapshotWrapper struct {
	Snapshots []Snapshot `json:"snapshots"`
}

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

// Get current cluster settings for exclusion
func (c *Client) GetClusterExcludeSettings() (ExcludeSettings, error) {
	body, err := handleErrWithBytes(c.buildGetRequest(clusterSettingsPath))

	if err != nil {
		return ExcludeSettings{}, err
	}

	excludedArray := gjson.GetManyBytes(body, "transient.cluster.routing.allocation.exclude._ip", "transient.cluster.routing.allocation.exclude._name", "transient.cluster.routing.allocation.exclude._host")

	excludeSettings := excludeSettingsFromJson(excludedArray)
	return excludeSettings, nil
}

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

func (c *Client) GetNodes() ([]Node, error) {
	var nodes []Node

	agent := c.buildGetRequest("_cat/nodes?h=master,role,name,ip,id")
	err := handleErrWithStruct(agent, &nodes)

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (c *Client) GetIndices() ([]Index, error) {
	var indices []Index
	err := handleErrWithStruct(c.buildGetRequest("_cat/indices?h=health,status,index,pri,rep,store.size,docs.count"), &indices)

	if err != nil {
		return nil, err
	}

	return indices, nil
}

func (c *Client) GetHealth() ([]ClusterHealth, error) {
	var health []ClusterHealth
	err := handleErrWithStruct(c.buildGetRequest("_cat/health?h=cluster,status,relo,init,unassign,pending_tasks,active_shards_percent"), &health)

	if err != nil {
		return nil, err
	}

	for i := range health {
		health[i].Message = captionHealth(health[i].Status)
	}

	return health, nil
}

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

func (c *Client) GetSnapshots(repository string) ([]Snapshot, error) {

	var snapshotWrapper snapshotWrapper

	err := handleErrWithStruct(c.buildGetRequest(fmt.Sprintf("_snapshot/%s/_all", repository)), &snapshotWrapper)

	if err != nil {
		return nil, err
	}

	return snapshotWrapper.Snapshots, nil
}

func (c *Client) GetSnapshotStatus(repository string, snapshot string) (Snapshot, error) {

	var snapshotWrapper snapshotWrapper

	err := handleErrWithStruct(c.buildGetRequest(fmt.Sprintf("_snapshot/%s/%s", repository, snapshot)), &snapshotWrapper)

	if err != nil {
		return Snapshot{}, err
	}

	return snapshotWrapper.Snapshots[0], nil
}
