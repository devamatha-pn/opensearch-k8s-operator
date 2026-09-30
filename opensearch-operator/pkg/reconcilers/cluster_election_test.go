package reconcilers

import (
	"testing"

	"github.com/opensearch-project/opensearch-k8s-operator/opensearch-operator/opensearch-gateway/responses"
)

// The seed pod must not be allowed to vouch for its own deletion.
//
// The bootstrap pod carries node.role=m, so it appears in
// `_cat/cluster_manager?format=json` exactly like any real manager, and in a
// fresh cluster it HOLDS the election until enough real managers form a quorum
// among themselves. A gate that asks only "is some manager elected?" is
// therefore self-satisfying: the seed votes for itself and the gate can never
// fire, so the seed survives forever.
//
// CONFIRMED LIVE on staging-eks, 2026-09-30, on a green cluster: bootstrap-0 was
// still the elected cluster manager 3h13m after formation. That left four
// manager-eligible nodes, raising quorum from 2-of-3 to 3-of-4 and keeping a
// node alive that nobody intended to pay for.
//
// The earlier regression for this gate was a source-contract test: it read
// cluster.go and asserted the SHAPE of the code. That passed while the
// behaviour was wrong in precisely this case. These tests are behavioural and
// would have failed against the old implementation.
func TestHasElectedNonBootstrapManager(t *testing.T) {
	const seed = "os-demo-bootstrap-0"

	tests := []struct {
		name     string
		managers []responses.CatClusterManagerResponse
		want     bool
	}{
		{
			// The exact live payload that left the seed alive for hours.
			name: "the seed electing itself is not evidence a real manager formed",
			managers: []responses.CatClusterManagerResponse{
				{ID: "kBaFnMqjTryg2zdJHMwd8g", Node: seed, Host: seed, Elected: "*"},
			},
			want: false,
		},
		{
			name: "a real manager elected alongside the seed is removable",
			managers: []responses.CatClusterManagerResponse{
				{ID: "kBaFnMqjTryg2zdJHMwd8g", Node: seed, Host: seed, Elected: "*"},
				{ID: "nO7", Node: "os-demo-cluster-managers-0", Elected: "*"},
			},
			want: true,
		},
		{
			name: "no manager elected yet keeps the seed",
			managers: []responses.CatClusterManagerResponse{
				{ID: "kBaFnMqjTryg2zdJHMwd8g", Node: seed, Host: seed, Elected: "-"},
				{ID: "nO7", Node: "os-demo-cluster-managers-0", Elected: "-"},
				{ID: "nO8", Node: "os-demo-cluster-managers-1", Elected: "-"},
			},
			want: false,
		},
		{
			name:     "an empty response keeps the seed",
			managers: nil,
			want:     false,
		},
		{
			// OpenSearch has reported the seed under `host` in some versions
			// rather than `node`, so the seed must be recognised on any of the
			// identity fields it can appear in.
			name: "the seed is matched on any identity field it reports",
			managers: []responses.CatClusterManagerResponse{
				{ID: "abc", Host: seed, Elected: "*"},
			},
			want: false,
		},
		{
			name: "the seed is matched on the name field too",
			managers: []responses.CatClusterManagerResponse{
				{ID: "abc", Name: seed, Elected: "*"},
			},
			want: false,
		},
		{
			name: "a manager elected only by hostname is still a real manager",
			managers: []responses.CatClusterManagerResponse{
				{ID: "nO7", Host: "os-demo-cluster-managers-0.os-demo-headless", Elected: "*"},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasElectedNonBootstrapManager(tc.managers, seed); got != tc.want {
				t.Fatalf("hasElectedNonBootstrapManager() = %v, want %v", got, tc.want)
			}
		})
	}
}
