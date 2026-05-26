/* Dark Pawns — World Map  (v3 — static positions, no force sim)
 *
 * Architecture:
 *   1. Fetch /map/map-index.json on load  → instant zone list + room→zone lookup
 *   2. World map view                     → fetch /map/world-map.json (9590 rooms)
 *                                            Canvas render, pan/pinch/zoom via D3
 *                                            Centroids, zone links, BFS, zone hulls
 *                                            all computed once at load time
 *   3. Zone click (from world map or list) → fetch /map/zone-{id}.json (lazy, cached)
 *                                            SVG render with pre-baked BFS x,y
 *   4. Room click                          → show detail panel (no extra fetch)
 *
 * Touch/mobile: D3 zoom handles pointer events (mouse + touch + trackpad pinch)
 * on both canvas and SVG elements. No force simulation anywhere.
 */
(function () {
  'use strict';

  // ── State ──────────────────────────────────────────────────────────────────
  const zoneCache = {};        // zone_id → loaded zone JSON
  let   indexData   = null;    // {zones: [{id, name, rooms}]}
  let   worldMapData = null;   // {rooms: [{id, x, y, sector, zone_id}], links: [{s, t}]}
  let   wmRoomPosMap = {};     // room_id → room object (for O(1) hit-testing)
  let   zoneCentroids = {};    // zone_id → {x, y, name, count}
  let   worldZoneLinks = [];   // [{z1, z2, count}]
  let   reachableRooms = {};   // room_id → true
  let   zoneHulls = {};        // zone_id → {hull, ds, bx0, by0, bx1, by1}

  let currentZoneId  = null;   // null = world map
  let selectedRoomId = null;
  let pathStartId    = null;
  let pathMode       = false;
  let currentRooms   = [];     // rooms in the current zone (with x,y)
  let currentLinks   = [];     // links [{s, t}] in the current zone

  // canvas transform state (world map)
  let cvTransform = d3.zoomIdentity;
  let viewMode = 'grid';       // 'grid' | 'constellation'
  let roomDegrees = {};        // room_id → degree (exit count)

  // ── Sector colors ──────────────────────────────────────────────────────────
  const SECTOR_COLOR = {
    0: '#4a4540', 1: '#a8201a', 2: '#8c905c',  3: '#3e5c38',
    4: '#b08b5c', 5: '#6b5e50', 6: '#4a7a96',  7: '#2c5282',
    8: '#1a365d', 9: '#63b3ed', 10: '#d0a868', 11: '#e05a47',
    12: '#8d7a5b', 13: '#cbd5e0', 14: '#2c5282', 15: '#5d705c',
  };
  const SECTOR_NAME = {
    0: 'inside', 1: 'city', 2: 'field', 3: 'forest', 4: 'hills',
    5: 'mountain', 6: 'water', 7: 'deep water', 8: 'underwater',
    9: 'flying', 10: 'desert', 11: 'fire', 12: 'earth',
    13: 'wind', 14: 'water', 15: 'swamp',
  };
  function sectorColor(s) { return SECTOR_COLOR[s] ?? '#a8201a'; }

  const DIR_FULL = { n: 'north', e: 'east', s: 'south', w: 'west', u: 'up', d: 'down' };

  // ── DOM refs ───────────────────────────────────────────────────────────────
  const $zoneSearch  = document.getElementById('zone-search');
  const $zoneList    = document.getElementById('zone-list');
  const $roomSearch  = document.getElementById('room-search');
  const $roomResults = document.getElementById('room-search-results');
  const $zoneTitle   = document.getElementById('zone-title');
  const $loading     = document.getElementById('map-loading');
  const $loadingMsg  = document.getElementById('map-loading-msg');
  const $empty       = document.getElementById('map-empty');
  const $legend      = document.getElementById('map-legend');
  const $tooltip     = document.getElementById('map-tooltip');
  const $detail      = document.getElementById('map-detail');
  const $detailName  = document.getElementById('detail-room-name');
  const $detailVnum  = document.getElementById('detail-vnum');
  const $detailDesc  = document.getElementById('detail-desc');
  const $detailExits = document.getElementById('detail-exits');
  const $detailMobs  = document.getElementById('detail-mobs');
  const $pathStatus  = document.getElementById('path-status');
  const $sidebar     = document.getElementById('map-sidebar');
  const $btnSidebar  = document.getElementById('btn-sidebar');
  const $btnSetStart = document.getElementById('btn-set-start');
  const $btnClrPath  = document.getElementById('btn-clear-path');
  const $btnPath     = document.getElementById('btn-path');
  const $btnView     = document.getElementById('btn-view');
  const $svgWrap     = document.getElementById('map-svg-wrap');
  const $canvasWrap  = document.getElementById('map-canvas-wrap');
  const canvasEl     = document.getElementById('map-canvas');
  const ctx          = canvasEl ? canvasEl.getContext('2d') : null;

  // ── D3 SVG setup (zone view) ───────────────────────────────────────────────
  const svgEl  = document.getElementById('map-svg');
  const svg    = d3.select(svgEl);
  const gRoot  = svg.append('g').attr('id', 'map-g');
  const gLinks = gRoot.append('g').attr('class', 'links-layer');
  const gNodes = gRoot.append('g').attr('class', 'nodes-layer');

  const svgZoom = d3.zoom()
    .scaleExtent([0.05, 12])
    .on('zoom', ev => gRoot.attr('transform', ev.transform));
  svg.call(svgZoom);

  // ── Canvas zoom (world map) ────────────────────────────────────────────────
  const canvasZoom = d3.zoom()
    .scaleExtent([0.04, 12])
    .on('zoom', ev => {
      cvTransform = ev.transform;
      drawWorldMap();
    });

  // ── Helpers ────────────────────────────────────────────────────────────────
  function svgDims() {
    const r = svgEl.getBoundingClientRect();
    return { w: r.width || 800, h: r.height || 600 };
  }
  function showLoading(msg) {
    if ($loadingMsg) $loadingMsg.textContent = msg;
    $loading.style.display = 'flex';
  }
  function hideLoading() {
    $loading.style.display = 'none';
  }
  function escHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;').replace(/</g, '&lt;')
      .replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  // ── Init ───────────────────────────────────────────────────────────────────
  function init() {
    showLoading('Loading…');

    // Wire sidebar toggle (mobile)
    if ($btnSidebar) {
      $btnSidebar.addEventListener('click', () => {
        $sidebar.classList.toggle('open');
      });
    }

    // Wire zoom buttons
    document.getElementById('btn-zoom-in')?.addEventListener('click', () => {
      const target = currentZoneId === null
        ? d3.select(canvasEl)
        : svg;
      const z = currentZoneId === null ? canvasZoom : svgZoom;
      target.transition().duration(200).call(z.scaleBy, 1.4);
    });
    document.getElementById('btn-zoom-out')?.addEventListener('click', () => {
      const target = currentZoneId === null
        ? d3.select(canvasEl)
        : svg;
      const z = currentZoneId === null ? canvasZoom : svgZoom;
      target.transition().duration(200).call(z.scaleBy, 1 / 1.4);
    });
    document.getElementById('btn-reset')?.addEventListener('click', () => {
      if (currentZoneId === null) fitCanvasToWorldMap();
      else fitSvgToZone();
    });
    $btnPath?.addEventListener('click', togglePathMode);
    $btnView?.addEventListener('click', toggleViewMode);
    $btnSetStart?.addEventListener('click', () => setPathStart(selectedRoomId));
    $btnClrPath?.addEventListener('click', resetPath);
    document.getElementById('detail-close')?.addEventListener('click', deselectRoom);

    // Zone search
    $zoneSearch?.addEventListener('input', e => renderZoneList(e.target.value));

    // Room search
    $roomSearch?.addEventListener('input', e => handleRoomSearch(e.target.value));

    // Close sidebar on outside tap (mobile)
    document.addEventListener('click', e => {
      if (window.innerWidth > 768) return;
      if ($sidebar.classList.contains('open') &&
          !$sidebar.contains(e.target) &&
          e.target !== $btnSidebar) {
        $sidebar.classList.remove('open');
      }
    });

    // Fetch index (map-index.json avoids conflict with Hugo's generated /map/index.json)
    fetch('/map/map-index.json')
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        indexData = data;
        hideLoading();
        $empty.style.display = 'none';
        renderZoneList('');

        // Deep-link routing
        if (window.location.hash === '#constellation') {
          viewMode = 'constellation';
          $btnView?.classList.add('active');
        } else {
          viewMode = 'grid';
          $btnView?.classList.remove('active');
        }

        const params  = new URLSearchParams(window.location.search);
        const roomArg = params.get('room');
        const zoneArg = params.get('zone');

        if (zoneArg) {
          selectZone(parseInt(zoneArg, 10));
        } else if (roomArg) {
          // need to find which zone the room is in — check each zone's JSON
          // Fallback: show world overview, then load
          selectWorldOverview();
          resolveRoomDeepLink(parseInt(roomArg, 10));
        } else {
          selectWorldOverview();
        }
      })
      .catch(err => {
        if ($loadingMsg) $loadingMsg.textContent = `Failed: ${err.message}`;
      });
  }

  // ── Deep-link room resolve ─────────────────────────────────────────────────
  // index.json includes room_zones: {roomId → zoneId} built at precompute time.
  // We wait for the zone fetch to complete before jumping — no setTimeout hack.
  function resolveRoomDeepLink(roomId) {
    if (!indexData) return;
    const zoneId = indexData.room_zones?.[String(roomId)];
    if (zoneId !== undefined) {
      selectZone(zoneId, { pushHistory: false }).then(() => {
        const room = currentRooms.find(r => r.id === roomId);
        if (room) { selectRoom(room); jumpToRoom(roomId); }
        else jumpToRoom(roomId);
      });
    }
  }

  // ── URL / history helpers ──────────────────────────────────────────────────
  function pushState(params) {
    const url = new URL(window.location.href);
    // Clear and rebuild cleanly
    url.search = '';
    for (const [k, v] of Object.entries(params)) {
      if (v !== null && v !== undefined) url.searchParams.set(k, v);
    }
    history.pushState({}, '', url);
  }
  function replaceState(params) {
    const url = new URL(window.location.href);
    url.search = '';
    for (const [k, v] of Object.entries(params)) {
      if (v !== null && v !== undefined) url.searchParams.set(k, v);
    }
    history.replaceState({}, '', url);
  }

  // Handle browser back/forward
  window.addEventListener('popstate', () => {
    const p     = new URLSearchParams(window.location.search);
    const zoneArg = p.get('zone');
    const roomArg = p.get('room');

    if (window.location.hash === '#constellation') {
      viewMode = 'constellation';
      $btnView?.classList.add('active');
    } else {
      viewMode = 'grid';
      $btnView?.classList.remove('active');
    }

    if (zoneArg) {
      selectZone(parseInt(zoneArg, 10), { pushHistory: false }).then(() => {
        if (roomArg) jumpToRoom(parseInt(roomArg, 10));
      });
    } else {
      selectWorldOverview({ pushHistory: false });
    }
  });

  // ── Copy link to clipboard ─────────────────────────────────────────────────
  function copyRoomLink(roomId) {
    const url = `${location.origin}${location.pathname}?room=${roomId}`;
    if (navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(url).then(() => flashCopyBtn('Copied!')).catch(() => flashCopyBtn('Copy failed'));
    } else {
      // Fallback for older mobile browsers
      const ta = document.createElement('textarea');
      ta.value = url;
      ta.style.cssText = 'position:fixed;opacity:0';
      document.body.appendChild(ta);
      ta.select();
      try { document.execCommand('copy'); flashCopyBtn('Copied!'); }
      catch { flashCopyBtn('Copy failed'); }
      ta.remove();
    }
  }
  function flashCopyBtn(msg) {
    const btn = document.getElementById('detail-copy-link');
    if (!btn) return;
    const orig = btn.textContent;
    btn.textContent = msg;
    setTimeout(() => { btn.textContent = orig; }, 1800);
  }

  // ── Zone list ──────────────────────────────────────────────────────────────
  function renderZoneList(filter) {
    if (!indexData) return;
    const q = filter.toLowerCase().trim();
    const zones = indexData.zones.filter(z =>
      !q || z.name.toLowerCase().includes(q)
    );

    if (!zones.length) {
      $zoneList.innerHTML = '<div style="padding:.5rem .75rem;font-size:.65rem;color:var(--ink-muted)">No zones match</div>';
      return;
    }

    const totalRooms = indexData.zones.reduce((s, z) => s + z.rooms, 0);
    let html = '';
    if (!q) {
      html += `<div class="zone-item${currentZoneId === null ? ' active' : ''}" id="item-world-map">
        <span class="zone-name">🌍 WORLD MAP</span>
        <span class="zone-count">${totalRooms}</span>
      </div>`;
    }
    html += zones.map(z => `
      <div class="zone-item${z.id === currentZoneId ? ' active' : ''}" data-zone="${z.id}">
        <span class="zone-name">${escHtml(z.name)}</span>
        <span class="zone-count">${z.rooms}</span>
      </div>`).join('');

    $zoneList.innerHTML = html;

    document.getElementById('item-world-map')?.addEventListener('click', selectWorldOverview);
    $zoneList.querySelectorAll('.zone-item[data-zone]').forEach(el =>
      el.addEventListener('click', () => selectZone(+el.dataset.zone))
    );
  }

  function updateZoneListActive() {
    $zoneList.querySelectorAll('.zone-item').forEach(el => {
      if (el.id === 'item-world-map') {
        el.classList.toggle('active', currentZoneId === null);
      } else {
        el.classList.toggle('active', +el.dataset.zone === currentZoneId);
      }
    });
  }

  // ── Fetch zone (cached) ────────────────────────────────────────────────────
  function fetchZone(zoneId) {
    if (zoneCache[zoneId]) return Promise.resolve(zoneCache[zoneId]);
    return fetch(`/map/zone-${zoneId}.json`)
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => { zoneCache[zoneId] = data; return data; });
  }

  // ── World overview — Canvas ────────────────────────────────────────────────
  function selectWorldOverview({ pushHistory = true } = {}) {
    currentZoneId = null;
    deselectRoom();
    resetPath();
    updateZoneListActive();

    if (pushHistory) pushState({});
    $zoneTitle.textContent = 'WORLD MAP';
    $legend.classList.add('visible');
    $svgWrap.style.display    = 'none';
    $canvasWrap.style.display = '';
    if (window.innerWidth <= 768) $sidebar.classList.remove('open');

    if (worldMapData) {
      resizeCanvas();
      drawWorldMap();
      return;
    }

    showLoading('Loading world map…');
    fetch('/map/world-map.json')
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        worldMapData = data;
        wmRoomPosMap = {};
        for (const room of data.rooms) wmRoomPosMap[room.id] = room;

        // Compute room degrees for Constellation View
        roomDegrees = {};
        for (const lk of data.links) {
          roomDegrees[lk.s] = (roomDegrees[lk.s] || 0) + 1;
          roomDegrees[lk.t] = (roomDegrees[lk.t] || 0) + 1;
        }

        // 1. Compute Zone Centroids (DP-312 / DP-316)
        const sums = {};
        for (const room of data.rooms) {
          if (room.zone_id !== undefined) {
            const z = sums[room.zone_id] ??= { sumX: 0, sumY: 0, count: 0 };
            z.sumX += room.x;
            z.sumY += room.y;
            z.count++;
          }
        }
        zoneCentroids = {};
        for (const [zoneId, z] of Object.entries(sums)) {
          const indexZone = indexData?.zones.find(iz => iz.id === +zoneId);
          zoneCentroids[zoneId] = {
            x: z.sumX / z.count,
            y: z.sumY / z.count,
            name: indexZone ? indexZone.name : `Zone ${zoneId}`,
            count: z.count,
          };
        }

        // 2. Compute Inter-Zone Link Counts (DP-315)
        const zoneLinksMap = {};
        for (const lk of data.links) {
          const s = wmRoomPosMap[lk.s];
          const t = wmRoomPosMap[lk.t];
          if (s && t && s.zone_id !== undefined && t.zone_id !== undefined && s.zone_id !== t.zone_id) {
            const key = s.zone_id < t.zone_id ? `${s.zone_id}-${t.zone_id}` : `${t.zone_id}-${s.zone_id}`;
            zoneLinksMap[key] = (zoneLinksMap[key] || 0) + 1;
          }
        }
        worldZoneLinks = [];
        for (const [key, count] of Object.entries(zoneLinksMap)) {
          const [z1, z2] = key.split('-').map(Number);
          worldZoneLinks.push({ z1, z2, count });
        }

        // 3. Run BFS from room 0 to find Reachable Rooms (DP-313)
        reachableRooms = { 0: true };
        const adj = {};
        for (const room of data.rooms) adj[room.id] = [];
        for (const lk of data.links) {
          adj[lk.s]?.push(lk.t);
          adj[lk.t]?.push(lk.s);
        }
        const queue = [0];
        let head = 0;
        while (head < queue.length) {
          const u = queue[head++];
          const neighbors = adj[u] || [];
          for (const v of neighbors) {
            if (!reachableRooms[v]) {
              reachableRooms[v] = true;
              queue.push(v);
            }
          }
        }

        // 4. Per-Zone Convex Hulls for territory shading (DP-314 rework)
        const roomsByZone = {};
        for (const room of data.rooms) (roomsByZone[room.zone_id] ??= []).push(room);
        zoneHulls = {};
        for (const [zoneId, rooms] of Object.entries(roomsByZone)) {
          const sectorCount = {};
          for (const r of rooms) sectorCount[r.sector] = (sectorCount[r.sector] || 0) + 1;
          let ds = 0, dsMax = 0;
          for (const [s, n] of Object.entries(sectorCount)) {
            if (n > dsMax) { dsMax = n; ds = +s; }
          }
          const hull = convexHull(rooms);
          if (hull.length < 3) continue;
          let bx0 = Infinity, by0 = Infinity, bx1 = -Infinity, by1 = -Infinity;
          for (const p of hull) {
            if (p.x < bx0) bx0 = p.x; if (p.x > bx1) bx1 = p.x;
            if (p.y < by0) by0 = p.y; if (p.y > by1) by1 = p.y;
          }
          zoneHulls[+zoneId] = { hull, ds, bx0, by0, bx1, by1 };
        }

        hideLoading();
        resizeCanvas();
        d3.select(canvasEl).call(canvasZoom);
        fitCanvasToWorldMap();
        drawWorldMap();
        attachCanvasEvents();
      })
      .catch(err => {
        if ($loadingMsg) $loadingMsg.textContent = `Failed: ${err.message}`;
      });
  }

  function resizeCanvas() {
    if (!canvasEl) return;
    const rect = canvasEl.parentElement.getBoundingClientRect();
    const dpr  = window.devicePixelRatio || 1;
    canvasEl.width  = rect.width  * dpr;
    canvasEl.height = rect.height * dpr;
    canvasEl.style.width  = rect.width  + 'px';
    canvasEl.style.height = rect.height + 'px';
    ctx.scale(dpr, dpr);
  }

  window.addEventListener('resize', () => {
    if (currentZoneId === null && worldMapData) {
      resizeCanvas();
      drawWorldMap();
    }
  });

  // ── Dot radius in world coords that renders at desired screen pixels ─────────
  function dotRadius(k) {
    if (k < 0.15) return 1.5 / k;
    if (k < 0.5)  return 2.0 / k;
    if (k < 1.5)  return 2.5 / k;
    if (k < 4)    return 3.5 / k;
    return 5.0 / k;
  }

  // Monotone Chain algorithm for Convex Hull (O(N log N)) (DP-314)
  function convexHull(points) {
    if (points.length <= 1) return points;
    const sorted = points.slice().sort((a, b) => a.x !== b.x ? a.x - b.x : a.y - b.y);
    const cross = (o, a, b) => (a.x - o.x) * (b.y - o.y) - (a.y - o.y) * (b.x - o.x);
    const lower = [];
    for (const p of sorted) {
      while (lower.length >= 2 && cross(lower[lower.length - 2], lower[lower.length - 1], p) <= 0) {
        lower.pop();
      }
      lower.push(p);
    }
    const upper = [];
    for (let i = sorted.length - 1; i >= 0; i--) {
      const p = sorted[i];
      while (upper.length >= 2 && cross(upper[upper.length - 2], upper[upper.length - 1], p) <= 0) {
        upper.pop();
      }
      upper.push(p);
    }
    upper.pop();
    lower.pop();
    return lower.concat(upper);
  }
  function toggleViewMode() {
    viewMode = viewMode === 'grid' ? 'constellation' : 'grid';
    $btnView?.classList.toggle('active', viewMode === 'constellation');

    // Sync hash
    if (viewMode === 'constellation') {
      window.location.hash = 'constellation';
    } else {
      history.replaceState(null, document.title, window.location.pathname + window.location.search);
    }

    if (currentZoneId !== null) {
      selectWorldOverview();
    } else if (worldMapData) {
      drawWorldMap();
    }
  }

  function drawConstellation() {
    if (!ctx || !worldMapData) return;
    const dpr = window.devicePixelRatio || 1;
    const W   = canvasEl.width  / dpr;
    const H   = canvasEl.height / dpr;
    const { x: tx, y: ty, k } = cvTransform;

    ctx.clearRect(0, 0, W, H);
    ctx.save();
    ctx.translate(tx, ty);
    ctx.scale(k, k);

    // Viewport bounds in world coords (with generous padding to avoid edge flicker)
    const pad = 80 / k;
    const vx0 = -tx / k - pad,  vx1 = (W - tx) / k + pad;
    const vy0 = -ty / k - pad,  vy1 = (H - ty) / k + pad;

    // ── 1. Graph Edges (with zoom alpha fade-in to prevent popping) ────────────
    if (k >= 0.05) {
      ctx.save();
      
      // 1a. Inter-Zone Links (faint oxblood threads representing major highways)
      const interAlpha = Math.min(0.20, Math.max(0, (k - 0.05) * 1.5));
      ctx.strokeStyle = `rgba(168, 32, 26, ${interAlpha.toFixed(3)})`;
      ctx.lineWidth   = 1.0 / k;
      ctx.beginPath();
      for (const lk of worldMapData.links) {
        const s = wmRoomPosMap[lk.s];
        const t = wmRoomPosMap[lk.t];
        if (!s || !t || s.zone_id === t.zone_id) continue;
        if ((s.x < vx0 && t.x < vx0) || (s.x > vx1 && t.x > vx1)) continue;
        if ((s.y < vy0 && t.y < vy0) || (s.y > vy1 && t.y > vy1)) continue;
        ctx.moveTo(s.x, s.y);
        ctx.lineTo(t.x, t.y);
      }
      ctx.stroke();

      // 1b. Intra-Zone Links (super-fine charcoal threads representing standard paths)
      if (k >= 0.08) {
        const intraAlpha = Math.min(0.08, Math.max(0, (k - 0.08) * 0.8));
        ctx.strokeStyle = `rgba(26, 22, 20, ${intraAlpha.toFixed(3)})`;
        ctx.lineWidth   = 0.5 / k;
        ctx.beginPath();
        for (const lk of worldMapData.links) {
          const s = wmRoomPosMap[lk.s];
          const t = wmRoomPosMap[lk.t];
          if (!s || !t || s.zone_id !== t.zone_id) continue;
          if ((s.x < vx0 && t.x < vx0) || (s.x > vx1 && t.x > vx1)) continue;
          if ((s.y < vy0 && t.y < vy0) || (s.y > vy1 && t.y > vy1)) continue;
          ctx.moveTo(s.x, s.y);
          ctx.lineTo(t.x, t.y);
        }
        ctx.stroke();
      }

      ctx.restore();
    }

    // ── 2. Graph Nodes (Double-Circle "Ink-Blot" Draw) ────────────────────────
    const TAU = Math.PI * 2;
    
    // Group rooms inside viewport by sector and reachability for batch drawing
    const reachableBySector = {};
    const unreachableRooms = [];

    for (const r of worldMapData.rooms) {
      if (r.x < vx0 || r.x > vx1 || r.y < vy0 || r.y > vy1) continue;
      if (reachableRooms[r.id]) {
        (reachableBySector[r.sector] ??= []).push(r);
      } else {
        unreachableRooms.push(r);
      }
    }

    // 2a. Draw Unreachable Rooms first (ghostly, barely printed ink)
    if (unreachableRooms.length > 0) {
      ctx.save();
      // Outer extremely faint ink bleeding
      ctx.fillStyle = 'rgba(26, 22, 20, 0.03)';
      ctx.beginPath();
      for (const r of unreachableRooms) {
        const deg = roomDegrees[r.id] || 0;
        const rVal = deg <= 1 ? 1.8 : (deg <= 4 ? 3.0 : 4.5);
        const dr = rVal / k;
        ctx.moveTo(r.x + dr * 2.5, r.y);
        ctx.arc(r.x, r.y, dr * 2.5, 0, TAU);
      }
      ctx.fill();

      // Inner ghostly charcoal core
      ctx.fillStyle = 'rgba(74, 69, 64, 0.15)';
      ctx.beginPath();
      for (const r of unreachableRooms) {
        const deg = roomDegrees[r.id] || 0;
        const rVal = deg <= 1 ? 1.8 : (deg <= 4 ? 3.0 : 4.5);
        const dr = rVal / k;
        ctx.moveTo(r.x + dr, r.y);
        ctx.arc(r.x, r.y, dr, 0, TAU);
      }
      ctx.fill();
      ctx.restore();
    }

    // 2b. Draw Reachable Rooms in two passes
    // Pass 1: Outer soft ink bleed/halo for all reachable nodes
    ctx.save();
    ctx.fillStyle = 'rgba(26, 22, 20, 0.08)';
    ctx.beginPath();
    for (const sector in reachableBySector) {
      for (const r of reachableBySector[sector]) {
        const deg = roomDegrees[r.id] || 0;
        const rVal = deg <= 1 ? 1.8 : (deg <= 4 ? 3.0 : 4.5);
        const dr = rVal / k;
        ctx.moveTo(r.x + dr * 2.5, r.y);
        ctx.arc(r.x, r.y, dr * 2.5, 0, TAU);
      }
    }
    ctx.fill();
    ctx.restore();

    // Pass 2: Inner solid sector cores
    for (const sector in reachableBySector) {
      ctx.save();
      ctx.fillStyle = SECTOR_COLOR[+sector] ?? '#a8201a';
      ctx.beginPath();
      for (const r of reachableBySector[sector]) {
        const deg = roomDegrees[r.id] || 0;
        const rVal = deg <= 1 ? 1.8 : (deg <= 4 ? 3.0 : 4.5);
        const dr = rVal / k;
        ctx.moveTo(r.x + dr, r.y);
        ctx.arc(r.x, r.y, dr, 0, TAU);
      }
      ctx.fill();
      ctx.restore();
    }

    // ── 3. Selected Room Highlight ─────────────────────────────────────────────
    const selRoom = selectedRoomId !== null ? wmRoomPosMap[selectedRoomId] : null;
    if (selRoom) {
      const deg = roomDegrees[selRoom.id] || 0;
      const rVal = deg <= 1 ? 1.8 : (deg <= 4 ? 3.0 : 4.5);
      const dr = rVal / k;
      ctx.beginPath();
      ctx.arc(selRoom.x, selRoom.y, dr * 2.5, 0, TAU);
      ctx.strokeStyle = '#a8201a';
      ctx.lineWidth   = 2.0 / k;
      ctx.stroke();
    }

    // ── 4. Zone Name Labels ───────────────────────────────────────────────────
    if (k >= 0.12 && zoneCentroids) {
      ctx.save();
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      
      const fontSize = k >= 1.0 ? 12 : 10;
      ctx.font = k >= 1.0 
        ? `bold ${fontSize/k}px 'DM Serif Display', Georgia, serif` 
        : `${fontSize/k}px 'DM Serif Display', Georgia, serif`;
      
      ctx.fillStyle = k >= 1.0 
        ? 'rgba(168, 32, 26, 0.85)' // full oxblood at close zoom
        : 'rgba(86, 80, 74, 0.55)';  // muted ink-muted at medium zoom
        
      for (const [zoneId, z] of Object.entries(zoneCentroids)) {
        if (z.x < vx0 || z.x > vx1 || z.y < vy0 || z.y > vy1) continue;
        ctx.fillText(z.name.toUpperCase(), z.x, z.y - 12 / k);
      }
      ctx.restore();
    }

    ctx.restore();
  }
  function drawWorldMap() {
    if (!ctx || !worldMapData) return;
    if (viewMode === 'constellation') {
      drawConstellation();
      return;
    }
    const dpr = window.devicePixelRatio || 1;
    const W   = canvasEl.width  / dpr;
    const H   = canvasEl.height / dpr;
    const { x: tx, y: ty, k } = cvTransform;

    ctx.clearRect(0, 0, W, H);
    ctx.save();
    ctx.translate(tx, ty);
    ctx.scale(k, k);

    // Viewport bounds in world coords (with generous padding to avoid edge flicker)
    const pad = 80 / k;
    const vx0 = -tx / k - pad,  vx1 = (W - tx) / k + pad;
    const vy0 = -ty / k - pad,  vy1 = (H - ty) / k + pad;

    // ── 1. Per-Zone Territory Shading (DP-314) ─────────────────────────────────
    // Hull per zone (pre-computed at load), colored by that zone's dominant sector.
    // Avoids global-sector hulls which span 97%+ of the world for common sectors.
    if (k > 0.08 && Object.keys(zoneHulls).length > 0) {
      ctx.save();
      ctx.globalAlpha = 0.06;
      for (const entry of Object.values(zoneHulls)) {
        if (entry.bx1 < vx0 || entry.bx0 > vx1 || entry.by1 < vy0 || entry.by0 > vy1) continue;
        ctx.fillStyle = SECTOR_COLOR[entry.ds] ?? '#a8201a';
        ctx.beginPath();
        ctx.moveTo(entry.hull[0].x, entry.hull[0].y);
        for (let i = 1; i < entry.hull.length; i++) ctx.lineTo(entry.hull[i].x, entry.hull[i].y);
        ctx.closePath();
        ctx.fill();
      }
      ctx.restore();
    }

    // ── 2. Inter-Zone Links (DP-315) ──────────────────────────────────────────
    if (k > 0.12 && worldZoneLinks) {
      ctx.save();
      ctx.strokeStyle = 'rgba(139, 0, 0, 0.06)'; // elegant thin oxblood lines
      for (const zl of worldZoneLinks) {
        const c1 = zoneCentroids[zl.z1];
        const c2 = zoneCentroids[zl.z2];
        if (!c1 || !c2) continue;
        if ((c1.x < vx0 && c2.x < vx0) || (c1.x > vx1 && c2.x > vx1)) continue;
        if ((c1.y < vy0 && c2.y < vy0) || (c1.y > vy1 && c2.y > vy1)) continue;
        
        ctx.lineWidth = Math.min(zl.count * 0.4, 3.5) / k;
        ctx.beginPath();
        ctx.moveTo(c1.x, c1.y);
        ctx.lineTo(c2.x, c2.y);
        ctx.stroke();
      }
      ctx.restore();
    }

    // ── 3. Intra-Zone Links ────────────────────────────────────────────────────
    if (k > 0.1) {
      const alpha = Math.min(0.2, Math.max(0.04, k * 0.25));
      ctx.strokeStyle = `rgba(26,22,20,${alpha.toFixed(2)})`;
      ctx.lineWidth   = 0.7 / k;
      ctx.beginPath();
      for (const lk of worldMapData.links) {
        const s = wmRoomPosMap[lk.s];
        const t = wmRoomPosMap[lk.t];
        if (!s || !t) continue;
        if ((s.x < vx0 && t.x < vx0) || (s.x > vx1 && t.x > vx1)) continue;
        if ((s.y < vy0 && t.y < vy0) || (s.y > vy1 && t.y > vy1)) continue;
        ctx.moveTo(s.x, s.y);
        ctx.lineTo(t.x, t.y);
      }
      ctx.stroke();
    }

    // ── 4. Rooms with Unreachable Dimming (DP-313) ─────────────────────────────
    const dr = dotRadius(k);
    const TAU = Math.PI * 2;

    const bySector = {};
    for (const room of worldMapData.rooms) {
      if (room.x < vx0 || room.x > vx1 || room.y < vy0 || room.y > vy1) continue;
      const s = room.sector;
      (bySector[s] ??= []).push(room);
    }

    for (const [sector, rooms] of Object.entries(bySector)) {
      // 4a. Draw Reachable rooms in sector colors
      ctx.fillStyle = SECTOR_COLOR[+sector] ?? '#a8201a';
      ctx.beginPath();
      for (const r of rooms) {
        if (reachableRooms[r.id]) {
          ctx.moveTo(r.x + dr, r.y);
          ctx.arc(r.x, r.y, dr, 0, TAU);
        }
      }
      ctx.fill();

      // 4b. Draw Unreachable rooms dimmed (ghostly parchment overlay)
      ctx.fillStyle = 'rgba(74, 69, 64, 0.18)'; 
      ctx.beginPath();
      for (const r of rooms) {
        if (!reachableRooms[r.id]) {
          ctx.moveTo(r.x + dr, r.y);
          ctx.arc(r.x, r.y, dr, 0, TAU);
        }
      }
      ctx.fill();
    }

    // ── 5. Selected Room Highlight ─────────────────────────────────────────────
    const selRoom = selectedRoomId !== null ? wmRoomPosMap[selectedRoomId] : null;
    if (selRoom) {
      ctx.beginPath();
      ctx.arc(selRoom.x, selRoom.y, dr * 2.5, 0, TAU);
      ctx.strokeStyle = '#a8201a';
      ctx.lineWidth   = 2 / k;
      ctx.stroke();
    }

    // ── 6. Zone Name Labels (DP-312) ───────────────────────────────────────────
    if (k >= 0.2 && zoneCentroids) {
      ctx.save();
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      
      const fontSize = k >= 1.0 ? 12 : 10;
      ctx.font = k >= 1.0 
        ? `bold ${fontSize/k}px "IM Fell English", Georgia, serif` 
        : `${fontSize/k}px "IM Fell English", Georgia, serif`;
      
      ctx.fillStyle = k >= 1.0 
        ? 'rgba(139, 0, 0, 0.85)' // full opacity oxblood at close zoom
        : 'rgba(74, 69, 64, 0.55)'; // muted charcoal at medium zoom
        
      for (const [zoneId, z] of Object.entries(zoneCentroids)) {
        if (z.x < vx0 || z.x > vx1 || z.y < vy0 || z.y > vy1) continue;
        // Float label slightly above the calculated centroid
        ctx.fillText(z.name.toUpperCase(), z.x, z.y - 12 / k);
      }
      ctx.restore();
    }

    ctx.restore();
  }

  function fitCanvasToWorldMap() {
    if (!worldMapData || !canvasEl) return;
    const rooms = worldMapData.rooms;
    if (!rooms.length) return;
    const W = canvasEl.width  / (window.devicePixelRatio || 1);
    const H = canvasEl.height / (window.devicePixelRatio || 1);

    let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
    for (const r of rooms) {
      if (r.x < minX) minX = r.x;  if (r.x > maxX) maxX = r.x;
      if (r.y < minY) minY = r.y;  if (r.y > maxY) maxY = r.y;
    }
    const pad = 50;
    const k   = Math.max(0.02, Math.min(10,
      Math.min((W - pad * 2) / (maxX - minX + 1), (H - pad * 2) / (maxY - minY + 1))
    ));
    const tx  = W / 2 - k * ((minX + maxX) / 2);
    const ty  = H / 2 - k * ((minY + maxY) / 2);
    d3.select(canvasEl).call(canvasZoom.transform,
      d3.zoomIdentity.translate(tx, ty).scale(k));
  }

  // Canvas hit-test: find nearest room under pointer (world map)
  function worldMapHitTest(clientX, clientY) {
    if (!worldMapData || !canvasEl) return null;
    const rect = canvasEl.getBoundingClientRect();
    const { x: tx, y: ty, k } = cvTransform;
    const wx = (clientX - rect.left - tx) / k;
    const wy = (clientY - rect.top  - ty) / k;
    // Threshold: 14px in screen coords converted to world units
    const threshold = 14 / k;
    let nearest = null, minDist = threshold;
    for (const room of worldMapData.rooms) {
      const d = Math.hypot(room.x - wx, room.y - wy);
      if (d < minDist) { minDist = d; nearest = room; }
    }
    return nearest;
  }

  function attachCanvasEvents() {
    if (!canvasEl) return;

    // Click / tap: find room → load its zone → enter zone view → select room
    canvasEl.addEventListener('click', e => {
      const room = worldMapHitTest(e.clientX, e.clientY);
      if (!room) return;
      selectZone(room.zone_id).then(() => {
        const zoneRoom = currentRooms.find(r => r.id === room.id);
        if (zoneRoom) { selectRoom(zoneRoom); jumpToRoom(room.id); }
      });
    });

    // Hover tooltip (desktop): show zone name + sector
    canvasEl.addEventListener('mousemove', e => {
      const room = worldMapHitTest(e.clientX, e.clientY);
      if (room) {
        canvasEl.style.cursor = 'pointer';
        const zoneName = indexData?.zones.find(z => z.id === room.zone_id)?.name
                         ?? `Zone ${room.zone_id}`;
        showTooltip(e.clientX, e.clientY,
          `<strong>${escHtml(zoneName)}</strong>${SECTOR_NAME[room.sector] ?? 'unknown'}`);
      } else {
        canvasEl.style.cursor = '';
        hideTooltip();
      }
    });
    canvasEl.addEventListener('mouseleave', hideTooltip);
  }

  // ── Zone view — SVG with pre-baked positions ───────────────────────────────
  // Returns a Promise that resolves after the zone is rendered.
  function selectZone(zoneId, { pushHistory = true } = {}) {
    currentZoneId = zoneId;
    deselectRoom();
    resetPath();
    updateZoneListActive();

    if (pushHistory) pushState({ zone: zoneId });
    if (window.innerWidth <= 768) $sidebar.classList.remove('open');

    $svgWrap.style.display    = '';
    $canvasWrap.style.display = 'none';
    $legend.classList.remove('visible');

    showLoading('Loading zone…');

    return fetchZone(zoneId)
      .then(data => {
        renderZone(data);
        hideLoading();
        const z = indexData?.zones.find(z => z.id === zoneId);
        $zoneTitle.textContent = z ? z.name.toUpperCase() : `ZONE ${zoneId}`;
      })
      .catch(err => {
        if ($loadingMsg) $loadingMsg.textContent = `Failed: ${err.message}`;
      });
  }

  function renderZone(zoneData) {
    currentRooms = zoneData.rooms;
    currentLinks = zoneData.links;

    const roomMap = {};
    for (const r of currentRooms) roomMap[r.id] = r;

    // ── Links ──
    gLinks.selectAll('line')
      .data(currentLinks, d => `${d.s}-${d.t}`)
      .join('line')
      .attr('class', 'exit-line')
      .attr('x1', d => roomMap[d.s]?.x ?? 0)
      .attr('y1', d => roomMap[d.s]?.y ?? 0)
      .attr('x2', d => roomMap[d.t]?.x ?? 0)
      .attr('y2', d => roomMap[d.t]?.y ?? 0);

    // ── Nodes ──
    gNodes.selectAll('circle')
      .data(currentRooms, d => d.id)
      .join('circle')
      .attr('class', 'room-node')
      .attr('r', 5)
      .attr('cx', d => d.x)
      .attr('cy', d => d.y)
      .attr('fill', d => sectorColor(d.sector))
      .attr('stroke', 'var(--ink)')
      .attr('stroke-width', 1.2)
      .on('mouseover', onNodeHover)
      .on('mousemove', onNodeMove)
      .on('mouseout',  onNodeOut)
      .on('click',     onNodeClick)
      .on('touchend',  onNodeTouch);

    fitSvgToZone();
  }

  function fitSvgToZone() {
    if (!currentRooms.length) return;
    const { w, h } = svgDims();
    let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
    for (const r of currentRooms) {
      if (r.x < minX) minX = r.x;  if (r.x > maxX) maxX = r.x;
      if (r.y < minY) minY = r.y;  if (r.y > maxY) maxY = r.y;
    }
    const pad = 40;
    const k   = Math.max(0.05, Math.min(3,
      Math.min((w - pad * 2) / (maxX - minX + 1), (h - pad * 2) / (maxY - minY + 1))
    ));
    const cx  = (minX + maxX) / 2;
    const cy  = (minY + maxY) / 2;
    svg.transition().duration(300).call(
      svgZoom.transform,
      d3.zoomIdentity.translate(w / 2 - k * cx, h / 2 - k * cy).scale(k)
    );
  }

  // ── Room interaction ───────────────────────────────────────────────────────
  function onNodeHover(event, d) {
    d3.select(this).raise();
    showTooltip(event.clientX, event.clientY,
      `<strong>${escHtml(d.name)}</strong>#${d.id} · ${SECTOR_NAME[d.sector] ?? 'unknown'}`);
  }
  function onNodeMove(event) {
    moveTooltip(event.clientX, event.clientY);
  }
  function onNodeOut() { hideTooltip(); }
  function onNodeClick(event, d) {
    event.stopPropagation();
    selectRoom(d);
  }
  function onNodeTouch(event, d) {
    event.preventDefault();
    event.stopPropagation();
    selectRoom(d);
  }

  svg.on('click', () => {
    if (selectedRoomId !== null) deselectRoom();
  });

  function selectRoom(room) {
    selectedRoomId = room.id;

    // Reflect room in URL without adding a history entry — room selection is
    // a detail within a zone page, not a separate navigation step.
    if (currentZoneId !== null) {
      replaceState({ zone: currentZoneId, room: room.id });
    }

    gNodes.selectAll('circle')
      .classed('selected', d => d.id === room.id)
      .attr('stroke-width', d => d.id === room.id ? 3 : 1.2);

    showDetail(room);

    if (window.innerWidth <= 768) {
      $detail.classList.add('open');
      $detail.classList.remove('hidden');
    } else {
      $detail.classList.remove('hidden');
    }
  }

  function deselectRoom() {
    selectedRoomId = null;
    // Restore URL to just the zone (drop the room param)
    if (currentZoneId !== null) replaceState({ zone: currentZoneId });
    gNodes.selectAll('circle')
      .classed('selected', false)
      .attr('stroke-width', 1.2);
    $detail.classList.add('hidden');
    $detail.classList.remove('open');
  }

  function showDetail(room) {
    $detailName.textContent = room.name;
    $detailVnum.textContent = `#${room.id} · ${SECTOR_NAME[room.sector] ?? ''}`;
    $detailDesc.textContent = room.desc ?? '';

    // Wire copy-link button
    const $copyBtn = document.getElementById('detail-copy-link');
    if ($copyBtn) {
      $copyBtn.onclick = () => copyRoomLink(room.id);
    }

    // Exits
    const exits = (room.exits ?? []).filter(e => ['n','s','e','w','u','d'].includes(e.d));
    $detailExits.innerHTML = exits.length
      ? `<h4>Exits</h4>` + exits.map(ex => {
          const target = currentRooms.find(r => r.id === ex.t);
          return `<div class="exit-item" data-target="${ex.t}" tabindex="0" role="button">
            <span class="exit-dir">${ex.d}</span>
            <span>${escHtml(target?.name ?? `Room #${ex.t}`)}</span>
            <span class="exit-dest">#${ex.t}</span>
          </div>`;
        }).join('')
      : '<p style="font-size:.7rem;color:var(--ink-muted)">No exits.</p>';

    $detailExits.querySelectorAll('.exit-item').forEach(el => {
      const action = () => {
        const tid = +el.dataset.target;
        const tr  = currentRooms.find(r => r.id === tid);
        if (tr) { selectRoom(tr); jumpToRoom(tid); }
      };
      el.addEventListener('click', action);
      el.addEventListener('keydown', e => { if (e.key === 'Enter' || e.key === ' ') action(); });
    });

    $detailMobs.innerHTML = '';

    // Path buttons
    if (pathMode && pathStartId && pathStartId !== room.id) {
      $btnSetStart.style.display = 'none';
      $btnClrPath.style.display  = '';
      $pathStatus.textContent    = `Path from #${pathStartId} → #${room.id}`;
      drawPath(pathStartId, room.id);
    } else {
      $btnSetStart.style.display = '';
      $btnClrPath.style.display  = 'none';
      $pathStatus.textContent    = pathMode ? 'Click "Set Start" to begin' : '';
    }
  }

  function jumpToRoom(roomId) {
    const room = currentRooms.find(r => r.id === roomId);
    if (!room) return;
    const { w, h } = svgDims();
    // Get current scale
    const t = d3.zoomTransform(svgEl);
    svg.transition().duration(400).call(
      svgZoom.transform,
      d3.zoomIdentity.translate(w / 2 - t.k * room.x, h / 2 - t.k * room.y).scale(t.k)
    );
  }

  // ── Tooltip ────────────────────────────────────────────────────────────────
  function showTooltip(cx, cy, html) {
    $tooltip.innerHTML  = html;
    $tooltip.style.display = 'block';
    moveTooltip(cx, cy);
  }
  function moveTooltip(cx, cy) {
    const tw = $tooltip.offsetWidth;
    const th = $tooltip.offsetHeight;
    const vw = window.innerWidth;
    const vh = window.innerHeight;
    let left = cx + 14;
    let top  = cy - 10;
    if (left + tw > vw - 8) left = cx - tw - 14;
    if (top  + th > vh - 8) top  = cy - th - 10;
    $tooltip.style.left = left + 'px';
    $tooltip.style.top  = top  + 'px';
  }
  function hideTooltip() {
    $tooltip.style.display = 'none';
  }

  // ── Pathfinding ────────────────────────────────────────────────────────────
  function togglePathMode() {
    pathMode = !pathMode;
    $btnPath.classList.toggle('active', pathMode);
    if (!pathMode) resetPath();
    else $pathStatus.textContent = 'Select a room to set path start';
    if ($detail.classList.contains('hidden')) return;
    const room = currentRooms.find(r => r.id === selectedRoomId);
    if (room) showDetail(room);
  }

  function setPathStart(roomId) {
    if (!roomId) return;
    pathStartId = roomId;
    $pathStatus.textContent = `Start set: #${roomId} — now click destination`;
    // Highlight start node
    gNodes.selectAll('circle')
      .classed('path-start', d => d.id === roomId);
  }

  function resetPath() {
    pathStartId = null;
    gLinks.selectAll('line').classed('in-path', false);
    gNodes.selectAll('circle').classed('path-node path-start', false);
    if ($btnClrPath) $btnClrPath.style.display = 'none';
    if ($btnSetStart) $btnSetStart.style.display = '';
    if ($pathStatus) $pathStatus.textContent = '';
  }

  function drawPath(startId, endId) {
    // BFS pathfinding (single zone)
    const roomMap = {};
    for (const r of currentRooms) roomMap[r.id] = r;

    const adj = {};
    for (const r of currentRooms) adj[r.id] = [];
    for (const lk of currentLinks) {
      adj[lk.s]?.push(lk.t);
      adj[lk.t]?.push(lk.s);
    }

    const visited = { [startId]: null };
    const queue   = [startId];
    while (queue.length) {
      const cur = queue.shift();
      if (cur === endId) break;
      for (const nb of (adj[cur] ?? [])) {
        if (!(nb in visited)) {
          visited[nb] = cur;
          queue.push(nb);
        }
      }
    }

    if (!(endId in visited)) {
      $pathStatus.textContent = 'No path found in this zone';
      return;
    }

    // Trace back
    const pathSet = new Set();
    const edgeSet = new Set();
    let cur = endId;
    while (visited[cur] !== null && visited[cur] !== undefined) {
      pathSet.add(cur);
      const prev = visited[cur];
      edgeSet.add(`${Math.min(cur, prev)}-${Math.max(cur, prev)}`);
      cur = prev;
    }
    pathSet.add(startId);

    gNodes.selectAll('circle')
      .classed('path-node',  d => pathSet.has(d.id))
      .classed('path-start', d => d.id === startId);

    gLinks.selectAll('line')
      .classed('in-path', d => edgeSet.has(`${Math.min(d.s, d.t)}-${Math.max(d.s, d.t)}`));
  }

  // ── Room search ────────────────────────────────────────────────────────────
  function handleRoomSearch(q) {
    if (!q.trim() || !currentRooms.length) {
      $roomResults.style.display = 'none';
      $roomResults.innerHTML = '';
      return;
    }
    const ql = q.toLowerCase();
    const hits = currentRooms.filter(r =>
      r.name.toLowerCase().includes(ql) || String(r.id).includes(ql)
    ).slice(0, 20);

    if (!hits.length) {
      $roomResults.style.display = 'none';
      return;
    }
    $roomResults.style.display = 'block';
    $roomResults.innerHTML = hits.map(r => `
      <div class="room-result" data-id="${r.id}" tabindex="0" role="button">
        <div>${escHtml(r.name)}</div>
        <div class="room-result-zone">#${r.id}</div>
      </div>`).join('');

    $roomResults.querySelectorAll('.room-result').forEach(el => {
      const action = () => {
        const room = currentRooms.find(r => r.id === +el.dataset.id);
        if (room) { selectRoom(room); jumpToRoom(room.id); }
        $roomResults.style.display = 'none';
        $roomSearch.value = '';
      };
      el.addEventListener('click', action);
      el.addEventListener('keydown', e => { if (e.key === 'Enter') action(); });
    });
  }

  // ── Boot ───────────────────────────────────────────────────────────────────
  init();

})();
