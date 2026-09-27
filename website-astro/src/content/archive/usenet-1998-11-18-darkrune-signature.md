---
title: "Who owns a character?"
description: "A post by Stephen C. Thompson in a MUD administration thread carries a contemporary Dark Pawns signature for mud.darkrune.org."
kind: "usenet-thread"
sortDate: 1998-11-18
dateLabel: "November 18, 1998"
publishedAt: 1998-11-18
sourceSite: "Usenet"
sourceUrl: "https://usenetarchives.com/view.php?id=rec.games.mud.admin&mid=PDM2MzlEQkYyLjgzNzFEMzVAYWx1bW5pLnByaW5jZXRvbi5lZHU%2B"
captureUrl: "https://usenetarchives.com/view.php?id=rec.games.mud.admin&mid=PDM2MzlEQkYyLjgzNzFEMzVAYWx1bW5pLnByaW5jZXRvbi5lZHU%2B"
recoveredAt: 2026-09-25
textKind: "edited-excerpt"
source: "UsenetArchives capture identified by captureUrl and Message-ID"
voiceLayer: "frontline"
board: "rec.games.mud.admin"
messageId: "<3639DBF2.8371D35@alumni.princeton.edu>"
postCount: 1
completeness: "partial"
completenessNote: "Only post 31 is reproduced. The 147-post thread concerned character ownership and linked-list algorithms; this post is retained for its contemporary Dark Pawns signature."
participants:
  - name: "stevet"
    role: "unknown"
    posts: 1
draft: false
---

*Transcript note: recovered from UsenetArchives. Email addresses and personal contact URLs are redacted; spelling, punctuation and the remaining text are preserved.*

### stevet — Nov 18, 1998 19:00 UTC

<pre class="usenet-transcript">Jon A. Lambert  wrote:
›
▶

On 18 Nov 1998 19:08:27 GMT, J C Lawrence said:
(8 lines)
› On 18 Nov 1998 19:08:27 GMT, J C Lawrence said:
››
››  Given a singly linked list of objects and a pointer to the first
›› node on the list, what is the most efficient (time __and__
›› resources) algorithm you can come up with to detect if the list is
›› circular?
››

› Hmmm, how about walking the list copying the pointers into an STL set
› container.  As soon as you hit a &quot;bad&quot; insert, you have the answer.
› I think it&#x27;s fairly inexpensive.  :)

That isn&#x27;t bad.  But it seems excessive if all you are doing is checking to
see if it is circular.  I was actually asked this question once, and came up
with this solution:

Take pointer A and move it two slots down the list, take pointer B and move it
one slot.  If pointer A is ever equal to pointer B (on either of the two slots
in each move) then the list loops, if not then it doesn&#x27;t.  n time right?  And
only a few pointer assignments and 1 comparison per node.

-Steve

Signature
  Stephen C. Thompson
  [email redacted]

[personal URL redacted]
  Dark Pawns:  mud.darkrune.org 4000</pre>
