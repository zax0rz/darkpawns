-- Dark Pawns for Mudlet: the "dp" command.
--
-- The package adds no game commands and no gameplay triggers: Dark Pawns
-- already accepts n/e/s/w/u/d and command abbreviations, and a client alias
-- that shadowed one would change what the game hears. "dp" only manages the
-- package itself.

function DarkPawns.command(argument)
  argument = (argument or ""):lower()
  if argument == "hide" then
    DarkPawns.ui.hide()
  elseif argument == "show" then
    DarkPawns.ui.build()
  elseif argument == "map" then
    DarkPawns.map.download()
  elseif argument == "clear" then
    if DarkPawns.ui.chat then
      DarkPawns.ui.chat:clear()
    end
  else
    cecho(string.format("\n<ansi_white>[ Dark Pawns %s ]<reset>\n", DarkPawns.version))
    cecho("  <yellow>dp hide<reset>   - hide the dock\n")
    cecho("  <yellow>dp show<reset>   - bring it back\n")
    cecho("  <yellow>dp clear<reset>  - clear the chat window\n")
    cecho("  <yellow>dp map<reset>    - load the world map (replaces this profile's map)\n")
    cecho("  Double-click a room on the map to walk there.\n")
  end
end

-- Uninstalling the package removes its scripts but not the windows and
-- handlers they made at runtime; take those down with it.
DarkPawns.on("uninstall", "sysUninstallPackage", function(_, name)
  if name ~= DarkPawns.packageName then
    return
  end
  DarkPawns.ui.hide()
  DarkPawns.offAll()
end)
