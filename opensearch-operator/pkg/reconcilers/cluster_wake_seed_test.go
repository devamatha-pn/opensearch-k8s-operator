package reconcilers

import (
	"testing"

	opensearchv1 "github.com/opensearch-project/opensearch-k8s-operator/opensearch-operator/api/opensearch.org/v1"
)

// A formed cluster that wakes from sleep has its cluster-manager pools back
// above 0 replicas but no manager has been elected yet. If the seed pod is
// missing at that moment, the operator must NOT create a new one: a fresh seed
// forms its own one-node cluster with a new cluster UUID, the managers (which
// keep their old UUID on their PVCs) refuse to join it, and the election gate
// then never passes, leaving the seed and its node up all day.
//
// CONFIRMED LIVE on bellxco-dev, 2026-11-12: bootstrap-0 was created at the
// 08:00 IST wake, formed uuid iGzHybi9..., while the managers carried
// od47wH2I... and rejected it with "different cluster uuid". The managers
// elected a leader among themselves 13 minutes later.
func TestShouldMaintainBootstrapPod(t *testing.T) {
	tests := []struct {
		name            string
		seedExists      bool
		managerReplicas int32
		want            bool
	}{
		{
			name:            "wake with 3 managers and no seed does not create one",
			seedExists:      false,
			managerReplicas: 3,
			want:            false,
		},
		{
			name:            "wake with 5 managers and no seed does not create one",
			seedExists:      false,
			managerReplicas: 5,
			want:            false,
		},
		{
			name:            "an existing seed is always kept, even with 3 managers",
			seedExists:      true,
			managerReplicas: 3,
			want:            true,
		},
		{
			// A single manager cannot elect without the seed that holds its
			// initial voting configuration, so behaviour is unchanged there.
			name:            "a single manager still gets its seed created",
			seedExists:      false,
			managerReplicas: 1,
			want:            true,
		},
		{
			name:            "two managers still get their seed created",
			seedExists:      false,
			managerReplicas: 2,
			want:            true,
		},
		{
			name:            "an existing seed is kept for a single manager",
			seedExists:      true,
			managerReplicas: 1,
			want:            true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldMaintainBootstrapPod(tc.seedExists, tc.managerReplicas); got != tc.want {
				t.Fatalf("shouldMaintainBootstrapPod(%v, %d) = %v, want %v", tc.seedExists, tc.managerReplicas, got, tc.want)
			}
		})
	}
}

func TestClusterManagerReplicas(t *testing.T) {
	pool := func(name string, replicas int32, roles ...string) opensearchv1.NodePool {
		return opensearchv1.NodePool{Component: name, Replicas: replicas, Roles: roles}
	}
	cluster := func(pools ...opensearchv1.NodePool) *opensearchv1.OpenSearchCluster {
		cr := &opensearchv1.OpenSearchCluster{}
		cr.Spec.General.Version = "2.19.1"
		cr.Spec.NodePools = pools
		return cr
	}

	tests := []struct {
		name string
		cr   *opensearchv1.OpenSearchCluster
		want int32
	}{
		{
			name: "the bellxco-dev shape: 3 managers and 3 data nodes",
			cr: cluster(
				pool("cluster-managers", 3, "cluster_manager"),
				pool("data", 3, "data", "ingest"),
			),
			want: 3,
		},
		{
			name: "data pools are not counted",
			cr:   cluster(pool("data", 5, "data")),
			want: 0,
		},
		{
			name: "several manager pools are added together",
			cr: cluster(
				pool("a", 1, "cluster_manager"),
				pool("b", 2, "cluster_manager", "data"),
			),
			want: 3,
		},
		{
			name: "the legacy master role name counts as a manager",
			cr:   cluster(pool("m", 3, "master")),
			want: 3,
		},
		{
			name: "a cluster with no pools has no managers",
			cr:   cluster(),
			want: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := clusterManagerReplicas(tc.cr); got != tc.want {
				t.Fatalf("clusterManagerReplicas() = %d, want %d", got, tc.want)
			}
		})
	}
}
