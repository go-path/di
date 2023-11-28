// Copyright (c) 2021 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// from https://github.com/uber-go/dig/blob/master/internal/graph/graph.go

package di

import "reflect"

// graph represents a simple interface for representation
// of a directed graph.
// It is assumed that each node in the graph is uniquely
// identified with an incremental positive integer (i.e. 1, 2, 3...).
// A value of 0 for a node represents a sentinel error value.
type graph struct {
	nodes []*provider // all the nodes defined in the graph.
}

// add adds a new value to the graph and returns its order.
func (g *graph) add(node *provider) int {
	order := len(g.nodes)
	g.nodes = append(g.nodes, node)
	return order
}

// order returns the total number of nodes in the graph
func (g *graph) order() int { return len(g.nodes) }

// edgesFrom returns the indices of nodes that are dependencies of node u.
//
// To do that, it retrieves the providers of the constructor's
// parameters and reports their orders.
func (g *graph) edgesFrom(u int) []int {
	var orders []int
	p := g.nodes[u]
	for i, param := range p.params {
		if p.useContext && i == 0 {
			// ignore context
			continue
		}
		orders = append(orders, getParamOrder(param)...)
	}
	return orders
}

// getParamOrder returns the order(s) of a parameter type.
func getParamOrder(param reflect.Type) []int {
	var orders []int
	for _, p := range providers[param] {
		orders = append(orders, p.order)
	}
	return orders
}

// isAcyclic uses depth-first search to find cycles
// in a generic graph represented by graph interface.
// If a cycle is found, it returns a list of nodes that
// are in the cyclic path, identified by their orders.
func (g *graph) isAcyclic() (bool, []int) {
	// cycleStart is a node that introduces a cycle in
	// the graph. Values in the range [1, g.order()) mean
	// that there exists a cycle in g.
	info := newCycleInfo(g.order())

	for i := 0; i < g.order(); i++ {
		info.reset()

		cycle := isAcyclic(g, i, info, nil /* cycle path */)
		if len(cycle) > 0 {
			return false, cycle
		}
	}

	return true, nil
}

// isAcyclic traverses the given graph starting from a specific node
// using depth-first search using recursion. If a cycle is detected,
// it returns the node that contains the "last" edge that introduces
// a cycle.
// For example, running isAcyclic starting from 1 on the following
// graph will return 3.
//
//	1 -> 2 -> 3 -> 1
func isAcyclic(g *graph, u int, info cycleInfo, path []int) []int {
	// We've already verified that there are no cycles from this node.
	if info[u].visited {
		return nil
	}
	info[u].visited = true
	info[u].onStack = true

	path = append(path, u)
	for _, v := range g.edgesFrom(u) {
		if !info[v].visited {
			if cycle := isAcyclic(g, v, info, path); len(cycle) > 0 {
				return cycle
			}
		} else if info[v].onStack {
			// We've found a cycle, and we have a full path back.
			// Prune it down to just the cyclic nodes.
			cycle := path
			for i := len(cycle) - 1; i >= 0; i-- {
				if cycle[i] == v {
					cycle = cycle[i:]
					break
				}
			}

			// Complete the cycle by adding this node to it.
			return append(cycle, v)
		}
	}
	info[u].onStack = false
	return nil
}

// cycleNode keeps track of a single node's info for cycle detection.
type cycleNode struct {
	visited bool
	onStack bool
}

// cycleInfo contains information about each node while we're trying to find
// cycles.
type cycleInfo []cycleNode

func newCycleInfo(order int) cycleInfo {
	return make(cycleInfo, order)
}

func (info cycleInfo) reset() {
	for i := range info {
		info[i].onStack = false
	}
}
