#ifndef lualib_h
#define lualib_h
#include "lua.h"
void lua_baselibopen(lua_State *L);
void lua_iolibopen(lua_State *L);
void lua_strlibopen(lua_State *L);
void lua_mathlibopen(lua_State *L);
void lua_dblibopen(lua_State *L);
#endif
