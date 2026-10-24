/* Minimal Lua 4.0 API declarations, reconstructed to link against the
   prebuilt liblua.a shipped in src/lua/lib (headers were not in the tree). */
#ifndef lua_h
#define lua_h
#include <stddef.h>
#define LUA_API extern
#define LUA_MULTRET (-1)
#define LUA_TUSERDATA 0
#define LUA_TNIL 1
#define LUA_TNUMBER 2
#define LUA_TSTRING 3
#define LUA_TTABLE 4
#define LUA_TFUNCTION 5
typedef struct lua_State lua_State;
typedef int (*lua_CFunction)(lua_State *L);
LUA_API lua_State *lua_open(int stacksize);
LUA_API void lua_close(lua_State *L);
LUA_API int lua_gettop(lua_State *L);
LUA_API void lua_settop(lua_State *L, int index);
LUA_API void lua_pushvalue(lua_State *L, int index);
LUA_API int lua_type(lua_State *L, int index);
LUA_API int lua_isnumber(lua_State *L, int index);
LUA_API int lua_isstring(lua_State *L, int index);
LUA_API double lua_tonumber(lua_State *L, int index);
LUA_API const char *lua_tostring(lua_State *L, int index);
LUA_API void *lua_touserdata(lua_State *L, int index);
LUA_API void lua_pushnil(lua_State *L);
LUA_API void lua_pushnumber(lua_State *L, double n);
LUA_API void lua_pushstring(lua_State *L, const char *s);
LUA_API void lua_pushcclosure(lua_State *L, lua_CFunction fn, int n);
LUA_API void lua_pushusertag(lua_State *L, void *u, int tag);
LUA_API void lua_getglobal(lua_State *L, const char *name);
LUA_API void lua_gettable(lua_State *L, int index);
LUA_API void lua_newtable(lua_State *L);
LUA_API void lua_setglobal(lua_State *L, const char *name);
LUA_API void lua_settable(lua_State *L, int index);
LUA_API int lua_call(lua_State *L, int nargs, int nresults);
LUA_API int lua_dofile(lua_State *L, const char *filename);
LUA_API int lua_dostring(lua_State *L, const char *str);
#define lua_pop(L,n) lua_settop(L, -(n)-1)
#define lua_pushcfunction(L,f) lua_pushcclosure(L, f, 0)
#define lua_register(L,n,f) (lua_pushcfunction(L, f), lua_setglobal(L, n))
#define lua_pushuserdata(L,u) lua_pushusertag(L, u, 0)
#define lua_istable(L,n) (lua_type(L,n) == LUA_TTABLE)
#define lua_isnil(L,n) (lua_type(L,n) == LUA_TNIL)
#define lua_isfunction(L,n) (lua_type(L,n) == LUA_TFUNCTION)
#endif
