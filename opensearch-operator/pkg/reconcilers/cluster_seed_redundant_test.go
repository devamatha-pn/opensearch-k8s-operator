package reconcilers

import "testing"

// The seed pod only exists to help a new cluster form. Once the cluster formed
// and every cluster-manager StatefulSet has all replicas Ready, a cluster with
// 3 or more managers can elect among themselves (2 of 3 is a majority), so the
// operator removes the seed without waiting for the election query.
//
// CONFIRMED LIVE on bellxco-dev, 2026-11-12: the real managers elected a leader
// at 02:43:47 UTC, yet the operator kept logging "Keeping bootstrap pod until a
// cluster manager is elected" for hours and bootstrap-0 (an extra r6g.medium
// node) stayed up. The election query never turned true.
func TestSeedIsRedundant(t *testing.T) {
	tests := []struct {
		name              string
		initialized       bool
		managerReplicas   int32
		managerPoolsReady bool
		want              bool
	}{
		{
			name:              "formed cluster, 3 managers all Ready: seed is redundant",
			initialized:       true,
			managerReplicas:   3,
			managerPoolsReady: true,
			want:              true,
		},
		{
			name:              "formed cluster, 5 managers all Ready: seed is redundant",
			initialized:       true,
			managerReplicas:   5,
			managerPoolsReady: true,
			want:              true,
		},
		{
			name:              "managers not all Ready yet (wake in progress): keep the seed",
			initialized:       true,
			managerReplicas:   3,
			managerPoolsReady: false,
			want:              false,
		},
		{
			name:              "cluster not initialized yet (fresh create): keep the seed",
			initialized:       false,
			managerReplicas:   3,
			managerPoolsReady: true,
			want:              false,
		},
		{
			// A single manager cannot elect without the seed, which holds its
			// initial voting configuration. Confirmed live on staging-eks.
			name:              "a single manager always needs the seed until a real election",
			initialized:       true,
			managerReplicas:   1,
			managerPoolsReady: true,
			want:              false,
		},
		{
			name:              "two managers still need the seed",
			initialized:       true,
			managerReplicas:   2,
			managerPoolsReady: true,
			want:              false,
		},
		{
			name:              "no managers at all is never redundant",
			initialized:       true,
			managerReplicas:   0,
			managerPoolsReady: true,
			want:              false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := seedIsRedundant(tc.initialized, tc.managerReplicas, tc.managerPoolsReady)
			if got != tc.want {
				t.Fatalf("seedIsRedundant(%v, %d, %v) = %v, want %v",
					tc.initialized, tc.managerReplicas, tc.managerPoolsReady, got, tc.want)
			}
		})
	}
}
