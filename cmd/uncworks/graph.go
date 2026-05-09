// graph.go — uncworks graph: print the run execution tree.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func runGraph(args []string) error {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	jsonOut := fs.Bool("json", false, "Output as JSON instead of ASCII tree")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks graph <run-id> [flags]\n\nPrint the execution tree for a run.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("run ID argument required")
	}
	id := fs.Arg(0)

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	req := connect.NewRequest(&apiv1.GetRunGraphRequest{Id: id})
	resp, err := client.GetRunGraph(context.Background(), req)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	if *jsonOut {
		type nodeJSON struct {
			Name  string `json:"name"`
			Phase string `json:"phase"`
			Role  string `json:"role,omitempty"`
		}
		type edgeJSON struct {
			Parent string `json:"parent"`
			Child  string `json:"child"`
		}
		type graphJSON struct {
			Nodes []nodeJSON `json:"nodes"`
			Edges []edgeJSON `json:"edges"`
		}
		g := graphJSON{}
		for _, n := range resp.Msg.GetNodes() {
			g.Nodes = append(g.Nodes, nodeJSON{
				Name:  n.GetName(),
				Phase: phaseLabel(n.GetPhase()),
				Role:  n.GetRole(),
			})
		}
		for _, e := range resp.Msg.GetEdges() {
			g.Edges = append(g.Edges, edgeJSON{Parent: e.GetParent(), Child: e.GetChild()})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(g)
	}

	printGraph(id, resp.Msg)
	return nil
}

// printGraph renders the RunGraph as an ASCII tree.
func printGraph(_ string, graph *apiv1.RunGraph) {
	nodeByName := make(map[string]*apiv1.RunGraphNode, len(graph.GetNodes()))
	for _, n := range graph.GetNodes() {
		nodeByName[n.GetName()] = n
	}

	kids := make(map[string][]string)
	for _, e := range graph.GetEdges() {
		kids[e.GetParent()] = append(kids[e.GetParent()], e.GetChild())
	}

	parentSet := make(map[string]bool)
	for _, e := range graph.GetEdges() {
		parentSet[e.GetChild()] = true
	}

	var roots []string
	for _, n := range graph.GetNodes() {
		if !parentSet[n.GetName()] {
			roots = append(roots, n.GetName())
		}
	}
	if len(roots) == 0 && len(graph.GetNodes()) > 0 {
		roots = []string{graph.GetNodes()[0].GetName()}
	}

	for _, root := range roots {
		fmt.Printf("▶ %s\n", graphNodeLabel(root, nodeByName))
		graphPrintChildren(root, "  ", nodeByName, kids)
	}
}

// graphPrintChildren prints all children of parent using box-drawing connectors.
// indent is the prefix for each child's connector line.
func graphPrintChildren(parent, indent string, nodes map[string]*apiv1.RunGraphNode, kids map[string][]string) {
	children := kids[parent]
	for i, child := range children {
		isLast := i == len(children)-1
		connector := "├─ "
		childIndent := indent + "│  "
		if isLast {
			connector = "└─ "
			childIndent = indent + "   "
		}
		fmt.Printf("%s%s%s\n", indent, connector, graphNodeLabel(child, nodes))
		graphPrintChildren(child, childIndent, nodes, kids)
	}
}

func graphNodeLabel(name string, nodes map[string]*apiv1.RunGraphNode) string {
	node := nodes[name]
	if node == nil {
		return name + " [?]"
	}
	phase := phaseLabel(node.GetPhase())
	if role := node.GetRole(); role != "" {
		return fmt.Sprintf("%s (%s) [%s]", name, role, phase)
	}
	return fmt.Sprintf("%s [%s]", name, phase)
}
