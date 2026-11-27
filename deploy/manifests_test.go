package deploy

import (
	"os"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

func TestMetricsServiceMatchesController(t *testing.T) {
	svc := &corev1.Service{}
	readManifest(t, "501-service-metrics.yaml", svc)

	deploy := &appsv1.Deployment{}
	readManifest(t, "500-controller.yaml", deploy)

	if len(svc.Spec.Selector) == 0 {
		t.Fatal("service has no selector")
	}
	if len(svc.Spec.Ports) == 0 {
		t.Fatal("service has no ports")
	}

	podLabels := deploy.Spec.Template.Labels
	for key, value := range svc.Spec.Selector {
		if podLabels[key] != value {
			t.Errorf("service selector %s=%s does not match controller pod labels %v", key, value, podLabels)
		}
	}

	for _, port := range svc.Spec.Ports {
		if port.TargetPort.Type != intstr.String {
			t.Fatalf("service port %q: expected named targetPort, got %v", port.Name, port.TargetPort)
		}

		found := false
		for _, container := range deploy.Spec.Template.Spec.Containers {
			for _, containerPort := range container.Ports {
				if containerPort.Name == port.TargetPort.StrVal {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("service targetPort %q does not match any named container port in the controller", port.TargetPort.StrVal)
		}
	}
}

func readManifest(t *testing.T, name string, obj interface{}) {
	t.Helper()

	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	if err := yaml.UnmarshalStrict(data, obj); err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
}
