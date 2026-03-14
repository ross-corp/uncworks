package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// DebugHandler manages debug pod endpoints for scaling deployments up/down
// and providing connection information.
type DebugHandler struct {
	k8sClient  runtimeclient.Client
	restConfig *rest.Config
	namespace  string
}

// NewDebugHandler creates a new DebugHandler.
func NewDebugHandler(k8sClient runtimeclient.Client, restConfig *rest.Config, namespace string) *DebugHandler {
	return &DebugHandler{
		k8sClient:  k8sClient,
		restConfig: restConfig,
		namespace:  namespace,
	}
}

// RegisterDebugHandlers registers the debug pod REST endpoints on the given mux.
func (d *DebugHandler) RegisterDebugHandlers(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/runs/{id}/debug", d.handleStartDebug)
	mux.HandleFunc("DELETE /api/v1/runs/{id}/debug", d.handleStopDebug)
	mux.HandleFunc("GET /api/v1/runs/{id}/connect", d.handleConnect)
}

// ConnectInfo is the JSON response for the connect endpoint.
type ConnectInfo struct {
	PodName   string `json:"podName"`
	Namespace string `json:"namespace"`
	Command   string `json:"command"`
}

// handleStartDebug scales a run's Deployment to 1 in debug mode.
func (d *DebugHandler) handleStartDebug(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	crd, err := d.lookupAgentRun(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("agent run %q not found: %v", runID, err)})
		return
	}

	deployName := crd.Status.DeploymentName
	if deployName == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Sprintf("agent run %q has no deployment", runID)})
		return
	}

	// Get the Deployment.
	deploy := &appsv1.Deployment{}
	if err := d.k8sClient.Get(r.Context(), runtimeclient.ObjectKey{
		Namespace: d.namespace,
		Name:      deployName,
	}, deploy); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("deployment %q not found: %v", deployName, err)})
		return
	}

	// Patch Deployment: set replicas=1, add debug annotation, add debug env var to sidecar.
	replicas := int32(1)
	deploy.Spec.Replicas = &replicas

	if deploy.Annotations == nil {
		deploy.Annotations = make(map[string]string)
	}
	deploy.Annotations["aot.uncworks.io/mode"] = "debug"

	// Also set the annotation on the pod template so the sidecar can read it.
	if deploy.Spec.Template.Annotations == nil {
		deploy.Spec.Template.Annotations = make(map[string]string)
	}
	deploy.Spec.Template.Annotations["aot.uncworks.io/mode"] = "debug"

	// Add AOT_DEBUG_MODE=true env var to the sidecar container.
	d.ensureDebugEnvVar(deploy, true)

	if err := d.k8sClient.Update(r.Context(), deploy); err != nil {
		log.Printf("failed to update deployment %s for debug: %v", deployName, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to start debug: " + err.Error()})
		return
	}

	// Update CRD status debugActive=true via status subresource.
	crd.Status.DebugActive = true
	if err := d.k8sClient.Status().Update(r.Context(), crd); err != nil {
		log.Printf("failed to update AgentRun %s status for debug: %v", runID, err)
		// Non-fatal: deployment was already patched.
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "debug started"})
}

// handleStopDebug scales a run's Deployment to 0 and removes debug mode.
func (d *DebugHandler) handleStopDebug(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	crd, err := d.lookupAgentRun(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("agent run %q not found: %v", runID, err)})
		return
	}

	deployName := crd.Status.DeploymentName
	if deployName == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Sprintf("agent run %q has no deployment", runID)})
		return
	}

	// Get the Deployment.
	deploy := &appsv1.Deployment{}
	if err := d.k8sClient.Get(r.Context(), runtimeclient.ObjectKey{
		Namespace: d.namespace,
		Name:      deployName,
	}, deploy); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("deployment %q not found: %v", deployName, err)})
		return
	}

	// Patch Deployment: set replicas=0, remove debug annotation.
	replicas := int32(0)
	deploy.Spec.Replicas = &replicas

	delete(deploy.Annotations, "aot.uncworks.io/mode")
	delete(deploy.Spec.Template.Annotations, "aot.uncworks.io/mode")

	// Remove AOT_DEBUG_MODE env var from the sidecar container.
	d.ensureDebugEnvVar(deploy, false)

	if err := d.k8sClient.Update(r.Context(), deploy); err != nil {
		log.Printf("failed to update deployment %s to stop debug: %v", deployName, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to stop debug: " + err.Error()})
		return
	}

	// Update CRD status debugActive=false via status subresource.
	crd.Status.DebugActive = false
	if err := d.k8sClient.Status().Update(r.Context(), crd); err != nil {
		log.Printf("failed to update AgentRun %s status after debug stop: %v", runID, err)
		// Non-fatal: deployment was already patched.
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "debug stopped"})
}

// handleConnect returns connection info for attaching to a run's pod.
func (d *DebugHandler) handleConnect(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	crd, err := d.lookupAgentRun(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("agent run %q not found: %v", runID, err)})
		return
	}

	podName := crd.Status.PodName
	if podName == "" {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "no pod available for this run; start a debug session first"})
		return
	}

	info := ConnectInfo{
		PodName:   podName,
		Namespace: d.namespace,
		Command:   fmt.Sprintf("kubectl port-forward -n %s pod/%s 50052:50052", d.namespace, podName),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(info)
}

// lookupAgentRun retrieves the AgentRun CRD by run ID.
func (d *DebugHandler) lookupAgentRun(ctx context.Context, runID string) (*aotv1alpha1.AgentRun, error) {
	crd := &aotv1alpha1.AgentRun{}
	if err := d.k8sClient.Get(ctx, runtimeclient.ObjectKey{
		Namespace: d.namespace,
		Name:      runID,
	}, crd); err != nil {
		return nil, err
	}
	return crd, nil
}

// ensureDebugEnvVar adds or removes the AOT_DEBUG_MODE env var on the
// rpc-gateway (sidecar) container in the Deployment's pod template.
func (d *DebugHandler) ensureDebugEnvVar(deploy *appsv1.Deployment, add bool) {
	const containerName = "rpc-gateway"
	const envName = "AOT_DEBUG_MODE"

	containers := deploy.Spec.Template.Spec.Containers
	for i := range containers {
		if containers[i].Name != containerName {
			continue
		}
		if add {
			// Add env var if not already present.
			found := false
			for j := range containers[i].Env {
				if containers[i].Env[j].Name == envName {
					containers[i].Env[j].Value = "true"
					found = true
					break
				}
			}
			if !found {
				containers[i].Env = append(containers[i].Env, corev1.EnvVar{
					Name:  envName,
					Value: "true",
				})
			}
		} else {
			// Remove env var.
			envs := containers[i].Env
			for j := range envs {
				if envs[j].Name == envName {
					envs = append(envs[:j], envs[j+1:]...)
					containers[i].Env = envs
					break
				}
			}
		}
		break
	}
}
