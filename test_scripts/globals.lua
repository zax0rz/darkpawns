-- Global constants for Dark Pawns MUD scripts
-- Based on original globals.lua

function default()
    -- Direction constants
    NORTH = 0
    EAST = 1
    SOUTH = 2
    WEST = 3
    UP = 4
    DOWN = 5
    
    -- Message types for act()
    TO_ROOM = 1
    TO_VICT = 2
    TO_NOTVICT = 3
    TO_CHAR = 4
    
    -- Boolean constants
    TRUE = 1
    FALSE = 0
    NIL = nil
    
    -- Level constants
    LVL_IMMORT = 31
    
    -- Mob script trigger bitmask values (from structs.h)
    MS_NONE = 0
    MS_BRIBE = 1
    MS_GREET = 2
    MS_ONGIVE = 4
    MS_SOUND = 8
    MS_DEATH = 16
    MS_ONPULSE_ALL = 32
    MS_ONPULSE_PC = 64
    MS_FIGHTING = 128
    MS_ONCMD = 256
    
    -- Room script trigger bitmask values
    RS_NONE = 0
    RS_ENTER = 1
    RS_ONPULSE = 2
    RS_ONDROP = 4
    RS_ONGET = 8
    RS_ONCMD = 16
    
    -- Object script trigger bitmask values
    OS_NONE = 0
    OS_ONCMD = 1
    OS_ONPULSE = 2
end