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
package di

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraphIsAcyclic(t *testing.T) {
	testCases := [][]testProvider{
		// 0
		{
			// Edges is an adjacency list representation of
			// a directed graph. i.e. edges[u] is a list of
			// nodes that node u has edges pointing to.
			{t0, testParams()},
		},
		// 0 --> 1 --> 2
		{
			{t0, testParams(t1)},
			{t1, testParams(t2)},
			{t2, nil},
		},
		// 0 ---> 1 -------> 2
		// |                 ^
		// '-----------------'
		{
			{t0, testParams(t1, t2)},
			{t1, testParams(t2)},
			{t2, nil},
		},
		// 0 --> 1 --> 2    4 --> 5
		// |           ^    ^
		// +-----------'    |
		// '---------> 3 ---'
		{
			{t0, testParams(t1, t2, t3)},
			{t1, testParams(t2)},
			{t2, nil},
			{t3, testParams(t4)},
			{t4, testParams(t5)},
			{t5, nil},
		},
	}
	for _, tt := range testCases {
		providers = map[reflect.Type][]*provider{}
		g := &graph{}
		for _, pt := range tt {
			p := &provider{
				params: pt.params,
			}
			p.order = g.add(p)
			providers[pt.key] = append(providers[pt.key], p)
		}
		ok, cycle := g.isAcyclic()
		assert.True(t, ok, "expected acyclic, got cycle %v", cycle)
	}
}

func TestGraphIsCyclic(t *testing.T) {
	testCases := []struct {
		providers []testProvider
		cycle     []int
	}{
		//
		// 0 ---> 1 ---> 2 ---> 3
		// ^                    |
		// '--------------------'
		{
			providers: []testProvider{
				{t0, testParams(t1)},
				{t1, testParams(t2)},
				{t2, testParams(t3)},
				{t3, testParams(t0)},
			},
			cycle: []int{0, 1, 2, 3, 0},
		},
		//
		// 0 ---> 1 ---> 2
		//        ^      |
		//        '------'
		{
			providers: []testProvider{
				{t0, testParams(t1)},
				{t1, testParams(t2)},
				{t2, testParams(t1)},
			},
			cycle: []int{1, 2, 1},
		},
		//
		// 0 ---> 1 ---> 2 ----> 3
		// |      ^      |       ^
		// |      '------'       |
		// '---------------------'
		{
			providers: []testProvider{
				{t0, testParams(t1, t3)},
				{t1, testParams(t2)},
				{t2, testParams(t1, t3)},
				{t3, nil},
			},
			cycle: []int{1, 2, 1},
		},
	}
	for _, tt := range testCases {
		providers = map[reflect.Type][]*provider{}
		g := &graph{}
		for _, pt := range tt.providers {
			p := &provider{params: pt.params}
			p.order = g.add(p)
			providers[pt.key] = append(providers[pt.key], p)
		}
		ok, c := g.isAcyclic()
		assert.False(t, ok)
		assert.Equal(t, tt.cycle, c)
	}
}
