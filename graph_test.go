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

type testGraphService0 uint8
type testGraphService1 uint8
type testGraphService2 uint8
type testGraphService3 uint8
type testGraphService4 uint8
type testGraphService5 uint8

type testGraphFactoryData struct {
	key    reflect.Type
	params []reflect.Type
}

var (
	t0 = Key[testGraphService0]() // 0
	t1 = Key[testGraphService1]() // 1
	t2 = Key[testGraphService2]() // 2
	t3 = Key[testGraphService3]() // 3
	t4 = Key[testGraphService4]() // 4
	t5 = Key[testGraphService5]() // 5
)

func testGraphFactoryParams(types ...reflect.Type) []reflect.Type {
	params := make([]reflect.Type, 0, len(types))
	params = append(params, types...)
	return params
}

func TestGraphIsAcyclic(t *testing.T) {
	testCases := [][]testGraphFactoryData{
		// 0
		{
			// Edges is an adjacency list representation of
			// a directed graph. i.e. edges[u] is a list of
			// nodes that node u has edges pointing to.
			{t0, testGraphFactoryParams()},
		},
		// 0 --> 1 --> 2
		{
			{t0, testGraphFactoryParams(t1)},
			{t1, testGraphFactoryParams(t2)},
			{t2, nil},
		},
		// 0 ---> 1 -------> 2
		// |                 ^
		// '-----------------'
		{
			{t0, testGraphFactoryParams(t1, t2)},
			{t1, testGraphFactoryParams(t2)},
			{t2, nil},
		},
		// 0 --> 1 --> 2    4 --> 5
		// |           ^    ^
		// +-----------'    |
		// '---------> 3 ---'
		{
			{t0, testGraphFactoryParams(t1, t2, t3)},
			{t1, testGraphFactoryParams(t2)},
			{t2, nil},
			{t3, testGraphFactoryParams(t4)},
			{t4, testGraphFactoryParams(t5)},
			{t5, nil},
		},
	}
	for _, tt := range testCases {
		c := New(nil).(*container)
		g := c.graph
		for _, pt := range tt {
			p := &Factory{
				parameterKeys: pt.params,
			}
			p.order = g.add(p)
			c.factories[pt.key] = append(c.factories[pt.key], p)
		}
		ok, cycle := g.isAcyclic()
		assert.True(t, ok, "expected acyclic, got cycle %v", cycle)
	}
}

func TestGraphIsCyclic(t *testing.T) {
	testCases := []struct {
		providers []testGraphFactoryData
		cycle     []int
	}{
		//
		// 0 ---> 1 ---> 2 ---> 3
		// ^                    |
		// '--------------------'
		{
			providers: []testGraphFactoryData{
				{t0, testGraphFactoryParams(t1)},
				{t1, testGraphFactoryParams(t2)},
				{t2, testGraphFactoryParams(t3)},
				{t3, testGraphFactoryParams(t0)},
			},
			cycle: []int{0, 1, 2, 3, 0},
		},
		//
		// 0 ---> 1 ---> 2
		//        ^      |
		//        '------'
		{
			providers: []testGraphFactoryData{
				{t0, testGraphFactoryParams(t1)},
				{t1, testGraphFactoryParams(t2)},
				{t2, testGraphFactoryParams(t1)},
			},
			cycle: []int{1, 2, 1},
		},
		//
		// 0 ---> 1 ---> 2 ----> 3
		// |      ^      |       ^
		// |      '------'       |
		// '---------------------'
		{
			providers: []testGraphFactoryData{
				{t0, testGraphFactoryParams(t1, t3)},
				{t1, testGraphFactoryParams(t2)},
				{t2, testGraphFactoryParams(t1, t3)},
				{t3, nil},
			},
			cycle: []int{1, 2, 1},
		},
	}
	for _, tt := range testCases {
		c := New(nil)
		cc := c.(*container)
		g := cc.graph
		for _, pt := range tt.providers {
			p := &Factory{parameterKeys: pt.params}
			p.order = g.add(p)
			cc.factories[pt.key] = append(cc.factories[pt.key], p)
		}
		ok, cycle := g.isAcyclic()
		assert.False(t, ok)
		assert.Equal(t, tt.cycle, cycle)
	}
}
