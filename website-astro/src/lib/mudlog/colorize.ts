// Rebuilds the colour of a plain-text Dark Pawns log.
//
// The surviving logs were captured by zMUD, which wrote the words and dropped
// the ANSI codes. The server decided every colour itself, by message type, and
// those decisions are still in the C source. Each rule below names the call it
// reproduces. A rule also records the `color` level the C gates it on
// (screen.h: C_OFF 0, C_SPR 1, C_NRM 2, C_CMP 3; clr() is level >= gate), so
// the page can show the log at any of the four settings a player could choose.
//
// What this cannot know, and does not guess: anything the log does not show.
// Prompt numbers are coloured by their share of the character's maximum
// (comm.c:1081-1117), and the log never prints a maximum, so the highest value
// the log shows for each stat stands in for it.

import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

export type Color = 'red' | 'green' | 'yellow' | 'blue' | 'magenta' | 'cyan' | 'white';
export type Level = 1 | 2 | 3;

export interface Run {
  text: string;
  color?: Color;
  level?: Level;
}

export type Line = Run[];

export interface WorldText {
  mobLongs: Set<string>;
  objLongs: Set<string>;
}

const plain = (text: string): Run => ({ text });
const tint = (text: string, color: Color, level: Level): Run => ({ text, color, level });

const escapeRe = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

// act() substitutions, as far as the log can show them: a name is any text,
// a pronoun is one of three words.
function actTemplate(template: string, verbs?: { singular: string; plural: string }): RegExp {
  let re = escapeRe(template.trim());
  if (verbs) {
    re = re.replace(/#w/g, verbs.singular).replace(/#W/g, verbs.plural);
  }
  re = re
    .replace(/\\\$[eE]/g, '(?:he|she|it)')
    .replace(/\\\$[mM]/g, '(?:him|her|it)')
    .replace(/\\\$[sS]/g, '(?:his|her|its)')
    .replace(/\\\$[a-zA-Z]/g, '.+?');
  return new RegExp(`^${re}$`);
}

// fight.c:86-100, attack_hit_text.
const HIT_VERBS = [
  ['hit', 'hits'], ['sting', 'stings'], ['whip', 'whips'], ['slash', 'slashes'],
  ['bite', 'bites'], ['bludgeon', 'bludgeons'], ['crush', 'crushes'], ['pound', 'pounds'],
  ['claw', 'claws'], ['maul', 'mauls'], ['thrash', 'thrashes'], ['pierce', 'pierces'],
  ['blast', 'blasts'], ['punch', 'punches'], ['stab', 'stabs'],
];

// fight.c:905-973, dam_weapons: [to_char, to_victim] for each damage band.
const DAM_WEAPONS = [
  ['You try to #w $N, but miss.', '$n tries to #w you, but misses.'],
  ['You scratch $N as you #w $M.', '$n scratches you as $e #W you.'],
  ['You barely #w $N.', '$n barely #W you.'],
  ['You #w $N.', '$n #W you.'],
  ['You #w $N hard.', '$n #W you hard.'],
  ['You #w $N very hard.', '$n #W you very hard.'],
  ['You #w $N extremely hard.', '$n #W you extremely hard.'],
  ['You massacre $N to small fragments with your #w.', '$n massacres you to small fragments with $s #w.'],
  ['You OBLITERATE $N with your deadly #w!!', '$n OBLITERATES you with $s deadly #w!!'],
  ['You EVISCERATE $N with your incredible #w!!', '$n EVISCERATES you with $s incredible #w!!'],
  ['You DESTROY $N with your ungodly #w!!', '$n DESTROYS you with $s ungodly #w!!'],
  ['You ROCK THE HELL OUT OF $N with your ultimate #w!!', '$n ROCKS THE HELL OUT OF you with $s ultimate #w!!'],
];

interface DamagePatterns {
  attacker: RegExp[];
  victim: RegExp[];
}

function weaponPatterns(): DamagePatterns {
  const verbs = HIT_VERBS.map(([singular, plural]) => ({ singular, plural }));
  const alt = {
    singular: `(?:${verbs.map((v) => v.singular).join('|')})`,
    plural: `(?:${verbs.map((v) => v.plural).join('|')})`,
  };
  return {
    attacker: DAM_WEAPONS.map(([toChar]) => actTemplate(toChar, alt)),
    victim: DAM_WEAPONS.map(([, toVictim]) => actTemplate(toVictim, alt)),
  };
}

// lib/misc/messages: twelve lines per record, death/miss/hit/god, each as
// (attacker, victim, room). fight.c:1047-1086 sends the attacker's line in
// yellow and the victim's in red at C_CMP; the god lines carry no colour.
function skillPatterns(file: string): DamagePatterns {
  const out: DamagePatterns = { attacker: [], victim: [] };
  if (!existsSync(file)) return out;
  const lines = readFileSync(file, 'latin1').split(/\r?\n/);
  for (let i = 0; i < lines.length; i++) {
    if (lines[i] !== 'M') continue;
    const record = lines.slice(i + 2, i + 14);
    i += 13;
    for (const [index, side] of [[0, 'attacker'], [1, 'victim'], [3, 'attacker'], [4, 'victim'], [6, 'attacker'], [7, 'victim']] as const) {
      const template = record[index]?.trim();
      if (template && template !== '#') out[side].push(actTemplate(template));
    }
  }
  return out;
}

// do_gen_comm, act.comm.c:1172-1202. The speaker's own line is coloured at
// C_CMP (1264), everyone else's at C_NRM (1291).
const CHANNELS: Record<string, Color> = {
  holler: 'yellow', shout: 'yellow', gossip: 'yellow', auction: 'magenta', congrat: 'green', newbie: 'yellow',
};
const CHANNEL_NAMES = Object.keys(CHANNELS).join('|');
const OWN_CHANNEL = new RegExp(`^You (${CHANNEL_NAMES}), '`);
const OTHER_CHANNEL = new RegExp(`^.+ (${CHANNEL_NAMES})s, '`);

// comm.c:1066-1157: optional invisibility level, the three stats, target and
// tank status, AFK, then "> ".
const PROMPT = /^(i\d+ )?(?:(\d+)H )?(?:(\d+)M )?(?:(\d+)V )?((?:[^>]*?:\([^)]*\) )*)((?:AFK |INACTIVE )?)> ?(.*)$/;

function promptParts(line: string) {
  const m = PROMPT.exec(line);
  if (!m || (!m[2] && !m[3] && !m[4])) return null;
  return {
    invis: m[1] ?? '',
    stats: [m[2], m[3], m[4]] as (string | undefined)[],
    status: m[5],
    flags: m[6],
    rest: m[7],
  };
}

const CHAR_IN_ROOM = /\b(?:is|are) (?:standing|sitting|resting|sleeping|lying|floating|flying|here|meditating|fighting)\b|^\.\.\.(?:he|she|it) /;
const OBJ_IN_ROOM = /\bhere\.\s*$|\bground\.\s*$|\bfloor\.\s*$/;

// act.informative.c:1168-1300, do_score. The name is green at C_CMP (1184);
// current hit/mana/move are green and their maximums red, all C_CMP (1192-1201);
// experience, carried gold and bank gold are green at C_CMP (1261-1266); the
// experience still needed is green at C_CMP (1273). Everything else do_score
// prints carries no colour.
const SCORE_VITALS = /^Hit points: (?<hit>\d+)\((?<maxhit>\d+)\)  (?:Mana|Mind\/Psi) points: (?<mana>\d+)\((?<maxmana>\d+)\)  Movement points: (?<move>\d+)\((?<maxmove>\d+)\)$/d;
const SCORE_EXP = /^Experience: {4}(?<exp>\d+) points$/d;
const SCORE_COINS = /^Coins carried: (?<gold>\d+) gold coins {4}Coins in bank: (?<bank>\d+) gold coins$/d;
const SCORE_NEED = /^You need (?<need>\d+) exp to reach your next level\.$/d;
const SCORE_NAME = /^(?<name>.+?) {2,}Age: (?<age>\d+) years$/d;

// act.informative.c:1395-1412: after "Spells affecting you:", each spell name is
// printed cyan at C_NRM in 24-column fields, two to a line.
const AFFECTS_HEADER = /^Spells affecting you:$/;

// Colour the named groups of a match. `d` gives each group's offsets, so a
// repeated value is never mistaken for an earlier one.
function colorGroups(text: string, re: RegExp, colorFor: (name: string) => Color | null): Line {
  const m = re.exec(text);
  if (!m || !m.indices) return [plain(text)];
  const runs: Run[] = [];
  let last = 0;
  for (const name of Object.keys(m.groups ?? {})) {
    const idx = m.indices.groups?.[name];
    if (!idx) continue;
    if (idx[0] > last) runs.push(plain(text.slice(last, idx[0])));
    const color = colorFor(name);
    const slice = text.slice(idx[0], idx[1]);
    runs.push(color ? tint(slice, color, 3) : plain(slice));
    last = idx[1];
  }
  if (last < text.length) runs.push(plain(text.slice(last)));
  return runs;
}

export class Colorizer {
  private weapons = weaponPatterns();
  private skills: DamagePatterns;
  private world: WorldText;

  constructor(world: WorldText, messagesFile = resolve(process.cwd(), '../lib/misc/messages')) {
    this.world = world;
    this.skills = skillPatterns(messagesFile);
  }

  colorize(raw: string): Line[] {
    const lines = raw.replace(/\r/g, '').replace(/\n$/, '').split('\n');
    const prompts = lines.map(promptParts);

    // The stand-in maximum for each prompt stat (see the header).
    const max = [0, 0, 0];
    for (const p of prompts) {
      p?.stats.forEach((v, i) => { if (v) max[i] = Math.max(max[i], Number(v)); });
    }

    const out: Line[] = lines.map((text) => [plain(text)]);
    const claimed = new Set<number>();

    // Rooms first, because a room block is only recognisable as a whole: its
    // exits line anchors it, and the title and contents sit around that.
    lines.forEach((text, e) => {
      if (!/^\s*\[ Exits: .*\]\s*$/.test(text)) return;
      this.room(lines, prompts, e, out, claimed);
    });

    // do_score's affect list is anchored by its header line.
    lines.forEach((text, i) => {
      if (AFFECTS_HEADER.test(text.trim())) this.affects(lines, i, out, claimed);
    });

    lines.forEach((text, i) => {
      const p = prompts[i];
      if (p) {
        out[i] = this.prompt(p, max, claimed.has(i) ? out[i] : null);
        return;
      }
      if (claimed.has(i)) return;
      out[i] = this.message(text);
    });
    return out;
  }

  // do_score's affect list: the spell names run cyan at C_NRM in 24-column
  // fields (act.informative.c:1405).
  private affects(lines: string[], header: number, out: Line[], claimed: Set<number>) {
    for (let i = header + 1; i < lines.length; i++) {
      const text = lines[i];
      if (text.trim() === '' || !/^ {5}/.test(text) || promptParts(text)) break;
      const runs: Run[] = [plain(text.slice(0, 5))];
      for (let p = 5; p < text.length; p += 24) {
        const chunk = text.slice(p, p + 24);
        runs.push(chunk.trim() ? tint(chunk, 'cyan', 2) : plain(chunk));
      }
      claimed.add(i);
      out[i] = runs;
    }
  }

  private room(lines: string[], prompts: ReturnType<typeof promptParts>[], e: number, out: Line[], claimed: Set<number>) {
    let s = e - 1;
    while (s >= 0 && lines[s].trim() !== '' && !prompts[s]) s--;
    // s is the prompt or blank line before the block. A title the server sent
    // straight after a prompt lands on the prompt's own line in the capture.
    const before = prompts[s];
    const titleOnPrompt = before && /^[A-Z[]/.test(before.rest) && !/[.!?']$/.test(before.rest);
    const first = s + 1;

    // act.informative.c:765 sends cyan at C_NRM before the title.
    if (titleOnPrompt) {
      claimed.add(s);
      out[s] = [tint(before.rest, 'cyan', 2)];
    } else if (first < e) {
      claimed.add(first);
      out[first] = [tint(lines[first], 'cyan', 2)];
    }

    // do_auto_exits, act.informative.c:662-678: cyan at C_NRM; a closed door
    // an immortal can see is red at C_CMP.
    claimed.add(e);
    out[e] = lines[e].split(/(\([a-z]+\))/).filter(Boolean).map((part) =>
      part.startsWith('(') ? tint(part, 'red', 3) : tint(part, 'cyan', 2));

    // act.informative.c:824-835: objects green, then people yellow, both C_NRM.
    let seenChar = false;
    for (let i = e + 1; i < lines.length; i++) {
      const text = lines[i];
      const bare = text.trim();
      if (bare === '' || prompts[i]) break;
      const isChar = this.world.mobLongs.has(bare) || CHAR_IN_ROOM.test(bare);
      const isObj = !seenChar && (this.world.objLongs.has(bare) || OBJ_IN_ROOM.test(bare));
      if (isChar) seenChar = true;
      else if (!isObj) break;
      claimed.add(i);
      out[i] = [tint(text, isChar ? 'yellow' : 'green', 2)];
    }
  }

  private prompt(p: NonNullable<ReturnType<typeof promptParts>>, max: number[], titled: Line | null): Line {
    const runs: Run[] = [];
    if (p.invis) runs.push(plain(p.invis));
    (['H', 'M', 'V'] as const).forEach((suffix, i) => {
      const value = p.stats[i];
      if (!value) return;
      const share = max[i] ? Number(value) / max[i] : 1;
      const color: Color = share >= 0.75 ? 'green' : share >= 0.33 ? 'yellow' : 'red';
      runs.push(tint(value, color, 3), plain(`${suffix} `));
    });
    // Target and tank status, comm.c:1128-1145; AFK, 1149.
    if (p.status) runs.push(tint(p.status.trimEnd(), 'red', 3), plain(' '));
    if (p.flags) runs.push(tint(p.flags.trimEnd(), 'red', 3), plain(' '));
    runs.push(plain('> '));
    if (titled) runs.push(...titled);
    else if (p.rest) runs.push(...this.message(p.rest));
    return runs;
  }

  private message(text: string): Line {
    const bare = text.trim();
    let m: RegExpExecArray | null;

    if ((m = OWN_CHANNEL.exec(bare))) return [tint(text, CHANNELS[m[1]], 3)];
    if ((m = OTHER_CHANNEL.exec(bare))) return [tint(text, CHANNELS[m[1]], 2)];

    // do_gsay, act.comm.c:848 (to the group, C_CMP) and 865 (to yourself, C_SPR).
    if (/^You tell the group, '/.test(bare)) return [tint(text, 'white', 1)];
    if (/^.+ tells the group, '/.test(bare)) return [tint(text, 'white', 3)];

    // do_ctell, act.comm.c:1535 and 1556, cyan at C_CMP.
    if (/^(?:You|.+) tells? your clan(?: \(\d+\) )?, '/.test(bare)) return [tint(text, 'cyan', 3)];

    // perform_tell, act.comm.c:875 (received, C_NRM) and 887 (sent, C_CMP).
    if (/^.+ tells you, '/.test(bare)) return [tint(text, 'red', 2)];
    if (/^You tell .+, '/.test(bare)) return [tint(text, 'red', 3)];

    for (const re of [...this.weapons.attacker, ...this.skills.attacker]) {
      if (re.test(bare)) return [tint(text, 'yellow', 3)];
    }
    for (const re of [...this.weapons.victim, ...this.skills.victim]) {
      if (re.test(bare)) return [tint(text, 'red', 3)];
    }
    const score = this.score(text);
    if (score) return score;
    return [plain(text)];
  }

  // do_score (act.informative.c:1168-1300). Only the lines the C colours are
  // claimed here; the rest of the score sheet stays plain.
  private score(text: string): Line | null {
    if (SCORE_VITALS.test(text)) return colorGroups(text, SCORE_VITALS, (name) => (name.startsWith('max') ? 'red' : 'green'));
    if (SCORE_EXP.test(text)) return colorGroups(text, SCORE_EXP, () => 'green');
    if (SCORE_COINS.test(text)) return colorGroups(text, SCORE_COINS, () => 'green');
    if (SCORE_NEED.test(text)) return colorGroups(text, SCORE_NEED, () => 'green');
    if (SCORE_NAME.test(text)) return colorGroups(text, SCORE_NAME, (name) => (name === 'name' ? 'green' : null));
    return null;
  }
}

// Long descriptions of every mob and object in the shipped world, so a room's
// contents can be told apart by what they are rather than how they read.
export function loadWorldText(databaseFile = resolve(process.cwd(), '../website/static/data/database.json')): WorldText {
  const db = JSON.parse(readFileSync(databaseFile, 'utf8')) as {
    mobs: Record<string, { l?: string }>;
    items: Record<string, { l?: string }>;
  };
  const longs = (rows: Record<string, { l?: string }>) =>
    new Set(Object.values(rows).map((row) => row.l?.trim()).filter((l): l is string => Boolean(l)));
  return { mobLongs: longs(db.mobs), objLongs: longs(db.items) };
}
