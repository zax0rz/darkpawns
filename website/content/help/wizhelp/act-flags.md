---
title: "'ACT FLAGS'"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/act-flags']
---

Each control character is preceded by a '$'.

$n - Write name, short description, or "someone", for ch, depending on whether 
	ch is a PC, a NPC, or an invisible PC/NPC.
$N - Like $n, except insert the text for victim
$m - "him", "her", or "it", depending on the gender of ch.
$M - Like $m, for vict.
$s - "his", "her", or "it", depending on the gender of ch.
$S - Like $s, for vict.
$e - "he", "she", "it", depending on the gender of ch.
$E - Like $e, for vict.
$o - Name or "something" for obj, depending on visibility.
$O - Like $o, for vict_obj. i.e. A wand of invis pointed at a sword; sword is $O
$p - Short description or "something" for obj.
$P - Like $p for vict_obj.
$a - "an" or "a", depending on the first character of obj's name.
$A - Like $a, for vict_obj.
$T - Prints the string pointed to by vict_obj.
$F - Processes the string pointed to by vict_obj with fname() prior to printing.
$$ - Print the character '$'.


Example:
Drakos is is holding a wand of invis and uses it on Xira.
$n points $p at $N, and $E vanishes!
would print:
Drakos points a wand of invis at Xira, and she vanishes!

wizonly