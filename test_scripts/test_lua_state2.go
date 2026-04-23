//go:build ignore

package testscripts

import (
	"fmt"
	"github.com/yuin/gopher-lua"
)

func test_lua_state2() {
	L := lua.NewState()
	defer L.Close()
	
	// Test 1: SetGlobal then GetGlobal
	fmt.Println("=== Test 1: SetGlobal/GetGlobal ===")
	L.SetGlobal("test", lua.LString("hello"))
	
	fmt.Printf("Stack top before GetGlobal: %d\n", L.GetTop())
	L.GetGlobal("test")
	fmt.Printf("Stack top after GetGlobal: %d\n", L.GetTop())
	
	if L.GetTop() > 0 {
		val := L.Get(-1)
		fmt.Printf("Value at top: %v (type: %v)\n", val, val.Type())
		L.Pop(1)
	}
	
	// Test 2: What does GetGlobal return?
	fmt.Println("\n=== Test 2: GetGlobal return value ===")
	L.GetGlobal("test")
	val := L.Get(-1)
	fmt.Printf("GetGlobal returns? Actually we get from stack: %v\n", val)
	L.Pop(1)
	
	// Test 3: Check gopher-lua documentation
	fmt.Println("\n=== Test 3: Direct call ===")
	// Actually, let's check the type
	L.GetGlobal("test")
	if L.Get(-1).Type() == lua.LTString {
		fmt.Println("It's a string on the stack")
	}
	L.Pop(1)
}