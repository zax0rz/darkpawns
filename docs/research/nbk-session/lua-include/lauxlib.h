#ifndef lauxlib_h
#define lauxlib_h
#include "lua.h"
struct luaL_reg { const char *name; lua_CFunction func; };
void luaL_openlib(lua_State *L, const struct luaL_reg *l, int n);
#define luaL_openl(L,a) luaL_openlib(L, a, (sizeof(a)/sizeof(a[0])))
#endif
