-- Dark Pawns for Mudlet: the dock.
--
-- The dock is the website's /play page brought into Mudlet: a Paper-Deep
-- chassis around the game's dark canvas. The chassis carries the lockup (the
-- site header's own, as a picture), the character
-- line, and the gauges; the map and the chat window are game surfaces, so they
-- keep the dark canvas, like the terminal on the site.
--
-- Colours are DESIGN.md tokens only. The gauges carry their numbers as ink
-- text on paper beside a thin bar, so no reading depends on colour and every
-- piece of text clears WCAG AA contrast.

-- A dock belongs to one package version. Mudlet upgrades a package inside
-- the running profile, so the new version loads into the Lua state that
-- still holds the old version's dock (hidden by its uninstall). Reusing it
-- would show the old dock again; instead the new version starts a fresh
-- table and gives its widgets names of its own.
if DarkPawns.ui and DarkPawns.ui.version ~= DarkPawns.version then
  if DarkPawns.ui.hide then
    DarkPawns.ui.hide()
  end
  DarkPawns.ui = nil
end
DarkPawns.ui = DarkPawns.ui or { version = DarkPawns.version }
local ui = DarkPawns.ui

-- A widget name unique to this version.
local function id(name)
  return "DarkPawns-" .. DarkPawns.version .. "." .. name
end

ui.palette = {
  paper = "#EFE7D6",
  paperDeep = "#E5DAC1",
  ink = "#1A1614",
  inkMuted = "#56504A",
  oxblood = "#A8201A",
  canvas = "#0a0908",
}

-- DESIGN.md typography, with its own fallbacks: players rarely have the
-- site's fonts installed, and Georgia or the system serif stands in. The
-- display face appears only in the lockup, which is a picture.
ui.fonts = {
  body = "'Source Serif 4', Georgia, serif",
  mono = "'JetBrains Mono', 'Fira Code', monospace",
}

ui.dockPercent = 32 -- share of the window width the dock takes

-- The lockup, as the site header draws it: the pawn, then DARK over PAWNS in
-- DM Serif Display. A package cannot install fonts and a label cannot draw
-- SVG, so scripts/render_mudlet_lockup.py renders the header's own drawing
-- and CSS to a picture, at the CSS size and at twice it. The dock shows it
-- unscaled; Qt takes the @2x file on high-density screens. Written to the
-- profile directory at load, because a label image has to be a file.
ui.lockupPNG = [[
{{LOCKUP_PNG_BASE64}}
]]
ui.lockup2xPNG = [[
{{LOCKUP_2X_PNG_BASE64}}
]]
ui.lockupHeight = 55 -- lockup.png's height: the header row is exactly the picture

local function decodeBase64(data)
  local alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
  local lookup = {}
  for i = 1, #alphabet do
    lookup[alphabet:sub(i, i)] = i - 1
  end
  -- Decoded in runs of 1024 groups, so no single concat holds the image.
  local parts, out, bits, count = {}, {}, 0, 0
  for char in data:gmatch("[%w+/]") do
    bits = bits * 64 + lookup[char]
    count = count + 1
    if count == 4 then
      out[#out + 1] = string.char(math.floor(bits / 65536) % 256, math.floor(bits / 256) % 256, bits % 256)
      bits, count = 0, 0
      if #out == 1024 then
        parts[#parts + 1] = table.concat(out)
        out = {}
      end
    end
  end
  if count == 3 then
    out[#out + 1] = string.char(math.floor(bits / 1024) % 256, math.floor(bits / 4) % 256)
  elseif count == 2 then
    out[#out + 1] = string.char(math.floor(bits / 16) % 256)
  end
  parts[#parts + 1] = table.concat(out)
  return table.concat(parts)
end
ui.decodeBase64 = decodeBase64

local function writeImage(path, data)
  local file = io.open(path, "wb")
  if not file then
    return false
  end
  file:write(decodeBase64(data))
  file:close()
  return true
end

local function writeLockup()
  local path = getMudletHomeDir() .. "/darkpawns-lockup.png"
  if not writeImage(path, ui.lockupPNG) then
    return nil
  end
  writeImage(getMudletHomeDir() .. "/darkpawns-lockup@2x.png", ui.lockup2xPNG)
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
    ui.map:raise()
    return
  end

  -- The chassis: Paper-Deep, an Ink rule on the edge facing the game text.
  ui.dock = Geyser.Label:new({
    name = id("dock"),
    x = "-" .. ui.dockPercent .. "%", y = 0,
    width = ui.dockPercent .. "%", height = "100%",
  })
  ui.dock:setStyleSheet(string.format(
    "background-color: %s; border-left: 1px solid %s;", ui.palette.paperDeep, ui.palette.ink))

  ui.box = Geyser.VBox:new({
    name = id("box"), x = "4%", y = "1%", width = "92%", height = "98%",
  }, ui.dock)

  -- The lockup, drawn at its own size from the left edge: never stretched.
  ui.header = Geyser.Label:new({
    name = id("header"), height = ui.lockupHeight, v_policy = Geyser.Fixed,
  }, ui.box)
  local lockup = writeLockup()
  if lockup then
    ui.header:setStyleSheet(string.format(
      [[background-color: transparent; background-image: url("%s"); background-repeat: no-repeat; background-position: left center;]],
      lockup))
  else
    transparent(ui.header)
  end

  ui.status = Geyser.Label:new({ name = id("status"), height = 26, v_policy = Geyser.Fixed }, ui.box)
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
      name = id("reading." .. spec.key), height = 20, v_policy = Geyser.Fixed,
    }, ui.box)
    transparent(reading)
    reading.label = spec.label
    ui.readings[spec.key] = reading

    local gauge = Geyser.Gauge:new({
      name = id("gauge." .. spec.key), height = 8, v_policy = Geyser.Fixed,
    }, ui.box)
    gauge.front:setStyleSheet(string.format("background-color: %s; border: none;", spec.colour))
    gauge.back:setStyleSheet(string.format(
      "background-color: %s; border: 1px solid %s;", ui.palette.paper, ui.palette.ink))
    gauge.text:setStyleSheet("background-color: transparent;")
    gauge:setValue(0, 1, "")
    ui.gauges[spec.key] = gauge
  end

  eyebrow(id("mapLabel"), "MAP")
  ui.map = Geyser.Mapper:new({ name = id("map"), v_stretch_factor = 3 }, ui.box)

  eyebrow(id("chatLabel"), "CHAT")
  ui.chat = Geyser.MiniConsole:new({
    name = id("chat"),
    v_stretch_factor = 2,
    color = ui.palette.canvas,
    fontSize = 10,
    autoWrap = true,
    scrollBar = true,
  }, ui.box)

  -- The profile has one map widget, made when the first dock was built.
  -- Placing it doesn't raise it, and a dock built later (after an upgrade)
  -- is drawn over it, so bring it back to the top.
  ui.map:raise()
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
