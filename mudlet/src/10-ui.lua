-- Dark Pawns for Mudlet: the dock.
--
-- The dock is the website's /play page brought into Mudlet: a Paper-Deep
-- chassis around the game's dark canvas. The chassis carries the lockup (the
-- pawn, then DARK over PAWNS with Oxblood on PAWNS alone), the character
-- line, and the gauges; the map and the chat window are game surfaces, so they
-- keep the dark canvas, like the terminal on the site.
--
-- Colours are DESIGN.md tokens only. The gauges carry their numbers as ink
-- text on paper beside a thin bar, so no reading depends on colour and every
-- piece of text clears WCAG AA contrast.

DarkPawns.ui = DarkPawns.ui or {}
local ui = DarkPawns.ui

ui.palette = {
  paper = "#EFE7D6",
  paperDeep = "#E5DAC1",
  ink = "#1A1614",
  inkMuted = "#56504A",
  oxblood = "#A8201A",
  canvas = "#0a0908",
}

-- DESIGN.md typography, with its own fallbacks: players rarely have the
-- site's fonts installed, and Georgia or the system serif stands in.
ui.fonts = {
  display = "'DM Serif Display', Georgia, serif",
  body = "'Source Serif 4', Georgia, serif",
  mono = "'JetBrains Mono', 'Fira Code', monospace",
}

ui.dockPercent = 32 -- share of the window width the dock takes

-- The pawn, rasterized from the site header's canonical drawing by
-- cmd/mudlet-package (Mudlet 5.0 labels cannot show SVG). Written to the
-- profile directory at load, because a label image has to be a file.
ui.pawnPNG = [[
{{PAWN_PNG_BASE64}}
]]

local function decodeBase64(data)
  local alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
  local lookup = {}
  for i = 1, #alphabet do
    lookup[alphabet:sub(i, i)] = i - 1
  end
  local out, bits, count = {}, 0, 0
  for char in data:gmatch("[%w+/]") do
    bits = bits * 64 + lookup[char]
    count = count + 1
    if count == 4 then
      out[#out + 1] = string.char(math.floor(bits / 65536) % 256, math.floor(bits / 256) % 256, bits % 256)
      bits, count = 0, 0
    end
  end
  if count == 3 then
    out[#out + 1] = string.char(math.floor(bits / 1024) % 256, math.floor(bits / 4) % 256)
  elseif count == 2 then
    out[#out + 1] = string.char(math.floor(bits / 16) % 256)
  end
  return table.concat(out)
end
ui.decodeBase64 = decodeBase64

local function writePawn()
  local path = getMudletHomeDir() .. "/darkpawns-pawn.png"
  local file = io.open(path, "wb")
  if not file then
    return nil
  end
  file:write(decodeBase64(ui.pawnPNG))
  file:close()
  return path
end

local function dockWidth()
  local width = getMainWindowSize()
  return math.floor(width * ui.dockPercent / 100)
end

local function transparent(label)
  label:setStyleSheet("background-color: transparent;")
end

-- An eyebrow: the site's small uppercase section label in Oxblood.
local function eyebrow(name, text)
  local label = Geyser.Label:new({ name = name, height = 22, v_policy = Geyser.Fixed }, ui.box)
  transparent(label)
  label:echo(string.format(
    [[<span style="font-family: %s; font-size: 11px; font-weight: bold; color: %s;">%s</span>]],
    ui.fonts.mono, ui.palette.oxblood, text))
  return label
end

function ui.build()
  if ui.dock then
    ui.dock:show()
    ui.layout()
    return
  end

  -- The chassis: Paper-Deep, an Ink rule on the edge facing the game text.
  ui.dock = Geyser.Label:new({
    name = "DarkPawns.dock",
    x = "-" .. ui.dockPercent .. "%", y = 0,
    width = ui.dockPercent .. "%", height = "100%",
  })
  ui.dock:setStyleSheet(string.format(
    "background-color: %s; border-left: 1px solid %s;", ui.palette.paperDeep, ui.palette.ink))

  ui.box = Geyser.VBox:new({
    name = "DarkPawns.box", x = "4%", y = "1%", width = "92%", height = "98%",
  }, ui.dock)

  -- The lockup.
  ui.header = Geyser.HBox:new({ name = "DarkPawns.header", height = 56, v_policy = Geyser.Fixed }, ui.box)
  ui.pawn = Geyser.Label:new({ name = "DarkPawns.pawn", width = 33, h_policy = Geyser.Fixed }, ui.header)
  transparent(ui.pawn)
  local pawnFile = writePawn()
  if pawnFile then
    ui.pawn:setBackgroundImage(pawnFile)
  end
  ui.wordmark = Geyser.Label:new({ name = "DarkPawns.wordmark" }, ui.header)
  transparent(ui.wordmark)
  ui.wordmark:echo(string.format(
    [[<div style="font-family: %s; font-size: 21px; line-height: 82%%; color: %s; padding-left: 8px;">DARK<br><span class="accent" style="color: %s;">PAWNS</span></div>]],
    ui.fonts.display, ui.palette.ink, ui.palette.oxblood))

  ui.status = Geyser.Label:new({ name = "DarkPawns.status", height = 26, v_policy = Geyser.Fixed }, ui.box)
  ui.status:setStyleSheet(string.format(
    "background-color: transparent; border-top: 1px solid %s;", ui.palette.ink))
  ui.status:echo(string.format([[<span style="font-family: %s; color: %s;">Not in the game yet</span>]],
    ui.fonts.body, ui.palette.inkMuted))

  -- Gauges: the reading in ink text, the bar as a thin rule of colour. HP is
  -- Oxblood, mana Ink, movement Ink-Muted: all three read by label first.
  ui.gauges, ui.readings = {}, {}
  for _, spec in ipairs({
    { key = "hp", label = "HP", colour = ui.palette.oxblood },
    { key = "mp", label = "MANA", colour = ui.palette.ink },
    { key = "mv", label = "MOVE", colour = ui.palette.inkMuted },
  }) do
    local reading = Geyser.Label:new({
      name = "DarkPawns.reading." .. spec.key, height = 20, v_policy = Geyser.Fixed,
    }, ui.box)
    transparent(reading)
    reading.label = spec.label
    ui.readings[spec.key] = reading

    local gauge = Geyser.Gauge:new({
      name = "DarkPawns.gauge." .. spec.key, height = 8, v_policy = Geyser.Fixed,
    }, ui.box)
    gauge.front:setStyleSheet(string.format("background-color: %s; border: none;", spec.colour))
    gauge.back:setStyleSheet(string.format(
      "background-color: %s; border: 1px solid %s;", ui.palette.paper, ui.palette.ink))
    gauge.text:setStyleSheet("background-color: transparent;")
    gauge:setValue(0, 1, "")
    ui.gauges[spec.key] = gauge
  end

  eyebrow("DarkPawns.mapLabel", "MAP")
  ui.map = Geyser.Mapper:new({ name = "DarkPawns.map", v_stretch_factor = 3 }, ui.box)

  eyebrow("DarkPawns.chatLabel", "CHAT")
  ui.chat = Geyser.MiniConsole:new({
    name = "DarkPawns.chat",
    v_stretch_factor = 2,
    color = ui.palette.canvas,
    fontSize = 10,
    autoWrap = true,
    scrollBar = true,
  }, ui.box)

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

local function setReading(reading, current, maximum)
  reading:echo(string.format(
    [[<table width="100%%" cellspacing="0" cellpadding="0"><tr>]] ..
    [[<td style="font-family: %s; font-size: 11px; font-weight: bold; color: %s;">%s</td>]] ..
    [[<td align="right" style="font-family: %s; font-size: 12px; color: %s;">%d / %d</td>]] ..
    [[</tr></table>]],
    ui.fonts.mono, ui.palette.inkMuted, reading.label, ui.fonts.mono, ui.palette.ink, current, maximum))
  reading.value = { current, maximum }
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
      gauge:setValue(math.max(current, 0), math.max(maximum, 1), "")
      setReading(ui.readings[key], current, maximum)
    end
  end
end

-- Char.Status: {"name","level","race","class","gold"}
function ui.updateStatus()
  local status = gmcp.Char and gmcp.Char.Status
  if not (ui.status and status) then
    return
  end
  ui.status:echo(string.format(
    [[<span style="font-family: %s; color: %s;"><b style="color: %s;">%s</b> &middot; level %s %s %s &middot; %s gold</span>]],
    ui.fonts.body, ui.palette.inkMuted, ui.palette.ink,
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
