package reconcilers

import (
	"testing"

	opensearchv1 "github.com/opensearch-project/opensearch-k8s-operator/opensearch-operator/api/opensearch.org/v1"
)

// A formed cluster that is put to sleep has every cluster-manager pool at 0
// replicas. No manager can be elected, so the bootstrap gate would keep and even
// re-create the seed pod all night, holding its node up. The sleep case must be
// recognised, and the wake case (pools back above 0) must NOT be.
func TestAllClusterManagerPoolsScaledToZero(t *testing.T) {
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
		want bool
	}{
		{
			name: "asleep: managers and data at 0",
			cr:   cluster(pool("cluster-managers", 0, "cluster_manager"), pool("data", 0, "data", "ingest")),
			want: true,
		},
		{
			name: "asleep: legacy master role name",
			cr:   cluster(pool("masters", 0, "master"), pool("data", 0, "data")),
			want: true,
		},
		{
			name: "awake: managers back after wake",
			cr:   cluster(pool("cluster-managers", 3, "cluster_manager"), pool("data", 0, "data")),
			want: false,
		},
		{
			name: "awake: a single manager replica",
			cr:   cluster(pool("cluster-managers", 1, "cluster_manager")),
			want: false,
		},
		{
			name: "mixed manager pools: one still running",
			cr:   cluster(pool("a", 0, "cluster_manager"), pool("b", 1, "cluster_manager")),
			want: false,
		},
		{
			name: "data at 0 but managers running is not asleep",
			cr:   cluster(pool("cluster-managers", 3, "cluster_manager"), pool("data", 0, "data")),
			want: false,
		},
		{
			name: "no manager pool is never treated as asleep",
			cr:   cluster(pool("data", 0, "data")),
			want: false,
		},
		{
			name: "no pools at all is never treated as asleep",
			cr:   cluster(),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := allClusterManagerPoolsScaledToZero(tc.cr); got != tc.want {
				t.Fatalf("allClusterManagerPoolsScaledToZero() = %v, want %v", got, tc.want)
			}
		})
	}
}
