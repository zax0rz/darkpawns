-- Dark Pawns for Mudlet: the mapper.
--
-- The server offers the whole world as a Mudlet map (GMCP Client.Map). A
-- profile with no map loads it on first connect, and a profile that loaded
-- it before takes each new version. A map the player built by hand is never
-- replaced without asking: "dp map" loads the full one on request.
--
-- Rooms the download doesn't have (new builds) are still mapped as you walk,
-- from Room.Info:
--   {"num":3001,"name":"...","area":"...","environment":"City","exits":{"n":3002}}
-- Room numbers are the game's own, so the map never guesses which room you
-- are in: a room seen once is recognised forever, and a new room is placed
-- one step from the room whose exit leads to it. The server leaves closed
-- doors out of "exits" (as the game's own exit list does), so an exit that
-- is missing now is never removed from the map.

DarkPawns.map = DarkPawns.map or {}
local map = DarkPawns.map

map.offsets = {
  n = { 0, 1, 0 }, s = { 0, -1, 0 }, e = { 1, 0, 0 }, w = { -1, 0, 0 },
  u = { 0, 0, 1 }, d = { 0, 0, -1 },
}

-- Environment names from Room.Info, coloured for the map. Ids start above
-- Mudlet's reserved range so they never collide with its defaults, and match
-- the ids the server's downloadable map uses (pkg/mudletmap).
map.environments = {
  Inside = { 257, 110, 100, 90 },
  City = { 258, 150, 140, 125 },
  Field = { 259, 125, 150, 80 },
  Forest = { 260, 60, 110, 60 },
  Hills = { 261, 140, 120, 70 },
  Mountain = { 262, 120, 110, 105 },
  ["Water Swim"] = { 263, 80, 120, 170 },
  ["Water Noswim"] = { 264, 50, 80, 150 },
  Underwater = { 265, 30, 50, 110 },
  Flying = { 266, 170, 190, 210 },
  Desert = { 267, 200, 170, 100 },
  Fire = { 268, 207, 75, 66 },
  Earth = { 269, 110, 85, 60 },
  Wind = { 270, 190, 200, 200 },
  Water = { 271, 70, 110, 160 },
  Unknown = { 272, 90, 90, 90 },
}

-- pending[destination] = list of {from, direction}: exits seen before the
-- room they lead to has been mapped.
map.pending = map.pending or {}
map.previous = map.previous or nil

local function areaId(name)
  if name == nil or name == "" then
    name = "Dark Pawns"
  end
  local areas = getAreaTable()
  if areas[name] then
    return areas[name]
  end
  return addAreaName(name)
end

function map.defineEnvironments()
  for _, env in pairs(map.environments) do
    setCustomEnvColor(env[1], env[2], env[3], env[4], 255)
  end
end

local function applyEnvironment(room, environment)
  local env = map.environments[environment] or map.environments.Unknown
  setRoomEnv(room, env[1])
end

-- placeRoom picks coordinates for a new room: one step from a mapped room
-- whose exit leads here, in the same area; otherwise the origin of its area.
local function placeRoom(room, area)
  for _, link in ipairs(map.pending[room] or {}) do
    if roomExists(link.from) and getRoomArea(link.from) == area then
      local x, y, z = getRoomCoordinates(link.from)
      local step = map.offsets[link.direction]
      return x + step[1], y + step[2], z + step[3]
    end
  end
  return 0, 0, 0
end

local function linkExits(room, exits)
  for direction, destination in pairs(exits) do
    if map.offsets[direction] then
      if roomExists(destination) then
        setExit(room, destination, direction)
      else
        setExitStub(room, direction, true)
        map.pending[destination] = map.pending[destination] or {}
        table.insert(map.pending[destination], { from = room, direction = direction })
      end
    end
  end
end

function map.onRoomInfo()
  local info = gmcp.Room and gmcp.Room.Info
  local room = info and tonumber(info.num)
  if not room then
    return
  end
  local area = areaId(info.area)

  if not roomExists(room) then
    addRoom(room)
    setRoomArea(room, area)
    setRoomCoordinates(room, placeRoom(room, area))
  end
  setRoomName(room, info.name or "")
  applyEnvironment(room, info.environment)

  -- Exits recorded earlier that lead here can now be drawn.
  for _, link in ipairs(map.pending[room] or {}) do
    if roomExists(link.from) then
      setExitStub(link.from, link.direction, false)
      setExit(link.from, room, link.direction)
    end
  end
  map.pending[room] = nil

  linkExits(room, info.exits or {})
  map.previous = room
  centerview(room)
end

-- Mudlet calls doSpeedWalk when a map room is double-clicked; the game
-- understands the same one-letter and up/down directions getPath returns.
function doSpeedWalk()
  if not getPath(speedWalkFrom, speedWalkTo) then
    cecho("\n<red>[ Dark Pawns ] No mapped path to that room.<reset>\n")
    return
  end
  for _, direction in ipairs(speedWalkDir) do
    send(direction, false)
  end
end

-- The downloadable world map -------------------------------------------------

map.versionKey = "darkpawns.mapVersion"

local function loadedVersion()
  local version = getMapUserData(map.versionKey)
  if version == "" then
    return nil
  end
  return version
end

function map.download()
  if not map.offer then
    cecho("\n<red>[ Dark Pawns ] The server hasn't offered a map yet. Connect and try again.<reset>\n")
    return
  end
  map.file = getMudletHomeDir() .. "/darkpawns-map.xml"
  cecho("\n<ansi_white>[ Dark Pawns ] Downloading the world map...<reset>\n")
  downloadFile(map.file, map.offer.url)
end

-- Client.Map: {"url": ..., "version": ...}
function map.onClientMap()
  local offer = gmcp.Client and gmcp.Client.Map
  if not (offer and offer.url and offer.version) then
    return
  end
  map.offer = offer
  local have = loadedVersion()
  if have == offer.version then
    return
  end
  if have or next(getRooms()) == nil then
    map.download()
  else
    cecho("\n<ansi_white>[ Dark Pawns ] A map of the whole world is available. Type <yellow>dp map<ansi_white> to load it; it replaces this profile's map.<reset>\n")
  end
end

function map.onDownloaded(_, file)
  if file ~= map.file then
    return
  end
  local ok, err = loadMap(file)
  if not ok then
    cecho(string.format("\n<red>[ Dark Pawns ] The map downloaded but didn't load: %s<reset>\n", tostring(err)))
    return
  end
  setMapUserData(map.versionKey, map.offer.version)
  map.defineEnvironments()
  cecho("\n<ansi_white>[ Dark Pawns ] World map loaded.<reset>\n")
  if map.previous and roomExists(map.previous) then
    centerview(map.previous)
  end
end

function map.onDownloadError(_, message, file)
  if file ~= nil and file ~= map.file then
    return
  end
  if map.file then
    cecho(string.format("\n<red>[ Dark Pawns ] The world map didn't download: %s<reset>\n", tostring(message)))
  end
end

map.defineEnvironments()
DarkPawns.on("map.room", "gmcp.Room.Info", map.onRoomInfo)
DarkPawns.on("map.offer", "gmcp.Client.Map", map.onClientMap)
DarkPawns.on("map.downloaded", "sysDownloadDone", map.onDownloaded)
DarkPawns.on("map.downloadError", "sysDownloadError", map.onDownloadError)
