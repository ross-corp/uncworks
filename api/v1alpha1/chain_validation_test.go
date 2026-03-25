package v1alpha1

import "testing"

func TestValidateChainDAG_Linear(t *testing.T) {
	steps := []ChainStep{
		{Name: "A", TemplateRef: "t"},
		{Name: "B", TemplateRef: "t", DependsOn: []string{"A"}},
		{Name: "C", TemplateRef: "t", DependsOn: []string{"B"}},
	}
	if err := ValidateChainDAG(steps); err != nil {
		t.Fatalf("expected no error for linear DAG, got: %v", err)
	}
}

func TestValidateChainDAG_Diamond(t *testing.T) {
	steps := []ChainStep{
		{Name: "A", TemplateRef: "t"},
		{Name: "B", TemplateRef: "t", DependsOn: []string{"A"}},
		{Name: "C", TemplateRef: "t", DependsOn: []string{"A"}},
		{Name: "D", TemplateRef: "t", DependsOn: []string{"B", "C"}},
	}
	if err := ValidateChainDAG(steps); err != nil {
		t.Fatalf("expected no error for diamond DAG, got: %v", err)
	}
}

func TestValidateChainDAG_Cycle(t *testing.T) {
	steps := []ChainStep{
		{Name: "A", TemplateRef: "t", DependsOn: []string{"B"}},
		{Name: "B", TemplateRef: "t", DependsOn: []string{"A"}},
	}
	if err := ValidateChainDAG(steps); err == nil {
		t.Fatal("expected cycle error, got nil")
	}
}

func TestValidateChainDAG_UndefinedDep(t *testing.T) {
	steps := []ChainStep{
		{Name: "B", TemplateRef: "t", DependsOn: []string{"nonexistent"}},
	}
	if err := ValidateChainDAG(steps); err == nil {
		t.Fatal("expected error for undefined dependency, got nil")
	}
}

func TestValidateChainDAG_UndefinedContextFrom(t *testing.T) {
	steps := []ChainStep{
		{Name: "A", TemplateRef: "t"},
		{Name: "B", TemplateRef: "t", DependsOn: []string{"A"}, ContextFrom: "ghost"},
	}
	if err := ValidateChainDAG(steps); err == nil {
		t.Fatal("expected error for undefined contextFrom, got nil")
	}
}

func TestValidateChainDAG_UndefinedBranchFrom(t *testing.T) {
	steps := []ChainStep{
		{Name: "A", TemplateRef: "t"},
		{Name: "B", TemplateRef: "t", DependsOn: []string{"A"}, BranchFrom: "ghost"},
	}
	if err := ValidateChainDAG(steps); err == nil {
		t.Fatal("expected error for undefined branchFrom, got nil")
	}
}

func TestValidateChainDAG_DuplicateName(t *testing.T) {
	steps := []ChainStep{
		{Name: "A", TemplateRef: "t"},
		{Name: "A", TemplateRef: "t2"},
	}
	if err := ValidateChainDAG(steps); err == nil {
		t.Fatal("expected error for duplicate step name, got nil")
	}
}
