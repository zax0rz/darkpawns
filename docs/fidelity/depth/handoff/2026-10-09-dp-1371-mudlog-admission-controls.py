#!/usr/bin/env python3
"""R5h controls for the DP-1371 PR 2 admission sites.

Each case disables exactly its producer, or restores the pre-fix host-string
behaviour, requires the named test's assertion failure -- never a build
failure -- then restores and re-proves green.

    python3 docs/fidelity/depth/handoff/2026-10-09-dp-1371-mudlog-admission-controls.py \
        --output /absolute/evidence/path
"""
import argparse
import pathlib
import subprocess

LISTENER = "pkg/telnet/listener.go"
BANS = "pkg/game/bans.go"
CLOSE = "pkg/session/close_descriptor.go"
MUD_HOST = "pkg/session/mud_host.go"

# comm.c:1552-1555. Removing the producer leaves the function compiling.
DNS_FAILURE_LOG = "\tgame.MudLog(fmt.Sprintf(\"DNS lookup failed on %s.\", padded), game.MudlogComplete, game.LVL_GOD, true) // comm.c:1554\n"

# comm.c:1527-1529: with the nameserver slow C never resolves. Forcing the
# resolved branch in identifyConnection is the host-string switch this control
# disables (the accept loop's copy of the gate is three tabs deep; this is the
# one-tab gate inside identifyConnection itself).
SLOW_GATE_OLD = "\tif nameserverSlow {\n\t\t// C's wildhost and double_wild are both defined on this branch\n"
SLOW_GATE_NEW = "\tif false {\n\t\t// C's wildhost and double_wild are both defined on this branch\n"

# comm.c:1572: the refusal bytes.
REFUSAL_BYTES_OLD = "\tif !isTLSConn(conn) {\n\t\t_, _ = conn.Write([]byte(\"Sorry, your site is banned.\\r\\n\"))\n\t}\n"
REFUSAL_BYTES_NEW = "\tif false {\n\t\t_, _ = conn.Write([]byte(\"Sorry, your site is banned.\\r\\n\"))\n\t}\n"

# comm.c:1573-1574: the refusal producer.
REFUSAL_LOG = "\tgame.MudLog(fmt.Sprintf(\"Connection attempt denied from [%s]\", host), game.MudlogComplete, game.LVL_GOD, true)\n"

# ban.c:205-209.
BAN_LOG = "\tMudLog(fmt.Sprintf(\"%s has banned %s for %s players.\", actorName, rawSite, banTypeName(banType)),\n\t\tMudlogNormal, max(LVL_GOD, invis), true)\n"

# ban.c:237-243: the acknowledgement reaches the actor before the producer.
UNBAN_ORDER_OLD = "\tack(\"Site unbanned.\\r\\n\")\n\tMudLog(fmt.Sprintf(\"%s removed the %s-player ban on %s.\", actorName, banTypeName(removed.BanType), removed.Site),\n\t\tMudlogNormal, max(LVL_GOD, invis), true)\n"
UNBAN_ORDER_NEW = "\tMudLog(fmt.Sprintf(\"%s removed the %s-player ban on %s.\", actorName, banTypeName(removed.BanType), removed.Site),\n\t\tMudlogNormal, max(LVL_GOD, invis), true)\n\tack(\"Site unbanned.\\r\\n\")\n"

# comm.c:2142, then comm.c:2137-2138.
DESCRIPTOR_LOG = "\t\t\tgame.MudLog(\"Losing descriptor without char.\", game.MudlogComplete, game.LVL_IMMORT, true)\n\t\t\treturn\n"

PLAYER_LOG = "\t\tgame.MudLog(fmt.Sprintf(\"Losing player: %s.\", s.descriptorCharacterName()), game.MudlogComplete, game.LVL_IMMORT, true)\n"

# comm.c:1536-1539: the zero-padded quad.
PADDING_OLD = "\t\treturn fmt.Sprintf(\"%03d.%03d.%03d.%03d\", ip[0], ip[1], ip[2], ip[3])\n"
PADDING_NEW = "\t\treturn fmt.Sprintf(\"%d.%d.%d.%d\", ip[0], ip[1], ip[2], ip[3])\n"

CASES = [
    {
        "case": "dns-failure-log",
        "test": "TestIdentifyConnectionFailedLookupLogs",
        "package": "./pkg/telnet",
        "assertion": "producer bytes",
        "patches": [{"path": LISTENER, "new": "", "old": DNS_FAILURE_LOG}],
    },
    {
        "case": "dns-slow-gate",
        "test": "TestIdentifyConnectionSlowPerformsNoLookup",
        "package": "./pkg/telnet",
        "assertion": "a reverse lookup ran with nameserver_is_slow set",
        "patches": [{"path": LISTENER, "new": SLOW_GATE_NEW, "old": SLOW_GATE_OLD}],
    },
    {
        "case": "refusal-bytes",
        "test": "TestBannedConnectionRefusalBytesAndLog",
        "package": "./pkg/telnet",
        "assertion": "refusal bytes",
        "patches": [{"path": LISTENER, "new": REFUSAL_BYTES_NEW, "old": REFUSAL_BYTES_OLD}],
    },
    {
        "case": "refusal-log",
        "test": "TestBannedConnectionRefusalBytesAndLog",
        "package": "./pkg/telnet",
        "assertion": "producer = []",
        "patches": [{"path": LISTENER, "new": "\t_ = host\n", "old": REFUSAL_LOG}],
    },
    {
        "case": "ban-producer",
        "test": "TestBanMudlogProducerBytesAndOrder",
        "package": "./pkg/session",
        "assertion": "ban actor stream",
        "patches": [{"path": BANS, "new": "\t_ = rawSite\n", "old": BAN_LOG}],
    },
    {
        "case": "unban-order",
        "test": "TestBanMudlogProducerBytesAndOrder",
        "package": "./pkg/session",
        "assertion": "unban actor stream",
        "patches": [{"path": BANS, "new": UNBAN_ORDER_NEW, "old": UNBAN_ORDER_OLD}],
    },
    {
        "case": "losing-descriptor",
        "test": "TestLoseDescriptorWithoutChar",
        "package": "./pkg/session",
        "assertion": "observer = \"\"",
        "patches": [{"path": CLOSE, "new": "\t\t\treturn\n", "old": DESCRIPTOR_LOG}],
    },
    {
        "case": "losing-player",
        "test": "TestLosePlayerOnMenuExit",
        "package": "./pkg/session",
        "assertion": "observer = \"\"",
        "patches": [{
            "path": CLOSE,
            "new": "\t\t_ = fmt.Sprintf(\"Losing player: %s.\", s.descriptorCharacterName())\n",
            "old": PLAYER_LOG,
        }],
    },
    {
        "case": "host-padding",
        "test": "TestMudHostPrefersResolvedName",
        "package": "./pkg/session",
        "assertion": "MudHost() = \"192.0.2.10\"",
        "patches": [{"path": MUD_HOST, "new": PADDING_NEW, "old": PADDING_OLD}],
    },
]


def run(case, stage, root):
    result = subprocess.run(
        ["go", "test", case["package"], "-run", "^" + case["test"] + "$", "-count=1"],
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
    )
    (root / f"{case['case']}-{stage}.txt").write_text(result.stdout + "\nEXIT=" + str(result.returncode) + "\n")
    assert "[build failed]" not in result.stdout, result.stdout
    if stage == "revert":
        assert result.returncode != 0, result.stdout
        assert "--- FAIL: " + case["test"] in result.stdout, result.stdout
        assert case["assertion"] in result.stdout, result.stdout
    else:
        assert result.returncode == 0, result.stdout


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True)
    root = pathlib.Path(parser.parse_args().output)
    root.mkdir(parents=True, exist_ok=True)
    for case in CASES:
        originals = {p["path"]: pathlib.Path(p["path"]).read_text() for p in case["patches"]}
        reverted = dict(originals)
        for patch in case["patches"]:
            assert reverted[patch["path"]].count(patch["old"]) == 1, (case["case"], patch["path"])
            reverted[patch["path"]] = reverted[patch["path"]].replace(patch["old"], patch["new"])
        try:
            for stage, sources in (("green", originals), ("revert", reverted), ("restore", originals)):
                for path, source in sources.items():
                    pathlib.Path(path).write_text(source)
                run(case, stage, root)
            print(case["case"] + ": 1 -> 0 -> 1", flush=True)
        finally:
            for path, source in originals.items():
                pathlib.Path(path).write_text(source)


if __name__ == "__main__":
    main()
