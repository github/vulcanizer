package vulcan

import (
	"strings"

	"github.com/tidwall/gjson"
)

func ExcludeSettingsFromJson(settings []gjson.Result) *ExcludeSettings {
	excludeSettings := &ExcludeSettings{}

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
func captionHealth(health string) (caption string) {

	switch health {
	case "red":
		caption := "The cluster is red: One or more primary shards is not allocated on an index or indices. Please check for missing instances and return them to service if possible."
		return caption
	case "yellow":
		caption := "The cluster is yellow: One or more replica shards is not allocated on an index or indices. Please check for missing instances and return them to service if possible."
		return caption
	case "green":
		caption := "The cluster is green: All primary and replica shards are allocated. This does NOT mean the cluster is otherwise healthy."
		return caption
	default:
		return health
	}
}
