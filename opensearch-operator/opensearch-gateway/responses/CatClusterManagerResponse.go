package responses

// CatClusterManagerResponse models a single row of
// `GET /_cat/cluster_manager?format=json`.
//
// The elected field is the signal that matters for bootstrap: a cluster
// manager that is merely running has not necessarily completed an election.
// `elected` is only true once a cluster manager has won a vote and the
// cluster has formed, which is the point at which the bootstrap seed is no
// longer required.
//
// Node and Host are both the node id the manager votes under, and both carry
// the bootstrap pod's name when the seed is present. Live payload from
// staging-eks on 2026-09-30, where the seed was still the elected manager:
//
//	[{"id":"kBaFnMqjTryg2zdJHMwd8g",
//	  "host":"os-…-bootstrap-0","ip":"10.1.48.17",
//	  "node":"os-…-bootstrap-0"}]
//
// Node was not modelled originally and was silently dropped during decoding;
// the bootstrap gate compares against Host as well so it does not depend on
// that, but the field is kept so the identity the API actually returns is
// available.
type CatClusterManagerResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Host    string `json:"host"`
	Node    string `json:"node"`
	Elected string `json:"elected"`
}
