-- Dark Pawns for Mudlet: core.
--
-- Everything this package shows comes from the game's GMCP messages
-- (docs/gmcp.md in the Dark Pawns repository). The package never sends
-- game commands on its own and never reads game text: the gauges, map and
-- chat window are drawn from data the server already sent alongside the text.

DarkPawns = DarkPawns or {}
DarkPawns.version = "{{VERSION}}"
DarkPawns.packageName = "darkpawns"

-- The GMCP modules this package reads. Mudlet enables Char and Room on its
-- own; Comm.Channel has to be asked for.
DarkPawns.modules = { "Char 1", "Room 1", "Comm.Channel 1" }

-- Named anonymous event handlers, so reinstalling or reloading the package
-- replaces its handlers instead of stacking a second copy of each.
DarkPawns.handlers = DarkPawns.handlers or {}

function DarkPawns.on(key, event, fn)
  if DarkPawns.handlers[key] then
    killAnonymousEventHandler(DarkPawns.handlers[key])
  end
  DarkPawns.handlers[key] = registerAnonymousEventHandler(event, fn)
end

function DarkPawns.offAll()
  for key, id in pairs(DarkPawns.handlers) do
    killAnonymousEventHandler(id)
    DarkPawns.handlers[key] = nil
  end
end

function DarkPawns.negotiate(_, protocol)
  if protocol ~= "GMCP" then
    return
  end
  sendGMCP("Core.Supports.Add " .. yajl.to_string(DarkPawns.modules))
end

DarkPawns.on("negotiate", "sysProtocolEnabled", DarkPawns.negotiate)

-- A package installed mid-session (the server's Client.GUI offer arrives
-- after GMCP is already up) has missed sysProtocolEnabled, so ask now. The
-- gmcp table outlives a disconnect, so check the connection too.
local _, _, connected = getConnectionInfo()
if connected and type(gmcp) == "table" and next(gmcp) then
  DarkPawns.negotiate(nil, "GMCP")
end
