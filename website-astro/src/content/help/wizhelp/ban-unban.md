---
title: "BAN UNBAN"
description: "Usage: ban [<all | new | select> <site>]"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/ban-unban']
---

Usage: ban [&lt;all | new | select&gt; &lt;site&gt;]
       unban &lt;site&gt;

These commands prevent anyone from a site with a hostname containing the
site substring from logging in to the game.  You may ban a site to ALL, [NEW](/database#mob-16302)
or SELECT players.  Banning a site to [NEW](/database#mob-16302) players prevents any new players
from registering.  Banning a site to ALL players disallows ANY connections
from that site.  Banning a site SELECTively allows only players with site-ok
flags to log in from that site.  Ban with no argument returns a list of
currently banned sites.

Unban removes the ban.

Examples:

  > ban all whitehouse.gov
  > unban ai.mit.edu

See also: [WIZLOCK](/help/wizhelp/wizlock/)
wizonly