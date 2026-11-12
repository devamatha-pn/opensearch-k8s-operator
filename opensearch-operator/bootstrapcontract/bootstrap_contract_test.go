package bootstrapcontract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	opensearchv1 "github.com/opensearch-project/opensearch-k8s-operator/opensearch-operator/api/opensearch.org/v1"
	"github.com/opensearch-project/opensearch-k8s-operator/opensearch-operator/pkg/builders"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBootstrapDataUsesEmptyDir(t *testing.T) {
	cluster := opensearchv1.OpenSearchCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "contract-cluster",
			Namespace: "contract-namespace",
		},
	}

	pod := builders.NewBootstrapPod(&cluster, nil, nil)
	for _, volume := range pod.Spec.Volumes {
		if volume.Name != "data" {
			continue
		}
		if volume.EmptyDir == nil {
			t.Fatal("bootstrap data volume must use emptyDir")
		}
		if volume.PersistentVolumeClaim != nil {
			t.Fatal("bootstrap data volume must not reference a PVC")
		}
		return
	}

	t.Fatal("bootstrap pod is missing the data volume")
}

func TestExistingBootstrapPodIsNotReconciledAsMutable(t *testing.T) {
	reconcilerSource, err := os.ReadFile(filepath.Join("..", "pkg", "reconcilers", "cluster.go"))
	if err != nil {
		t.Fatalf("read bootstrap reconciler: %v", err)
	}

	source := string(reconcilerSource)
	start := strings.Index(source, "func (r *ClusterReconciler) reconcileBootstrapPod")
	end := strings.Index(source[start:], "func (r *ClusterReconciler) recreateBootstrapPod")
	if start == -1 || end == -1 {
		t.Fatal("bootstrap reconciliation boundaries not found")
	}
	bootstrapReconciler := source[start : start+end]
	if strings.Contains(bootstrapReconciler, "ReconcileResource(updatePod, reconciler.StatePresent)") {
		t.Fatal("existing bootstrap Pods must not be reconciled with StatePresent")
	}
	if !strings.Contains(bootstrapReconciler, "BootstrapPodNeedsRecreation") {
		t.Fatal("existing bootstrap Pods must be explicitly treated as immutable")
	}
}

// A cluster must not lose its seed node on pod readiness alone.
//
// Status.Initialized is derived from builders.AllMastersReady, which only checks
// StatefulSet pod readiness. The bootstrap pod also seeds the initial voting
// configuration under its own node id, so a lone cluster manager reports Ready
// while it is still waiting to reach that very seed. Deleting the seed at that
// point strands the cluster permanently: the manager logs "an election requires a
// node with id [<bootstrap>]" and "which is not a quorum" and can never form, so
// Dashboards are never created.
//
// CONFIRMED LIVE on staging-eks, 2026-09-29: two services on the same cluster and
// same operator. The 3-cluster-manager service formed cleanly and served
// Dashboards, because 2 of 3 is a majority and does not need the seed. The
// 1-cluster-manager service never formed, never got Dashboards, and logged
// exactly the messages above, and it stayed that way indefinitely.
//
// The contract is therefore: deletion requires an actual election, and the seed
// is retained (with a requeue) until then.
func TestBootstrapPodIsOnlyDeletedOnceAClusterManagerIsElected(t *testing.T) {
	reconcilerSource, err := os.ReadFile(filepath.Join("..", "pkg", "reconcilers", "cluster.go"))
	if err != nil {
		t.Fatalf("read cluster reconciler: %v", err)
	}

	source := string(reconcilerSource)

	// The deletion gate must be reachable only through the election check.
	gateStart := strings.Index(source, "if r.instance.Status.Initialized {")
	if gateStart == -1 {
		t.Fatal("Initialized gate not found in Reconcile")
	}
	gateEnd := strings.Index(source[gateStart:], "for _, nodePool := range r.instance.Spec.NodePools")
	if gateEnd == -1 {
		t.Fatal("end of the Initialized gate not found")
	}
	gate := source[gateStart : gateStart+gateEnd]

	if !strings.Contains(gate, "r.hasElectedClusterManager()") {
		t.Fatal("bootstrap deletion must be gated on hasElectedClusterManager, not pod readiness alone")
	}
	if !strings.Contains(gate, "reconcileBootstrapPod(bootstrapPod)") {
		t.Fatal("the seed must be retained while no cluster manager is elected")
	}
	// A formed cluster that wakes from sleep must not get a NEW seed: it would form
	// its own cluster uuid, which the real managers reject.
	if !strings.Contains(gate, "shouldMaintainBootstrapPod(") {
		t.Fatal("a missing seed must not be re-created on an already formed cluster with enough managers")
	}
	// An election completes inside OpenSearch with no Kubernetes event to
	// trigger a watch, so the gate must poll rather than wait.
	if !strings.Contains(gate, "RequeueAfter: bootstrapElectionRequeueInterval") {
		t.Fatal("the election check must requeue; an election raises no Kubernetes event")
	}

	// The check itself must treat any error as "not elected" and keep the seed.
	electionStart := strings.Index(source, "func (r *ClusterReconciler) hasElectedClusterManager() bool {")
	if electionStart == -1 {
		t.Fatal("hasElectedClusterManager not found")
	}
	electionEnd := strings.Index(source[electionStart:], "func (r *ClusterReconciler) reconcileBootstrapPod")
	if electionEnd == -1 {
		t.Fatal("end of hasElectedClusterManager not found")
	}
	election := source[electionStart : electionStart+electionEnd]

	ruleStart := strings.Index(source, "func hasElectedNonBootstrapManager(")
	if ruleStart == -1 {
		t.Fatal("hasElectedNonBootstrapManager not found")
	}
	rule := source[ruleStart:]
	if !strings.Contains(election, "hasElectedNonBootstrapManager(managers") || !strings.Contains(rule, `manager.Elected != "*"`) {
		t.Fatal("an election must be read from the elected field, not inferred from readiness")
	}
	if strings.Count(election, "return false") < 2 {
		t.Fatal("client and request errors must both fail closed, keeping the seed pod")
	}
	if strings.Contains(election, "return true") == false {
		t.Fatal("an elected cluster manager must allow the seed to be deleted")
	}
}
