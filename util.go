package vulcanizer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

func excludeSettingsFromJSON(settings []gjson.Result) ExcludeSettings {
	excludeSettings := ExcludeSettings{}

	if settings[0].String() == "" {
		excludeSettings.Ips = []string{}
	} else {
		excludeSettings.Ips = strings.Split(settings[0].String(), ",")
	}

	if settings[1].String() == "" {
		excludeSettings.Names = []string{}
	} else {
		excludeSettings.Names = strings.Split(settings[1].String(), ",")
	}

	if settings[2].String() == "" {
		excludeSettings.Hosts = []string{}
	} else {
		excludeSettings.Hosts = strings.Split(settings[2].String(), ",")
	}

	return excludeSettings
}

// Returns caption based on cluster health explaining the meaning of this state.
func captionHealth(clusterHealth ClusterHealth) (caption string) {
	unhealthyIndexList := make([]string, 0, len(clusterHealth.UnhealthyIndices))
	for _, index := range clusterHealth.UnhealthyIndices {
		status := fmt.Sprintf("%s is %s. %d shards are unassigned.", index.Name, index.Status, index.UnassignedShards)
		unhealthyIndexList = append(unhealthyIndexList, status)
	}

	switch clusterHealth.Status {
	case "red":
		caption := fmt.Sprintf("The cluster is red: One or more primary shards is not allocated on an index or indices. Please check for missing instances and return them to service if possible.\n%s", strings.Join(unhealthyIndexList, "\n"))
		return caption
	case "yellow":
		caption := fmt.Sprintf("The cluster is yellow: One or more replica shards is not allocated on an index or indices. Please check for missing instances and return them to service if possible.\n%s", strings.Join(unhealthyIndexList, "\n"))
		return caption
	case "green":
		caption := "The cluster is green: All primary and replica shards are allocated. This does NOT mean the cluster is otherwise healthy."
		return caption
	default:
		return clusterHealth.Status
	}
}

func enrichNodesWithAllocations(nodes []Node, allocations []DiskAllocation) []Node {
	var enrichedNodes []Node
	nodeAllocation := make(map[string]DiskAllocation)
	for _, alloc := range allocations {
		nodeAllocation[alloc.Node] = alloc
	}
	for _, node := range nodes {
		enrichedNode := Node{
			Name:        node.Name,
			IP:          node.IP,
			ID:          node.ID,
			Role:        node.Role,
			Master:      node.Master,
			Jdk:         node.Jdk,
			Version:     node.Version,
			Shards:      nodeAllocation[node.Name].Shards,
			DiskIndices: nodeAllocation[node.Name].DiskIndices,
			DiskUsed:    nodeAllocation[node.Name].DiskUsed,
			DiskAvail:   nodeAllocation[node.Name].DiskAvail,
			DiskTotal:   nodeAllocation[node.Name].DiskTotal,
			DiskPercent: nodeAllocation[node.Name].DiskPercent,
		}
		enrichedNodes = append(enrichedNodes, enrichedNode)
	}
	return enrichedNodes
}

func combineErrors(errs []error) error {
	errorText := []string{}
	for _, err := range errs {
		errorText = append(errorText, err.Error())
	}
	return errors.New(strings.Join(errorText, "\n"))
}

func escapeIndexName(index string) string {
	return strings.Replace(index, ".", "\\.", -1)
}
