//go:build ignore

package main

import (
	"fmt"
	"github.com/yuin/gopher-lua"
)

func main() {
	L := lua.NewState()
	defer L.Close()
	
	// Set a global
	L.SetGlobal("test", lua.LString("hello"))
	
	// Get it back
	fmt.Printf("Stack top before GetGlobal: %d\n", L.GetTop())
	L.GetGlobal("test")
	fmt.Printf("Stack top after GetGlobal: %d\n", L.GetTop())
	
	if L.GetTop() > 0 {
		val := L.Get(-1)
		fmt.Printf("Value: %v (type: %v)\n", val, val.Type())
		L.Pop(1)
	}
	
	// Try getting a non-existent global
	fmt.Printf("\nStack top before GetGlobal non-existent: %d\n", L.GetTop())
	L.GetGlobal("nonexistent")
	fmt.Printf("Stack top after GetGlobal non-existent: %d\n", L.GetTop())
	
	if L.GetTop() > 0 {
		val := L.Get(-1)
		fmt.Printf("Value: %v (type: %v)\n", val, val.Type())
		L.Pop(1)
	}
}