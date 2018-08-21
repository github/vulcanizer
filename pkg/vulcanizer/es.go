package vulcan

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

const ClusterSettingsPath = "_cluster/settings"

func buildGetRequest(server string, port int, path string) *gorequest.SuperAgent {
	return gorequest.New().Get(fmt.Sprintf("http://%s:%v/%s", server, port, path)).Set("Accept", "application/json")
}

func buildPutRequest(server string, port int, path string) *gorequest.SuperAgent {
	return gorequest.New().Put(fmt.Sprintf("http://%s:%v/%s", server, port, path))
}

// Get current cluster settings for exclusion
func GetClusterExcludeSettings(server string, port int) *ExcludeSettings {
	_, body, _ := buildGetRequest(server, port, ClusterSettingsPath).End()

	excludedArray := gjson.GetMany(body, "transient.cluster.routing.allocation.exclude._ip", "transient.cluster.routing.allocation.exclude._name", "transient.cluster.routing.allocation.exclude._host")

	excludeSettings := ExcludeSettingsFromJson(excludedArray)
	return excludeSettings
}

func DrainServer(server string, port int, serverToDrain string, namesExcluded string) (excludedServers string) {

	var drainList string = ""

	if namesExcluded != "None" {
		drainList = serverToDrain + "," + namesExcluded

	} else {
		drainList = serverToDrain
	}

	_, body, _ := buildPutRequest(server, port, ClusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(`{"transient" : { "cluster.routing.allocation.exclude._name" : "` + drainList + `"}}`).
		End()

	drainingServers := gjson.Get(body, "transient.cluster.routing.allocation.exclude._name")

	return drainingServers.String()
}

func FillOneServer(server string, port int, serverToFill string) (serverFilling, excludedServers string) {

	// Get the current list of strings
	excludeSettings := GetClusterExcludeSettings(server, port)

	serverToFill = strings.TrimSpace(serverToFill)

	newNamesDrained := []string{}
	for _, s := range excludeSettings.Names {
		if s != serverToFill {
			newNamesDrained = append(newNamesDrained, s)
		}
	}

	_, body, _ := buildPutRequest(server, port, ClusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(`{"transient" : { "cluster.routing.allocation.exclude._name" : "` + strings.Join(newNamesDrained, ",") + `"}}`).
		End()

	drainingServers := gjson.Get(body, "transient.cluster.routing.allocation.exclude._name")

	return serverToFill, drainingServers.String()
}

func FillAll(server string, port int) *ExcludeSettings {

	_, body, _ := buildPutRequest(server, port, ClusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(`{"transient" : { "cluster.routing.allocation.exclude" : { "_name" :  "", "_ip" : "", "_host" : ""}}}`).
		End()

	excludedArray := gjson.GetMany(body, "transient.cluster.routing.allocation.exclude._ip", "transient.cluster.routing.allocation.exclude._name", "transient.cluster.routing.allocation.exclude._host")

	return ExcludeSettingsFromJson(excludedArray)
}

func GetNodes(server string, port int) ([][]string, []string) {
	_, body, _ := buildGetRequest(server, port, "_cat/nodes?h=master,role,name,ip,id").End()

	results := [][]string{}
	headers := []string{"master", "role", "name", "ip", "id"}

	gjson.Parse(body).ForEach(func(key, value gjson.Result) bool {
		result := []string{
			value.Get("master").String(),
			value.Get("role").String(),
			value.Get("name").String(),
			value.Get("ip").String(),
			value.Get("id").String(),
		}

		results = append(results, result)
		return true // keep iterating
	})

	return results, headers
}

func GetIndices(server string, port int) ([][]string, []string) {
	_, body, _ := buildGetRequest(server, port, "_cat/indices?h=health,status,index,pri,rep,store.size,docs.count").End()

	results := [][]string{}
	headers := []string{"health", "status", "name", "primary shards", "replicas", "index size", "docs"}

	gjson.Parse(body).ForEach(func(key, value gjson.Result) bool {
		result := []string{
			value.Get("health").String(),
			value.Get("status").String(),
			value.Get("index").String(),
			value.Get("pri").String(),
			value.Get("rep").String(),
			value.Get("store\\.size").String(),
			value.Get("docs\\.count").String(),
		}

		results = append(results, result)
		return true // keep iterating
	})

	return results, headers
}

func GetHealth(server string, port int) (string, [][]string, []string) {
	_, body, _ := buildGetRequest(server, port, "_cat/health?h=status,relo,init,unassign,pending_tasks,active_shards_percent").End()

	results := [][]string{}
	headers := []string{"status", "relocating", "init", "unassigned", "active shards %"}

	gjson.Parse(body).ForEach(func(key, value gjson.Result) bool {
		result := []string{
			value.Get("status").String(),
			value.Get("relo").String(),
			value.Get("init").String(),
			value.Get("unassign").String(),
			value.Get("active_shards_percent").String(),
		}

		results = append(results, result)
		return true // keep iterating
	})

	status := results[0][0]
	caption := captionHealth(status)

	return caption, results, headers
}

func GetSettings(server string, port int) ([][]string, []string) {
	_, body, _ := buildGetRequest(server, port, ClusterSettingsPath).End()

	results := [][]string{}
	headers := []string{"setting", "value"}

	settings, _ := flatten.FlattenString(body, "", flatten.DotStyle)

	settingsMap, _ := gjson.Parse(settings).Value().(map[string]interface{})
	keys := []string{}

	for k, v := range settingsMap {
		strValue := v.(string)
		if strValue != "" {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)

	for _, k := range keys {
		setting := []string{k, settingsMap[k].(string)}
		results = append(results, setting)
	}

	return results, headers
}

func SetAllocation(server string, port int, allocation string) string {

	var allocationSetting string

	if allocation == "enable" {
		allocationSetting = "all"
	} else {
		allocationSetting = "none"
	}

	_, body, _ := buildPutRequest(server, port, ClusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(fmt.Sprintf(`{"transient" : { "cluster.routing.allocation.enable" : "%s"}}`, allocationSetting)).
		End()

	allocationVal := gjson.Get(body, "transient.cluster.routing.allocation.enable")

	return allocationVal.String()
}

func SetSetting(server string, port int, setting string, value string) (string, string, error) {

	_, settingsBody, _ := buildGetRequest(server, port, ClusterSettingsPath).End()

	existingValues := gjson.GetMany(settingsBody, fmt.Sprintf("transient.%s", setting), fmt.Sprintf("persistent.%s", setting))

	response, body, errs := buildPutRequest(server, port, ClusterSettingsPath).
		Set("Content-Type", "application/json").
		Send(fmt.Sprintf(`{"transient" : { "%s" : "%s"}}`, setting, value)).
		End()

	if errs != nil && len(errs) > 0 {
		errorText := []string{}
		for _, err := range errs {
			errorText = append(errorText, err.Error())
		}
		return "", "", errors.New(strings.Join(errorText, "\n"))
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

func GetSnapshots(server string, port int, cluster string) ([][]string, []string) {

	_, body, _ := buildGetRequest(server, port, fmt.Sprintf("_snapshot/%s/_all", cluster)).End()

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

func GetSnapshotStatus(server string, port int, cluster string, snapshot string) ([][]string, []string) {
	_, body, _ := buildGetRequest(server, port, fmt.Sprintf("_snapshot/%s/%s", cluster, snapshot)).End()

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

func PerformSnapshotsCheck(server string, port int, cluster string) ([]string, []string) {
	_, body, _ := buildGetRequest(server, port, fmt.Sprintf("_snapshot/%s/_all", cluster)).End()

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
