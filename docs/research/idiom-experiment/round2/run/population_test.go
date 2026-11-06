package main

import (
	"go/ast"
	"go/parser"
	"reflect"
	"testing"
)

func TestOutputArgumentIndicesByCallShape(t *testing.T) {
	tests := []struct {
		name string
		call string
		want []int
	}{
		{name: "DreamContext selector", call: `ctx.SendToChar("message")`, want: []int{0}},
		{name: "DreamContext room selector", call: `ctx.SendToRoom("message")`, want: []int{0}},
		{name: "ordinary SendToChar call", call: `SendToChar(ch, "message")`, want: []int{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.call)
			if err != nil {
				t.Fatal(err)
			}
			call, ok := expr.(*ast.CallExpr)
			if !ok {
				t.Fatalf("parsed %T, want *ast.CallExpr", expr)
			}
			if got := outputArgumentIndices(call.Fun, len(call.Args)); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("outputArgumentIndices() = %v, want %v", got, tt.want)
			}
		})
	}
}
