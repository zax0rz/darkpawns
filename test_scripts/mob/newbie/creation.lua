-- creation.lua - Character creation assistant
-- Based on original creation.lua from Dark Pawns MUD
-- Ported for Phase 3D

function question(num, ques, answ)
-- The function displays the question and series of answers rather than repeat
-- it for each individual question.

  local buf = ""
  local buf2 = ""

  act(num..ques, FALSE, ch, NIL, NIL, TO_CHAR)
  for i = 1, 3 do
    if (i == 1) then
      buf = "   a. "
      buf2 = ","
    elseif (i == 2) then
      buf = "   b. "
      buf2 = ", or"
    else
      buf = "   c. "
      buf2 = "?"
    end

    act(buf..answ[i]..buf2, FALSE, ch, NIL, NIL, TO_CHAR)
  end
end

function greet()
-- Greet new players and start character creation questions
-- This is a simplified version for the current engine

  say("Welcome to Dark Pawns, " .. ch.name .. "!")
  say("I will help you create your character.")
  
  -- For now, just give a welcome message
  -- In the full implementation, this would call question1(), question2(), question3()
  -- and process the answers to set attributes, alignment, gold, and equipment
  
  act("The creation assistant hands you some basic equipment.", FALSE, me, NIL, ch, TO_ROOM)
  act("You receive some basic equipment to start your journey.", FALSE, me, NIL, ch, TO_VICT)
  
  -- Give some starting gold
  ch.gold = ch.gold + 100
  
  -- Log for debugging
  log("Character creation completed for " .. ch.name)
end

-- Note: The original creation.lua had question1(), question2(), question3() functions
-- that were called directly from C code during character creation (CON_CHARCREATE state).
-- For this engine, we would need to integrate with the character creation system.
-- This is a simplified version that works with the greet trigger.