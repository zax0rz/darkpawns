# A tolerant normalizer hid a date bug

*2026-11-21, Claude. Ledger PF-151.*

GLM's L1 finish reported that `Current machine time: %s` (`src/act.wizard.c:1818`) can never be witnessed in a census dump, because the dump normalizer replaces wall-clock text. That is true. But reading the Go side of the same line showed something the census had no way to see.

C formats the time with `asctime()`, whose day of the month is `%3d`. On the 1st through the 9th it prints two spaces after the month: `Thu Nov  5 07:08:09 2026`. Go used the layout `"Mon Jan 2 15:04:05 2006"`, which prints one space. The same layout was in three places: `date`, `uptime`, and the mail reader's `Date:` line (`src/mail.c:429`). Two other Go sites (`stat`, boards) already used `_2` correctly.

The normalizer's wall-clock pattern (`internal/oraclediff/normalize.go:25`) matches the day with ` +\d{1,2}`, so it accepts one space or many. Both engines' lines collapse to `<WALL_CLOCK>`, and the difference disappears before any comparison. No census, on any day of the month, could have failed on this.

This is the second time a normalizer's tolerance has hidden a real byte bug. The first was the trailing-space trim (PF-122), which hid 31 divergences. The pattern is the same: a normalizer written to remove noise also removes the signal next to it. The tighter rule is that a normalizer should accept only the exact shape C produces (here `(?: \d|[1-3]\d)` for the day), so a Go string with the wrong shape stays visible as a diff.

The fix (`common.Asctime`, pinned by a unit test that fails on the old layout) is in the same change as this note. Tightening the normalizer is census tooling, so it is Zach's decision.
