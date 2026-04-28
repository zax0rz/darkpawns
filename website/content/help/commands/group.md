---
title: "GROUP"
description: "Usage: group [player]"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/group']
---

Usage: group [player]
       group all

GROUP with no arguments shows the status of your group.  GROUP <player>
adds or removes <player> from your group.  GROUP ALL adds everyone who is
following you to your group.

If you are a group member, then you will share experience points for killing
monsters equally among the group.  Experience points will only be shared among
group members that are in the same location, at the time of the killing
blow.  Sharing is regardless of whether the group members have been
participating in the fight.

Group with no argument shows a list of people with whom you are grouped.
"group all" groups all people who are following you.

To make a group, a "leader" must be selected.  All members of the group
must then "follow" the leader using the follow command.  The leader should
NOT follow anyone at all!  The leader can now toggle group membership for
each player in the group, by using "group <player_name>".  The leader
should remember to include himself in the group.

Example:

  Quifael and Redferne wishes to group with Dragon.

  Quifael : "follow dragon"
  Redferne: "follow dragon"
  (Dragon should be following no one but himself)
  Dragon  : "group dragon"
  Dragon  : "group redferne"
  Dragon  : "group quifael"

Later when Dragon get upset at Quifael (because he is sleeping all the time)
he types : "Group Quifael" to kick Quifael out of the group.

See also: EXPERIENCE, UNGROUP, XP