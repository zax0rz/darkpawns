package testscripts

import (
	"fmt"
	"github.com/yuin/gopher-lua"
)

func test_getfield() {
	L := lua.NewState()
	defer L.Close()
	
	// Create a table
	tbl := L.NewTable()
	tbl.RawSetString("hp", lua.LNumber(100))
	tbl.RawSetString("gold", lua.LNumber(50))
	
	// Push table onto stack
	L.Push(tbl)
	fmt.Printf("Stack top after pushing table: %d\n", L.GetTop())
	
	// Try GetField
	val := L.GetField(tbl, "hp")
	fmt.Printf("GetField returned: %v (type: %v)\n", val, val.Type())
	fmt.Printf("Stack top after GetField: %d\n", L.GetTop())
	
	// Try with table on stack
	tbl2 := L.Get(-1)
	fmt.Printf("\nTable from stack: %v (type: %v)\n", tbl2, tbl2.Type())
	val2 := L.GetField(tbl2, "gold")
	fmt.Printf("GetField returned: %v (type: %v)\n", val2, val2.Type())
	fmt.Printf("Stack top: %d\n", L.GetTop())
	
	// Pop table
	L.Pop(1)
	fmt.Printf("Stack top after pop: %d\n", L.GetTop())
}