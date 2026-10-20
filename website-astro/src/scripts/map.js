/* Dark Pawns — Interactive World Map & Obsidian Constellation Atlas
 *
 * Modes:
 *   - ✦ Graph (3D World / 2D Plane):
 *       - 3D World (/map/world-sphere.json): Interactive 3D Obsidian sphere where users can
 *         drag to rotate in 3D, scroll/pinch to zoom, Shift+drag / right-drag / 2-finger pan,
 *         hover any node to illuminate its connected exit web, and click any room to inspect
 *         its description, exits, mobs, and items.
 *       - 2D Plane (/map/world-graph.json): Planar force-directed Obsidian constellation.
 *   - ▦ Grid (Point-Connected MUD Map):
 *       - World Overview (/map/world-map.json): Stitched 2D cardinal layout of all 91 zones.
 *       - Zone Grid (/map/zone-{id}.json): Classic point-connected MUD map with N/S/E/W/U/D links.
 */
(function () {
  'use strict';

  // ── State ──────────────────────────────────────────────────────────────────
  const zoneCache = {};        // zone_id → loaded zone JSON
  let indexData = null;        // {zones: [{id, name, rooms}], room_zones: {}, room_names: {}}
  let worldMapData = null;     // {rooms: [{id, x, y, sector, zone_id}], links: [{s, t}]}
  let wmRoomPosMap = {};       // room_id → room object (for O(1) hit-testing)
  let worldAdj = {};           // room_id → Set of connected room_ids across the world
  let zoneCentroids = {};      // zone_id → {x, y, name, count}
  let worldZoneLinks = [];     // [{z1, z2, count}]
  let reachableRooms = {};     // room_id → true
  let zoneHulls = {};          // zone_id → {hull, ds, bx0, by0, bx1, by1}

  let currentZoneId = null;    // null = world map; number = focused/selected zone
  let selectedRoomId = null;
  let hoveredRoomId = null;
  let pathStartId = null;
  let pathMode = false;
  let currentRooms = [];       // rooms in the current zone (with x,y)
  let currentLinks = [];       // links [{s, t}] in the current zone

  // Canvas transform state
  let cvTransform = d3.zoomIdentity;
  let viewMode = 'globe';      // 'globe' (3D Obsidian World) | 'graph' (2D Obsidian Plane) | 'grid' (MUD Grid)
  let graphDim = '3d';         // '3d' | '2d' — sub-mode of ✦ Graph
  let roomDegrees = {};        // room_id → degree (exit count)
  let roomInhabitants = {};    // room_id → { mobs: [], items: [] }

  // 2D Obsidian Graph State
  let graphData = null;        // loaded from world-graph.json
  let graphRoomMap = null;     // room_id → projected node {x,y,degree,sector,zone}

  // 3D Obsidian Globe State
  let globeData = null;        // loaded from world-sphere.json
  let globeRoomMap = null;     // room_id → sphere node {id,x,y,z,sector,zone}
  let globeRotation = [0.3, -0.18]; // [theta, phi]
  let globePan = [0, 0];       // [panX, panY] screen-space offset for panning zoomed-in 3D view
  let isDraggingGlobe = false;
  let isPanningGlobe = false;
  let lastMousePos = { x: 0, y: 0 };
  let dragMoveDistance = 0;
  let globeAnimFrame = null;

  // Touch pinch/pan state for 3D Globe
  let touchPinchStartDist = 0;
  let touchPinchStartScale = 1;
  let touchPinchLastMid = { x: 0, y: 0 };

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

  // Lifted sector palette for the dark Obsidian graph canvas
  const SECTOR_COLOR_DARK = {
    0: '#9a948e', 1: '#cf4b42', 2: '#b8c07e',  3: '#6fa061',
    4: '#d0ac7d', 5: '#a39481', 6: '#7fb3d1',  7: '#5c83b8',
    8: '#4a6b9e', 9: '#8fc9f2', 10: '#e0c08a', 11: '#ee7a66',
    12: '#b09c7e', 13: '#dbe2ea', 14: '#5c83b8', 15: '#87a086',
  };
  function sectorColorDark(s) { return SECTOR_COLOR_DARK[s] ?? '#cf4b42'; }

  // ── World Text Reflow Helper (matches src/lib/world-text.mjs) ──────────────
  const DRAWING_RE = /\*--|--\*|\|\s{2,}\|/;
  function reflowWorldText(text) {
    const source = String(text ?? '').replace(/\r\n?/g, '\n').replace(/\s+$/, '');
    if (!source) return { drawing: false, paragraphs: [] };
    if (DRAWING_RE.test(source)) return { drawing: true, paragraphs: [source] };

    const paragraphs = [];
    let current = [];
    const flush = () => {
      if (current.length) paragraphs.push(current.join(' ').replace(/\s+/g, ' ').trim());
      current = [];
    };
    for (const line of source.split('\n')) {
      if (!line.trim()) {
        flush();
        continue;
      }
      if (/^\s/.test(line)) flush();
      current.push(line.trim());
    }
    flush();
    return { drawing: false, paragraphs };
  }

  function renderReflowedDesc(text) {
    const { drawing, paragraphs } = reflowWorldText(text);
    if (!paragraphs.length) return '';
    if (drawing) return `<pre>${escHtml(paragraphs[0])}</pre>`;
    return paragraphs.map(p => `<p>${escHtml(p)}</p>`).join('');
  }

  // ── DOM refs ───────────────────────────────────────────────────────────────
  const $zoneSearch   = document.getElementById('zone-search');
  const $zoneList     = document.getElementById('zone-list');
  const $roomSearch   = document.getElementById('room-search');
  const $roomResults  = document.getElementById('room-search-results');
  const $zoneTitle    = document.getElementById('zone-title');
  const $loading      = document.getElementById('map-loading');
  const $loadingMsg   = document.getElementById('map-loading-msg');
  const $empty        = document.getElementById('map-empty');
  const $legend       = document.getElementById('map-legend');
  const $tooltip      = document.getElementById('map-tooltip');
  const $detail       = document.getElementById('map-detail');
  const $detailName   = document.getElementById('detail-room-name');
  const $detailVnum   = document.getElementById('detail-vnum');
  const $detailDesc   = document.getElementById('detail-desc');
  const $detailExits  = document.getElementById('detail-exits');
  const $detailMobs   = document.getElementById('detail-mobs');
  const $pathStatus   = document.getElementById('path-status');
  const $sidebar      = document.getElementById('map-sidebar');
  const $btnSidebar   = document.getElementById('btn-sidebar');
  const $btnSetStart  = document.getElementById('btn-set-start');
  const $btnClrPath   = document.getElementById('btn-clear-path');
  const $btnPath      = document.getElementById('btn-path');
  const $btnViewGraph = document.getElementById('btn-view-graph');
  const $btnViewGrid  = document.getElementById('btn-view-grid');
  const $graphHud     = document.getElementById('map-graph-hud');
  const $btnGraph3d   = document.getElementById('btn-graph-3d');
  const $btnGraph2d   = document.getElementById('btn-graph-2d');
  const $graphHudHint = document.getElementById('graph-hud-hint');
  const $svgWrap      = document.getElementById('map-svg-wrap');
  const $canvasWrap   = document.getElementById('map-canvas-wrap');
  const canvasEl      = document.getElementById('map-canvas');
  const ctx           = canvasEl ? canvasEl.getContext('2d') : null;

  // ── D3 SVG setup (zone point-connected grid view) ──────────────────────────
  const svgEl  = document.getElementById('map-svg');
  const svg    = d3.select(svgEl);
  const gRoot  = svg.append('g').attr('id', 'map-g');
  const gLinks = gRoot.append('g').attr('class', 'links-layer');
  const gNodes = gRoot.append('g').attr('class', 'nodes-layer');

  const svgZoom = d3.zoom()
    .scaleExtent([0.05, 12])
    .on('zoom', ev => gRoot.attr('transform', ev.transform));
  svg.call(svgZoom);

  // ── Canvas zoom (world map / 2D graph / 3D globe wheel zoom) ───────────────
  const canvasZoom = d3.zoom()
    .scaleExtent([0.04, 18])
    .filter(event => {
      if (viewMode === 'globe') {
        // In 3D globe mode, wheel zooms smoothly; pointer/touch drag is handled by 3D trackball + pan
        return event.type === 'wheel';
      }
      return !event.button;
    })
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
    return String(s ?? '')
      .replace(/&/g, '&amp;').replace(/</g, '&lt;')
      .replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }
  function getRoomName(roomId) {
    return indexData?.room_names?.[String(roomId)] || `Room #${roomId}`;
  }
  function getZoneName(zoneId) {
    return indexData?.zones?.find(z => z.id === zoneId)?.name || `Zone ${zoneId}`;
  }

  function isShowingZoneSvg() {
    return currentZoneId !== null && viewMode === 'grid';
  }

  // ── Init ───────────────────────────────────────────────────────────────────
  function init() {
    showLoading('Loading…');

    if ($btnSidebar) {
      $btnSidebar.addEventListener('click', () => {
        $sidebar.classList.toggle('open');
      });
    }

    document.getElementById('btn-zoom-in')?.addEventListener('click', () => {
      const target = isShowingZoneSvg() ? svg : d3.select(canvasEl);
      const z = isShowingZoneSvg() ? svgZoom : canvasZoom;
      target.transition().duration(200).call(z.scaleBy, 1.4);
    });

    document.getElementById('btn-zoom-out')?.addEventListener('click', () => {
      const target = isShowingZoneSvg() ? svg : d3.select(canvasEl);
      const z = isShowingZoneSvg() ? svgZoom : canvasZoom;
      target.transition().duration(200).call(z.scaleBy, 1 / 1.4);
    });

    document.getElementById('btn-reset')?.addEventListener('click', () => {
      if (isShowingZoneSvg()) {
        fitSvgToZone();
      } else if (viewMode === 'globe') {
        globePan = [0, 0];
        globeRotation = [0.3, -0.18];
        d3.select(canvasEl).transition().duration(280).call(canvasZoom.transform, d3.zoomIdentity);
      } else if (viewMode === 'graph') {
        fitCanvasToGraph();
      } else {
        fitCanvasToWorldMap();
      }
    });

    $btnPath?.addEventListener('click', togglePathMode);

    // Top-level view mode buttons: ✦ Graph vs ▦ Grid
    $btnViewGraph?.addEventListener('click', () => {
      setViewMode(graphDim === '2d' ? 'graph' : 'globe');
    });
    $btnViewGrid?.addEventListener('click', () => {
      setViewMode('grid');
    });

    // Sub-mode switcher inside ✦ Graph HUD: 3D World vs 2D Plane
    $btnGraph3d?.addEventListener('click', () => {
      graphDim = '3d';
      setViewMode('globe');
    });
    $btnGraph2d?.addEventListener('click', () => {
      graphDim = '2d';
      setViewMode('graph');
    });

    document.getElementById('btn-expand-zone')?.addEventListener('click', () => {
      const roomId = selectedRoomId;
      const zId = roomId !== null
        ? (indexData?.room_zones?.[String(roomId)] ?? currentZoneId)
        : currentZoneId;
      if (zId !== null && zId !== undefined) {
        viewMode = 'grid';
        syncViewBtn();
        selectZone(zId).then(() => {
          if (roomId !== null) {
            const zRoom = currentRooms.find(r => r.id === roomId);
            if (zRoom) { selectRoom(zRoom); jumpToRoom(roomId); }
          }
        });
      }
    });

    $btnSetStart?.addEventListener('click', () => setPathStart(selectedRoomId));
    $btnClrPath?.addEventListener('click', resetPath);
    document.getElementById('detail-close')?.addEventListener('click', deselectRoom);

    $zoneSearch?.addEventListener('input', e => renderZoneList(e.target.value));
    $roomSearch?.addEventListener('input', e => handleRoomSearch(e.target.value));

    document.addEventListener('click', e => {
      if (window.innerWidth > 768) return;
      if ($sidebar.classList.contains('open') &&
          !$sidebar.contains(e.target) &&
          e.target !== $btnSidebar) {
        $sidebar.classList.remove('open');
      }
    });

    // Fetch database.json in the background to index room mobs & items
    fetch('/data/database.json')
      .then(r => r.ok ? r.json() : null)
      .then(db => {
        if (!db) return;
        const byRoom = {};
        const ensure = (rid) => (byRoom[rid] || (byRoom[rid] = { mobs: [], items: [] }));
        Object.values(db.mobs || {}).forEach(m => {
          (m.spw || []).forEach(s => {
            if (s.room !== undefined) {
              ensure(s.room).mobs.push({
                v: m.v,
                s: m.s || m.k,
                lvl: m.lvl,
                alg: m.alg,
                shop: Boolean(m.shop),
                drpCount: (m.drp || []).length,
              });
            }
          });
        });
        Object.values(db.items || {}).forEach(it => {
          (it.rms || []).forEach(rm => {
            if (rm.room !== undefined) {
              ensure(rm.room).items.push({
                v: it.v,
                s: it.s || it.k,
                type: it.type,
              });
            }
          });
        });
        roomInhabitants = byRoom;
        if (selectedRoomId !== null) {
          renderRoomInhabitants(selectedRoomId);
        }
      })
      .catch(() => {});

    fetch('/map/map-index.json')
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        indexData = data;
        hideLoading();
        $empty.style.display = 'none';
        renderZoneList('');

        const params = new URLSearchParams(window.location.search);
        const roomArg = params.get('room');
        const zoneArg = params.get('zone');
        const hash = window.location.hash;

        if (hash === '#grid') {
          viewMode = 'grid';
        } else if (hash === '#2d' || hash === '#graph') {
          graphDim = '2d';
          viewMode = 'graph';
        } else if (zoneArg || roomArg) {
          // Deep-links to a specific ?zone= or ?room= open directly in the classic 2D point-connected MUD grid
          viewMode = 'grid';
        } else {
          // Default to the rich 3D Obsidian World
          graphDim = '3d';
          viewMode = 'globe';
        }
        syncViewBtn();

        if (zoneArg) {
          selectZone(parseInt(zoneArg, 10));
        } else if (roomArg) {
          resolveRoomDeepLink(parseInt(roomArg, 10));
        } else {
          selectWorldOverview({ pushHistory: false });
        }
      })
      .catch(err => {
        if ($loadingMsg) $loadingMsg.textContent = `Failed: ${err.message}`;
      });
  }

  function resolveRoomDeepLink(roomId) {
    if (!indexData) return;
    const zoneId = indexData.room_zones?.[String(roomId)];
    if (zoneId !== undefined) {
      viewMode = 'grid';
      syncViewBtn();
      selectZone(zoneId, { pushHistory: false }).then(() => {
        const room = currentRooms.find(r => r.id === roomId);
        if (room) { selectRoom(room); jumpToRoom(roomId); }
        else jumpToRoom(roomId);
      });
    } else {
      selectWorldOverview({ pushHistory: false });
    }
  }

  function pushState(params) {
    const url = new URL(window.location.href);
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

  window.addEventListener('popstate', () => {
    const p = new URLSearchParams(window.location.search);
    const zoneArg = p.get('zone');
    const roomArg = p.get('room');
    const hash = window.location.hash;

    if (hash === '#grid') {
      viewMode = 'grid';
    } else if (hash === '#2d' || hash === '#graph') {
      graphDim = '2d';
      viewMode = 'graph';
    } else if (zoneArg || roomArg) {
      viewMode = 'grid';
    } else {
      graphDim = '3d';
      viewMode = 'globe';
    }
    syncViewBtn();

    if (zoneArg) {
      selectZone(parseInt(zoneArg, 10), { pushHistory: false }).then(() => {
        if (roomArg) jumpToRoom(parseInt(roomArg, 10));
      });
    } else if (roomArg) {
      resolveRoomDeepLink(parseInt(roomArg, 10));
    } else {
      selectWorldOverview({ pushHistory: false });
    }
  });

  function copyRoomLink(roomId) {
    const url = `${location.origin}${location.pathname}?room=${roomId}`;
    if (navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(url).then(() => flashCopyBtn('Copied!')).catch(() => flashCopyBtn('Copy failed'));
    } else {
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
      !q || z.name.toLowerCase().includes(q) || String(z.id).includes(q)
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
        <a class="zone-record-link" href="/zones/${z.id}/" title="Zone record page" aria-label="Zone record page for ${escHtml(z.name)}">▤</a>
        <span class="zone-count">${z.rooms}</span>
      </div>`).join('');

    $zoneList.innerHTML = html;

    document.getElementById('item-world-map')?.addEventListener('click', () => {
      selectWorldOverview();
    });
    $zoneList.querySelectorAll('.zone-item[data-zone]').forEach(el =>
      el.addEventListener('click', () => {
        const zid = +el.dataset.zone;
        onSidebarZoneClick(zid);
      })
    );
    $zoneList.querySelectorAll('.zone-record-link').forEach(el =>
      el.addEventListener('click', e => e.stopPropagation())
    );
  }

  function onSidebarZoneClick(zoneId) {
    if (viewMode === 'globe' || viewMode === 'graph') {
      // In Obsidian Graph mode, clicking a zone focuses & highlights that zone cluster in the graph!
      currentZoneId = zoneId;
      updateZoneListActive();
      const zName = getZoneName(zoneId);
      $zoneTitle.textContent = `${zName.toUpperCase()} · GRAPH`;
      if (window.innerWidth <= 768) $sidebar.classList.remove('open');

      if (viewMode === 'globe') {
        focusGlobeOnZone(zoneId);
      } else {
        focusGraph2DOnZone(zoneId);
      }
      return;
    }
    selectZone(zoneId);
  }

  function focusGlobeOnZone(zoneId) {
    if (!globeData) {
      loadGlobeAndDraw(() => focusGlobeOnZone(zoneId));
      return;
    }
    const zNode = globeData.zones?.find(z => z.id === zoneId);
    if (!zNode) return;

    // Rotate zNode (x, y, z) to face the camera (x_rot = 0, y_rot = 0, z_rot > 0)
    const targetTheta = -Math.atan2(zNode.x, zNode.z);
    const z1 = Math.hypot(zNode.x, zNode.z);
    const targetPhi = Math.atan2(zNode.y, z1);

    const startTheta = globeRotation[0];
    const startPhi = globeRotation[1];
    const startPanX = globePan[0];
    const startPanY = globePan[1];
    const startK = cvTransform.k;
    const targetK = Math.max(startK, 2.3);

    if (globeAnimFrame) cancelAnimationFrame(globeAnimFrame);
    const t0 = performance.now();
    const dur = 480;

    function step(now) {
      const p = Math.min(1, (now - t0) / dur);
      const ease = 1 - Math.pow(1 - p, 3);
      globeRotation[0] = startTheta + (targetTheta - startTheta) * ease;
      globeRotation[1] = startPhi + (targetPhi - startPhi) * ease;
      globePan[0] = startPanX * (1 - ease);
      globePan[1] = startPanY * (1 - ease);
      const kNow = startK + (targetK - startK) * ease;
      d3.select(canvasEl).call(canvasZoom.transform, d3.zoomIdentity.scale(kNow));
      if (p < 1) {
        globeAnimFrame = requestAnimationFrame(step);
      } else {
        globeAnimFrame = null;
        drawWorldMap();
      }
    }
    globeAnimFrame = requestAnimationFrame(step);
  }

  function focusGraph2DOnZone(zoneId) {
    if (!graphData || !canvasEl) {
      loadGraphAndDraw(() => focusGraph2DOnZone(zoneId));
      return;
    }
    const zRooms = graphData.rooms.filter(r => r.zone === zoneId);
    if (!zRooms.length) return;
    const dpr = window.devicePixelRatio || 1;
    const W = canvasEl.width / dpr;
    const H = canvasEl.height / dpr;
    let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
    for (const r of zRooms) {
      if (r.x < minX) minX = r.x; if (r.x > maxX) maxX = r.x;
      if (r.y < minY) minY = r.y; if (r.y > maxY) maxY = r.y;
    }
    const pad = 80;
    const k = Math.max(0.8, Math.min(8,
      Math.min((W - pad * 2) / (maxX - minX + 20), (H - pad * 2) / (maxY - minY + 20))
    ));
    const tx = W / 2 - k * ((minX + maxX) / 2);
    const ty = H / 2 - k * ((minY + maxY) / 2);
    d3.select(canvasEl).transition().duration(420).call(
      canvasZoom.transform,
      d3.zoomIdentity.translate(tx, ty).scale(k)
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

  function loadGlobeAndDraw(cb) {
    if (globeData) {
      drawWorldMap();
      if (cb) cb();
      return;
    }
    showLoading('Loading 3D world graph…');
    fetch('/map/world-sphere.json')
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        globeData = data;
        globeRoomMap = {};
        for (const r of data.rooms) {
          globeRoomMap[r.id] = r;
        }
        hideLoading();
        drawWorldMap();
        if (cb) cb();
      })
      .catch(err => {
        if ($loadingMsg) $loadingMsg.textContent = `Failed to load 3D world: ${err.message}`;
      });
  }

  function loadGraphAndDraw(cb) {
    if (graphData) {
      drawWorldMap();
      if (cb) cb();
      return;
    }
    showLoading('Loading 2D constellation graph…');
    fetch('/map/world-graph.json')
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        graphData = data;
        graphRoomMap = {};
        for (const r of data.rooms) {
          r.degree = roomDegrees[r.id] || 0;
          graphRoomMap[r.id] = r;
        }
        hideLoading();
        fitCanvasToGraph();
        drawWorldMap();
        if (cb) cb();
      })
      .catch(err => {
        if ($loadingMsg) $loadingMsg.textContent = `Failed to load graph: ${err.message}`;
      });
  }

  function fitCanvasToGraph() {
    if (!graphData || !canvasEl) return;
    const W = canvasEl.width  / (window.devicePixelRatio || 1);
    const H = canvasEl.height / (window.devicePixelRatio || 1);
    let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
    for (const r of graphData.rooms) {
      if (r.x < minX) minX = r.x;  if (r.x > maxX) maxX = r.x;
      if (r.y < minY) minY = r.y;  if (r.y > maxY) maxY = r.y;
    }
    const pad = 40;
    const k   = Math.max(0.02, Math.min(10,
      Math.min((W - pad * 2) / (maxX - minX + 1), (H - pad * 2) / (maxY - minY + 1))
    ));
    const tx  = W / 2 - k * ((minX + maxX) / 2);
    const ty  = H / 2 - k * ((minY + maxY) / 2);
    d3.select(canvasEl).call(canvasZoom.transform,
      d3.zoomIdentity.translate(tx, ty).scale(k));
  }

  // ── World overview — Canvas ────────────────────────────────────────────────
  function selectWorldOverview({ pushHistory = true } = {}) {
    currentZoneId = null;
    deselectRoom();
    resetPath();
    updateZoneListActive();

    if (pushHistory) pushState({});
    $zoneTitle.textContent = viewMode === 'grid' ? 'WORLD MAP · GRID' : 'WORLD MAP · OBSIDIAN GRAPH';
    $legend.classList.add('visible');
    $svgWrap.style.display    = 'none';
    $canvasWrap.style.display = '';
    syncViewBtn();
    if (window.innerWidth <= 768) $sidebar.classList.remove('open');

    if (worldMapData) {
      resizeCanvas();
      if (viewMode === 'globe') {
        globePan = [0, 0];
        d3.select(canvasEl).call(canvasZoom.transform, d3.zoomIdentity);
        loadGlobeAndDraw();
      } else if (viewMode === 'graph') {
        loadGraphAndDraw();
      } else {
        fitCanvasToWorldMap();
        drawWorldMap();
      }
      return;
    }

    showLoading('Loading world map…');
    fetch('/map/world-map.json')
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        worldMapData = data;
        wmRoomPosMap = {};
        worldAdj = {};
        for (const room of data.rooms) {
          wmRoomPosMap[room.id] = room;
          worldAdj[room.id] = new Set();
        }

        roomDegrees = {};
        for (const lk of data.links) {
          roomDegrees[lk.s] = (roomDegrees[lk.s] || 0) + 1;
          roomDegrees[lk.t] = (roomDegrees[lk.t] || 0) + 1;
          worldAdj[lk.s]?.add(lk.t);
          worldAdj[lk.t]?.add(lk.s);
        }

        // 1. Compute Zone Centroids
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

        // 2. Compute Inter-Zone Link Counts
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

        // 3. Run BFS from room 0 to find Reachable Rooms
        reachableRooms = { 0: true };
        const queue = [0];
        let head = 0;
        while (head < queue.length) {
          const u = queue[head++];
          const neighbors = worldAdj[u];
          if (neighbors) {
            for (const v of neighbors) {
              if (!reachableRooms[v]) {
                reachableRooms[v] = true;
                queue.push(v);
              }
            }
          }
        }

        // 4. Per-Zone Convex Hulls for territory shading
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
        if (viewMode === 'globe') {
          d3.select(canvasEl).call(canvasZoom.transform, d3.zoomIdentity);
        } else if (viewMode === 'grid') {
          fitCanvasToWorldMap();
        }
        attachCanvasEvents();

        if (viewMode === 'globe') {
          loadGlobeAndDraw();
        } else if (viewMode === 'graph') {
          loadGraphAndDraw();
        } else {
          drawWorldMap();
        }
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
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  }

  window.addEventListener('resize', () => {
    if (!isShowingZoneSvg() && worldMapData) {
      resizeCanvas();
      drawWorldMap();
    }
  });

  function dotRadius(k) {
    if (k < 0.15) return 1.5 / k;
    if (k < 0.5)  return 2.0 / k;
    if (k < 1.5)  return 2.5 / k;
    if (k < 4)    return 3.5 / k;
    return 5.0 / k;
  }

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

  function syncViewBtn() {
    const isGraph = viewMode === 'globe' || viewMode === 'graph';
    $btnViewGraph?.classList.toggle('active', isGraph);
    $btnViewGrid?.classList.toggle('active', viewMode === 'grid');

    if ($graphHud) {
      $graphHud.hidden = !isGraph || isShowingZoneSvg();
    }
    $btnGraph3d?.classList.toggle('active', viewMode === 'globe');
    $btnGraph2d?.classList.toggle('active', viewMode === 'graph');

    if ($graphHudHint) {
      $graphHudHint.textContent = viewMode === 'globe'
        ? 'Drag to rotate 3D · Scroll to zoom · Shift+drag to pan · Hover or click any room'
        : 'Drag to pan · Scroll to zoom · Hover or click any room';
    }

    const $expandSection = document.getElementById('detail-expand-section');
    if ($expandSection) {
      $expandSection.style.display = isShowingZoneSvg() ? 'none' : '';
    }
  }

  function setViewMode(mode) {
    const prevZone = currentZoneId;
    const prevRoom = selectedRoomId;
    viewMode = mode;
    if (mode === 'globe') graphDim = '3d';
    if (mode === 'graph') graphDim = '2d';
    syncViewBtn();

    if (mode === 'grid') {
      history.replaceState(null, document.title, window.location.pathname + window.location.search + '#grid');
      if (prevZone !== null) {
        selectZone(prevZone, { pushHistory: false }).then(() => {
          if (prevRoom !== null) {
            const zRoom = currentRooms.find(r => r.id === prevRoom);
            if (zRoom) { selectRoom(zRoom); jumpToRoom(prevRoom); }
          }
        });
        return;
      }
      $svgWrap.style.display = 'none';
      $canvasWrap.style.display = '';
      $zoneTitle.textContent = 'WORLD MAP · GRID';
      if (worldMapData) {
        resizeCanvas();
        fitCanvasToWorldMap();
        drawWorldMap();
      } else {
        selectWorldOverview({ pushHistory: false });
      }
      return;
    }

    // Switching to ✦ Graph ('globe' 3D or 'graph' 2D)
    const hashSuffix = mode === 'graph' ? '#2d' : '';
    history.replaceState(null, document.title, window.location.pathname + window.location.search + hashSuffix);

    $svgWrap.style.display = 'none';
    $canvasWrap.style.display = '';
    $legend.classList.add('visible');
    if (prevZone !== null) {
      $zoneTitle.textContent = `${getZoneName(prevZone).toUpperCase()} · GRAPH`;
    } else {
      $zoneTitle.textContent = 'WORLD MAP · OBSIDIAN GRAPH';
    }

    const ensureWorldLoaded = worldMapData
      ? Promise.resolve()
      : new Promise(resolve => {
          selectWorldOverview({ pushHistory: false });
          const check = setInterval(() => {
            if (worldMapData) { clearInterval(check); resolve(); }
          }, 50);
        });

    ensureWorldLoaded.then(() => {
      resizeCanvas();
      if (mode === 'globe') {
        globePan = [0, 0];
        d3.select(canvasEl).call(canvasZoom.transform, d3.zoomIdentity);
        loadGlobeAndDraw(() => {
          if (prevZone !== null) focusGlobeOnZone(prevZone);
        });
      } else {
        loadGraphAndDraw(() => {
          if (prevZone !== null) focusGraph2DOnZone(prevZone);
        });
      }
    });
  }

  // ── 3D Rotation Math ────────────────────────────────────────────────────────
  function rotate3D(p, theta, phi) {
    const cosT = Math.cos(theta), sinT = Math.sin(theta);
    const x1 = p.x * cosT - p.z * sinT;
    const z1 = p.x * sinT + p.z * cosT;

    const cosP = Math.cos(phi), sinP = Math.sin(phi);
    const y2 = p.y * cosP - z1 * sinP;
    const z2 = p.y * sinP + z1 * cosP;

    return { x: x1, y: y2, z: z2 };
  }

  // ── 3D Volumetric Obsidian World Renderer ───────────────────────────────────
  function drawGlobe() {
    if (!ctx || !globeData || !worldMapData) return;
    const dpr = window.devicePixelRatio || 1;
    const W   = canvasEl.width  / dpr;
    const H   = canvasEl.height / dpr;

    // Full-bleed dark Obsidian plate
    ctx.fillStyle = '#0a0908';
    ctx.fillRect(0, 0, W, H);

    const cx = W / 2 + globePan[0];
    const cy = H / 2 + globePan[1];
    const R = Math.min(W, H) * 0.44 * cvTransform.k;

    ctx.save();

    // Subtle warm volumetric glow at the center of the 3D constellation
    const glow = ctx.createRadialGradient(cx, cy, 0, cx, cy, R * 1.15);
    glow.addColorStop(0, 'rgba(42, 34, 26, 0.72)');
    glow.addColorStop(0.55, 'rgba(20, 16, 13, 0.45)');
    glow.addColorStop(1, 'rgba(10, 9, 8, 0)');
    ctx.fillStyle = glow;
    ctx.fillRect(0, 0, W, H);

    // 1. Rotate & Project All 9,590 Rooms in 3D space with perspective depth
    const roomProj = {};
    const visibleRooms = [];
    for (const r of globeData.rooms) {
      const rot = rotate3D(r, globeRotation[0], globeRotation[1]);
      // Camera sits at z = +2.8 looking toward the origin; perspective scale = 2.6 / (2.6 - z)
      const zClamped = Math.max(-1.6, Math.min(1.6, rot.z));
      const persp = 2.6 / (2.6 - zClamped * 0.55);
      const sx = cx + R * rot.x * persp;
      const sy = cy + R * rot.y * persp;

      const proj = {
        id: r.id,
        sector: r.sector,
        zone: r.zone,
        x_rot: rot.x,
        y_rot: rot.y,
        z_rot: rot.z,
        persp,
        sx,
        sy,
        degree: roomDegrees[r.id] || 0,
      };
      roomProj[r.id] = proj;
      visibleRooms.push(proj);
    }

    const activeFocusId = hoveredRoomId ?? selectedRoomId;
    const activeNeighbors = activeFocusId !== null ? worldAdj[activeFocusId] : null;
    const lodLeaves = cvTransform.k >= 1.25;

    // 2. Connective Edges (Luminous 3D web + Obsidian-style neighbor highlight)
    ctx.lineWidth = 0.6;
    ctx.strokeStyle = activeFocusId !== null
      ? 'rgba(239, 231, 214, 0.06)'
      : 'rgba(239, 231, 214, 0.14)';
    ctx.beginPath();

    const highlightEdges = [];
    const zoneFocusEdges = [];

    for (const lk of worldMapData.links) {
      const s = roomProj[lk.s];
      const t = roomProj[lk.t];
      if (!s || !t) continue;

      if (activeFocusId !== null && (lk.s === activeFocusId || lk.t === activeFocusId)) {
        highlightEdges.push([s, t]);
        continue;
      }
      if (currentZoneId !== null && s.zone === currentZoneId && t.zone === currentZoneId) {
        zoneFocusEdges.push([s, t]);
        continue;
      }
      if (!lodLeaves && s.degree <= 1 && t.degree <= 1) continue;

      ctx.moveTo(s.sx, s.sy);
      ctx.lineTo(t.sx, t.sy);
    }
    ctx.stroke();

    // Focused zone edges
    if (zoneFocusEdges.length > 0) {
      ctx.lineWidth = 1.05;
      ctx.strokeStyle = 'rgba(207, 75, 66, 0.45)';
      ctx.beginPath();
      for (const [s, t] of zoneFocusEdges) {
        ctx.moveTo(s.sx, s.sy);
        ctx.lineTo(t.sx, t.sy);
      }
      ctx.stroke();
    }

    // Active hovered/selected room's direct exit web
    if (highlightEdges.length > 0) {
      ctx.lineWidth = 1.75;
      ctx.strokeStyle = '#cf4b42';
      ctx.beginPath();
      for (const [s, t] of highlightEdges) {
        ctx.moveTo(s.sx, s.sy);
        ctx.lineTo(t.sx, t.sy);
      }
      ctx.stroke();
    }

    // 3. Luminous Nodes (Sorted back-to-front by z_rot for genuine 3D occlusion & depth)
    visibleRooms.sort((a, b) => a.z_rot - b.z_rot);
    const zoomScale = Math.min(2.4, Math.max(0.85, Math.sqrt(cvTransform.k)));

    for (const r of visibleRooms) {
      const isFocused = r.id === activeFocusId;
      const isNeighbor = Boolean(activeNeighbors && activeNeighbors.has(r.id));
      const inFocusedZone = currentZoneId !== null && r.zone === currentZoneId;

      if (!lodLeaves && r.degree <= 1 && !isFocused && !isNeighbor && !inFocusedZone) continue;

      // Normalize depth into [0.22, 1.0] so back clusters recede into atmospheric haze
      const depthNorm = Math.max(0.22, Math.min(1.0, (r.z_rot + 1.05) / 2.0));
      const baseR = r.degree <= 1 ? 1.45 : (r.degree <= 4 ? 2.35 : 3.8);
      const dr = baseR * r.persp * zoomScale * (0.72 + 0.35 * depthNorm);

      if (r.degree >= 6 || isFocused || isNeighbor) {
        ctx.fillStyle = (isFocused || isNeighbor)
          ? 'rgba(207, 75, 66, 0.34)'
          : `rgba(207, 75, 66, ${(0.18 * depthNorm).toFixed(3)})`;
        ctx.beginPath();
        ctx.arc(r.sx, r.sy, dr * 2.2, 0, Math.PI * 2);
        ctx.fill();
      }

      ctx.globalAlpha = (isFocused || isNeighbor || inFocusedZone)
        ? 1.0
        : (activeFocusId !== null ? 0.32 * depthNorm : depthNorm);
      ctx.fillStyle = isFocused
        ? '#efe7d6'
        : isNeighbor
        ? '#cf4b42'
        : sectorColorDark(r.sector);

      ctx.beginPath();
      ctx.arc(r.sx, r.sy, isFocused ? dr * 1.45 : dr, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.globalAlpha = 1.0;

    // 4. Selected / Hovered Room Ring
    for (const ringId of [selectedRoomId, hoveredRoomId]) {
      if (ringId === null) continue;
      const sel = roomProj[ringId];
      if (sel) {
        ctx.beginPath();
        ctx.arc(sel.sx, sel.sy, 8.5 * zoomScale, 0, Math.PI * 2);
        ctx.strokeStyle = ringId === selectedRoomId ? '#cf4b42' : '#efe7d6';
        ctx.lineWidth = 2.0;
        ctx.stroke();
      }
    }

    // 5. Floating Zone Typographic Labels (decluttered, foreground-weighted)
    if (cvTransform.k >= 0.35 && globeData.zones) {
      const labelAll = cvTransform.k >= 1.85;
      ctx.save();
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';

      const candidates = [];
      for (const z of globeData.zones) {
        const isCurrentZone = z.id === currentZoneId;
        if (!labelAll && !isCurrentZone && (z.count || 0) < 120) continue;
        const rot = rotate3D(z, globeRotation[0], globeRotation[1]);
        if (rot.z < -0.15 && !isCurrentZone) continue;
        const zClamped = Math.max(-1.6, Math.min(1.6, rot.z));
        const persp = 2.6 / (2.6 - zClamped * 0.55);
        const sx = cx + R * rot.x * persp;
        const sy = cy + R * rot.y * persp;
        candidates.push({ ...z, sx, sy, z_rot: rot.z, isCurrentZone });
      }

      candidates.sort((a, b) => (b.isCurrentZone - a.isCurrentZone) || ((b.count || 0) - (a.count || 0)));
      const placed = [];
      const minGap = labelAll ? 58 : 105;

      for (const z of candidates) {
        if (!z.isCurrentZone && placed.some(p => Math.hypot(p.sx - z.sx, p.sy - z.sy) < minGap)) continue;
        placed.push(z);
        const fs = z.isCurrentZone ? 13 : (cvTransform.k >= 1.2 ? 11.5 : 11);
        ctx.font = `${z.isCurrentZone || cvTransform.k >= 1.0 ? 'bold ' : ''}${fs}px 'DM Serif Display', Georgia, serif`;
        ctx.lineWidth = 3.4;
        ctx.strokeStyle = 'rgba(10, 9, 8, 0.88)';
        ctx.strokeText(z.name.toUpperCase(), z.sx, z.sy - 10);
        const depthAlpha = Math.max(0.5, Math.min(0.95, 0.65 + 0.3 * z.z_rot));
        ctx.fillStyle = z.isCurrentZone
          ? '#cf4b42'
          : `rgba(239, 231, 214, ${depthAlpha.toFixed(2)})`;
        ctx.fillText(z.name.toUpperCase(), z.sx, z.sy - 10);
      }
      ctx.restore();
    }

    ctx.restore();
  }

  // ── 2D Obsidian Constellation Graph Renderer ────────────────────────────────
  function drawGraph() {
    if (!ctx || !graphData || !worldMapData) return;
    const dpr = window.devicePixelRatio || 1;
    const W   = canvasEl.width  / dpr;
    const H   = canvasEl.height / dpr;
    const { x: tx, y: ty, k } = cvTransform;

    ctx.fillStyle = '#0a0908';
    ctx.fillRect(0, 0, W, H);
    const glow = ctx.createRadialGradient(W / 2, H / 2, 0, W / 2, H / 2, Math.max(W, H) * 0.55);
    glow.addColorStop(0, 'rgba(38, 32, 25, 0.6)');
    glow.addColorStop(1, 'rgba(10, 9, 8, 0)');
    ctx.fillStyle = glow;
    ctx.fillRect(0, 0, W, H);

    ctx.save();
    ctx.translate(tx, ty);
    ctx.scale(k, k);

    const pad = 60 / k;
    const vx0 = -tx / k - pad,  vx1 = (W - tx) / k + pad;
    const vy0 = -ty / k - pad,  vy1 = (H - ty) / k + pad;
    const inView = p => p.x >= vx0 && p.x <= vx1 && p.y >= vy0 && p.y <= vy1;

    const lodLeaves = k >= 1.2;
    const allLabels = k >= 2.5;
    const activeFocusId = hoveredRoomId ?? selectedRoomId;
    const activeNeighbors = activeFocusId !== null ? worldAdj[activeFocusId] : null;

    // 1. The Web — all edges, plus highlighted neighbor web
    ctx.lineWidth = 0.7 / k;
    ctx.strokeStyle = activeFocusId !== null
      ? 'rgba(239, 231, 214, 0.08)'
      : 'rgba(239, 231, 214, 0.16)';
    ctx.beginPath();
    const highlightEdges = [];
    for (const lk of worldMapData.links) {
      const s = graphRoomMap[lk.s];
      const t = graphRoomMap[lk.t];
      if (!s || !t) continue;
      if (!inView(s) && !inView(t)) continue;
      if (activeFocusId !== null && (lk.s === activeFocusId || lk.t === activeFocusId)) {
        highlightEdges.push([s, t]);
        continue;
      }
      if (!lodLeaves && s.degree <= 1 && t.degree <= 1) continue;
      ctx.moveTo(s.x, s.y);
      ctx.lineTo(t.x, t.y);
    }
    ctx.stroke();

    if (highlightEdges.length > 0) {
      ctx.lineWidth = 1.8 / k;
      ctx.strokeStyle = '#cf4b42';
      ctx.beginPath();
      for (const [s, t] of highlightEdges) {
        ctx.moveTo(s.x, s.y);
        ctx.lineTo(t.x, t.y);
      }
      ctx.stroke();
    }

    // 2. Hub halos
    for (const r of graphData.rooms) {
      if (r.degree < 6) continue;
      if (!inView(r)) continue;
      const dr = 5.5 / k;
      ctx.fillStyle = 'rgba(207, 75, 66, 0.22)';
      ctx.beginPath();
      ctx.arc(r.x, r.y, dr * 2.2, 0, Math.PI * 2);
      ctx.fill();
    }

    // 3. Nodes — sector-colored, sized by degree
    for (const r of graphData.rooms) {
      const isFocused = r.id === activeFocusId;
      const isNeighbor = Boolean(activeNeighbors && activeNeighbors.has(r.id));
      const inFocusedZone = currentZoneId !== null && r.zone === currentZoneId;
      if (!lodLeaves && r.degree <= 1 && !isFocused && !isNeighbor && !inFocusedZone) continue;
      if (!inView(r)) continue;
      const base = r.degree <= 1 ? 1.4 : (r.degree <= 4 ? 2.4 : 4.0);
      const dr = Math.max(base, base / k * 0.35);
      ctx.fillStyle = isFocused
        ? '#efe7d6'
        : isNeighbor
        ? '#cf4b42'
        : sectorColorDark(r.sector);
      ctx.beginPath();
      ctx.arc(r.x, r.y, isFocused ? dr * 1.4 : dr, 0, Math.PI * 2);
      ctx.fill();
    }

    // 4. Selection / Hover ring
    for (const ringId of [selectedRoomId, hoveredRoomId]) {
      if (ringId === null) continue;
      const sel = graphRoomMap[ringId];
      if (sel) {
        ctx.beginPath();
        ctx.arc(sel.x, sel.y, 9 / k, 0, Math.PI * 2);
        ctx.strokeStyle = ringId === selectedRoomId ? '#cf4b42' : '#efe7d6';
        ctx.lineWidth = 1.8 / k;
        ctx.stroke();
      }
    }

    // 5. Zone labels
    ctx.save();
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    const labelZones = graphData.zones
      .filter(z => allLabels || z.id === currentZoneId || (z.count || 0) >= 130)
      .filter(z => inView(z))
      .sort((a, b) => (b.count || 0) - (a.count || 0));
    const placed = [];
    const minGap = allLabels ? 60 / k : 130 / k;
    for (const z of labelZones) {
      if (z.id !== currentZoneId && placed.some(p => Math.hypot(p.x - z.x, p.y - z.y) < minGap)) continue;
      placed.push(z);
      const fs = (allLabels ? 11 : 13) / k;
      ctx.font = `${allLabels ? '' : 'bold '}${fs}px 'DM Serif Display', Georgia, serif`;
      ctx.lineWidth = 3.5 / k;
      ctx.strokeStyle = 'rgba(10, 9, 8, 0.85)';
      ctx.strokeText(z.name.toUpperCase(), z.x, z.y);
      ctx.fillStyle = z.id === currentZoneId ? '#cf4b42' : 'rgba(239, 231, 214, 0.88)';
      ctx.fillText(z.name.toUpperCase(), z.x, z.y);
    }
    ctx.restore();

    ctx.restore();
  }

  // ── World Map Dispatcher & Stitched 2D Grid Renderer ────────────────────────
  function drawWorldMap() {
    if (!ctx || !worldMapData) return;
    if (viewMode === 'globe') {
      drawGlobe();
      return;
    }
    if (viewMode === 'graph') {
      drawGraph();
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

    const pad = 80 / k;
    const vx0 = -tx / k - pad,  vx1 = (W - tx) / k + pad;
    const vy0 = -ty / k - pad,  vy1 = (H - ty) / k + pad;

    // 1. Per-Zone Territory Shading
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

    // 2. Inter-Zone Links
    if (k > 0.12 && worldZoneLinks) {
      ctx.save();
      ctx.strokeStyle = 'rgba(139, 0, 0, 0.08)';
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

    // 3. Intra-Zone Links
    if (k > 0.1) {
      const alpha = Math.min(0.24, Math.max(0.06, k * 0.25));
      ctx.strokeStyle = `rgba(26,22,20,${alpha.toFixed(2)})`;
      ctx.lineWidth   = 0.75 / k;
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

    // 4. Rooms
    const dr = dotRadius(k);
    const TAU = Math.PI * 2;
    const bySector = {};
    for (const room of worldMapData.rooms) {
      if (room.x < vx0 || room.x > vx1 || room.y < vy0 || room.y > vy1) continue;
      const s = room.sector;
      (bySector[s] ??= []).push(room);
    }

    for (const [sector, rooms] of Object.entries(bySector)) {
      ctx.fillStyle = SECTOR_COLOR[+sector] ?? '#a8201a';
      ctx.beginPath();
      for (const r of rooms) {
        if (reachableRooms[r.id]) {
          ctx.moveTo(r.x + dr, r.y);
          ctx.arc(r.x, r.y, dr, 0, TAU);
        }
      }
      ctx.fill();

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

    // 5. Selected Room Highlight
    const selRoom = selectedRoomId !== null ? wmRoomPosMap[selectedRoomId] : null;
    if (selRoom) {
      ctx.beginPath();
      ctx.arc(selRoom.x, selRoom.y, dr * 2.5, 0, TAU);
      ctx.strokeStyle = '#a8201a';
      ctx.lineWidth   = 2 / k;
      ctx.stroke();
    }

    // 6. Zone Name Labels
    if (k >= 0.18 && zoneCentroids) {
      ctx.save();
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      const fontSize = k >= 1.0 ? 12 : 10;
      ctx.font = k >= 1.0
        ? `bold ${fontSize / k}px 'DM Serif Display', Georgia, serif`
        : `${fontSize / k}px 'DM Serif Display', Georgia, serif`;
      ctx.fillStyle = k >= 1.0
        ? 'rgba(168, 32, 26, 0.88)'
        : 'rgba(74, 69, 64, 0.65)';
      for (const [, z] of Object.entries(zoneCentroids)) {
        if (z.x < vx0 || z.x > vx1 || z.y < vy0 || z.y > vy1) continue;
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

  // ── Hit-Testing across Grid, 2D Graph, and 3D Globe ─────────────────────────
  function worldMapHitTest(clientX, clientY) {
    if (!worldMapData || !canvasEl) return null;
    const rect = canvasEl.getBoundingClientRect();
    const { x: tx, y: ty, k } = cvTransform;
    const wx = (clientX - rect.left - tx) / k;
    const wy = (clientY - rect.top  - ty) / k;
    const threshold = 14 / k;
    let nearest = null, minDist = threshold;
    for (const room of worldMapData.rooms) {
      const d = Math.hypot(room.x - wx, room.y - wy);
      if (d < minDist) { minDist = d; nearest = room; }
    }
    return nearest;
  }

  function graphHitTest(clientX, clientY) {
    if (!graphData || !canvasEl) return null;
    const rect = canvasEl.getBoundingClientRect();
    const { x: tx, y: ty, k } = cvTransform;
    const wx = (clientX - rect.left - tx) / k;
    const wy = (clientY - rect.top  - ty) / k;
    const threshold = 13 / k;
    let nearest = null, minDist = threshold;
    for (const room of graphData.rooms) {
      const d = Math.hypot(room.x - wx, room.y - wy);
      if (d < minDist) { minDist = d; nearest = room; }
    }
    return nearest;
  }

  function globeHitTest(clientX, clientY) {
    if (!globeData || !canvasEl) return null;
    const rect = canvasEl.getBoundingClientRect();
    const W = rect.width;
    const H = rect.height;
    const cx = W / 2 + globePan[0];
    const cy = H / 2 + globePan[1];
    const R = Math.min(W, H) * 0.44 * cvTransform.k;

    const mx = clientX - rect.left;
    const my = clientY - rect.top;

    let nearest = null;
    let minDist = 16; // screen pixels

    for (const r of globeData.rooms) {
      const rot = rotate3D(r, globeRotation[0], globeRotation[1]);
      if (rot.z < -1.2) continue;
      const zClamped = Math.max(-1.2, Math.min(1.2, rot.z));
      const persp = 2.6 / (2.6 - zClamped * 0.55);
      const sx = cx + R * rot.x * persp;
      const sy = cy + R * rot.y * persp;

      const dist = Math.hypot(sx - mx, sy - my);
      if (dist < minDist) {
        minDist = dist;
        nearest = r;
      }
    }
    return nearest;
  }

  function selectGlobeRoom(room) {
    selectedRoomId = room.id;
    currentZoneId = room.zone;
    updateZoneListActive();
    replaceState({ room: room.id });
    drawWorldMap();

    showLoading('Loading room details…');
    fetchZone(room.zone)
      .then(zoneData => {
        hideLoading();
        currentRooms = zoneData.rooms;
        currentLinks = zoneData.links;
        const zRoom = currentRooms.find(r => r.id === room.id);
        if (zRoom) {
          showDetail(zRoom);
          if (window.innerWidth <= 768) {
            $detail.classList.add('open');
            $detail.classList.remove('hidden');
          } else {
            $detail.classList.remove('hidden');
          }
        }
      })
      .catch(err => {
        hideLoading();
        console.error('Failed to load room zone details:', err);
      });
  }

  function attachCanvasEvents() {
    if (!canvasEl) return;

    canvasEl.addEventListener('contextmenu', e => {
      if (viewMode === 'globe') e.preventDefault();
    });

    canvasEl.addEventListener('click', e => {
      if (viewMode === 'globe') {
        if (dragMoveDistance > 6) return;
        const room = globeHitTest(e.clientX, e.clientY);
        if (room) selectGlobeRoom(room);
        else deselectRoom();
        return;
      }

      if (viewMode === 'graph') {
        const room = graphHitTest(e.clientX, e.clientY);
        if (room) selectGlobeRoom(room);
        else deselectRoom();
        return;
      }

      const room = worldMapHitTest(e.clientX, e.clientY);
      if (!room) return;
      selectZone(room.zone_id).then(() => {
        const zoneRoom = currentRooms.find(r => r.id === room.id);
        if (zoneRoom) { selectRoom(zoneRoom); jumpToRoom(room.id); }
      });
    });

    // 3D Globe trackball rotate (left drag) and pan (Shift+drag or right/middle drag)
    canvasEl.addEventListener('mousedown', e => {
      if (viewMode !== 'globe') return;
      isDraggingGlobe = true;
      isPanningGlobe = e.shiftKey || e.button === 1 || e.button === 2;
      dragMoveDistance = 0;
      lastMousePos = { x: e.clientX, y: e.clientY };
    });

    window.addEventListener('mousemove', e => {
      if (isShowingZoneSvg()) return;

      if (viewMode === 'globe') {
        if (isDraggingGlobe) {
          const dx = e.clientX - lastMousePos.x;
          const dy = e.clientY - lastMousePos.y;
          dragMoveDistance += Math.hypot(dx, dy);
          if (isPanningGlobe || e.shiftKey) {
            globePan[0] += dx;
            globePan[1] += dy;
          } else {
            const sensitivity = 0.0055 / Math.max(1, Math.sqrt(cvTransform.k) * 0.65);
            globeRotation[0] += dx * sensitivity;
            globeRotation[1] = Math.max(-1.48, Math.min(1.48, globeRotation[1] + dy * sensitivity));
          }
          lastMousePos = { x: e.clientX, y: e.clientY };
          hideTooltip();
          drawWorldMap();
        } else {
          const room = globeHitTest(e.clientX, e.clientY);
          const nextHoverId = room ? room.id : null;
          if (nextHoverId !== hoveredRoomId) {
            hoveredRoomId = nextHoverId;
            drawWorldMap();
          }
          if (room) {
            canvasEl.style.cursor = 'pointer';
            const roomName = getRoomName(room.id);
            const zoneName = getZoneName(room.zone);
            showTooltip(
              e.clientX,
              e.clientY,
              `<strong>${escHtml(roomName)}</strong>#${room.id} · ${escHtml(zoneName)} · ${escHtml(SECTOR_NAME[room.sector] ?? 'unknown')}`
            );
          } else {
            canvasEl.style.cursor = '';
            hideTooltip();
          }
        }
        return;
      }

      if (viewMode === 'graph') {
        const room = graphHitTest(e.clientX, e.clientY);
        const nextHoverId = room ? room.id : null;
        if (nextHoverId !== hoveredRoomId) {
          hoveredRoomId = nextHoverId;
          drawWorldMap();
        }
        if (room) {
          canvasEl.style.cursor = 'pointer';
          const roomName = getRoomName(room.id);
          const zoneName = getZoneName(room.zone);
          showTooltip(
            e.clientX,
            e.clientY,
            `<strong>${escHtml(roomName)}</strong>#${room.id} · ${escHtml(zoneName)} · ${escHtml(SECTOR_NAME[room.sector] ?? 'unknown')}`
          );
        } else {
          canvasEl.style.cursor = '';
          hideTooltip();
        }
        return;
      }

      // Grid world overview hover
      const room = worldMapHitTest(e.clientX, e.clientY);
      if (room) {
        canvasEl.style.cursor = 'pointer';
        const roomName = getRoomName(room.id);
        const zoneName = getZoneName(room.zone_id);
        showTooltip(
          e.clientX,
          e.clientY,
          `<strong>${escHtml(roomName)}</strong>#${room.id} · ${escHtml(zoneName)} · ${escHtml(SECTOR_NAME[room.sector] ?? 'unknown')}`
        );
      } else {
        canvasEl.style.cursor = '';
        hideTooltip();
      }
    });

    window.addEventListener('mouseup', () => {
      isDraggingGlobe = false;
      isPanningGlobe = false;
    });

    // Mobile Touch Support for 3D Globe (1-finger rotate, 2-finger pinch-zoom + pan)
    canvasEl.addEventListener('touchstart', e => {
      if (viewMode !== 'globe') return;
      if (e.touches.length === 1) {
        isDraggingGlobe = true;
        dragMoveDistance = 0;
        lastMousePos = { x: e.touches[0].clientX, y: e.touches[0].clientY };
      } else if (e.touches.length === 2) {
        isDraggingGlobe = false;
        const dx = e.touches[1].clientX - e.touches[0].clientX;
        const dy = e.touches[1].clientY - e.touches[0].clientY;
        touchPinchStartDist = Math.hypot(dx, dy) || 1;
        touchPinchStartScale = cvTransform.k;
        touchPinchLastMid = {
          x: (e.touches[0].clientX + e.touches[1].clientX) / 2,
          y: (e.touches[0].clientY + e.touches[1].clientY) / 2,
        };
      }
    }, { passive: true });

    canvasEl.addEventListener('touchmove', e => {
      if (viewMode !== 'globe') return;
      if (e.touches.length === 1 && isDraggingGlobe) {
        const dx = e.touches[0].clientX - lastMousePos.x;
        const dy = e.touches[0].clientY - lastMousePos.y;
        dragMoveDistance += Math.hypot(dx, dy);
        const sensitivity = 0.0055 / Math.max(1, Math.sqrt(cvTransform.k) * 0.65);
        globeRotation[0] += dx * sensitivity;
        globeRotation[1] = Math.max(-1.48, Math.min(1.48, globeRotation[1] + dy * sensitivity));
        lastMousePos = { x: e.touches[0].clientX, y: e.touches[0].clientY };
        drawWorldMap();
      } else if (e.touches.length === 2) {
        const dx = e.touches[1].clientX - e.touches[0].clientX;
        const dy = e.touches[1].clientY - e.touches[0].clientY;
        const dist = Math.hypot(dx, dy) || 1;
        const midX = (e.touches[0].clientX + e.touches[1].clientX) / 2;
        const midY = (e.touches[0].clientY + e.touches[1].clientY) / 2;
        globePan[0] += midX - touchPinchLastMid.x;
        globePan[1] += midY - touchPinchLastMid.y;
        touchPinchLastMid = { x: midX, y: midY };

        const nextScale = Math.max(0.35, Math.min(14, touchPinchStartScale * (dist / touchPinchStartDist)));
        d3.select(canvasEl).call(canvasZoom.transform, d3.zoomIdentity.scale(nextScale));
      }
    }, { passive: true });

    canvasEl.addEventListener('touchend', () => {
      isDraggingGlobe = false;
    });

    canvasEl.addEventListener('mouseleave', () => {
      if (hoveredRoomId !== null) {
        hoveredRoomId = null;
        drawWorldMap();
      }
      hideTooltip();
    });
  }

  // ── Zone view — Point-Connected MUD Grid (SVG) ─────────────────────────────
  function selectZone(zoneId, { pushHistory = true } = {}) {
    currentZoneId = zoneId;
    viewMode = 'grid';
    deselectRoom();
    resetPath();
    updateZoneListActive();
    syncViewBtn();

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

    gLinks.selectAll('line')
      .data(currentLinks, d => `${d.s}-${d.t}`)
      .join('line')
      .attr('class', 'exit-line')
      .attr('x1', d => roomMap[d.s]?.x ?? 0)
      .attr('y1', d => roomMap[d.s]?.y ?? 0)
      .attr('x2', d => roomMap[d.t]?.x ?? 0)
      .attr('y2', d => roomMap[d.t]?.y ?? 0);

    gNodes.selectAll('circle')
      .data(currentRooms, d => d.id)
      .join('circle')
      .attr('class', 'room-node')
      .attr('r', 5.5)
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
    if (currentZoneId !== null && isShowingZoneSvg()) replaceState({ zone: currentZoneId });
    gNodes.selectAll('circle')
      .classed('selected', false)
      .attr('stroke-width', 1.2);
    $detail.classList.add('hidden');
    $detail.classList.remove('open');
    if (!isShowingZoneSvg() && worldMapData) drawWorldMap();
  }

  function showDetail(room) {
    $detailName.textContent = room.name;
    $detailVnum.textContent = `#${room.id} · ${SECTOR_NAME[room.sector] ?? ''}`;
    $detailDesc.innerHTML = renderReflowedDesc(room.desc ?? '');

    const $recordLink = document.getElementById('detail-record-link');
    if ($recordLink) {
      $recordLink.href = `/rooms/${room.id}/`;
      $recordLink.style.display = '';
    }

    const $copyBtn = document.getElementById('detail-copy-link');
    if ($copyBtn) {
      $copyBtn.onclick = () => copyRoomLink(room.id);
    }

    const exits = (room.exits ?? []).filter(e => ['n','s','e','w','u','d'].includes(e.d));
    $detailExits.innerHTML = exits.length
      ? `<h4>Exits</h4>` + exits.map(ex => {
          const target = currentRooms.find(r => r.id === ex.t);
          const targetName = target?.name ?? getRoomName(ex.t);
          return `<div class="exit-item" data-target="${ex.t}" tabindex="0" role="button">
            <span class="exit-dir">${ex.d}</span>
            <span>${escHtml(targetName)}</span>
            <span class="exit-dest">#${ex.t}</span>
          </div>`;
        }).join('')
      : '<p style="font-size:.7rem;color:var(--ink-muted)">No exits.</p>';

    $detailExits.querySelectorAll('.exit-item').forEach(el => {
      const action = () => {
        const tid = +el.dataset.target;
        const tr = currentRooms.find(r => r.id === tid);
        if (tr) {
          if (isShowingZoneSvg()) {
            selectRoom(tr);
            jumpToRoom(tid);
          } else {
            selectGlobeRoom({ id: tid, zone: indexData?.room_zones?.[String(tid)] ?? currentZoneId, sector: tr.sector });
          }
        } else {
          const targetZone = indexData?.room_zones?.[String(tid)];
          if (targetZone !== undefined) {
            if (isShowingZoneSvg()) {
              selectZone(targetZone).then(() => {
                const ntr = currentRooms.find(r => r.id === tid);
                if (ntr) { selectRoom(ntr); jumpToRoom(tid); }
              });
            } else {
              selectGlobeRoom({ id: tid, zone: targetZone, sector: 0 });
            }
          }
        }
      };
      el.addEventListener('click', action);
      el.addEventListener('keydown', e => { if (e.key === 'Enter' || e.key === ' ') action(); });
    });

    renderRoomInhabitants(room.id);

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

  function renderRoomInhabitants(roomId) {
    if (!$detailMobs) return;
    const info = roomInhabitants[roomId];
    if (!info || ((info.mobs || []).length === 0 && (info.items || []).length === 0)) {
      $detailMobs.innerHTML = '';
      return;
    }

    const parts = [];
    if (info.mobs && info.mobs.length > 0) {
      parts.push(`<h4>Mobs Here (${info.mobs.length})</h4>`);
      info.mobs.forEach(m => {
        const alignTag = m.alg >= 300 ? 'Good' : m.alg <= -300 ? 'Evil' : 'Neutral';
        const shopTag = m.shop ? ' · Shop' : '';
        parts.push(`
          <a class="inhabitant-item" href="/database/#mob-${m.v}">
            <span class="inhabitant-name">${escHtml(m.s)}</span>
            <span class="inhabitant-meta">Lvl ${m.lvl} · ${alignTag}${shopTag} · #${m.v}</span>
          </a>
        `);
      });
    }

    if (info.items && info.items.length > 0) {
      parts.push(`<h4 style="margin-top:0.5rem">Items Here (${info.items.length})</h4>`);
      info.items.forEach(it => {
        parts.push(`
          <a class="inhabitant-item" href="/database/#item-${it.v}">
            <span class="inhabitant-name">${escHtml(it.s)}</span>
            <span class="inhabitant-meta">${escHtml(it.type)} · #${it.v}</span>
          </a>
        `);
      });
    }

    $detailMobs.innerHTML = parts.join('');
  }

  function jumpToRoom(roomId) {
    const room = currentRooms.find(r => r.id === roomId);
    if (!room) return;
    const { w, h } = svgDims();
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

  // ── Room search (works both within a selected zone AND globally across all 9,590 rooms) ──
  function handleRoomSearch(q) {
    const trimmed = q.trim();
    if (!trimmed) {
      $roomResults.style.display = 'none';
      $roomResults.innerHTML = '';
      return;
    }
    const ql = trimmed.toLowerCase().replace(/^#/, '');
    let hits = [];

    if (isShowingZoneSvg() && currentRooms.length) {
      hits = currentRooms
        .filter(r => r.name.toLowerCase().includes(ql) || String(r.id).includes(ql))
        .slice(0, 20)
        .map(r => ({ id: r.id, name: r.name, zoneId: currentZoneId }));
    } else if (indexData?.room_names) {
      for (const [ridStr, rName] of Object.entries(indexData.room_names)) {
        if (rName.toLowerCase().includes(ql) || ridStr.includes(ql)) {
          const rid = parseInt(ridStr, 10);
          hits.push({
            id: rid,
            name: rName,
            zoneId: indexData.room_zones?.[ridStr],
          });
          if (hits.length >= 25) break;
        }
      }
    }

    if (!hits.length) {
      $roomResults.style.display = 'none';
      return;
    }
    $roomResults.style.display = 'block';
    $roomResults.innerHTML = hits.map(r => {
      const zLabel = r.zoneId !== undefined ? ` · ${getZoneName(r.zoneId)}` : '';
      return `
      <div class="room-result" data-id="${r.id}" data-zone="${r.zoneId ?? ''}" tabindex="0" role="button">
        <div>${escHtml(r.name)}</div>
        <div class="room-result-zone">#${r.id}${escHtml(zLabel)}</div>
      </div>`;
    }).join('');

    $roomResults.querySelectorAll('.room-result').forEach(el => {
      const action = () => {
        const rid = +el.dataset.id;
        const zid = el.dataset.zone ? +el.dataset.zone : indexData?.room_zones?.[String(rid)];
        if (isShowingZoneSvg() && zid === currentZoneId) {
          const room = currentRooms.find(r => r.id === rid);
          if (room) { selectRoom(room); jumpToRoom(room.id); }
        } else if (viewMode === 'globe' && globeRoomMap?.[rid]) {
          const gr = globeRoomMap[rid];
          focusGlobeOnZone(gr.zone);
          selectGlobeRoom(gr);
        } else if (viewMode === 'graph' && graphRoomMap?.[rid]) {
          const gr = graphRoomMap[rid];
          focusGraph2DOnZone(gr.zone);
          selectGlobeRoom(gr);
        } else if (zid !== undefined) {
          selectZone(zid).then(() => {
            const room = currentRooms.find(r => r.id === rid);
            if (room) { selectRoom(room); jumpToRoom(room.id); }
          });
        }
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
