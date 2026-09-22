-- Dark Pawns for Mudlet: the dock.
--
-- A right-hand dock with the character line, three gauges, the map and the
-- chat window. Colours follow the Dark Pawns site (Haunted Paperback): cream
-- text on the CRT canvas, oxblood for hit points. Mana and movement need two
-- more hues the site palette does not have; they are muted so the game text
-- stays the brightest thing on screen.

DarkPawns.ui = DarkPawns.ui or {}
local ui = DarkPawns.ui

ui.palette = {
  canvas = "#0a0908",
  paper = "#EFE7D6",
  muted = "#9C948A",
  rule = "#3A3430",
  hp = "#CF4B42",
  mana = "#6F86B8",
  move = "#B8964A",
}

ui.dockPercent = 32 -- share of the window width the dock takes

local function dockWidth()
  local width = getMainWindowSize()
  return math.floor(width * ui.dockPercent / 100)
end

local function gaugeStyle(colour)
  return string.format([[
    background-color: %s;
    border: 1px solid %s;
    border-radius: 2px;
  ]], colour, ui.palette.rule)
end

local function gaugeBackStyle()
  return string.format([[
    background-color: %s;
    border: 1px solid %s;
    border-radius: 2px;
  ]], ui.palette.canvas, ui.palette.rule)
end

function ui.build()
  if ui.dock then
    ui.dock:show()
    ui.layout()
    return
  end

  ui.dock = Geyser.VBox:new({
    name = "DarkPawns.dock",
    x = "-" .. ui.dockPercent .. "%", y = 0,
    width = ui.dockPercent .. "%", height = "100%",
  })

  ui.status = Geyser.Label:new({
    name = "DarkPawns.status",
    height = 28, v_policy = Geyser.Fixed,
    fgColor = ui.palette.paper,
    message = "Dark Pawns",
  }, ui.dock)
  ui.status:setStyleSheet(string.format([[
    background-color: %s;
    border-bottom: 1px solid %s;
    padding-left: 6px;
    font-family: serif;
  ]], ui.palette.canvas, ui.palette.rule))

  ui.gauges = {}
  for _, spec in ipairs({
    { key = "hp", label = "HP", colour = ui.palette.hp },
    { key = "mp", label = "Mana", colour = ui.palette.mana },
    { key = "mv", label = "Move", colour = ui.palette.move },
  }) do
    local gauge = Geyser.Gauge:new({
      name = "DarkPawns.gauge." .. spec.key,
      height = 22, v_policy = Geyser.Fixed,
    }, ui.dock)
    gauge.front:setStyleSheet(gaugeStyle(spec.colour))
    gauge.back:setStyleSheet(gaugeBackStyle())
    gauge.text:setStyleSheet("color: " .. ui.palette.paper .. "; padding-left: 6px;")
    gauge:setValue(0, 1, spec.label)
    gauge.label = spec.label
    ui.gauges[spec.key] = gauge
  end

  ui.map = Geyser.Mapper:new({
    name = "DarkPawns.map",
    v_stretch_factor = 3,
  }, ui.dock)

  ui.chat = Geyser.MiniConsole:new({
    name = "DarkPawns.chat",
    v_stretch_factor = 2,
    color = ui.palette.canvas,
    fontSize = 10,
    autoWrap = true,
    scrollBar = true,
  }, ui.dock)

  ui.layout()
end

function ui.layout()
  setBorderRight(dockWidth())
end

function ui.hide()
  if ui.dock then
    ui.dock:hide()
  end
  setBorderRight(0)
end

-- Char.Vitals: {"hp","maxhp","mp","maxmp","mv","maxmv"}
function ui.updateVitals()
  local vitals = gmcp.Char and gmcp.Char.Vitals
  if not (ui.gauges and vitals) then
    return
  end
  for key, gauge in pairs(ui.gauges) do
    local current, maximum = tonumber(vitals[key]), tonumber(vitals["max" .. key])
    if current and maximum then
      gauge:setValue(math.max(current, 0), math.max(maximum, 1),
        string.format("%s %d / %d", gauge.label, current, maximum))
    end
  end
end

-- Char.Status: {"name","level","race","class","gold"}
function ui.updateStatus()
  local status = gmcp.Char and gmcp.Char.Status
  if not (ui.status and status) then
    return
  end
  ui.status:echo(string.format("%s &middot; level %s %s %s &middot; %s gold",
    status.name or "", status.level or "?", status.race or "", status.class or "", status.gold or 0))
end

DarkPawns.on("ui.vitals", "gmcp.Char.Vitals", ui.updateVitals)
DarkPawns.on("ui.status", "gmcp.Char.Status", ui.updateStatus)
DarkPawns.on("ui.resize", "sysWindowResizeEvent", function() if ui.dock and not ui.dock.hidden then ui.layout() end end)

-- Mudlet's built-in starter interface draws its own gauges and chat for
-- games without one. Dark Pawns has one, so ask it to stand aside rather
-- than draw two of everything. (It does this by itself when the package
-- arrives through the server's Client.GUI offer; a hand import needs this.)
if BaseUI and type(BaseUI.standAside) == "function" and not (BaseUI.settings and BaseUI.settings.standingAside) then
  BaseUI.standAside(nil, DarkPawns.packageName)
end

ui.build()
ui.updateVitals()
ui.updateStatus()
