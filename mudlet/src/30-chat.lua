-- Dark Pawns for Mudlet: the chat window.
--
-- Comm.Channel.Text carries each line exactly as the game printed it:
--   {"channel":"gossip","talker":"Someone","text":"Someone gossips, 'hi'"}
-- The line also still appears in the main window; the chat window is a
-- scrollback of conversation only.

DarkPawns.chat = DarkPawns.chat or {}
local chat = DarkPawns.chat

-- A short channel tag ahead of each line, in the dock's muted ink.
function chat.onText()
  local message = gmcp.Comm and gmcp.Comm.Channel and gmcp.Comm.Channel.Text
  local console = DarkPawns.ui and DarkPawns.ui.chat
  if not (console and message and type(message.text) == "string") then
    return
  end
  local stamp = getTime(true, "hh:mm")
  console:decho(string.format("<156,148,138>%s %-6s<r> ", stamp, message.channel or ""))
  console:decho(ansi2decho(message.text) .. "\n")
end

DarkPawns.on("chat.text", "gmcp.Comm.Channel.Text", chat.onText)
