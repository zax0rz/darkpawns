# Docs pass 2: fresh-reader review

I read the revised pages as a new player, operator, or contributor landing from search. I reviewed Play, Host, and Project, plus the completed Build pages. I did not edit those pages.

## Play

**Purpose and audience:** `Your first hour` is a route for a brand-new character; `Connect a client` helps players keep Dark Pawns' intended three-character party connected across common clients. Both answer their audience quickly and link to relevant next steps.

**Search phrases:** “what to do after creating a Dark Pawns character”; “connect three characters to a MUD with TinTin++ or Mudlet.”

Reviewed `Play`, `Your first hour`, and `Connect a client`.

**Actionable issue:** `Your first hour` says a character can start in any hometown and advises using a local newbie area, then gives a Kir Drax'in-only sequence through its guilds and Guard Barracks. Mark the detailed route as conditional (“If your hometown is Kir Drax'in”) before its first step and state plainly that other hometowns need their own guild and training route. The opening and “Before you start” currently make the Kir Drax'in assumption easy to miss.

`Connect a client` is clear that a new player can begin with one character and add sessions later. Client-specific actions are concrete, and its telnet warning is easy to find. No material blocker found.

## Host

**Purpose and audience:** The overview sets hosting expectations; Install gets a new operator to a running instance; First login secures initial access; the remaining pages cover admin access, configuration, dashboards/logs, notifications, player records, welcome messages, updates, and connectivity failures.

**Search phrases:** “Dark Pawns server requirements and hosting cost”; “install Dark Pawns on Windows”; “first login admin password”; “change Dark Pawns telnet port”; “Discord notifications”; “players cannot connect to my MUD.”

Reviewed `Host`, `Hosting at a glance`, `Install Dark Pawns`, `Your first login`, `Admin console`, `Configuration settings`, `Dashboard and logs`, `Notifications`, `Player management`, `Edit the welcome messages`, `Update your server`, and `Players can't connect`.

**Actionable issues:**

- `Hosting at a glance` separates measured startup from planning estimates well. The “setup time” section still gives no setup-duration estimate; 0.42 seconds is process startup only. Either report a measured install-to-ready duration or say that duration has not been measured and point readers to the one-command local setup versus the longer public-networking path.
- `Install Dark Pawns` lists Windows as supported, but its first route is Linux/macOS and the Windows signed installer appears later under “Finish setup.” Put a visible Windows route near the opening so a Windows operator can choose it without scanning the whole page. The Docker method uses `--network host`; explain its platform prerequisites and Docker Desktop host-networking setting near that command, since the method is not uniformly available by default across the listed systems. The [Docker host network documentation](https://docs.docker.com/engine/network/drivers/host/) describes those constraints.
- `Notifications` and `Dashboard and logs` contain useful specifics, but dense operator vocabulary (for example, WHOD/HIGOD/GRGOD, `mudlog`, and file permissions) slows a first-time operator. Link first-use terms to a short glossary or add a brief plain-language definition where each term first appears. Keep implementation detail under the existing “How it works” treatment where practical.
- `Run a local server` says “Go at the version declared in go.mod,” which reads like a direction to visit a URL rather than install Go. Say “Install the Go version declared in `go.mod`.” Its alternate database example writes under `/var/lib/darkpawns`; tell a developer to create and grant access to that directory, or use a writable path, before using it.

The remaining Host pages have clear task titles and expected outcomes. `First login` explains why the initial account is privileged; the admin console maps pages to access levels; configuration, player management, MOTD, updates, and connection troubleshooting each provide a recognizable task path. The console page could define its first staff-level abbreviations inline, but the access table makes the practical gates discoverable.

## Project

**Purpose and audience:** The Project index routes developers to architecture, contributing rules, and research, while preserving project identity and history. Architecture and Port Fidelity explain the code and verification model; Contributing gives the practical rules for changes.

**Search phrases:** “Dark Pawns source architecture”; “how to contribute to Dark Pawns”; “how to verify the C to Go port.”

Reviewed `Project`, `Server architecture`, `Contribute to Dark Pawns`, and `Port fidelity workflow`.

**Actionable issue:** `Contribute to Dark Pawns` says “all four required checks” but its command block contains `make fmt`, build, vet, test, lint, and deadcode-ratchet. Separate the four required checks from formatting and the deadcode ratchet, or change the count and identify which commands are gates. As written, readers cannot tell which four the sentence means.

The index and Port Fidelity route are otherwise direct and make the read-only oracle boundary clear.

## Build

**Purpose and audience:** Build is for developers editing world prototypes or running a disposable local server. The edit pages explain the separation between live draft changes and saving zone files; the Lua page correctly states that script edits have no draft/undo protection.

**Search phrases:** “edit a room/object/mob in Dark Pawns”; “save zone edits”; “run a local development server.”

Reviewed `Build`, `Run a local server`, `Edit rooms`, `Edit objects`, `Edit mobs`, `Edit zones`, and `Edit Lua scripts`.

**Actionable issues:** The local-server wording and alternate database directory permission issue are listed under Host because they appear on that shared Build page. The Build index is sparse but its task links are clear. Edit pages consistently distinguish committing a draft to the running world from saving the underlying zone data, which is the distinction a new builder needs.
