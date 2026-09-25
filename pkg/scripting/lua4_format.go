package scripting

import (
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// luaFormat is Lua 4's format (lstrlib.c str_format), which C's boot_lua
// opens with lua_strlibopen. Lua 5.1's string.format takes the same
// directives, but gopher-lua passes each one to Go's fmt, which has no %u:
// format("%u", 2000) would come out as "%!u(int64=2000)". Lua 4 prints %u
// (and %o, %x, %X) as sprintf of (unsigned int) of the number, so those
// arguments are converted to their 32-bit unsigned value and %u becomes %d
// before string.format runs.
func luaFormat(L *lua.LState) int {
	format := L.CheckString(1)
	args := make([]lua.LValue, 0, L.GetTop())
	var out strings.Builder
	arg := 2
	for i := 0; i < len(format); i++ {
		c := format[i]
		out.WriteByte(c)
		if c != '%' {
			continue
		}
		if i+1 < len(format) && format[i+1] == '%' {
			out.WriteByte('%')
			i++
			continue
		}
		// flags, width and precision, then the conversion
		j := i + 1
		for j < len(format) && strings.IndexByte("-+ #0123456789.", format[j]) >= 0 {
			j++
		}
		if j >= len(format) {
			L.ArgError(1, "invalid format (missing conversion)")
		}
		conv := format[j]
		out.WriteString(format[i+1 : j])
		value := L.Get(arg)
		switch conv {
		case 'u', 'o', 'x', 'X':
			n := uint32(int64(L.CheckNumber(arg)))
			value = lua.LNumber(n)
			if conv == 'u' {
				conv = 'd'
			}
		}
		out.WriteByte(conv)
		args = append(args, value)
		arg++
		i = j
	}
	stringFormat := L.GetField(L.GetGlobal("string"), "format")
	L.Push(stringFormat)
	L.Push(lua.LString(out.String()))
	for _, v := range args {
		L.Push(v)
	}
	L.Call(1+len(args), 1)
	return 1
}
