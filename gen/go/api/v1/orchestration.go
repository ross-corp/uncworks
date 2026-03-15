// orchestration.go — Hand-written types for spec-orchestration-model.
// These supplement the generated api.pb.go until the next buf generate cycle.
package apiv1

import (
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// OrchestrationMode specifies how an agent run handles decomposition.
type OrchestrationMode int32

const (
	OrchestrationMode_ORCHESTRATION_MODE_UNSPECIFIED OrchestrationMode = 0
	OrchestrationMode_ORCHESTRATION_MODE_SINGLE      OrchestrationMode = 1
	OrchestrationMode_ORCHESTRATION_MODE_AUTO        OrchestrationMode = 2
	OrchestrationMode_ORCHESTRATION_MODE_MANUAL      OrchestrationMode = 3
)

// OrchestrationTask defines a single sub-task in a manual orchestration.
type OrchestrationTask struct {
	Name     string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Prompt   string   `protobuf:"bytes,2,opt,name=prompt,proto3" json:"prompt,omitempty"`
	RepoUrls []string `protobuf:"bytes,3,rep,name=repo_urls,json=repoUrls,proto3" json:"repo_urls,omitempty"`
}

func (x *OrchestrationTask) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *OrchestrationTask) GetPrompt() string {
	if x != nil {
		return x.Prompt
	}
	return ""
}

func (x *OrchestrationTask) GetRepoUrls() []string {
	if x != nil {
		return x.RepoUrls
	}
	return nil
}

// Orchestration contains the task list for manual orchestration mode.
type Orchestration struct {
	Tasks []*OrchestrationTask `protobuf:"bytes,1,rep,name=tasks,proto3" json:"tasks,omitempty"`
}

func (x *Orchestration) GetTasks() []*OrchestrationTask {
	if x != nil {
		return x.Tasks
	}
	return nil
}

// GetRunGraphRequest is the request message for GetRunGraph RPC.
type GetRunGraphRequest struct {
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *GetRunGraphRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

// RunGraphNode represents a single node in the run graph.
type RunGraphNode struct {
	Name        string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Phase       AgentRunPhase          `protobuf:"varint,2,opt,name=phase,proto3,enum=aot.api.v1.AgentRunPhase" json:"phase,omitempty"`
	Role        string                 `protobuf:"bytes,3,opt,name=role,proto3" json:"role,omitempty"`
	StartedAt   *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=started_at,json=startedAt,proto3" json:"started_at,omitempty"`
	CompletedAt *timestamppb.Timestamp `protobuf:"bytes,5,opt,name=completed_at,json=completedAt,proto3" json:"completed_at,omitempty"`
}

func (x *RunGraphNode) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *RunGraphNode) GetPhase() AgentRunPhase {
	if x != nil {
		return x.Phase
	}
	return AgentRunPhase_AGENT_RUN_PHASE_UNSPECIFIED
}

func (x *RunGraphNode) GetRole() string {
	if x != nil {
		return x.Role
	}
	return ""
}

func (x *RunGraphNode) GetStartedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.StartedAt
	}
	return nil
}

func (x *RunGraphNode) GetCompletedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.CompletedAt
	}
	return nil
}

// RunGraphEdge represents a parent-child edge in the run graph.
type RunGraphEdge struct {
	Parent string `protobuf:"bytes,1,opt,name=parent,proto3" json:"parent,omitempty"`
	Child  string `protobuf:"bytes,2,opt,name=child,proto3" json:"child,omitempty"`
}

func (x *RunGraphEdge) GetParent() string {
	if x != nil {
		return x.Parent
	}
	return ""
}

func (x *RunGraphEdge) GetChild() string {
	if x != nil {
		return x.Child
	}
	return ""
}

// RunGraph is the response message for GetRunGraph RPC.
type RunGraph struct {
	Nodes []*RunGraphNode `protobuf:"bytes,1,rep,name=nodes,proto3" json:"nodes,omitempty"`
	Edges []*RunGraphEdge `protobuf:"bytes,2,rep,name=edges,proto3" json:"edges,omitempty"`
}

func (x *RunGraph) GetNodes() []*RunGraphNode {
	if x != nil {
		return x.Nodes
	}
	return nil
}

func (x *RunGraph) GetEdges() []*RunGraphEdge {
	if x != nil {
		return x.Edges
	}
	return nil
}
