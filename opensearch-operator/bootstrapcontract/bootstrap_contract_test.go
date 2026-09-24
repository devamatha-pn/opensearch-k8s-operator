package bootstrapcontract

import (
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
