// +build integration

package vulcanizer_test

import (
	"testing"
	"time"

	"github.com/github/vulcanizer"
)

func TestNodes(t *testing.T) {
	c := vulcanizer.NewClient("localhost", 49200)

	nodes, err := c.GetNodes()

	if err != nil {
		t.Fatalf("Error getting nodes: %s", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("Expected two nodes, got: %v", len(nodes))
	}
}

func TestIndices(t *testing.T) {
	c := vulcanizer.NewClient("localhost", 49200)

	indices, err := c.GetIndices()

	if err != nil {
		t.Fatalf("Error getting indices: %s", err)
	}

	if len(indices) != 1 {
		t.Fatalf("Expected 1 index, got: %v", len(indices))
	}

	if indices[0].DocumentCount != 10 {
		t.Fatalf("Expected 10 docs, got: %v", indices[0].DocumentCount)
	}

	if indices[0].ReplicaCount != 1 {
		t.Fatalf("Expected replica count of 1, got: %v", indices[0].ReplicaCount)
	}

	_, _, err = c.SetIndexSetting(indices[0].Name, "number_of_replicas", "2")
	if err != nil {
		t.Fatalf("Error setting index number_of_replicas: %s", err)
	}

	indexSettings, err := c.GetIndexSettings(indices[0].Name)
	if err != nil {
		t.Fatalf("Error getting index settings: %s", err)
	}

	for i := range indexSettings {
		setting := indexSettings[i]
		if setting.Setting == "number_of_replicas" && setting.Value != "2" {
			t.Fatalf("Expected replica count of 2, got: %v", setting)
		}
	}

	indices, err = c.GetIndices()
	if err != nil {
		t.Fatalf("Error getting indices: %s", err)
	}

	if indices[0].ReplicaCount != 2 {
		t.Fatalf("Expected replica count of 2, got: %v", indices[0].ReplicaCount)
	}
}

func TestAliases(t *testing.T) {
	c := vulcanizer.NewClient("localhost", 49200)

	aliases, err := c.GetAliases()

	if err != nil {
		t.Fatalf("Error getting aliases: %s", err)
	}

	if len(aliases) != 1 {
		t.Fatalf("Expected 1 index, got: %v", len(aliases))
	}

	if aliases[0].Name != "integration_test_alias" {
		t.Fatalf("Expected aliases with name integration_test_alias, got: %v", aliases[0].Name)
	}

	if aliases[0].IndexName != "integration_test" {
		t.Fatalf("Expected index name integration_test, got: %v", aliases[0].IndexName)
	}
}

func TestVerifyRepository(t *testing.T) {
	c := vulcanizer.NewClient("localhost", 49200)

	verified, err := c.VerifyRepository("backup-repo")

	if err != nil {
		t.Fatalf("Error verifying repositories: %s", err)
	}

	if !verified {
		t.Fatalf("Expected to backup-repo to be a verified repository")
	}
}

func TestSnapshots(t *testing.T) {
	c := vulcanizer.NewClient("localhost", 49200)

	repos, err := c.GetRepositories()
	if err != nil {
		t.Fatalf("Error getting repositories: %s", err)
	}

	if len(repos) != 1 {
		t.Fatalf("Expected 1 repository, got: %+v", repos)
	}

	if repos[0].Name != "backup-repo" || repos[0].Type != "fs" {
		t.Fatalf("Unexpected repository values, got: %+v", repos[0])
	}

	snapshots, err := c.GetSnapshots("backup-repo")

	if err != nil {
		t.Fatalf("Error getting snapshots: %s", err)
	}

	if len(snapshots) != 1 || snapshots[0].Name != "snapshot_1" {
		t.Fatalf("Did not retrieve expected snapshots: %+v", snapshots)
	}

	snapshot, err := c.GetSnapshotStatus("backup-repo", "snapshot_1")

	if err != nil {
		t.Fatalf("Error getting snapshot status: %s", err)
	}

	if snapshot.State != "SUCCESS" {
		t.Fatalf("Expected snapshot to be a success: %+v", snapshot)
	}

	err = c.SnapshotAllIndices("backup-repo", "snapshot_2")
	if err != nil {
		t.Fatalf("Error taking second snapshot: %s", err)
	}

	// Allow snapshot operation to complete
	time.Sleep(5 * time.Second)

	err = c.DeleteSnapshot("backup-repo", "snapshot_1")
	if err != nil {
		t.Fatalf("Error deleting snapshot: %s", err)
	}

	snapshots, err = c.GetSnapshots("backup-repo")
	if err != nil {
		t.Fatalf("Error getting snapshots after delete: %s", err)
	}

	if len(snapshots) != 1 || snapshots[0].Name != "snapshot_2" {
		t.Fatalf("Unexpected snapshots, got: %+v", snapshots)
	}

	err = c.RestoreSnapshotIndices("backup-repo", "snapshot_2", []string{"integration_test"}, "restored_", nil)

	// Let the restore complete
	time.Sleep(5 * time.Second)

	indices, err := c.GetIndices()

	if err != nil {
		t.Fatalf("Error getting indices: %s", err)
	}

	if len(indices) != 2 {
		t.Fatalf("Expected 2 indices: %+v", indices)
	}

	var foundOriginalIndex, foundRestoredIndex bool

	for _, i := range indices {
		if i.Name == "integration_test" {
			foundOriginalIndex = true
		} else if i.Name == "restored_integration_test" {
			foundRestoredIndex = true
		}
	}

	if !foundOriginalIndex || !foundRestoredIndex {
		t.Fatalf("Couldn't find expected indices: %+v", indices)
	}

	err = c.DeleteIndex("restored_integration_test")
	if err != nil {
		t.Fatalf("Error deleting restored_integration_test index: %+v", indices)
	}

	indices, err = c.GetIndices()
	if err != nil {
		t.Fatalf("Error getting indices after index deletion: %s", err)
	}

	if len(indices) != 1 {
		t.Fatalf("Expected 1 indices: %+v", indices)
	}
}

func TestAllocations(t *testing.T) {
	c := vulcanizer.NewClient("localhost", 49200)

	val, err := c.SetAllocation("disable")

	if err != nil {
		t.Fatalf("Error disabling allocation: %s", err)
	}

	if val != "none" {
		t.Fatalf("Expected allocation to be none, got %s", val)
	}

	val, err = c.SetAllocation("enable")

	if err != nil {
		t.Fatalf("Error enabling allocation: %s", err)
	}

	if val != "all" {
		t.Fatalf("Expected allocation to be all, got %s", val)
	}
}

func TestGetShards_AllNodes(t *testing.T) {
	c := vulcanizer.NewClient("localhost", 49200)

	val, err := c.GetShards(nil)

	if err != nil {
		t.Fatalf("Error fetching shards: %s", err)
	}

	if val == nil {
		t.Fatal("Expected a slice of Shard, got nil instead")
	}

	// Account for the unassigned replicas
	if len(val) != 15 {
		t.Fatalf("Expected 15 shards, got %d instead", len(val))
	}
}

func TestGetShards_Regexp(t *testing.T) {
	c := vulcanizer.NewClient("localhost", 49200)

	val, err := c.GetShards([]string{"vulcanizer-elasticsearch-v(\\d|\\d-\\d)"})

	if err != nil {
		t.Fatalf("Error fetching shards: %s", err)
	}

	if val == nil {
		t.Fatal("Expected a slice of Shard, got nil instead")
	}

	if len(val) != 10 {
		t.Fatalf("Expected 15 shards, got %d instead", len(val))
	}
}
