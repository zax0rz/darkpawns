/**
 * Dark Pawns Interactive Database & Map Codex
 * Connects Mobs, Items, Rooms, and Zones from /data/database.json and /map/zone-*.json
 */

import { reflowWorldText } from '../lib/world-text.mjs';

const MOB_RACES = [
  'Human', 'Elf', 'Dwarf', 'Kender', 'Centaur', 'Rakshasa', 'Troll',
  'Lycanthrope', 'Vampire', 'Undead', 'Dragon', 'Demon', 'Horse', 'Reptile',
  'Arachnid', 'Rodent', 'Other', 'Vegetable', 'Giant', 'Demi-god', 'Ogre',
  'Insect', 'Mammal', 'Fish', 'Avian', 'Magical Construct', 'Amphibian',
  'Humanoid', 'Faery', 'Ssaur', 'Minotaur',
];

const AFFECT_BIT_NAMES = [
  'BLIND', 'INVIS', 'DET-ALIGN', 'DET-INVIS', 'DET-MAGIC',
  'SENSE-LIFE', 'WATERWALK', 'SANCT', 'GROUP', 'CURSE', 'INFRA',
  'POISON', 'PROT-EVIL', 'PROT-GOOD', 'SLEEP', '!TRACK',
  'FLESH-ALTER', 'DODGE', 'SNEAK', 'HIDE', 'BERSERK', 'CHARM',
  'FOLLOW', 'WIMPY', 'KUJI-KIRI', 'CUTTHROAT', 'FLY', 'WEREWOLF',
  'VAMPIRE', 'MOUNTED', 'INVULN', 'FLAMING', 'NOTHING', 'HASTE',
  'SLOW', 'DREAM', 'WATERBREATHE', 'METALSKIN', 'ROBBED',
];

const ATTACK_TYPES = [
  'hit', 'sting', 'whip', 'slash', 'bite', 'bludgeon', 'crush',
  'pound', 'claw', 'maul', 'thrash', 'pierce', 'blast', 'punch', 'stab',
];

const DRINK_TYPES = [
  'water', 'beer', 'wine', 'ale', 'dark ale', 'whiskey', 'lemonade',
  'firebreather', 'local speciality', 'slime mold juice', 'milk',
  'tea', 'coffee', 'blood', 'salt water', 'clear water',
];

const SPELL_NAMES = {
  1: 'holy ward', 2: 'shift reality', 3: 'bless', 4: 'blindness', 5: 'burning hands',
  6: 'call lightning', 7: 'charm person', 8: 'chill touch', 9: 'clone', 10: 'color spray',
  11: 'control weather', 12: 'create food', 13: 'create water', 14: 'cure blind', 15: 'cure critic',
  16: 'cure light', 17: 'curse', 18: 'detect alignment', 19: 'detect invisibility', 20: 'detect magic',
  21: 'detect poison', 22: 'dispel evil', 23: 'earthquake', 24: 'enchant weapon', 25: 'energy drain',
  26: 'fireball', 27: 'harm', 28: 'heal', 29: 'invisibility', 30: 'lightning bolt',
  31: 'locate object', 32: 'flame arrow', 33: 'poison', 34: 'protection from evil', 35: 'remove curse',
  36: 'sanctuary', 37: 'shocking grasp', 38: 'sleep', 39: 'strength', 40: 'summon',
  41: 'meteor swarm', 42: 'word of recall', 43: 'remove poison', 44: 'sense life', 45: 'animate dead',
  46: 'dispel good', 47: 'holy shield', 48: 'group heal', 49: 'group recall', 50: 'infravision',
  51: 'waterwalk', 52: 'mass heal', 53: 'fly', 54: 'lycanthropy', 55: 'vampirism',
  56: 'sobriety', 57: 'group invisibility', 58: 'hellfire', 59: 'enchant armor', 60: 'identify',
  61: 'mind poke', 62: 'mind blast', 63: 'chameleon', 64: 'levitate', 65: 'metalskin',
  66: 'globe of invulnerability', 67: 'vitality', 68: 'invigorate', 69: 'lesser perception', 70: 'greater perception',
  71: 'mind attack', 72: 'adrenaline boost', 73: 'psychic shield', 74: 'change density', 75: 'acid blast',
  76: 'dominate', 77: 'cell adjustment', 78: 'zen', 79: 'mirror image', 80: 'mass dominate',
  81: 'divine intervention', 82: 'mind bar', 83: 'soul leech', 84: 'mindsight', 85: 'transparency',
  86: 'know align', 87: 'gate', 88: 'word of intellect', 89: 'lay hands', 90: 'mental lapse',
  91: 'smokescreen', 92: 'ray of disruption', 93: 'disintegration', 94: 'calliope', 95: 'protection from good',
  96: 'flame strike', 97: 'haste', 98: 'slow', 99: 'dream travel', 100: 'psiblast',
  101: 'glyph of summoning', 102: 'waterbreathe', 105: 'conjure elemental',
  202: 'fire breath', 203: 'gas breath', 204: 'frost breath', 205: 'acid breath', 206: 'lightning breath',
};

function escapeHtml(value) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function renderProseBlock(text) {
  const { drawing, paragraphs } = reflowWorldText(text);
  if (!paragraphs.length) return '';
  if (drawing) {
    return `<pre class="sheet-prose is-drawing">${escapeHtml(paragraphs[0])}</pre>`;
  }
  return `<div class="sheet-prose">${paragraphs.map((p) => `<p>${escapeHtml(p)}</p>`).join('')}</div>`;
}

function calcDiceAvg(diceStr) {
  if (!diceStr) return 0;
  const m = String(diceStr).match(/^(\d+)d(\d+)([+-]\d+)?$/i);
  if (!m) return 0;
  const num = parseInt(m[1], 10);
  const sides = parseInt(m[2], 10);
  const plus = m[3] ? parseInt(m[3], 10) : 0;
  return Math.round(num * ((sides + 1) / 2) + plus);
}

function formatAffect(aff) {
  if (!aff) return '';
  if (aff.location === 'PERM_SPELL') {
    const bitName = AFFECT_BIT_NAMES[aff.modifier] || `BIT_${aff.modifier}`;
    return `Permanent ${bitName}`;
  }
  if (aff.location === 'RACE_HATE') {
    const rName = MOB_RACES[aff.modifier] || `Race ${aff.modifier}`;
    return `Hates ${rName}`;
  }
  const sign = aff.modifier >= 0 ? '+' : '';
  return `${sign}${aff.modifier} ${aff.location}`;
}

function describeItemValues(item) {
  const [v0, v1, v2, v3] = item.val || [0, 0, 0, 0];
  switch (item.type) {
    case 'WEAPON': {
      const avg = (((v2 + 1) / 2.0) * v1).toFixed(1);
      const atk = ATTACK_TYPES[v3] || `type ${v3}`;
      return `${v1}d${v2} (${avg} avg, ${atk})`;
    }
    case 'ARMOR':
      return `AC apply ${v0}`;
    case 'LIGHT':
      return v2 === -1 ? 'Infinite hours' : `${v2} hours`;
    case 'SCROLL':
    case 'POTION': {
      const spells = [v1, v2, v3]
        .filter((s) => s > 0)
        .map((s) => SPELL_NAMES[s] || `spell #${s}`);
      return `Level ${v0}: ${spells.length ? spells.join(', ') : 'none'}`;
    }
    case 'WAND':
    case 'STAFF': {
      const spell = v3 > 0 ? (SPELL_NAMES[v3] || `spell #${v3}`) : 'none';
      return `Level ${v0} ${spell} (${v2}/${v1} charges)`;
    }
    case 'CONTAINER':
      return `Capacity ${v0}${v2 > 0 ? `, key #${v2}` : ''}`;
    case 'DRINKCON':
    case 'FOUNTAIN': {
      const liq = DRINK_TYPES[v2] || `liquid #${v2}`;
      return `${v1}/${v0} units of ${liq}${v3 ? ' (poisoned)' : ''}`;
    }
    case 'FOOD':
      return `Fills ${v0} hours${v3 ? ' (poisoned)' : ''}`;
    case 'MONEY':
      return `${v0} coins`;
    default:
      return `[${v0}, ${v1}, ${v2}, ${v3}]`;
  }
}

document.addEventListener('DOMContentLoaded', () => {
  const dbPage = document.getElementById('db-page');
  if (!dbPage) return;

  // State
  let dbData = { mobs: {}, items: {} };
  let mobsList = [];
  let itemsList = [];
  let zonesMap = {};
  let zoneRanges = [];
  const zoneCache = new Map();

  let currentTab = 'mobs';
  let currentId = null;
  let searchQuery = '';
  let sortMode = 'zone';

  let filters = {
    zone: '',
    levelRange: '',
    alignment: '',
    mobRole: '',
    itemType: '',
    itemWear: '',
    itemAffect: '',
  };

  // DOM Elements
  const tabMobs = document.getElementById('tab-mobs');
  const tabItems = document.getElementById('tab-items');
  const countMobs = document.getElementById('count-mobs');
  const countItems = document.getElementById('count-items');
  const searchInput = document.getElementById('db-search');
  const clearBtn = document.getElementById('db-clear-btn');
  const sortSelect = document.getElementById('db-sort');
  const summaryEl = document.getElementById('db-result-summary');

  const filterZone = document.getElementById('filter-zone');
  const filterLevelRange = document.getElementById('filter-level-range');
  const filterAlignment = document.getElementById('filter-alignment');
  const filterMobRole = document.getElementById('filter-mob-role');
  const filterItemType = document.getElementById('filter-item-type');
  const filterItemWear = document.getElementById('filter-item-wear');
  const filterItemAffect = document.getElementById('filter-item-affect');

  const mobFilterFields = document.querySelectorAll('.mob-only-filter');
  const itemFilterFields = document.querySelectorAll('.item-only-filter');

  const resultsList = document.getElementById('db-results-list');
  const loadingEl = document.getElementById('db-loading');
  const emptyStateEl = document.getElementById('db-empty-state');
  const detailViewEl = document.getElementById('db-detail-view');
  const contentPane = document.getElementById('db-content');

  function getZoneForRecord(record) {
    if (record.spw && record.spw.length > 0 && record.spw[0].zone !== undefined && record.spw[0].zone !== -1) {
      return record.spw[0].zone;
    }
    if (record.rms && record.rms.length > 0 && record.rms[0].zone !== undefined && record.rms[0].zone !== -1) {
      return record.rms[0].zone;
    }
    // Check if carried by a mob whose spawn zone is known
    if (record.mobs && record.mobs.length > 0) {
      const m = dbData.mobs[record.mobs[0].mob_vnum];
      if (m && m.spw && m.spw.length > 0 && m.spw[0].zone !== -1) {
        return m.spw[0].zone;
      }
    }
    const v = record.v;
    for (const zr of zoneRanges) {
      if (v >= zr.min && v <= zr.max) return zr.id;
    }
    return Math.floor(v / 100);
  }

  function isMobAggressive(mob) {
    if (!mob.act || !Array.isArray(mob.act)) return false;
    return mob.act.some((f) => f === 'AGGR' || f.startsWith('AGGR'));
  }

  async function init() {
    try {
      const [dbResp, mapResp] = await Promise.all([
        fetch('/data/database.json'),
        fetch('/map/map-index.json').catch(() => null),
      ]);
      if (!dbResp.ok) throw new Error(`HTTP ${dbResp.status}`);
      dbData = await dbResp.json();

      if (mapResp && mapResp.ok) {
        const mapIdx = await mapResp.json();
        if (mapIdx && Array.isArray(mapIdx.zones)) {
          mapIdx.zones.forEach((z) => {
            zonesMap[z.id] = z.name;
            if (z.minVnum !== undefined && z.maxVnum !== undefined) {
              zoneRanges.push({ id: z.id, min: z.minVnum, max: z.maxVnum });
            }
          });
        }
      }

      mobsList = Object.values(dbData.mobs || {}).map((m) => ({
        ...m,
        _hpAvg: calcDiceAvg(m.hp),
        _dmgAvg: calcDiceAvg(m.dmg),
        _zone: getZoneForRecord(m),
      }));

      itemsList = Object.values(dbData.items || {}).map((it) => ({
        ...it,
        _zone: getZoneForRecord(it),
      }));

      if (countMobs) countMobs.textContent = mobsList.length.toLocaleString('en-US');
      if (countItems) countItems.textContent = itemsList.length.toLocaleString('en-US');

      populateSelectOptions();
      if (loadingEl) loadingEl.hidden = true;

      applyUrlState();
      renderList();

      window.addEventListener('hashchange', () => {
        applyHashOnly();
      });
    } catch (err) {
      console.error('Failed to load database:', err);
      if (loadingEl) {
        loadingEl.textContent = 'Error loading world database. Please refresh the page.';
      }
    }
  }

  function populateSelectOptions() {
    // Zones
    const zoneSet = new Set();
    mobsList.forEach((m) => {
      if (m._zone !== undefined && m._zone !== -1) zoneSet.add(m._zone);
      (m.spw || []).forEach((s) => {
        if (s.zone !== undefined && s.zone !== -1) zoneSet.add(s.zone);
      });
    });
    itemsList.forEach((it) => {
      if (it._zone !== undefined && it._zone !== -1) zoneSet.add(it._zone);
    });

    const sortedZones = Array.from(zoneSet).sort((a, b) => {
      const nameA = (zonesMap[a] || `Zone ${a}`).replace(/^\([^)]+\)\s*/, '');
      const nameB = (zonesMap[b] || `Zone ${b}`).replace(/^\([^)]+\)\s*/, '');
      return nameA.localeCompare(nameB);
    });

    sortedZones.forEach((zid) => {
      const opt = document.createElement('option');
      opt.value = String(zid);
      opt.textContent = `${zonesMap[zid] || `Zone ${zid}`} (#${zid})`;
      filterZone.appendChild(opt);
    });

    // Item Types
    const typeCounts = {};
    const wearCounts = {};
    const affectCounts = {};

    itemsList.forEach((it) => {
      if (it.type) typeCounts[it.type] = (typeCounts[it.type] || 0) + 1;
      (it.wear || []).forEach((w) => {
        if (w !== 'TAKE') wearCounts[w] = (wearCounts[w] || 0) + 1;
      });
      (it.aff || []).forEach((a) => {
        if (a.location && a.location !== 'NONE') {
          affectCounts[a.location] = (affectCounts[a.location] || 0) + 1;
        }
      });
    });

    Object.keys(typeCounts)
      .sort()
      .forEach((t) => {
        const opt = document.createElement('option');
        opt.value = t;
        opt.textContent = `${t} (${typeCounts[t]})`;
        filterItemType.appendChild(opt);
      });

    Object.keys(wearCounts)
      .sort()
      .forEach((w) => {
        const opt = document.createElement('option');
        opt.value = w;
        opt.textContent = `${w} (${wearCounts[w]})`;
        filterItemWear.appendChild(opt);
      });

    Object.keys(affectCounts)
      .sort()
      .forEach((a) => {
        const opt = document.createElement('option');
        opt.value = a;
        opt.textContent = `${a} (${affectCounts[a]})`;
        filterItemAffect.appendChild(opt);
      });
  }

  function updateSortOptionsForTab() {
    const currentVal = sortSelect.value;
    if (currentTab === 'mobs') {
      sortSelect.innerHTML = `
        <option value="zone">By Zone</option>
        <option value="lvl-desc">Highest Level</option>
        <option value="lvl-asc">Lowest Level</option>
        <option value="hp-desc">Highest Hit Points</option>
        <option value="name">Alphabetical</option>
        <option value="vnum">By VNUM</option>
      `;
    } else {
      sortSelect.innerHTML = `
        <option value="zone">By Zone</option>
        <option value="name">Alphabetical</option>
        <option value="cost-desc">Highest Value</option>
        <option value="wt-asc">Lightest Weight</option>
        <option value="vnum">By VNUM</option>
      `;
    }
    if (Array.from(sortSelect.options).some((o) => o.value === currentVal)) {
      sortSelect.value = currentVal;
    } else {
      sortSelect.value = 'zone';
      sortMode = 'zone';
    }
  }

  function switchTab(tab, preserveSelection = false) {
    if (tab !== 'mobs' && tab !== 'items') return;
    currentTab = tab;

    tabMobs.classList.toggle('active', tab === 'mobs');
    tabMobs.setAttribute('aria-selected', tab === 'mobs' ? 'true' : 'false');
    tabItems.classList.toggle('active', tab === 'items');
    tabItems.setAttribute('aria-selected', tab === 'items' ? 'true' : 'false');

    mobFilterFields.forEach((el) => {
      el.hidden = tab !== 'mobs';
    });
    itemFilterFields.forEach((el) => {
      el.hidden = tab !== 'items';
    });

    searchInput.placeholder =
      tab === 'mobs'
        ? 'Filter mobs by name, keyword, zone, or #VNUM...'
        : 'Filter items by name, type, stat affect, or #VNUM...';

    updateSortOptionsForTab();
    if (!preserveSelection) {
      renderList();
    }
  }

  function applyUrlState() {
    const params = new URLSearchParams(window.location.search);
    const qParam = params.get('q');
    const tabParam = params.get('tab');
    const zoneParam = params.get('zone');
    const mobParam = params.get('mob');
    const itemParam = params.get('item');

    if (qParam) {
      searchQuery = qParam.trim().toLowerCase();
      searchInput.value = qParam.trim();
    }
    if (zoneParam) {
      filters.zone = zoneParam;
      filterZone.value = zoneParam;
    }

    const hash = window.location.hash;
    if (hash.startsWith('#mob-')) {
      const vnum = parseInt(hash.replace('#mob-', ''), 10);
      switchTab('mobs', true);
      if (dbData.mobs[vnum]) selectRecord(vnum, 'mobs', false);
      return;
    }
    if (hash.startsWith('#item-')) {
      const vnum = parseInt(hash.replace('#item-', ''), 10);
      switchTab('items', true);
      if (dbData.items[vnum]) selectRecord(vnum, 'items', false);
      return;
    }
    if (mobParam && dbData.mobs[mobParam]) {
      switchTab('mobs', true);
      selectRecord(parseInt(mobParam, 10), 'mobs', false);
      return;
    }
    if (itemParam && dbData.items[itemParam]) {
      switchTab('items', true);
      selectRecord(parseInt(itemParam, 10), 'items', false);
      return;
    }
    if (tabParam === 'items' || tabParam === 'mobs') {
      switchTab(tabParam, true);
    }

    // Default selection on desktop so the right pane isn't blank
    if (window.innerWidth > 960) {
      const defaultMob = dbData.mobs[3005] ? 3005 : mobsList[0]?.v;
      if (currentTab === 'mobs' && defaultMob) {
        selectRecord(defaultMob, 'mobs', false);
      } else if (currentTab === 'items' && itemsList[0]) {
        selectRecord(itemsList[0].v, 'items', false);
      }
    }
  }

  function applyHashOnly() {
    const hash = window.location.hash;
    if (hash.startsWith('#mob-')) {
      const vnum = parseInt(hash.replace('#mob-', ''), 10);
      if (dbData.mobs[vnum]) {
        if (currentTab !== 'mobs') switchTab('mobs', true);
        selectRecord(vnum, 'mobs', false);
        renderList();
      }
    } else if (hash.startsWith('#item-')) {
      const vnum = parseInt(hash.replace('#item-', ''), 10);
      if (dbData.items[vnum]) {
        if (currentTab !== 'items') switchTab('items', true);
        selectRecord(vnum, 'items', false);
        renderList();
      }
    }
  }

  function hasActiveFilters() {
    return Boolean(
      searchQuery ||
        filters.zone ||
        filters.levelRange ||
        filters.alignment ||
        filters.mobRole ||
        filters.itemType ||
        filters.itemWear ||
        filters.itemAffect
    );
  }

  function resetAllFilters() {
    searchQuery = '';
    searchInput.value = '';
    filters = {
      zone: '',
      levelRange: '',
      alignment: '',
      mobRole: '',
      itemType: '',
      itemWear: '',
      itemAffect: '',
    };
    filterZone.value = '';
    filterLevelRange.value = '';
    filterAlignment.value = '';
    filterMobRole.value = '';
    filterItemType.value = '';
    filterItemWear.value = '';
    filterItemAffect.value = '';
    renderList();
  }

  function getFilteredRecords() {
    const source = currentTab === 'mobs' ? mobsList : itemsList;
    const q = searchQuery.replace(/^#/, '');

    const filtered = source.filter((r) => {
      // Search query match
      if (q) {
        const zName = (zonesMap[r._zone] || '').toLowerCase();
        const rcn = (r.rcn || '').toLowerCase();
        const typeStr = (r.type || '').toLowerCase();
        const affStr = (r.aff || [])
          .map((a) => (typeof a === 'string' ? a : a.location || ''))
          .join(' ')
          .toLowerCase();
        const matchText =
          (r.s && r.s.toLowerCase().includes(q)) ||
          (r.k && r.k.toLowerCase().includes(q)) ||
          String(r.v).includes(q) ||
          zName.includes(q) ||
          rcn.includes(q) ||
          typeStr.includes(q) ||
          affStr.includes(q);
        if (!matchText) return false;
      }

      // Zone filter
      if (filters.zone) {
        const targetZone = parseInt(filters.zone, 10);
        let inZone = r._zone === targetZone;
        if (!inZone && r.spw) {
          inZone = r.spw.some((s) => s.zone === targetZone);
        }
        if (!inZone && r.rms) {
          inZone = r.rms.some((rm) => rm.zone === targetZone);
        }
        if (!inZone) return false;
      }

      if (currentTab === 'mobs') {
        if (filters.levelRange) {
          const [minL, maxL] = filters.levelRange.split('-').map(Number);
          if (r.lvl < minL || r.lvl > maxL) return false;
        }
        if (filters.alignment) {
          if (filters.alignment === 'good' && r.alg < 300) return false;
          if (filters.alignment === 'evil' && r.alg > -300) return false;
          if (filters.alignment === 'neutral' && (r.alg >= 300 || r.alg <= -300)) return false;
        }
        if (filters.mobRole) {
          if (filters.mobRole === 'shop' && !r.shop) return false;
          if (filters.mobRole === 'aggr' && !isMobAggressive(r)) return false;
          if (filters.mobRole === 'drops' && (!r.drp || r.drp.length === 0)) return false;
          if (filters.mobRole === 'script' && !r.scr) return false;
        }
      } else {
        if (filters.itemType && r.type !== filters.itemType) return false;
        if (filters.itemWear && (!r.wear || !r.wear.includes(filters.itemWear))) return false;
        if (filters.itemAffect && (!r.aff || !r.aff.some((a) => a.location === filters.itemAffect))) return false;
      }

      return true;
    });

    filtered.sort((a, b) => {
      if (sortMode === 'zone') {
        if (a._zone !== b._zone) return a._zone - b._zone;
        return a.v - b.v;
      }
      if (sortMode === 'vnum') return a.v - b.v;
      if (sortMode === 'name') return (a.s || '').localeCompare(b.s || '');
      if (currentTab === 'mobs') {
        if (sortMode === 'lvl-desc') return b.lvl - a.lvl || a.v - b.v;
        if (sortMode === 'lvl-asc') return a.lvl - b.lvl || a.v - b.v;
        if (sortMode === 'hp-desc') return b._hpAvg - a._hpAvg || b.lvl - a.lvl;
      } else {
        if (sortMode === 'cost-desc') return b.cst - a.cst || a.v - b.v;
        if (sortMode === 'wt-asc') return a.wt - b.wt || a.v - b.v;
      }
      return a.v - b.v;
    });

    return filtered;
  }

  function renderList() {
    const filtered = getFilteredRecords();
    const total = currentTab === 'mobs' ? mobsList.length : itemsList.length;

    clearBtn.hidden = !hasActiveFilters();
    summaryEl.textContent =
      filtered.length === total
        ? `Showing all ${total.toLocaleString('en-US')} ${currentTab}`
        : `Showing ${filtered.length.toLocaleString('en-US')} of ${total.toLocaleString('en-US')} ${currentTab}`;

    resultsList.innerHTML = '';

    if (filtered.length === 0) {
      const emptyLi = document.createElement('li');
      emptyLi.style.padding = '1.5rem 1rem';
      emptyLi.style.fontFamily = 'var(--font-mono)';
      emptyLi.style.fontSize = 'var(--text-xs)';
      emptyLi.style.color = 'var(--ink-muted)';
      emptyLi.textContent = 'No matching records found.';
      resultsList.appendChild(emptyLi);
      return;
    }

    const fragment = document.createDocumentFragment();
    let lastZone = null;

    filtered.forEach((r) => {
      if (sortMode === 'zone' && r._zone !== lastZone) {
        lastZone = r._zone;
        const zHeader = document.createElement('li');
        zHeader.className = 'db-zone-divider';
        const zTitle = zonesMap[lastZone] || `Zone #${lastZone}`;
        zHeader.innerHTML = `<span>${escapeHtml(zTitle)}</span><span>#${lastZone}</span>`;
        fragment.appendChild(zHeader);
      }

      const li = document.createElement('li');
      li.className = 'db-list-item';
      li.setAttribute('role', 'option');
      const isSelected = currentId === r.v;
      if (isSelected) {
        li.classList.add('active');
        li.setAttribute('aria-selected', 'true');
      } else {
        li.setAttribute('aria-selected', 'false');
      }

      if (currentTab === 'mobs') {
        const alignTag = r.alg >= 300 ? 'Good' : r.alg <= -300 ? 'Evil' : 'Neutral';
        const raceStr = r.rcn || MOB_RACES[r.rc] || '';
        const badges = [];
        if (r.shop) badges.push('<span class="db-mini-tag tag-accent">Shop</span>');
        if (isMobAggressive(r)) badges.push('<span class="db-mini-tag tag-accent">Aggr</span>');
        if (r.drp && r.drp.length > 0) badges.push(`<span class="db-mini-tag">${r.drp.length} eq</span>`);

        li.innerHTML = `
          <div class="db-item-row-top">
            <span class="db-item-title">${escapeHtml(r.s || r.k)}</span>
            <span class="db-item-vnum">#${r.v}</span>
          </div>
          <div class="db-item-row-meta">
            <span>Lvl ${r.lvl} · ${escapeHtml(raceStr)} · ${alignTag}</span>
            <span class="db-item-badges">${badges.join('')}</span>
          </div>
        `;
      } else {
        const wearSlots = (r.wear || []).filter((w) => w !== 'TAKE').slice(0, 2).join(', ');
        const badges = [];
        if (r.type === 'WEAPON' && r.val) {
          badges.push(`<span class="db-mini-tag">${r.val[1]}d${r.val[2]}</span>`);
        } else if (r.type === 'ARMOR' && r.val) {
          badges.push(`<span class="db-mini-tag">AC ${r.val[0]}</span>`);
        }
        if (r.aff && r.aff.length > 0) {
          const firstAff = formatAffect(r.aff[0]);
          badges.push(`<span class="db-mini-tag tag-accent">${escapeHtml(firstAff)}${r.aff.length > 1 ? ` +${r.aff.length - 1}` : ''}</span>`);
        }

        li.innerHTML = `
          <div class="db-item-row-top">
            <span class="db-item-title">${escapeHtml(r.s || r.k)}</span>
            <span class="db-item-vnum">#${r.v}</span>
          </div>
          <div class="db-item-row-meta">
            <span>${escapeHtml(r.type)}${wearSlots ? ` · ${escapeHtml(wearSlots)}` : ''}</span>
            <span class="db-item-badges">${badges.join('')}</span>
          </div>
        `;
      }

      li.addEventListener('click', () => {
        selectRecord(r.v, currentTab, true);
        resultsList.querySelectorAll('.db-list-item.active').forEach((el) => {
          el.classList.remove('active');
          el.setAttribute('aria-selected', 'false');
        });
        li.classList.add('active');
        li.setAttribute('aria-selected', 'true');
      });

      fragment.appendChild(li);
    });

    resultsList.appendChild(fragment);
  }

  function selectRecord(vnum, tab, updateHash = true) {
    currentId = vnum;
    const record = tab === 'mobs' ? dbData.mobs[vnum] : dbData.items[vnum];
    if (!record) return;

    if (updateHash) {
      const prefix = tab === 'mobs' ? 'mob' : 'item';
      history.replaceState(null, '', `#${prefix}-${vnum}`);
    }

    emptyStateEl.hidden = true;
    detailViewEl.hidden = false;
    dbPage.classList.add('mobile-detail-open');

    if (tab === 'mobs') {
      renderMobSheet(record);
    } else {
      renderItemSheet(record);
    }

    if (contentPane) contentPane.scrollTop = 0;
    if (window.innerWidth <= 960) window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  async function renderZoneMiniMap(containerId, zoneId, highlightRoomIds) {
    const container = document.getElementById(containerId);
    if (!container || zoneId === undefined || zoneId === -1) return;

    try {
      let zoneData = zoneCache.get(zoneId);
      if (!zoneData) {
        const resp = await fetch(`/map/zone-${zoneId}.json`);
        if (!resp.ok) throw new Error(`Zone ${zoneId} not found`);
        zoneData = await resp.json();
        zoneCache.set(zoneId, zoneData);
      }

      const rooms = zoneData.rooms || [];
      if (rooms.length === 0) {
        container.innerHTML = '<span class="xref-sub">No room coordinates in this zone.</span>';
        return;
      }

      const highlightSet = new Set(highlightRoomIds);
      const roomMap = new Map(rooms.map((rm) => [rm.id, rm]));

      // If the zone is very wide/tall and we have highlighted rooms in it,
      // crop the viewBox to a generous neighborhood around the highlighted rooms
      // so the local room topology and oxblood markers are crisp and legible.
      const highlightedNodes = rooms.filter((rm) => highlightSet.has(rm.id));
      let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;

      let fullMinX = Infinity, fullMaxX = -Infinity, fullMinY = Infinity, fullMaxY = -Infinity;
      rooms.forEach((rm) => {
        const x = rm.x || 0;
        const y = rm.y || 0;
        if (x < fullMinX) fullMinX = x;
        if (x > fullMaxX) fullMaxX = x;
        if (y < fullMinY) fullMinY = y;
        if (y > fullMaxY) fullMaxY = y;
      });

      const fullSpan = Math.max(fullMaxX - fullMinX, fullMaxY - fullMinY);
      if (highlightedNodes.length > 0 && fullSpan > 420) {
        let hMinX = Infinity, hMaxX = -Infinity, hMinY = Infinity, hMaxY = -Infinity;
        highlightedNodes.forEach((rm) => {
          const x = rm.x || 0;
          const y = rm.y || 0;
          if (x < hMinX) hMinX = x;
          if (x > hMaxX) hMaxX = x;
          if (y < hMinY) hMinY = y;
          if (y > hMaxY) hMaxY = y;
        });
        const neighborhoodMargin = 160;
        minX = Math.max(fullMinX, hMinX - neighborhoodMargin);
        maxX = Math.min(fullMaxX, hMaxX + neighborhoodMargin);
        minY = Math.max(fullMinY, hMinY - neighborhoodMargin);
        maxY = Math.min(fullMaxY, hMaxY + neighborhoodMargin);
      } else {
        minX = fullMinX;
        maxX = fullMaxX;
        minY = fullMinY;
        maxY = fullMaxY;
      }

      const pad = 28;
      const rawW = maxX - minX + pad * 2;
      const rawH = maxY - minY + pad * 2;
      const width = Math.max(140, rawW);
      const height = Math.max(95, rawH);
      const vx = minX - pad - (width - rawW) / 2;
      const vy = minY - pad - (height - rawH) / 2;

      // Scale node radius and stroke width proportionally to viewBox span
      const scale = Math.max(1, Math.min(width, height * 1.6) / 180);
      const rHighlight = (6.5 * scale).toFixed(1);
      const rNormal = (3.4 * scale).toFixed(1);
      const lineStroke = (1.5 * scale).toFixed(1);
      const ringStroke = (1.6 * scale).toFixed(1);

      const lines = [];
      rooms.forEach((rm) => {
        const x1 = rm.x || 0;
        const y1 = rm.y || 0;
        (rm.exits || []).forEach((ex) => {
          const target = roomMap.get(ex.t);
          if (target && target.id > rm.id) {
            lines.push(
              `<line x1="${x1}" y1="${y1}" x2="${target.x || 0}" y2="${target.y || 0}" stroke="#d8ceb8" stroke-width="${lineStroke}" />`
            );
          }
        });
      });

      const circles = rooms.map((rm) => {
        const isHighlighted = highlightSet.has(rm.id);
        const r = isHighlighted ? rHighlight : rNormal;
        const fill = isHighlighted ? '#a8201a' : '#766a5a';
        const stroke = isHighlighted ? '#f3ede0' : 'none';
        const sw = isHighlighted ? ringStroke : '0';
        return `<a href="/map/?room=${rm.id}" title="${escapeHtml(rm.name)} (#${rm.id})"><circle cx="${rm.x || 0}" cy="${rm.y || 0}" r="${r}" fill="${fill}" stroke="${stroke}" stroke-width="${sw}" /></a>`;
      });

      // Draw highlighted nodes last so they sit on top
      const normalCircles = [];
      const activeCircles = [];
      rooms.forEach((rm, idx) => {
        if (highlightSet.has(rm.id)) activeCircles.push(circles[idx]);
        else normalCircles.push(circles[idx]);
      });

      container.innerHTML = `
        <svg viewBox="${vx} ${vy} ${width} ${height}" preserveAspectRatio="xMidYMid meet" role="img" aria-label="Zone ${zoneId} map preview">
          <g>${lines.join('')}</g>
          <g>${normalCircles.join('')}</g>
          <g>${activeCircles.join('')}</g>
        </svg>
      `;
    } catch {
      container.innerHTML = '<span class="xref-sub">Zone map preview unavailable.</span>';
    }
  }

  function renderMobSheet(mob) {
    const sexStr = mob.sex === 1 ? 'Male' : mob.sex === 2 ? 'Female' : 'Neutral';
    const alignStr =
      mob.alg >= 300 ? `Good (${mob.alg})` : mob.alg <= -300 ? `Evil (${mob.alg})` : `Neutral (${mob.alg})`;
    const raceStr = mob.rcn || MOB_RACES[mob.rc] || `Race ${mob.rc}`;
    const thac0 = mob.thac0 !== undefined ? mob.thac0 : 20;
    const hitroll = 20 - thac0;
    const hitrollStr = hitroll >= 0 ? `+${hitroll}` : String(hitroll);
    const hpAvg = calcDiceAvg(mob.hp);
    const dmgAvg = calcDiceAvg(mob.dmg);
    const atkName = mob.atk || ATTACK_TYPES[mob.bha || 0] || 'hit';
    const stats = mob.stat || [11, 11, 11, 11, 11, 11];
    const statLabels = ['STR', 'INT', 'WIS', 'DEX', 'CON', 'CHA'];

    const actionFlags = mob.act || [];
    const affectFlags = mob.aff || [];
    const spawns = mob.spw || [];
    const drops = mob.drp || [];
    const primaryZone = spawns.length > 0 ? spawns[0].zone : mob._zone;
    const primaryZoneName = zonesMap[primaryZone] || `Zone #${primaryZone}`;
    const spawnRoomIds = spawns.map((s) => s.room);

    detailViewEl.innerHTML = `
      <article class="sheet">
        <button type="button" class="sheet-mobile-back" id="sheet-back-btn">← Back to mob list</button>

        <header class="sheet-header">
          <div class="sheet-meta-bar">
            <span class="sheet-eyebrow">Mob Record · VNUM #${mob.v} · ${escapeHtml(primaryZoneName)}</span>
            <div class="sheet-actions">
              <a href="/mobs/${mob.v}/">Permanent page</a>
              <span>·</span>
              <a href="/mobs/${mob.v}.json">JSON</a>
              <span>·</span>
              <button type="button" id="copy-record-link">Copy link</button>
            </div>
          </div>
          <h2 class="sheet-title">${escapeHtml(mob.s || mob.k)}</h2>
          ${mob.l ? `<p class="sheet-lede">${escapeHtml(mob.l)}</p>` : ''}
          <div class="sheet-keywords">Keywords: <code>${escapeHtml(mob.k)}</code></div>
        </header>

        <div class="sheet-grid">
          <!-- Left Column: Look Description + Complete Combat & Identity Stats -->
          <div>
            ${
              mob.d
                ? `
              <section class="sheet-section">
                <h2 class="sheet-section-title">Look Description</h2>
                ${renderProseBlock(mob.d)}
              </section>
            `
                : ''
            }

            <section class="sheet-section">
              <h2 class="sheet-section-title">Combat &amp; Identity Stats</h2>
              <dl class="sheet-dl">
                <dt>VNUM</dt>
                <dd>#${mob.v}</dd>
                <dt>Level</dt>
                <dd>${mob.lvl}</dd>
                <dt>Race</dt>
                <dd>${escapeHtml(raceStr)} (${mob.rc})</dd>
                <dt>Sex</dt>
                <dd>${sexStr}</dd>
                <dt>Alignment</dt>
                <dd>${alignStr}</dd>
                <dt>Hit Points</dt>
                <dd>${mob.hp} (${hpAvg.toLocaleString('en-US')} avg)</dd>
                <dt>Barehand Damage</dt>
                <dd>${mob.dmg} (${dmgAvg} avg · ${escapeHtml(atkName)})</dd>
                <dt>THAC0 / Hitroll</dt>
                <dd>${thac0} (${hitrollStr} hitroll)</dd>
                <dt>Armor Class</dt>
                <dd>${mob.ac} (${mob.ac / 10}/10)</dd>
                <dt>Gold Carried</dt>
                <dd>${mob.gld.toLocaleString('en-US')} coins</dd>
                <dt>Experience</dt>
                <dd>${mob.exp.toLocaleString('en-US')} xp</dd>
                <dt>Position</dt>
                <dd>${escapeHtml(mob.pos || 'Standing')}${mob.dpos && mob.dpos !== mob.pos ? ` (default: ${escapeHtml(mob.dpos)})` : ''}</dd>
                ${mob.scr ? `<dt>Lua Script</dt><dd>${escapeHtml(mob.scr)}</dd>` : ''}
              </dl>

              <div class="attr-strip" aria-label="Base Attributes">
                ${statLabels
                  .map(
                    (lbl, idx) => `
                  <div class="attr-cell">
                    <span class="attr-label">${lbl}</span>
                    <span class="attr-val">${stats[idx] ?? 11}</span>
                  </div>
                `
                  )
                  .join('')}
              </div>
            </section>

            <section class="sheet-section">
              <h2 class="sheet-section-title">Action &amp; Affect Flags</h2>
              ${
                actionFlags.length > 0 || affectFlags.length > 0
                  ? `
                <ul class="sheet-tags">
                  ${actionFlags.map((flg) => `<li class="sheet-tag">${escapeHtml(flg)}</li>`).join('')}
                  ${affectFlags.map((flg) => `<li class="sheet-tag tag-affect">${escapeHtml(flg)}</li>`).join('')}
                </ul>
              `
                  : '<p class="xref-empty">No special action or affect flags set.</p>'
              }
            </section>

            ${
              mob.noise
                ? `
              <section class="sheet-section">
                <h2 class="sheet-section-title">Ambient Room Noise</h2>
                ${renderProseBlock(mob.noise)}
              </section>
            `
                : ''
            }
          </div>

          <!-- Right Column: Map Spawns, Equipment Resets, & Shop Inventory -->
          <div>
            <section class="sheet-section">
              <h2 class="sheet-section-title">
                <span>World Map Spawns</span>
                <small>${spawns.length} ${spawns.length === 1 ? 'room' : 'rooms'}</small>
              </h2>

              ${
                spawns.length > 0
                  ? `
                <div class="minimap-card">
                  <div class="minimap-bar">
                    <span class="minimap-zone-title">${escapeHtml(primaryZoneName)}</span>
                    <a href="/map/?room=${spawns[0].room}">Open interactive map →</a>
                  </div>
                  <div class="minimap-stage" id="mob-minimap-stage">
                    <span class="xref-sub">Loading zone map...</span>
                  </div>
                  <div class="minimap-caption">
                    <span>Oxblood nodes mark spawn rooms in this zone</span>
                    <a href="/zones/${primaryZone}/">Zone #${primaryZone} record</a>
                  </div>
                </div>

                <ul class="xref-list">
                  ${spawns
                    .map((s) => {
                      const zTitle = zonesMap[s.zone] || `Zone #${s.zone}`;
                      return `
                      <li class="xref-item">
                        <div class="xref-main">
                          <a class="xref-link" href="/map/?room=${s.room}">${escapeHtml(s.name || `Room #${s.room}`)}</a>
                          <span class="xref-sub">
                            <a href="/rooms/${s.room}/">Room #${s.room}</a> · <a href="/zones/${s.zone}/">${escapeHtml(zTitle)}</a>
                          </span>
                        </div>
                        <div class="xref-actions">
                          <a class="xref-pill" href="/map/?room=${s.room}">Show on Map</a>
                        </div>
                      </li>
                    `;
                    })
                    .join('')}
                </ul>
              `
                  : '<p class="xref-empty">This mob does not spawn via standard zone reset rules.</p>'
              }
            </section>

            <section class="sheet-section">
              <h2 class="sheet-section-title">
                <span>Equipped &amp; Carried Items</span>
                <small>${drops.length} ${drops.length === 1 ? 'item' : 'items'}</small>
              </h2>
              ${
                drops.length > 0
                  ? `
                <ul class="xref-list">
                  ${drops
                    .map((d) => {
                      const fullItem = dbData.items[d.obj_vnum];
                      const itemSummary = fullItem ? `${fullItem.type} · ${describeItemValues(fullItem)}` : '';
                      const itemAffects =
                        fullItem && fullItem.aff && fullItem.aff.length
                          ? ' · ' + fullItem.aff.map(formatAffect).join(', ')
                          : '';
                      return `
                      <li class="xref-item">
                        <div class="xref-main">
                          <button type="button" class="xref-link js-open-item" data-vnum="${d.obj_vnum}">
                            ${escapeHtml(d.name)}
                          </button>
                          <span class="xref-sub">
                            #${d.obj_vnum} · ${escapeHtml(d.slot)}${itemSummary ? ` · ${escapeHtml(itemSummary)}` : ''}${escapeHtml(itemAffects)}
                          </span>
                        </div>
                        <div class="xref-actions">
                          <button type="button" class="xref-pill js-open-item" data-vnum="${d.obj_vnum}">Inspect Item</button>
                        </div>
                      </li>
                    `;
                    })
                    .join('')}
                </ul>
              `
                  : '<p class="xref-empty">No equipment or inventory items assigned in zone resets.</p>'
              }
            </section>

            ${
              mob.shop
                ? `
              <section class="sheet-section">
                <h2 class="sheet-section-title">
                  <span>Shopkeeper Inventory</span>
                  <small>Open ${escapeHtml(mob.shop.open_hours)} · Sell ×${mob.shop.sell_mult}</small>
                </h2>
                <ul class="xref-list">
                  ${(mob.shop.items_sold || [])
                    .map((si) => {
                      const fullItem = dbData.items[si.vnum];
                      const price = fullItem ? Math.floor(fullItem.cst * mob.shop.sell_mult) : null;
                      return `
                      <li class="xref-item">
                        <div class="xref-main">
                          <button type="button" class="xref-link js-open-item" data-vnum="${si.vnum}">
                            ${escapeHtml(si.name)}
                          </button>
                          <span class="xref-sub">
                            #${si.vnum}${fullItem ? ` · ${escapeHtml(fullItem.type)}` : ''}${price !== null ? ` · ${price.toLocaleString('en-US')} coins` : ''}
                          </span>
                        </div>
                        <div class="xref-actions">
                          <button type="button" class="xref-pill js-open-item" data-vnum="${si.vnum}">Inspect Item</button>
                        </div>
                      </li>
                    `;
                    })
                    .join('')}
                </ul>
              </section>
            `
                : ''
            }
          </div>
        </div>
      </article>
    `;

    bindSheetEvents();
    if (spawns.length > 0 && primaryZone !== undefined && primaryZone !== -1) {
      renderZoneMiniMap('mob-minimap-stage', primaryZone, spawnRoomIds);
    }
  }

  function renderItemSheet(item) {
    const wearFlags = item.wear || [];
    const extraFlags = item.extra || [];
    const affects = item.aff || [];
    const edescs = item.edesc || [];
    const loaders = item.mobs || [];
    const placements = item.rms || [];
    const shops = item.shp || [];
    const decodedProps = describeItemValues(item);

    // Collect all rooms where this item can be found (via mob carriers, ground placements, or shopkeepers)
    const huntRooms = [];
    let primaryZone = item._zone;

    placements.forEach((p) => {
      if (p.room !== undefined) {
        huntRooms.push(p.room);
        if (p.zone !== undefined && p.zone !== -1) primaryZone = p.zone;
      }
    });

    loaders.forEach((l) => {
      const m = dbData.mobs[l.mob_vnum];
      if (m && m.spw && m.spw.length > 0) {
        m.spw.forEach((s) => huntRooms.push(s.room));
        if (m.spw[0].zone !== undefined && m.spw[0].zone !== -1) {
          primaryZone = m.spw[0].zone;
        }
      }
    });

    const primaryZoneName = zonesMap[primaryZone] || `Zone #${primaryZone}`;

    detailViewEl.innerHTML = `
      <article class="sheet">
        <button type="button" class="sheet-mobile-back" id="sheet-back-btn">← Back to item list</button>

        <header class="sheet-header">
          <div class="sheet-meta-bar">
            <span class="sheet-eyebrow">Item Record · VNUM #${item.v} · ${escapeHtml(item.type)}</span>
            <div class="sheet-actions">
              <a href="/items/${item.v}/">Permanent page</a>
              <span>·</span>
              <a href="/items/${item.v}.json">JSON</a>
              <span>·</span>
              <button type="button" id="copy-record-link">Copy link</button>
            </div>
          </div>
          <h2 class="sheet-title">${escapeHtml(item.s || item.k)}</h2>
          ${item.l ? `<p class="sheet-lede">${escapeHtml(item.l)}</p>` : ''}
          <div class="sheet-keywords">Keywords: <code>${escapeHtml(item.k)}</code></div>
        </header>

        <div class="sheet-grid">
          <!-- Left Column: Complete Object Specifications, Affects & Flags -->
          <div>
            <section class="sheet-section">
              <h2 class="sheet-section-title">Object Specifications</h2>
              <dl class="sheet-dl">
                <dt>VNUM</dt>
                <dd>#${item.v}</dd>
                <dt>Item Type</dt>
                <dd>${escapeHtml(item.type)}</dd>
                <dt>Type Properties</dt>
                <dd>${escapeHtml(decodedProps)}</dd>
                <dt>Wear Flags</dt>
                <dd>${escapeHtml(wearFlags.join(', ') || 'None')}</dd>
                <dt>Weight</dt>
                <dd>${item.wt} lbs</dd>
                <dt>Base Value</dt>
                <dd>${item.cst.toLocaleString('en-US')} coins</dd>
                <dt>Load Probability</dt>
                <dd>${item.load}%</dd>
                <dt>Raw Values [0..3]</dt>
                <dd>[${(item.val || [0, 0, 0, 0]).join(', ')}]</dd>
                ${item.scr ? `<dt>Lua Script</dt><dd>${escapeHtml(item.scr)}</dd>` : ''}
              </dl>
            </section>

            <section class="sheet-section">
              <h2 class="sheet-section-title">Magical Affects</h2>
              ${
                affects.length > 0
                  ? `
                <ul class="sheet-tags">
                  ${affects.map((a) => `<li class="sheet-tag tag-affect">${escapeHtml(formatAffect(a))}</li>`).join('')}
                </ul>
              `
                  : '<p class="xref-empty">No magical stat modifiers on this item.</p>'
              }
            </section>

            <section class="sheet-section">
              <h2 class="sheet-section-title">Extra Flags</h2>
              ${
                extraFlags.length > 0
                  ? `
                <ul class="sheet-tags">
                  ${extraFlags.map((flg) => `<li class="sheet-tag">${escapeHtml(flg)}</li>`).join('')}
                </ul>
              `
                  : '<p class="xref-empty">No extra restriction or property flags.</p>'
              }
            </section>

            ${
              item.ad
                ? `
              <section class="sheet-section">
                <h2 class="sheet-section-title">Action Description</h2>
                ${renderProseBlock(item.ad)}
              </section>
            `
                : ''
            }

            ${
              edescs.length > 0
                ? `
              <section class="sheet-section">
                <h2 class="sheet-section-title">Extra Descriptions</h2>
                ${edescs
                  .map(
                    (ed) => `
                  <div style="margin-bottom: var(--space-sm);">
                    <div class="xref-sub" style="margin-bottom: 0.2rem;">Keywords: <code>${escapeHtml(ed.keywords)}</code></div>
                    ${renderProseBlock(ed.desc)}
                  </div>
                `
                  )
                  .join('')}
              </section>
            `
                : ''
            }
          </div>

          <!-- Right Column: Where to Find This Item (Mobs, Map Rooms, Containers, Shops) -->
          <div>
            ${
              huntRooms.length > 0 && primaryZone !== undefined && primaryZone !== -1
                ? `
              <section class="sheet-section">
                <h2 class="sheet-section-title">
                  <span>Zone Map Preview</span>
                  <small>${escapeHtml(primaryZoneName)}</small>
                </h2>
                <div class="minimap-card">
                  <div class="minimap-bar">
                    <span class="minimap-zone-title">${escapeHtml(primaryZoneName)}</span>
                    <a href="/map/?room=${huntRooms[0]}">Open interactive map →</a>
                  </div>
                  <div class="minimap-stage" id="item-minimap-stage">
                    <span class="xref-sub">Loading zone map...</span>
                  </div>
                  <div class="minimap-caption">
                    <span>Oxblood nodes mark rooms where this item loads</span>
                    <a href="/zones/${primaryZone}/">Zone #${primaryZone} record</a>
                  </div>
                </div>
              </section>
            `
                : ''
            }

            <section class="sheet-section">
              <h2 class="sheet-section-title">
                <span>Loaded / Carried by Mobs</span>
                <small>${loaders.length} ${loaders.length === 1 ? 'mob' : 'mobs'}</small>
              </h2>
              ${
                loaders.length > 0
                  ? `
                <ul class="xref-list">
                  ${loaders
                    .map((l) => {
                      const fullMob = dbData.mobs[l.mob_vnum];
                      const firstSpawn = fullMob && fullMob.spw && fullMob.spw.length > 0 ? fullMob.spw[0] : null;
                      const zName = firstSpawn ? zonesMap[firstSpawn.zone] || `Zone #${firstSpawn.zone}` : '';
                      return `
                      <li class="xref-item">
                        <div class="xref-main">
                          <button type="button" class="xref-link js-open-mob" data-vnum="${l.mob_vnum}">
                            ${escapeHtml(l.name)}
                          </button>
                          <span class="xref-sub">
                            Mob #${l.mob_vnum}${fullMob ? ` (Lvl ${fullMob.lvl})` : ''} · ${escapeHtml(l.slot)}
                            ${
                              firstSpawn
                                ? ` · Spawns in <a href="/map/?room=${firstSpawn.room}">${escapeHtml(firstSpawn.name || `Room #${firstSpawn.room}`)}</a> (${escapeHtml(zName)})`
                                : ''
                            }
                          </span>
                        </div>
                        <div class="xref-actions">
                          <button type="button" class="xref-pill js-open-mob" data-vnum="${l.mob_vnum}">Inspect Mob</button>
                          ${firstSpawn ? `<a class="xref-pill" href="/map/?room=${firstSpawn.room}">Map</a>` : ''}
                        </div>
                      </li>
                    `;
                    })
                    .join('')}
                </ul>
              `
                  : '<p class="xref-empty">No mobs load with this item in the zone reset files.</p>'
              }
            </section>

            <section class="sheet-section">
              <h2 class="sheet-section-title">
                <span>Room &amp; Container Placements</span>
                <small>${placements.length}</small>
              </h2>
              ${
                placements.length > 0
                  ? `
                <ul class="xref-list">
                  ${placements
                    .map((p) => {
                      if (p.container_vnum !== undefined) {
                        return `
                        <li class="xref-item">
                          <div class="xref-main">
                            <button type="button" class="xref-link js-open-item" data-vnum="${p.container_vnum}">
                              Inside: ${escapeHtml(p.name)}
                            </button>
                            <span class="xref-sub">Container Item #${p.container_vnum}</span>
                          </div>
                          <div class="xref-actions">
                            <button type="button" class="xref-pill js-open-item" data-vnum="${p.container_vnum}">Inspect Container</button>
                          </div>
                        </li>
                      `;
                      }
                      const zName = zonesMap[p.zone] || `Zone #${p.zone}`;
                      return `
                        <li class="xref-item">
                          <div class="xref-main">
                            <a class="xref-link" href="/map/?room=${p.room}">${escapeHtml(p.name || `Room #${p.room}`)}</a>
                            <span class="xref-sub">
                              <a href="/rooms/${p.room}/">Room #${p.room}</a> · <a href="/zones/${p.zone}/">${escapeHtml(zName)}</a>
                            </span>
                          </div>
                          <div class="xref-actions">
                            <a class="xref-pill" href="/map/?room=${p.room}">Show on Map</a>
                          </div>
                        </li>
                      `;
                    })
                    .join('')}
                </ul>
              `
                  : '<p class="xref-empty">Not placed directly on the ground or inside a container.</p>'
              }
            </section>

            ${
              shops.length > 0
                ? `
              <section class="sheet-section">
                <h2 class="sheet-section-title">
                  <span>Sold by Shopkeepers</span>
                  <small>${shops.length}</small>
                </h2>
                <ul class="xref-list">
                  ${shops
                    .map((sh) => {
                      const keeper = dbData.mobs[sh.keeper_vnum];
                      const firstSpawn = keeper && keeper.spw && keeper.spw.length > 0 ? keeper.spw[0] : null;
                      return `
                      <li class="xref-item">
                        <div class="xref-main">
                          <button type="button" class="xref-link js-open-mob" data-vnum="${sh.keeper_vnum}">
                            ${escapeHtml(sh.keeper_name)}
                          </button>
                          <span class="xref-sub">
                            Mob #${sh.keeper_vnum} · Price: ${sh.price.toLocaleString('en-US')} coins
                            ${firstSpawn ? ` · <a href="/map/?room=${firstSpawn.room}">${escapeHtml(firstSpawn.name)}</a>` : ''}
                          </span>
                        </div>
                        <div class="xref-actions">
                          <button type="button" class="xref-pill js-open-mob" data-vnum="${sh.keeper_vnum}">Shopkeeper</button>
                          ${firstSpawn ? `<a class="xref-pill" href="/map/?room=${firstSpawn.room}">Map</a>` : ''}
                        </div>
                      </li>
                    `;
                    })
                    .join('')}
                </ul>
              </section>
            `
                : ''
            }
          </div>
        </div>
      </article>
    `;

    bindSheetEvents();
    if (huntRooms.length > 0 && primaryZone !== undefined && primaryZone !== -1) {
      renderZoneMiniMap('item-minimap-stage', primaryZone, huntRooms);
    }
  }

  function bindSheetEvents() {
    const backBtn = document.getElementById('sheet-back-btn');
    if (backBtn) {
      backBtn.addEventListener('click', () => {
        dbPage.classList.remove('mobile-detail-open');
        window.scrollTo({ top: 0, behavior: 'smooth' });
      });
    }

    const copyBtn = document.getElementById('copy-record-link');
    if (copyBtn) {
      copyBtn.addEventListener('click', () => {
        navigator.clipboard.writeText(window.location.href).then(() => {
          copyBtn.textContent = 'Copied';
          setTimeout(() => {
            copyBtn.textContent = 'Copy link';
          }, 1500);
        });
      });
    }

    detailViewEl.querySelectorAll('.js-open-item').forEach((btn) => {
      btn.addEventListener('click', () => {
        const vnum = parseInt(btn.getAttribute('data-vnum'), 10);
        if (dbData.items[vnum]) {
          if (currentTab !== 'items') switchTab('items', true);
          selectRecord(vnum, 'items', true);
          renderList();
        }
      });
    });

    detailViewEl.querySelectorAll('.js-open-mob').forEach((btn) => {
      btn.addEventListener('click', () => {
        const vnum = parseInt(btn.getAttribute('data-vnum'), 10);
        if (dbData.mobs[vnum]) {
          if (currentTab !== 'mobs') switchTab('mobs', true);
          selectRecord(vnum, 'mobs', true);
          renderList();
        }
      });
    });
  }

  // Toolbar Event Listeners
  tabMobs.addEventListener('click', () => switchTab('mobs'));
  tabItems.addEventListener('click', () => switchTab('items'));

  searchInput.addEventListener('input', (e) => {
    searchQuery = e.target.value.trim().toLowerCase();
    renderList();
  });

  clearBtn.addEventListener('click', resetAllFilters);

  sortSelect.addEventListener('change', (e) => {
    sortMode = e.target.value;
    renderList();
  });

  filterZone.addEventListener('change', (e) => {
    filters.zone = e.target.value;
    renderList();
  });

  filterLevelRange.addEventListener('change', (e) => {
    filters.levelRange = e.target.value;
    renderList();
  });

  filterAlignment.addEventListener('change', (e) => {
    filters.alignment = e.target.value;
    renderList();
  });

  filterMobRole.addEventListener('change', (e) => {
    filters.mobRole = e.target.value;
    renderList();
  });

  filterItemType.addEventListener('change', (e) => {
    filters.itemType = e.target.value;
    renderList();
  });

  filterItemWear.addEventListener('change', (e) => {
    filters.itemWear = e.target.value;
    renderList();
  });

  filterItemAffect.addEventListener('change', (e) => {
    filters.itemAffect = e.target.value;
    renderList();
  });

  init();
});
