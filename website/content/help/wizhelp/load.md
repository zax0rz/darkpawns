---
title: "LOAD"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/load']
---

Usage: load <mob | obj> <virtual number>
       load <mob | obj> <name>

LOAD is used to create mobiles and objects.  The first argument specifies if
you are trying to load a mobile or an object; the second is the virtual
number or name.

Example:

  > load obj 8099
  You create a bulletin board.

  > load mob 8005
  You create the receptionist.

  > load mob warg
  You create the hybrid warg.

  > load obj ring
  You're not godly enough for that!  (This is because load-by-name attempts
	to load lowest-vnumbered object with that name, and Tracer's ring
	is above your load limit.)

See also: VNUM
wizonly