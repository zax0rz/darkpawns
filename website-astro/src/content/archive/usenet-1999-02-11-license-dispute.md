---
title: "[AD] [custom] Dark Pawns"
description: "A Dark Pawns advertisement becomes a seventeen-post public dispute over DikuMUD credits, source lineage and license compliance."
kind: "usenet-thread"
sortDate: 1999-02-11
dateLabel: "February 11-14, 1999"
publishedAt: 1999-02-11
sourceSite: "Usenet"
sourceUrl: "https://usenetarchives.com/view.php?id=rec.games.mud.diku&mid=PDdhMWZpMCQ3ZmUkMUBubnJwMS5kZWphbmV3cy5jb20%2B"
captureUrl: "https://usenetarchives.com/view.php?id=rec.games.mud.diku&mid=PDdhMWZpMCQ3ZmUkMUBubnJwMS5kZWphbmV3cy5jb20%2B"
recoveredAt: 2026-09-25
textKind: "verbatim"
source: "UsenetArchives capture identified by captureUrl and Message-ID"
voiceLayer: "frontline"
board: "rec.games.mud.diku"
messageId: "<7a1fi0$7fe$1@nnrp1.dejanews.com>"
postCount: 17
completeness: "complete"
contentWarning: "This thread contains hostile accusations and period-typical insults. It is preserved as a disputed public record, not as an editorial verdict."
participants:
  - name: "rparet"
    role: "unknown"
    posts: 4
  - name: "kavir"
    role: "unknown"
    posts: 7
  - name: "mspeck"
    role: "unknown"
    posts: 2
  - name: "adam"
    role: "unknown"
    posts: 1
  - name: "eryi"
    role: "unknown"
    posts: 1
  - name: "sommer"
    role: "unknown"
    posts: 1
  - name: "j_herlih"
    role: "unknown"
    posts: 1
draft: false
---

*Transcript note: recovered from UsenetArchives. Email addresses and personal contact URLs are redacted; spelling, punctuation and the remaining text are preserved.*

### rparet — Feb 11, 1999 19:00 UTC

<pre class="usenet-transcript">D A R K      P A W N S

                          pawns.darkrune.org 4300
                           207.229.21.130  4300
                    WHOD daemon: pawns.darkrune.org 4301
                    WWW: http://pawns.darkrune.org

    Dark Pawns is a completely original Dark Fantasy Multi User
    Environment that has been constantly played and developed for
    over five years. We believe that Dark Pawns offers one of the
    most unique gaming environments on the Internet.

    We have just released Dark Pawns version 2.2, the biggest source
    update in our five year history.

        A Partial List of Features:

        o  No Rent
        o  Vampirism, Lycanthropy, and other diseases for PCs.
        o  Seven different PC races, which languages for each
        o  Six initial PC classes, along with a completely new
           remort system and six remort classes.
        o  Unique skills and spells for each class
        o  Mounts for traversing our massive game world, spread
           across two continents
        o  In the spirit of games such as Final Fantasy and Ultima
           Dark Pawns encourages players to use a party of up to
           three multis.
        o  Dark Pawns strives to be a realistic environment,
           therefore PK is allowed, but carries certain penalties
           with it.
        o  Magical Tattoos
        o  An Interactive Arena system
        o  A Unique Behavioral AI for mobiles
        o  100% DP Original areas in a dark fantasy setting
        o  No cheesy ASCII art in our News Group ad

        Even though 2.2 has just been released, we are already working
        on the next incredible expansion of Dark Pawns.
        So, if you are tired of playing StockMUD then come try out
        someplace different. Come play Dark Pawns.


-----------== Posted via Deja News, The Discussion Network ==----------
http://www.dejanews.com/       Search, Read, Discuss, or Start Your Own</pre>

### kavir — Feb 11, 1999 19:00 UTC

<pre class="usenet-transcript">[email redacted] wrote:
›

› [snip]

› Dark Pawns is a completely original Dark Fantasy Multi User

So why does it say &quot;Based on CircleMUD 3.0, created by Jeremy Elson&quot;
on the login screen?

› Environment that has been constantly played and developed for
› over five years.

I didn&#x27;t realise Circlemud 3.0 patch 11 was that old.  Being curious,
I downloaded a copy of Circlemud 3.0 patch 11 (which is the version
your helpfiles say you are using) and noticed that the files were
date-stamped 1996.  Of course, someone could have touched them since
the release date, so I did a grep for &quot;96&quot; and came up with the
following:

interpreter.c:     * re-add the code to cut off duplicates when a player quits.
 JE 6 Feb 96
modify.c:* for CircleMUD.  All functions below are his.  --JE 8 Mar 96
random.c: * supported on the target system.  -JE 2/3/96
spell_parser.c: * or skill, look in class.c.  -JE 5 Feb 1996
sysdep.h: * 20 Mar 96: My quest is not yet over.  These definitions still cause

›
▶

We believe that Dark Pawns offers one of the
(5 lines)
› We believe that Dark Pawns offers one of the
› most unique gaming environments on the Internet.

› Forgive me if I show a little hesitation to believe you at this point.

›     We have just released Dark Pawns version 2.2, the biggest source
›     update in our five year history.

You mean &quot;almost three year history&quot;.

While we&#x27;re on the topic of honesty, I&#x27;d also like to draw your attention
to the license agreement, as you&#x27;ve broken every single request listed in
the second section (about giving credit) - except for two parts which I
cannot prove (distributing the doc files with your code and keeping the
headers intact).  Specifically, you&#x27;ve broken the following rules:

-- The text in the &#x27;credits&#x27; file distributed with CircleMUD must be
   preserved.  You may add your own credits to the file, but the existing
   text must not be removed, abridged, truncated, or changed in any way.
   This file must be displayed when the &#x27;credits&#x27; command is used from
   within the MUD.

You&#x27;ve replaced it with your own.

-- The &quot;CIRCLEMUD&quot; help entry must be maintained intact and unchanged, and
   displayed in its entirety when the &#x27;help circlemud&#x27; command is used.

You&#x27;ve replaced it with your own.

-- The login sequence must contain the names of the DikuMUD and CircleMUD
   creators.  The &#x27;login sequence&#x27; is defined as the text seen by players
   between the time they connect to the MUD and when they start to play
   the game itself.

You don&#x27;t credit the Diku team (which is also against the third section of
the Circle license: you must comply with the DikuMUD license).

-- Claims that any of the above requirements are inapplicable to a particular
   MUD for reasons such as &quot;our MUD is totally rewritten&quot; or similar are
   completely invalid.  If you can write a MUD completely from scratch then
   you are encouraged to do so by all means, but use of any part of the
   CircleMUD or DikuMUD source code requires that their respective licenses
   be followed, including the crediting requirements.

That speaks for itself.  Did you read the part where Jeremy Elson wrote:

   A great deal of work went into the creation of CircleMUD, and it was given
   to you completely free of charge; claiming that you wrote the MUD yourself
   is a slap in the face to everyone who worked to bring you a high quality
   product while asking for nothing but credit for their work in return.

You should have done.

   The first time you try to compile Circle, you will be asked to read the
   CircleMUD license.  Please read it!

I assume you realise that because of people like you, people like me are
not willing to release their source code?

KaVir.</pre>

### rparet — Feb 11, 1999 19:00 UTC

<pre class="usenet-transcript">In article [email redacted]
  Richard Woolcock  wrote:
›
▶

[email redacted] wrote:
(5 lines)
› [email redacted] wrote:
››
›
› [snip]
›
›› Dark Pawns is a completely original Dark Fantasy Multi User
›
› So why does it say &quot;Based on CircleMUD 3.0, created by Jeremy Elson&quot;
› on the login screen?

It says that because I would like to give Jeremy Elson credit because I
and the developers before me used his Circle mud 3.0 pl11 code as a model
in enhancing our now almost six year old Diku code.  In the course of your
reply, you seem to indicate that I personally am an unethical individual
because you feel that I took stock circle, modified it, and now claim it
as my own 100%.  This is simply not true.

›
▶

Environment that has been constantly played and developed for
(16 lines)
›
›› Environment that has been constantly played and developed for
›› over five years.
›
› I didn&#x27;t realise Circlemud 3.0 patch 11 was that old.  Being curious,
› I downloaded a copy of Circlemud 3.0 patch 11 (which is the version
› your helpfiles say you are using) and noticed that the files were
› date-stamped 1996.  Of course, someone could have touched them since
› the release date, so I did a grep for &quot;96&quot; and came up with the
› following:
›
› interpreter.c:     * re-add the code to cut off duplicates when a player
› quits.
›  JE 6 Feb 96
› modify.c:* for CircleMUD.  All functions below are his.  --JE 8 Mar 96
› random.c: * supported on the target system.  -JE 2/3/96
› spell_parser.c: * or skill, look in class.c.  -JE 5 Feb 1996
› sysdep.h: * 20 Mar 96: My quest is not yet over.  These definitions still
› cause
›

Again, the circle code was only used as a model to enhance the existing
diku code. If a musical artist borrows a bass line from another artist
for his song, it is only right that the first artist is given credit.
HOWEVER, this doesn&#x27;t mean that the song is owned by the first artist, and
because credit is given it doesn&#x27;t constitute intellectual property theft.



›
▶

We believe that Dark Pawns offers one of the
(42 lines)
›› We believe that Dark Pawns offers one of the
›› most unique gaming environments on the Internet.
›
› Forgive me if I show a little hesitation to believe you at this point.
›

› Understandable, I hope this message clears things up for you a bit.


››     We have just released Dark Pawns version 2.2, the biggest source
››     update in our five year history.
›
› You mean &quot;almost three year history&quot;.
›

› Almost six year history, actually. But we&#x27;ve been through this.


› While we&#x27;re on the topic of honesty, I&#x27;d also like to draw your attention
› to the license agreement, as you&#x27;ve broken every single request listed in
› the second section (about giving credit) - except for two parts which I
› cannot prove (distributing the doc files with your code and keeping the
› headers intact).  Specifically, you&#x27;ve broken the following rules:
›
› -- The text in the &#x27;credits&#x27; file distributed with CircleMUD must be
›    preserved.  You may add your own credits to the file, but the existing
›    text must not be removed, abridged, truncated, or changed in any way.
›    This file must be displayed when the &#x27;credits&#x27; command is used from
›    within the MUD.
›
› You&#x27;ve replaced it with your own.
›
› -- The &quot;CIRCLEMUD&quot; help entry must be maintained intact and unchanged, and
›    displayed in its entirety when the &#x27;help circlemud&#x27; command is used.
›
› You&#x27;ve replaced it with your own.
›
› -- The login sequence must contain the names of the DikuMUD and CircleMUD
›    creators.  The &#x27;login sequence&#x27; is defined as the text seen by players
›    between the time they connect to the MUD and when they start to play
›    the game itself.
›
› You don&#x27;t credit the Diku team (which is also against the third section of
› the Circle license: you must comply with the DikuMUD license).
›
› -- Claims that any of the above requirements are inapplicable to a particular
›    MUD for reasons such as &quot;our MUD is totally rewritten&quot; or similar are
›    completely invalid.  If you can write a MUD completely from scratch then
›    you are encouraged to do so by all means, but use of any part of the
›    CircleMUD or DikuMUD source code requires that their respective licenses
›    be followed, including the crediting requirements.
›
› That speaks for itself.  Did you read the part where Jeremy Elson wrote:
›
›    A great deal of work went into the creation of CircleMUD, and it was given
›    to you completely free of charge; claiming that you wrote the MUD yourself
›    is a slap in the face to everyone who worked to bring you a high quality
›    product while asking for nothing but credit for their work in return.
›
› You should have done.
›
›    The first time you try to compile Circle, you will be asked to read the
›    CircleMUD license.  Please read it!
›


Now we are speaking strictly of personal opinion here, so don&#x27;t get hot, this
is just how I feel:  I feel that I am not bound by the circlemud licence
because my mud is not derived from circle directly.  I am extremely grateful
to Jeremy for releasing the code to circle, which has helped myself and Dark
Pawns greatly, there is no dispute about that, but I would not say that the
mud wasn&#x27;t custom unless it wasn&#x27;t.  The mud has been developed by talented
people for almost six years, I myself being fortunate enough to develop it
for the last three, and I have put a considerable amount of time and effort
into making it the most unique ud that I could.


› I assume you realise that because of people like you, people like me are
› not willing to release their source code?

I understand that there are people like that, and I&#x27;d like you to understand
that I&#x27;m not one of them. I particularly understand that you have been burned
before, with the release of your stolen GodWars source and are no doubt still
bitter about it, which I agree with wholeheartedly. I&#x27;m just trying to say
that you don&#x27;t have all the facts, and you are attacking the wrong person. I
do my best to give everyone the credit they deserve, and you are right,
Jeremy Elson does deserve his credit, which he gets in my opening screen and
in the credits file.

I hope I cleared things up for you.


Rich



-----------== Posted via Deja News, The Discussion Network ==----------
http://www.dejanews.com/       Search, Read, Discuss, or Start Your Own</pre>

### mspeck — Feb 11, 1999 19:00 UTC

<pre class="usenet-transcript">[email redacted] wrote:

›
▶

In article &lt;[email redacted]&gt;,
(97 lines)
› In article [email redacted]
›   Richard Woolcock
 wrote:
›› [email redacted] wrote:
›››
››
›› [snip]
››
››› Dark Pawns is a completely original Dark Fantasy Multi User
››
›› So why does it say &quot;Based on CircleMUD 3.0, created by Jeremy Elson&quot;
›› on the login screen?
›
› It says that because I would like to give Jeremy Elson credit because I
› and the developers before me used his Circle mud 3.0 pl11 code as a model
› in enhancing our now almost six year old Diku code.  In the course of your
› reply, you seem to indicate that I personally am an unethical individual
› because you feel that I took stock circle, modified it, and now claim it
› as my own 100%.  This is simply not true.
›
››
››› Environment that has been constantly played and developed for
››› over five years.
››
›› I didn&#x27;t realise Circlemud 3.0 patch 11 was that old.  Being curious,
›› I downloaded a copy of Circlemud 3.0 patch 11 (which is the version
›› your helpfiles say you are using) and noticed that the files were
›› date-stamped 1996.  Of course, someone could have touched them since
›› the release date, so I did a grep for &quot;96&quot; and came up with the
›› following:
››
›› interpreter.c:     * re-add the code to cut off duplicates when a player
› quits.
››  JE 6 Feb 96
›› modify.c:* for CircleMUD.  All functions below are his.  --JE 8 Mar 96
›› random.c: * supported on the target system.  -JE 2/3/96
›› spell_parser.c: * or skill, look in class.c.  -JE 5 Feb 1996
›› sysdep.h: * 20 Mar 96: My quest is not yet over.  These definitions still
› cause
››
›
› Again, the circle code was only used as a model to enhance the existing
› diku code. If a musical artist borrows a bass line from another artist
› for his song, it is only right that the first artist is given credit.
› HOWEVER, this doesn&#x27;t mean that the song is owned by the first artist, and
› because credit is given it doesn&#x27;t constitute intellectual property theft.
›
››› We believe that Dark Pawns offers one of the
››› most unique gaming environments on the Internet.
››
›› Forgive me if I show a little hesitation to believe you at this point.
››
›
› Understandable, I hope this message clears things up for you a bit.
›
›››     We have just released Dark Pawns version 2.2, the biggest source
›››     update in our five year history.
››
›› You mean &quot;almost three year history&quot;.
››
›
› Almost six year history, actually. But we&#x27;ve been through this.
›
›› While we&#x27;re on the topic of honesty, I&#x27;d also like to draw your attention
›› to the license agreement, as you&#x27;ve broken every single request listed in
›› the second section (about giving credit) - except for two parts which I
›› cannot prove (distributing the doc files with your code and keeping the
›› headers intact).  Specifically, you&#x27;ve broken the following rules:
››
›› -- The text in the &#x27;credits&#x27; file distributed with CircleMUD must be
››    preserved.  You may add your own credits to the file, but the existing
››    text must not be removed, abridged, truncated, or changed in any way.
››    This file must be displayed when the &#x27;credits&#x27; command is used from
››    within the MUD.
››
›› You&#x27;ve replaced it with your own.
››
›› -- The &quot;CIRCLEMUD&quot; help entry must be maintained intact and unchanged, and
››    displayed in its entirety when the &#x27;help circlemud&#x27; command is used.
››
›› You&#x27;ve replaced it with your own.
››
›› -- The login sequence must contain the names of the DikuMUD and CircleMUD
››    creators.  The &#x27;login sequence&#x27; is defined as the text seen by players
››    between the time they connect to the MUD and when they start to play
››    the game itself.
››
›› You don&#x27;t credit the Diku team (which is also against the third section of
›› the Circle license: you must comply with the DikuMUD license).
››
›› -- Claims that any of the above requirements are inapplicable to a particular
››    MUD for reasons such as &quot;our MUD is totally rewritten&quot; or similar are
››    completely invalid.  If you can write a MUD completely from scratch then
››    you are encouraged to do so by all means, but use of any part of the
››    CircleMUD or DikuMUD source code requires that their respective licenses
››    be followed, including the crediting requirements.
››
›› That speaks for itself.  Did you read the part where Jeremy Elson wrote:
››
››    A great deal of work went into the creation of CircleMUD, and it was given
››    to you completely free of charge; claiming that you wrote the MUD yourself
››    is a slap in the face to everyone who worked to bring you a high quality
››    product while asking for nothing but credit for their work in return.
››
›› You should have done.
››
››    The first time you try to compile Circle, you will be asked to read the
››    CircleMUD license.  Please read it!
››
›
› Now we are speaking strictly of personal opinion here, so don&#x27;t get hot, this
› is just how I feel:  I feel that I am not bound by the circlemud licence
› because my mud is not derived from circle directly.  I am extremely grateful
› to Jeremy for releasing the code to circle, which has helped myself and Dark
› Pawns greatly, there is no dispute about that, but I would not say that the
› mud wasn&#x27;t custom unless it wasn&#x27;t.  The mud has been developed by talented
› people for almost six years, I myself being fortunate enough to develop it
› for the last three, and I have put a considerable amount of time and effort
› into making it the most unique ud that I could.
›
›› I assume you realise that because of people like you, people like me are
›› not willing to release their source code?
›
› I understand that there are people like that, and I&#x27;d like you to understand
› that I&#x27;m not one of them. I particularly understand that you have been burned
› before, with the release of your stolen GodWars source and are no doubt still
› bitter about it, which I agree with wholeheartedly. I&#x27;m just trying to say
› that you don&#x27;t have all the facts, and you are attacking the wrong person. I
› do my best to give everyone the credit they deserve, and you are right,
› Jeremy Elson does deserve his credit, which he gets in my opening screen and
› in the credits file.
›
› I hope I cleared things up for you.
›
› Rich

Stick it to him, Rich. I&#x27;m tired of snooty, soap-box types who leap on every
opportunity to make world-class asses of themselves. Three cheers for Rich!

Matthew Schuyler Peck</pre>

### adam — Feb 12, 1999 19:00 UTC

<pre class="usenet-transcript">Matthew Schuyler Peck   wrote:
›
▶

[email redacted] wrote:
(101 lines)
› [email redacted] wrote:
›
›› In article [email redacted]
››   Richard Woolcock
 wrote:
››› [email redacted] wrote:
››››
›››
››› [snip]
›››
›››› Dark Pawns is a completely original Dark Fantasy Multi User
›››
››› So why does it say &quot;Based on CircleMUD 3.0, created by Jeremy Elson&quot;
››› on the login screen?
››
›› It says that because I would like to give Jeremy Elson credit because I
›› and the developers before me used his Circle mud 3.0 pl11 code as a model
›› in enhancing our now almost six year old Diku code.  In the course of your
›› reply, you seem to indicate that I personally am an unethical individual
›› because you feel that I took stock circle, modified it, and now claim it
›› as my own 100%.  This is simply not true.
››
›››
›››› Environment that has been constantly played and developed for
›››› over five years.
›››
››› I didn&#x27;t realise Circlemud 3.0 patch 11 was that old.  Being curious,
››› I downloaded a copy of Circlemud 3.0 patch 11 (which is the version
››› your helpfiles say you are using) and noticed that the files were
››› date-stamped 1996.  Of course, someone could have touched them since
››› the release date, so I did a grep for &quot;96&quot; and came up with the
››› following:
›››
››› interpreter.c:     * re-add the code to cut off duplicates when a player
›› quits.
›››  JE 6 Feb 96
››› modify.c:* for CircleMUD.  All functions below are his.  --JE 8 Mar 96
››› random.c: * supported on the target system.  -JE 2/3/96
››› spell_parser.c: * or skill, look in class.c.  -JE 5 Feb 1996
››› sysdep.h: * 20 Mar 96: My quest is not yet over.  These definitions still
›› cause
›››
››
›› Again, the circle code was only used as a model to enhance the existing
›› diku code. If a musical artist borrows a bass line from another artist
›› for his song, it is only right that the first artist is given credit.
›› HOWEVER, this doesn&#x27;t mean that the song is owned by the first artist, and
›› because credit is given it doesn&#x27;t constitute intellectual property theft.
››
›››› We believe that Dark Pawns offers one of the
›››› most unique gaming environments on the Internet.
›››
››› Forgive me if I show a little hesitation to believe you at this point.
›››
››
›› Understandable, I hope this message clears things up for you a bit.
››
››››     We have just released Dark Pawns version 2.2, the biggest source
››››     update in our five year history.
›››
››› You mean &quot;almost three year history&quot;.
›››
››
›› Almost six year history, actually. But we&#x27;ve been through this.
››
››› While we&#x27;re on the topic of honesty, I&#x27;d also like to draw your attention
››› to the license agreement, as you&#x27;ve broken every single request listed in
››› the second section (about giving credit) - except for two parts which I
››› cannot prove (distributing the doc files with your code and keeping the
››› headers intact).  Specifically, you&#x27;ve broken the following rules:
›››
››› -- The text in the &#x27;credits&#x27; file distributed with CircleMUD must be
›››    preserved.  You may add your own credits to the file, but the existing
›››    text must not be removed, abridged, truncated, or changed in any way.
›››    This file must be displayed when the &#x27;credits&#x27; command is used from
›››    within the MUD.
›››
››› You&#x27;ve replaced it with your own.
›››
››› -- The &quot;CIRCLEMUD&quot; help entry must be maintained intact and unchanged, and
›››    displayed in its entirety when the &#x27;help circlemud&#x27; command is used.
›››
››› You&#x27;ve replaced it with your own.
›››
››› -- The login sequence must contain the names of the DikuMUD and CircleMUD
›››    creators.  The &#x27;login sequence&#x27; is defined as the text seen by players
›››    between the time they connect to the MUD and when they start to play
›››    the game itself.
›››
››› You don&#x27;t credit the Diku team (which is also against the third section of
››› the Circle license: you must comply with the DikuMUD license).
›››
››› -- Claims that any of the above requirements are inapplicable to a particular
›››    MUD for reasons such as &quot;our MUD is totally rewritten&quot; or similar are
›››    completely invalid.  If you can write a MUD completely from scratch then
›››    you are encouraged to do so by all means, but use of any part of the
›››    CircleMUD or DikuMUD source code requires that their respective licenses
›››    be followed, including the crediting requirements.
›››
››› That speaks for itself.  Did you read the part where Jeremy Elson wrote:
›››
›››    A great deal of work went into the creation of CircleMUD, and it was given
›››    to you completely free of charge; claiming that you wrote the MUD yourself
›››    is a slap in the face to everyone who worked to bring you a high quality
›››    product while asking for nothing but credit for their work in return.
›››
››› You should have done.
›››
›››    The first time you try to compile Circle, you will be asked to read the
›››    CircleMUD license.  Please read it!
›››
››
›› Now we are speaking strictly of personal opinion here, so don&#x27;t get hot, this
›› is just how I feel:  I feel that I am not bound by the circlemud licence
›› because my mud is not derived from circle directly.  I am extremely grateful
›› to Jeremy for releasing the code to circle, which has helped myself and Dark
›› Pawns greatly, there is no dispute about that, but I would not say that the
›› mud wasn&#x27;t custom unless it wasn&#x27;t.  The mud has been developed by talented
›› people for almost six years, I myself being fortunate enough to develop it
›› for the last three, and I have put a considerable amount of time and effort
›› into making it the most unique ud that I could.
››
››› I assume you realise that because of people like you, people like me are
››› not willing to release their source code?
››
›› I understand that there are people like that, and I&#x27;d like you to understand
›› that I&#x27;m not one of them. I particularly understand that you have been burned
›› before, with the release of your stolen GodWars source and are no doubt still
›› bitter about it, which I agree with wholeheartedly. I&#x27;m just trying to say
›› that you don&#x27;t have all the facts, and you are attacking the wrong person. I
›› do my best to give everyone the credit they deserve, and you are right,
›› Jeremy Elson does deserve his credit, which he gets in my opening screen and
›› in the credits file.
››
›› I hope I cleared things up for you.
››
›› Rich
›
› Stick it to him, Rich. I&#x27;m tired of snooty, soap-box types who leap on every
› opportunity to make world-class asses of themselves. Three cheers for Rich!
›
› Matthew Schuyler Peck
›

Yeah!  Teach him about USENET etiquette while you&#x27;re at it!

--Adam

Signature
[email redacted]
ICQ 27422092</pre>

### kavir — Feb 12, 1999 19:00 UTC

<pre class="usenet-transcript">Matthew Schuyler Peck wrote:
›
▶

[email redacted] wrote:
(100 lines)
›
› [email redacted] wrote:
›
›› In article [email redacted]
››   Richard Woolcock
 wrote:
››› [email redacted] wrote:
››››
›››
››› [snip]
›››
›››› Dark Pawns is a completely original Dark Fantasy Multi User
›››
››› So why does it say &quot;Based on CircleMUD 3.0, created by Jeremy Elson&quot;
››› on the login screen?
››
›› It says that because I would like to give Jeremy Elson credit because I
›› and the developers before me used his Circle mud 3.0 pl11 code as a model
›› in enhancing our now almost six year old Diku code.  In the course of your
›› reply, you seem to indicate that I personally am an unethical individual
›› because you feel that I took stock circle, modified it, and now claim it
›› as my own 100%.  This is simply not true.
››
›››
›››› Environment that has been constantly played and developed for
›››› over five years.
›››
››› I didn&#x27;t realise Circlemud 3.0 patch 11 was that old.  Being curious,
››› I downloaded a copy of Circlemud 3.0 patch 11 (which is the version
››› your helpfiles say you are using) and noticed that the files were
››› date-stamped 1996.  Of course, someone could have touched them since
››› the release date, so I did a grep for &quot;96&quot; and came up with the
››› following:
›››
››› interpreter.c:     * re-add the code to cut off duplicates when a player
›› quits.
›››  JE 6 Feb 96
››› modify.c:* for CircleMUD.  All functions below are his.  --JE 8 Mar 96
››› random.c: * supported on the target system.  -JE 2/3/96
››› spell_parser.c: * or skill, look in class.c.  -JE 5 Feb 1996
››› sysdep.h: * 20 Mar 96: My quest is not yet over.  These definitions still
›› cause
›››
››
›› Again, the circle code was only used as a model to enhance the existing
›› diku code. If a musical artist borrows a bass line from another artist
›› for his song, it is only right that the first artist is given credit.
›› HOWEVER, this doesn&#x27;t mean that the song is owned by the first artist, and
›› because credit is given it doesn&#x27;t constitute intellectual property theft.
››
›››› We believe that Dark Pawns offers one of the
›››› most unique gaming environments on the Internet.
›››
››› Forgive me if I show a little hesitation to believe you at this point.
›››
››
›› Understandable, I hope this message clears things up for you a bit.
››
››››     We have just released Dark Pawns version 2.2, the biggest source
››››     update in our five year history.
›››
››› You mean &quot;almost three year history&quot;.
›››
››
›› Almost six year history, actually. But we&#x27;ve been through this.
››
››› While we&#x27;re on the topic of honesty, I&#x27;d also like to draw your attention
››› to the license agreement, as you&#x27;ve broken every single request listed in
››› the second section (about giving credit) - except for two parts which I
››› cannot prove (distributing the doc files with your code and keeping the
››› headers intact).  Specifically, you&#x27;ve broken the following rules:
›››
››› -- The text in the &#x27;credits&#x27; file distributed with CircleMUD must be
›››    preserved.  You may add your own credits to the file, but the existing
›››    text must not be removed, abridged, truncated, or changed in any way.
›››    This file must be displayed when the &#x27;credits&#x27; command is used from
›››    within the MUD.
›››
››› You&#x27;ve replaced it with your own.
›››
››› -- The &quot;CIRCLEMUD&quot; help entry must be maintained intact and unchanged, and
›››    displayed in its entirety when the &#x27;help circlemud&#x27; command is used.
›››
››› You&#x27;ve replaced it with your own.
›››
››› -- The login sequence must contain the names of the DikuMUD and CircleMUD
›››    creators.  The &#x27;login sequence&#x27; is defined as the text seen by players
›››    between the time they connect to the MUD and when they start to play
›››    the game itself.
›››
››› You don&#x27;t credit the Diku team (which is also against the third section of
››› the Circle license: you must comply with the DikuMUD license).
›››
››› -- Claims that any of the above requirements are inapplicable to a particular
›››    MUD for reasons such as &quot;our MUD is totally rewritten&quot; or similar are
›››    completely invalid.  If you can write a MUD completely from scratch then
›››    you are encouraged to do so by all means, but use of any part of the
›››    CircleMUD or DikuMUD source code requires that their respective licenses
›››    be followed, including the crediting requirements.
›››
››› That speaks for itself.  Did you read the part where Jeremy Elson wrote:
›››
›››    A great deal of work went into the creation of CircleMUD, and it was given
›››    to you completely free of charge; claiming that you wrote the MUD yourself
›››    is a slap in the face to everyone who worked to bring you a high quality
›››    product while asking for nothing but credit for their work in return.
›››
››› You should have done.
›››
›››    The first time you try to compile Circle, you will be asked to read the
›››    CircleMUD license.  Please read it!
›››
››
›› Now we are speaking strictly of personal opinion here, so don&#x27;t get hot, this
›› is just how I feel:  I feel that I am not bound by the circlemud licence
›› because my mud is not derived from circle directly.  I am extremely grateful
›› to Jeremy for releasing the code to circle, which has helped myself and Dark
›› Pawns greatly, there is no dispute about that, but I would not say that the
›› mud wasn&#x27;t custom unless it wasn&#x27;t.  The mud has been developed by talented
›› people for almost six years, I myself being fortunate enough to develop it
›› for the last three, and I have put a considerable amount of time and effort
›› into making it the most unique ud that I could.
››
››› I assume you realise that because of people like you, people like me are
››› not willing to release their source code?
››
›› I understand that there are people like that, and I&#x27;d like you to understand
›› that I&#x27;m not one of them. I particularly understand that you have been burned
›› before, with the release of your stolen GodWars source and are no doubt still
›› bitter about it, which I agree with wholeheartedly. I&#x27;m just trying to say
›› that you don&#x27;t have all the facts, and you are attacking the wrong person. I
›› do my best to give everyone the credit they deserve, and you are right,
›› Jeremy Elson does deserve his credit, which he gets in my opening screen and
›› in the credits file.
››
›› I hope I cleared things up for you.
››
›› Rich
›
› Stick it to him, Rich. I&#x27;m tired of snooty, soap-box types who leap on every
› opportunity to make world-class asses of themselves. Three cheers for Rich!

So the fact that he doesn&#x27;t credit the authors of his codebase doesn&#x27;t
bother you, then?  Don&#x27;t you think we have enough Vryce&#x27;s already?

› Matthew Schuyler Peck

Do you mind if I just call you &quot;Kressilac&quot;?

KaVir.</pre>

### kavir — Feb 12, 1999 19:00 UTC

<pre class="usenet-transcript">[email redacted] wrote:
›
▶

In article &lt;[email redacted]&gt;,
(10 lines)
›
› In article [email redacted]
›   Richard Woolcock
 wrote:
›› [email redacted] wrote:
›››
››
›› [snip]
››
››› Dark Pawns is a completely original Dark Fantasy Multi User
››
›› So why does it say &quot;Based on CircleMUD 3.0, created by Jeremy Elson&quot;
›› on the login screen?
›
› It says that because I would like to give Jeremy Elson credit because I
› and the developers before me used his Circle mud 3.0 pl11 code as a model
› in enhancing our now almost six year old Diku code.

On your login screen it says &quot;Based on CircleMUD 3.0&quot;, and in the credits it
says:

Dark Pawns was originally based on CircleMUD 3.0 beta patch-level 11, developed
by Jeremy Elson.  Without Jeremy and the fine example his code was to build on,
Dark Pawns would never have existed. Thank you very much, Jeremy.

› In the course of your reply, you seem to indicate that I personally am an
› unethical individual because you feel that I took stock circle, modified
› it, and now claim it as my own 100%.  This is simply not true.

I made that assumption that your mud is based on Circle because:

1) The mud states that it is based on Circle 3.0 beta patch-level 11.
2) Your mud looks and feels like a stock Circle mud with a few modifications,
3) You advertised it in the rec.games.mud.diku newsgroup.

[snip]

› Again, the circle code was only used as a model to enhance the existing
› diku code.

So you do admit that you&#x27;re using diku code, despite not following their
license?

›
▶

If a musical artist borrows a bass line from another artist for his song,
(4 lines)
› If a musical artist borrows a bass line from another artist for his song,
› it is only right that the first artist is given credit.  HOWEVER, this
› doesn&#x27;t mean that the song is owned by the first artist, and because
› credit is given it doesn&#x27;t constitute intellectual property theft.

The mud is not a song.  The license is quite simple.  If you want to use
the code, you follow the license.  If you don&#x27;t want to follow the license,
you may not use the code.

[snip]

› Now we are speaking strictly of personal opinion here, so don&#x27;t get hot, this
› is just how I feel:  I feel that I am not bound by the circlemud licence
› because my mud is not derived from circle directly.

If you&#x27;ve used Circle code, then you must follow the Circle license.  If you&#x27;ve
used Diku code, then you must follow the Diku license.  This is not a matter of
personal opinion, this is a matter of law and ethics.  Once again, let me
repeat the quote from Jeremy:

   A great deal of work went into the creation of CircleMUD, and it was given
   to you completely free of charge; claiming that you wrote the MUD yourself
   is a slap in the face to everyone who worked to bring you a high quality
   product while asking for nothing but credit for their work in return.

The same applies to Diku.

[snip rest]

KaVir.</pre>

### eryi — Feb 12, 1999 19:00 UTC

<pre class="usenet-transcript">Chuckle..  Only one thing wrong with what you said, that is:

If you use CircleMUD, You have to follow BOTH the CircleMUD AND DikuMUD
Licenses..   The CircleMUD License requires full compliance with the DikuMUD
license as well as the additional things tacked on by the CircleMUD License
itself

Richard Woolcock  wrote in message
[email redacted]
›
▶

[email redacted] wrote:
(59 lines)
› [email redacted] wrote:
››
›› In article [email redacted]
››   Richard Woolcock
 wrote:
››› [email redacted] wrote:
››››
›››
››› [snip]
›››
›››› Dark Pawns is a completely original Dark Fantasy Multi User
›››
››› So why does it say &quot;Based on CircleMUD 3.0, created by Jeremy Elson&quot;
››› on the login screen?
››
›› It says that because I would like to give Jeremy Elson credit because I
›› and the developers before me used his Circle mud 3.0 pl11 code as a model
›› in enhancing our now almost six year old Diku code.
›
› On your login screen it says &quot;Based on CircleMUD 3.0&quot;, and in the credits
› it
› says:
›
› Dark Pawns was originally based on CircleMUD 3.0 beta patch-level 11,
› developed
› by Jeremy Elson.  Without Jeremy and the fine example his code was to build
› on,
› Dark Pawns would never have existed. Thank you very much, Jeremy.
›
›› In the course of your reply, you seem to indicate that I personally am an
›› unethical individual because you feel that I took stock circle, modified
›› it, and now claim it as my own 100%.  This is simply not true.
›
› I made that assumption that your mud is based on Circle because:
›
› 1) The mud states that it is based on Circle 3.0 beta patch-level 11.
› 2) Your mud looks and feels like a stock Circle mud with a few
› modifications,
› 3) You advertised it in the rec.games.mud.diku newsgroup.
›
› [snip]
›
›› Again, the circle code was only used as a model to enhance the existing
›› diku code.
›
› So you do admit that you&#x27;re using diku code, despite not following their
› license?
›
›› If a musical artist borrows a bass line from another artist for his song,
›› it is only right that the first artist is given credit.  HOWEVER, this
›› doesn&#x27;t mean that the song is owned by the first artist, and because
›› credit is given it doesn&#x27;t constitute intellectual property theft.
›
› The mud is not a song.  The license is quite simple.  If you want to use
› the code, you follow the license.  If you don&#x27;t want to follow the license,
› you may not use the code.
›
› [snip]
›
›› Now we are speaking strictly of personal opinion here, so don&#x27;t get hot,
›› this
›› is just how I feel:  I feel that I am not bound by the circlemud licence
›› because my mud is not derived from circle directly.
›
› If you&#x27;ve used Circle code, then you must follow the Circle license.  If
› you&#x27;ve
› used Diku code, then you must follow the Diku license.  This is not a
› matter of
› personal opinion, this is a matter of law and ethics.  Once again, let me
› repeat the quote from Jeremy:
›
›   A great deal of work went into the creation of CircleMUD, and it was
› given
›   to you completely free of charge; claiming that you wrote the MUD
› yourself
›   is a slap in the face to everyone who worked to bring you a high quality
›   product while asking for nothing but credit for their work in return.
›
› The same applies to Diku.
›
› [snip rest]
›
› KaVir.</pre>

### mspeck — Feb 12, 1999 19:00 UTC

<pre class="usenet-transcript">Richard,

I think you&#x27;ve really missed the boat and sunk to the bottom of the harbor on this
one. It is quite evident to me that what this person is saying is that he wrote a
MUD which was not a modified CircleMUD in any way shape or form (how could it be,
if it is six years old?) and at one point looked at CircleMUD 3.0bpl11 and used the
ideas for interface and whatnot to modify his own code. You keep harping on this
notion of him using CircleMUD code. He never indicates in any way that he used
CircleMUD code!

Jump down off your high horse, Richard, until you&#x27;ve taken a careful look at the
battlefield.

Matthew Schuyler Peck</pre>

### kavir — Feb 12, 1999 19:00 UTC

<pre class="usenet-transcript">Matthew Schuyler Peck wrote:
›
▶

Richard,
(5 lines)
›
› Richard,
›
› I think you&#x27;ve really missed the boat and sunk to the bottom of the harbor on this
› one. It is quite evident to me that what this person is saying is that he wrote a
› MUD which was not a modified CircleMUD in any way shape or form (how could it be,
› if it is six years old?)

Precisely, *IF* it is six years old (or five, as they claimed originally, or
seven, as they&#x27;ll probably claim next time).

› and at one point looked at CircleMUD 3.0bpl11 and used the
› ideas for interface and whatnot to modify his own code.
› You keep harping on this notion of him using CircleMUD code.

Wrong.  Read my post.  I explained why I made the *assumption* that he was
using Circle, and then explained how the same rules applied to his mud
anyway, because he was running a Diku; a Diku that credits the Circle
team (despite claiming not to use Circle code) without crediting the
Diku team (whose code he admits they are using).

› He never indicates in any way that he used CircleMUD code!

No, but the mud states in at least two places that is it BASED on the Circle
code.

› Jump down off your high horse, Richard, until you&#x27;ve taken a careful look at the
› battlefield.

I&#x27;ve obviously got a much better view of the battlefield up here.

KaVir.</pre>

### rparet — Feb 13, 1999 19:00 UTC

<pre class="usenet-transcript">› Precisely, *IF* it is six years old (or five, as they claimed originally, or
› seven, as they&#x27;ll probably claim next time).
›

Ok. Hold on a minute. I have held back from any personal attacks during the
course of this thread and I expect you to do the same. The mud is five years
old. It will be six years old in roughly three months.

›
▶

and at one point looked at CircleMUD 3.0bpl11 and used the
(8 lines)
›› and at one point looked at CircleMUD 3.0bpl11 and used the
›› ideas for interface and whatnot to modify his own code.
›› You keep harping on this notion of him using CircleMUD code.
›
› Wrong.  Read my post.  I explained why I made the *assumption* that he was
› using Circle, and then explained how the same rules applied to his mud
› anyway, because he was running a Diku; a Diku that credits the Circle
› team (despite claiming not to use Circle code) without crediting the
› Diku team (whose code he admits they are using).
›

Ok, what Matthrew said is essentially correct, but I do have to make a
concession: I can see how it may appear to be very unclear as to the nature of
the source, etc. Hopefully I can explain myself a little bit better in the
following paragraphs, but I think Matthew did a very nice job of suming it up.


›› He never indicates in any way that he used CircleMUD code!
›
› No, but the mud states in at least two places that is it BASED on the Circle
› code.
›

Ok. Point one.	In the mud it does say that the mud is based on the circle,
but not the code. This is my main point of concession, and also probably the
central issue regarding the unclarity of what DP actually &quot;is&quot; in terms of a
mud.  So here is an abbreviated version of why it is like that.  Just before
I took over the mud, three years or so ago, it was an awful mix of hybrid
code, obviously diku style based but I can&#x27;t honestly say it was diku for
sure, so the current admins had this brilliant idea, why not change our mud
to have the same look and feel as circle mud, and they even went so far as to
advertise it as a circle mud for a length of time.  You can see these ads by
doing a search on dejanews, and actually the last time I checked, one of the
original ads for DP is still archived from 1994, when the mud was at
knight.upf.org 4000. But between them leaving, and myself taking over, I&#x27;ve
released two source versions and taken the mud far beyond what it was.	I
suppose the whole &quot;based on CM 3.0 pl11&quot; comes from the fact that the look
and feel is very much based on the look and feel of circlemud, although I am
rather shocked to hear you say that my mud looks like stock circle with a few
modifications.	Play it for any length of time, or talk to any players, and
they will tell you that your assumption is completely off base.


›
▶

Jump down off your high horse, Richard, until you&#x27;ve taken a careful look at
(4 lines)
›› Jump down off your high horse, Richard, until you&#x27;ve taken a careful look at
›› the
›› battlefield.
›
› I&#x27;ve obviously got a much better view of the battlefield up here.


Actually, I think Matt&#x27;s message was right on the money. I think I can see
where you are coming from, and I will take some steps within the mud to
clarify the situation for anyone else who may be interested.

To sum it up, I have not broken any licence agreements, nor do I intend to.  I
am a mud developer and have been for some time.  I am neither an unethical
person or an unethical programmer, and I do not intend to breach any code of
ethics any time soon.


Thanks,

Rich


-----------== Posted via Deja News, The Discussion Network ==----------
http://www.dejanews.com/       Search, Read, Discuss, or Start Your Own</pre>

### kavir — Feb 13, 1999 19:00 UTC

<pre class="usenet-transcript">[email redacted] wrote:
›
▶

[snip]
(5 lines)
›

› [snip]

› To sum it up, I have not broken any licence agreements, nor do I intend to.  I
› am a mud developer and have been for some time.  I am neither an unethical
› person or an unethical programmer, and I do not intend to breach any code of
› ethics any time soon.

Okay, but I&#x27;m still a bit confused.  You said in your previous posts:

   &quot;[we] used his Circle mud 3.0 pl11 code as a model in enhancing our
    now almost six year old Diku code.&quot;

And:

   &quot;Again, the circle code was only used as a model to enhance the existing
    diku code.&quot;

Now perhaps I&#x27;m reading too much into your posts, but I get the impression
you are actually using Diku code.  Yet you do not follow the Diku license.

I&#x27;m sure you can understand how I might find that confusing, considering
you&#x27;ve just said that you &quot;have not broken any licence agreements&quot;.

KaVir.</pre>

### sommer — Feb 13, 1999 19:00 UTC

<pre class="usenet-transcript">Richard Woolcock wrote:

›
▶

Now perhaps I&#x27;m reading too much into your posts, but I get the impression
(4 lines)
› Now perhaps I&#x27;m reading too much into your posts, but I get the impression
› you are actually using Diku code.  Yet you do not follow the Diku license.
›
› I&#x27;m sure you can understand how I might find that confusing, considering
› you&#x27;ve just said that you &quot;have not broken any licence agreements&quot;.

Fwiw, I read this as &quot;we liked how Circle felt, so we adopted some of its
mannerisms, without actually using Circle code, and wanted to thank Jeremy in the
online helps for inspiring part of our game.&quot;

Close?

-Holly
Signature
&quot;Also in the funk hall of fame are split-personality cases the Commodores,
who alternated between the hard funk of `Machine Gun&#x27; and the Manilow-esque
gushy-goo of `Three Times a Lady&#x27; depending on how much freedom Lionel
Richie was given to wander alone in the Hallmark shop.&quot; -Rhino Records</pre>

### kavir — Feb 13, 1999 19:00 UTC

<pre class="usenet-transcript">Holly Sommer wrote:
›
▶

Richard Woolcock wrote:
(8 lines)
›
› Richard Woolcock wrote:
›
›› Now perhaps I&#x27;m reading too much into your posts, but I get the impression
›› you are actually using Diku code.  Yet you do not follow the Diku license.
››
›› I&#x27;m sure you can understand how I might find that confusing, considering
›› you&#x27;ve just said that you &quot;have not broken any licence agreements&quot;.
›
› Fwiw, I read this as &quot;we liked how Circle felt, so we adopted some of its
› mannerisms, without actually using Circle code, and wanted to thank Jeremy
› in the online helps for inspiring part of our game.&quot;

Yes Holly, I grasped that part.  My question is purely concerned with the
Diku code now.

KaVir.</pre>

### rparet — Feb 14, 1999 19:00 UTC

<pre class="usenet-transcript">In article [email redacted]
  Richard Woolcock  wrote:
›
▶

[email redacted] wrote:
(17 lines)
› [email redacted] wrote:
››
›
› [snip]
›
›› To sum it up, I have not broken any licence agreements, nor do I intend to.
››  I
›› am a mud developer and have been for some time.  I am neither an unethical
›› person or an unethical programmer, and I do not intend to breach any code of
›› ethics any time soon.
›
› Okay, but I&#x27;m still a bit confused.  You said in your previous posts:
›
›    &quot;[we] used his Circle mud 3.0 pl11 code as a model in enhancing our
›     now almost six year old Diku code.&quot;
›
› And:
›
›    &quot;Again, the circle code was only used as a model to enhance the existing
›     diku code.&quot;
›
› Now perhaps I&#x27;m reading too much into your posts, but I get the impression
› you are actually using Diku code.  Yet you do not follow the Diku license.
›
› I&#x27;m sure you can understand how I might find that confusing, considering
› you&#x27;ve just said that you &quot;have not broken any licence agreements&quot;.


I can understand why you might find it confusing, and it is my fault. Chalk
it up to language confusion.  When I said &quot;diku code&quot; I was refering to
&quot;diku&quot; in a generic sort of sense.  As in diku style, rather than LP style,
or MUME style, etc. It wasn&#x27;t my intention to say that the existing code was
DikuMUD Gamma 0.0, as you interpreted me as saying. I apologise for my
vagueness.

Rich

-----------== Posted via Deja News, The Discussion Network ==----------
http://www.dejanews.com/       Search, Read, Discuss, or Start Your Own</pre>

### kavir — Feb 14, 1999 19:00 UTC

<pre class="usenet-transcript">[email redacted] wrote:
›
▶

In article &lt;[email redacted]&gt;,
(25 lines)
›
› In article [email redacted]
›   Richard Woolcock
 wrote:
›› [email redacted] wrote:
›››
››
›› [snip]
››
››› To sum it up, I have not broken any licence agreements, nor do I intend to.
›  I
››› am a mud developer and have been for some time.  I am neither an unethical
››› person or an unethical programmer, and I do not intend to breach any code of
››› ethics any time soon.
››
›› Okay, but I&#x27;m still a bit confused.  You said in your previous posts:
››
››    &quot;[we] used his Circle mud 3.0 pl11 code as a model in enhancing our
››     now almost six year old Diku code.&quot;
››
›› And:
››
››    &quot;Again, the circle code was only used as a model to enhance the existing
››     diku code.&quot;
››
›› Now perhaps I&#x27;m reading too much into your posts, but I get the impression
›› you are actually using Diku code.  Yet you do not follow the Diku license.
››
›› I&#x27;m sure you can understand how I might find that confusing, considering
›› you&#x27;ve just said that you &quot;have not broken any licence agreements&quot;.
›
› I can understand why you might find it confusing, and it is my fault. Chalk
› it up to language confusion.  When I said &quot;diku code&quot; I was refering to
› &quot;diku&quot; in a generic sort of sense.  As in diku style, rather than LP style,
› or MUME style, etc. It wasn&#x27;t my intention to say that the existing code was
› DikuMUD Gamma 0.0, as you interpreted me as saying. I apologise for my
› vagueness.

I made no mention of what version of Diku you might be using.  Diku has
been available since 1991, and many of its derivatives (such as Circle 2,
Merc 2, Silly, etc) have been out since 1993, any of which would easily fall
into the 5 - nearly 6 - year age bracket that you mentioned.

However, despite your previous confusion when you said &quot;Just before I took
over the mud, three years or so ago, it was an awful mix of hybrid code,
obviously diku style based but I can&#x27;t honestly say it was diku for sure&quot;,
I&#x27;m glad to see you&#x27;ve finally checked the code and determined that it isn&#x27;t
Diku.

I&#x27;d also like to congratulate you on creating such a faithful Diku-lookalike;
it certainly had me fooled, and I&#x27;ve been playing Diku muds for years.

KaVir.</pre>

### j_herlih — Feb 14, 1999 19:00 UTC

<pre class="usenet-transcript">wrote:
›
▶

To sum it up, I have not broken any licence agreements, nor do I intend to.  I
(4 lines)
› To sum it up, I have not broken any licence agreements, nor do I intend to.  I
› am a mud developer and have been for some time.  I am neither an unethical
› person or an unethical programmer, and I do not intend to breach any code of
› ethics any time soon.

	Once again, you have a Diku-based mud that does not adhere to
the DikuMUD license.  Until you choose to adhere to a license that 99%
of DikuMUD administrators have no problem with, you get to share space
on the blacklist right along with Vryce and the other scum.  Enjoy!
Signature
-AxL    [email redacted] &quot;In Christianity, neither morality nor religion
   [email redacted]    Come into contact with reality at any point.&quot;
[personal URL redacted]
                                 - Nietzsche</pre>
