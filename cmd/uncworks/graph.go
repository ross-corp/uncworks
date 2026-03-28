// graph.go — uncworks graph: print the run execution tree.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func runGraph(args []string) error {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
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

	graph := resp.Msg
	printGraph(id, graph)
	return nil
}

// printGraph renders the RunGraph as an ASCII tree.
func printGraph(rootID string, graph *apiv1.RunGraph) {
	// Build a node map by name and a parent->children adjacency list.
	nodeByName := make(map[string]*apiv1.RunGraphNode, len(graph.GetNodes()))
	for _, n := range graph.GetNodes() {
		nodeByName[n.GetName()] = n
	}

	children := make(map[string][]string)
	for _, e := range graph.GetEdges() {
		children[e.GetParent()] = append(children[e.GetParent()], e.GetChild())
	}

	// Find the root node: the node whose name matches rootID or the first node
	// with no parent.
	parentSet := make(map[string]bool)
	for _, e := range graph.GetEdges() {
		parentSet[e.GetChild()] = true
	}

	// Collect root nodes (no incoming edges).
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
		printNode(root, "", true, nodeByName, children)
	}
}

func printNode(name, prefix string, isRoot bool, nodes map[string]*apiv1.RunGraphNode, children map[string][]string) {
	node := nodes[name]
	phase := "?"
	role := ""
	if node != nil {
		phase = phaseLabel(node.GetPhase())
		role = node.GetRole()
	}

	label := name
	if role != "" {
		label = fmt.Sprintf("%s (%s)", name, role)
	}

	if isRoot {
		fmt.Printf("▶ %s [%s]\n", label, phase)
	} else {
		fmt.Printf("%s%s [%s]\n", prefix, label, phase)
	}

	kids := children[name]
	for i, child := range kids {
		isLast := i == len(kids)-1
		connector := "├─ "
		childPrefix := prefix + "│  "
		if isLast {
			connector = "└─ "
			childPrefix = prefix + "   "
		}
		if isRoot {
			connector = "  " + connector
			childPrefix = "  " + childPrefix
		}
		fmt.Printf("%s", connector)
		printNode(child, childPrefix, false, nodes, children)
	}
}
