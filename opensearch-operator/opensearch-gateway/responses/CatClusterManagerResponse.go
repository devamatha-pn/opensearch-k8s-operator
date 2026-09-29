package responses

// CatClusterManagerResponse models a single row of
// `GET /_cat/cluster_manager?format=json`.
//
// The elected field is the signal that matters for bootstrap: a cluster
// manager that is merely running has not necessarily completed an election.
// `elected` is only true once a cluster manager has won a vote and the
// cluster has formed, which is the point at which the bootstrap seed is no
// longer required.
type CatClusterManagerResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Host    string `json:"host"`
	Elected string `json:"elected"`
}
