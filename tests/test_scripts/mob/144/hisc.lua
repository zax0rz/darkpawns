-- Test script based on original hisc.lua
oncmd = function()
    if argument then
        print("oncmd called with argument: " .. argument)
    else
        print("oncmd called with no argument")
    end
    return 1  -- TRUE
end

sound = function()
    -- Example sound function
    print("sound called")
end