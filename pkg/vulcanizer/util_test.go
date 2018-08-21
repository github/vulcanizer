package vulcanizer

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestExcludeSettingsFromJson_OneResult(t *testing.T) {
	body := `{"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_host":"excluded.host","_name":"excluded_name","_ip":"10.0.0.99"}}}}}}`
	excludedArray := gjson.GetMany(body, "transient.cluster.routing.allocation.exclude._ip", "transient.cluster.routing.allocation.exclude._name", "transient.cluster.routing.allocation.exclude._host")

	settings := ExcludeSettingsFromJson(excludedArray)

	if len(settings.Ips) != 1 && settings.Ips[0] != "10.0.0.99" {
		t.Fatalf("Ips should should contain 10.0.0.99, got %s", settings.Ips)
	}

	if len(settings.Names) != 1 && settings.Names[0] != "excluded_name" {
		t.Fatalf("Names should contain excluded_name, got %s", settings.Names)
	}

	if len(settings.Hosts) != 1 && settings.Hosts[0] != "excluded.host" {
		t.Fatalf("Hosts should contain excluded.host, got %s", settings.Hosts)
	}
}

func TestExcludeSettingsFromJson_NoResults(t *testing.T) {
	body := `{"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_host":"","_name":"","_ip":""}}}}}}`
	excludedArray := gjson.GetMany(body, "transient.cluster.routing.allocation.exclude._ip", "transient.cluster.routing.allocation.exclude._name", "transient.cluster.routing.allocation.exclude._host")

	settings := ExcludeSettingsFromJson(excludedArray)

	if len(settings.Ips) != 0 {
		t.Fatalf("Ips should be empty array, got %#v", settings.Ips)
	}

	if len(settings.Names) != 0 {
		t.Fatalf("Names should be empty array, got %s", settings.Names)
	}

	if len(settings.Hosts) != 0 {
		t.Fatalf("Hosts should be empty array, got %s", settings.Hosts)
	}
}
