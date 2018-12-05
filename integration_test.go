// +build integration

package vulcanizer_test

import (
	"testing"

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

func TestGetSnapshotStatus(t *testing.T) {
	c := vulcanizer.NewClient("localhost", 49200)

	snapshot, err := c.GetSnapshotStatus("backup-repo", "snapshot_1")

	if err != nil {
		t.Fatalf("Error getting snapshot status: %s", err)
	}

	if snapshot.State != "SUCCESS" {
		t.Fatalf("Expected snapshot to be a success: %+v", snapshot)
	}
}
