/* Dark Pawns — Interactive World Map
   Loads /map/world.json, renders zone graphs with D3 force layout. */

(function () {
  'use strict';

  // ── State ────────────────────────────────────────────────────────────
  let world = null;
  const zoneIndex = {};   // id -> zone object
  const roomIndex = {};   // id -> {room, zoneId}

  let currentZoneId = null;
  let selectedRoomId = null;
  let pathStartId = null;
  let pathMode = false;
  let simulation = null;
  let currentNodes = [];
  let currentLinks = [];

  // ── Sector colors ────────────────────────────────────────────────────
  const SECTOR_COLOR = {
    0: '#718096',  // inside
    1: '#4299e1',  // city
    2: '#48bb78',  // field
    3: '#276749',  // forest
    4: '#9f7aea',  // hills
    5: '#a0aec0',  // mountain
    6: '#63b3ed',  // water_swim
    7: '#2b6cb0',  // water_noswim
    8: '#2c5282',  // underwater
    9: '#90cdf4',  // flying
    10: '#f6ad55', // desert
    11: '#38a169', // swamp
  };
  function getSectorColor(s) { return SECTOR_COLOR[s] || '#4299e1'; }

  const DIR_FULL = { n: 'north', e: 'east', s: 'south', w: 'west', u: 'up', d: 'down' };

  // ── DOM refs ─────────────────────────────────────────────────────────
  const $zoneSearch   = document.getElementById('zone-search');
  const $zoneList     = document.getElementById('zone-list');
  const $roomSearch   = document.getElementById('room-search');
  const $roomResults  = document.getElementById('room-search-results');
  const $zoneTitle    = document.getElementById('zone-title');
  const $loading      = document.getElementById('map-loading');
  const $empty        = document.getElementById('map-empty');
  const $legend       = document.getElementById('map-legend');
  const $tooltip      = document.getElementById('map-tooltip');
  const $detail       = document.getElementById('map-detail');
  const $detailName   = document.getElementById('detail-room-name');
  const $detailVnum   = document.getElementById('detail-vnum');
  const $detailDesc   = document.getElementById('detail-desc');
  const $detailExits  = document.getElementById('detail-exits');
  const $pathStatus   = document.getElementById('path-status');
  const $sidebar      = document.getElementById('map-sidebar');
  const $btnSidebar   = document.getElementById('btn-sidebar');
  const $btnSetStart  = document.getElementById('btn-set-start');
  const $btnClearPath = document.getElementById('btn-clear-path');
  const $btnPath      = document.getElementById('btn-path');
  const $svgWrap      = document.getElementById('map-svg-wrap');

  // ── D3 SVG setup ─────────────────────────────────────────────────────
  const svgEl = document.getElementById('map-svg');
  const svg = d3.select(svgEl);
  const gRoot = svg.append('g').attr('id', 'map-g');
  const gLinks = gRoot.append('g').attr('class', 'links-layer');
  const gNodes = gRoot.append('g').attr('class', 'nodes-layer');

  const zoom = d3.zoom()
    .scaleExtent([0.05, 8])
    .on('zoom', (event) => {
      gRoot.attr('transform', event.transform);
    });
  svg.call(zoom);

  // click on SVG background → deselect
  svg.on('click', (event) => {
    if (event.target === svgEl || event.target.id === 'map-g') {
      deselectRoom();
    }
  });

  function svgDims() {
    return { w: svgEl.clientWidth || 800, h: svgEl.clientHeight || 600 };
  }

  // ── Boot ─────────────────────────────────────────────────────────────
  document.addEventListener('DOMContentLoaded', () => {
    bindControls();
    loadWorld();
  });

  // ── Load world data ──────────────────────────────────────────────────
  function loadWorld() {
    fetch('/map/world.json')
      .then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json();
      })
      .then(data => {
        world = data;
        buildIndex();
        $loading.style.display = 'none';
        $empty.style.display = 'flex';
        renderZoneList('');
      })
      .catch(err => {
        $loading.textContent = `Failed to load world data: ${err.message}`;
      });
  }

  function buildIndex() {
    world.zones.forEach(zone => {
      zoneIndex[zone.id] = zone;
      zone.rooms.forEach(room => {
        roomIndex[room.id] = { room, zoneId: zone.id };
      });
    });
  }

  // ── Zone sidebar ─────────────────────────────────────────────────────
  function renderZoneList(filter) {
    const q = filter.toLowerCase().trim();
    const zones = world.zones.filter(z =>
      !q || z.name.toLowerCase().includes(q)
    );

    if (!zones.length) {
      $zoneList.innerHTML = '<div style="padding:0.5rem 0.75rem;font-size:0.65rem;color:#4a5568">No zones match</div>';
      return;
    }

    $zoneList.innerHTML = zones.map(z => `
      <div class="zone-item${z.id === currentZoneId ? ' active' : ''}" data-zone="${z.id}">
        <span class="zone-name">${escHtml(z.name)}</span>
        <span class="zone-count">${z.rooms.length}</span>
      </div>`).join('');

    $zoneList.querySelectorAll('.zone-item').forEach(el => {
      el.addEventListener('click', () => selectZone(+el.dataset.zone));
    });
  }

  function selectZone(zoneId) {
    if (!zoneIndex[zoneId]) return;
    currentZoneId = zoneId;

    $zoneTitle.textContent = zoneIndex[zoneId].name;
    $empty.style.display = 'none';
    $legend.classList.add('visible');

    // Highlight active zone in list
    $zoneList.querySelectorAll('.zone-item').forEach(el => {
      el.classList.toggle('active', +el.dataset.zone === zoneId);
    });

    deselectRoom();
    resetPath();
    renderZone(zoneIndex[zoneId]);

    // On mobile: close sidebar after selection
    if (window.innerWidth <= 768) {
      $sidebar.classList.remove('open');
    }
  }

  // ── Zone rendering ───────────────────────────────────────────────────
  function renderZone(zone) {
    if (simulation) { simulation.stop(); simulation = null; }

    gLinks.selectAll('*').remove();
    gNodes.selectAll('*').remove();

    const rooms = zone.rooms;
    const zoneRoomSet = new Set(rooms.map(r => r.id));
    const { w, h } = svgDims();

    // Build nodes with initial positions spread in a circle
    const n = rooms.length;
    const spread = Math.min(w, h) * 0.35 + Math.sqrt(n) * 10;
    currentNodes = rooms.map((r, i) => {
      const angle = (i / n) * 2 * Math.PI;
      return {
        id: r.id,
        room: r,
        x: w / 2 + Math.cos(angle) * spread * (0.5 + Math.random() * 0.5),
        y: h / 2 + Math.sin(angle) * spread * (0.5 + Math.random() * 0.5),
      };
    });

    // Build links (deduplicated, intra-zone only)
    const linkSet = new Set();
    currentLinks = [];
    rooms.forEach(room => {
      room.exits.forEach(exit => {
        if (zoneRoomSet.has(exit.t)) {
          const key = Math.min(room.id, exit.t) + '-' + Math.max(room.id, exit.t);
          if (!linkSet.has(key)) {
            linkSet.add(key);
            currentLinks.push({ source: room.id, target: exit.t });
          }
        }
      });
    });

    // SVG elements
    const linkSel = gLinks.selectAll('line')
      .data(currentLinks)
      .join('line')
      .attr('class', 'exit-line');

    const nodeSel = gNodes.selectAll('circle')
      .data(currentNodes, d => d.id)
      .join('circle')
      .attr('class', 'room-node')
      .attr('r', 6)
      .attr('fill', d => getSectorColor(d.room.sector))
      .attr('stroke', '#1a202c')
      .attr('stroke-width', 1.5)
      .on('mouseover', onNodeHover)
      .on('mousemove', onNodeMove)
      .on('mouseout', onNodeOut)
      .on('click', onNodeClick)
      .call(d3.drag()
        .on('start', dragStart)
        .on('drag', dragging)
        .on('end', dragEnd)
      );

    // Force simulation
    const chargeStrength = n > 200 ? -25 : n > 80 ? -40 : -60;
    const linkDist = n > 200 ? 30 : n > 80 ? 35 : 45;
    const alphaDecay = n > 400 ? 0.05 : 0.028;

    simulation = d3.forceSimulation(currentNodes)
      .force('link', d3.forceLink(currentLinks).id(d => d.id).distance(linkDist).strength(0.6))
      .force('charge', d3.forceManyBody().strength(chargeStrength))
      .force('center', d3.forceCenter(w / 2, h / 2).strength(0.05))
      .force('collision', d3.forceCollide(10))
      .alphaDecay(alphaDecay)
      .on('tick', () => {
        linkSel
          .attr('x1', d => d.source.x)
          .attr('y1', d => d.source.y)
          .attr('x2', d => d.target.x)
          .attr('y2', d => d.target.y);
        nodeSel
          .attr('cx', d => d.x)
          .attr('cy', d => d.y);
      });

    // Reset zoom to fit
    setTimeout(() => fitToView(), 100);
  }

  function fitToView() {
    const { w, h } = svgDims();
    svg.transition().duration(300).call(
      zoom.transform,
      d3.zoomIdentity.translate(w / 2, h / 2).scale(1).translate(-w / 2, -h / 2)
    );
  }

  // ── Drag handlers ─────────────────────────────────────────────────────
  let dragging_ = false;

  function dragStart(event, d) {
    dragging_ = false;
    if (!event.active) simulation && simulation.alphaTarget(0.3).restart();
    d.fx = d.x; d.fy = d.y;
  }

  function dragging(event, d) {
    dragging_ = true;
    d.fx = event.x; d.fy = event.y;
  }

  function dragEnd(event, d) {
    if (!event.active) simulation && simulation.alphaTarget(0);
    d.fx = null; d.fy = null;
  }

  // ── Node interaction ──────────────────────────────────────────────────
  function onNodeHover(event, d) {
    const tip = $tooltip;
    tip.style.display = 'block';
    const desc = d.room.desc ? d.room.desc.substring(0, 120) + (d.room.desc.length > 120 ? '…' : '') : '';
    tip.innerHTML = `<strong>#${d.id} ${escHtml(d.room.name)}</strong>${desc ? `<span style="color:#718096">${escHtml(desc)}</span>` : ''}`;
    moveTooltip(event);
  }

  function onNodeMove(event) { moveTooltip(event); }

  function onNodeOut() { $tooltip.style.display = 'none'; }

  function moveTooltip(event) {
    const x = event.clientX + 12, y = event.clientY - 8;
    const tip = $tooltip;
    const rect = tip.getBoundingClientRect();
    tip.style.left = (x + rect.width > window.innerWidth ? x - rect.width - 20 : x) + 'px';
    tip.style.top = (y + rect.height > window.innerHeight ? y - rect.height : y) + 'px';
  }

  function onNodeClick(event, d) {
    event.stopPropagation();
    if (dragging_) { dragging_ = false; return; }
    $tooltip.style.display = 'none';

    if (pathMode && pathStartId !== null && pathStartId !== d.id) {
      // Find path
      const path = findPath(pathStartId, d.id);
      if (path) {
        highlightPath(path);
        $pathStatus.textContent = `Path found: ${path.length} rooms`;
      } else {
        $pathStatus.textContent = 'No path found between these rooms.';
      }
      pathMode = false;
      $btnPath.classList.remove('active');
    }

    selectRoom(d.id);
  }

  // ── Room selection & detail panel ─────────────────────────────────────
  function selectRoom(roomId) {
    selectedRoomId = roomId;
    const entry = roomIndex[roomId];
    if (!entry) return;
    const { room } = entry;

    // Update visual state
    gNodes.selectAll('.room-node').classed('selected', d => d.id === roomId);

    // Populate detail panel
    $detailName.textContent = room.name;
    $detailVnum.textContent = `#${room.id}`;
    $detailDesc.textContent = room.desc || '';

    // Exits
    if (room.exits && room.exits.length > 0) {
      $detailExits.innerHTML = `<h4>Exits</h4>` +
        room.exits.map(exit => {
          const dest = roomIndex[exit.t];
          const destName = dest ? dest.room.name : `#${exit.t}`;
          const sameZone = dest && dest.zoneId === currentZoneId;
          return `<div class="exit-item" data-dest="${exit.t}" title="Go to #${exit.t}">
            <span class="exit-dir">${exit.d}</span>
            <span>${escHtml(destName)}</span>
            ${!sameZone ? `<span class="exit-dest">[other zone]</span>` : ''}
          </div>`;
        }).join('');

      $detailExits.querySelectorAll('.exit-item').forEach(el => {
        el.addEventListener('click', () => jumpToRoom(+el.dataset.dest));
      });
    } else {
      $detailExits.innerHTML = `<h4>Exits</h4><div style="font-size:0.65rem;color:#4a5568">Dead end</div>`;
    }

    // Path buttons
    if (pathStartId !== null) {
      $btnSetStart.textContent = 'Find path to here';
      $btnSetStart.classList.add('start');
    } else {
      $btnSetStart.textContent = 'Set as path start';
      $btnSetStart.classList.remove('start');
    }

    $detail.classList.remove('hidden');
    if (window.innerWidth <= 768) $detail.classList.add('open');
  }

  function deselectRoom() {
    selectedRoomId = null;
    gNodes.selectAll('.room-node').classed('selected', false);
    $detail.classList.add('hidden');
    $detail.classList.remove('open');
  }

  function jumpToRoom(roomId) {
    const entry = roomIndex[roomId];
    if (!entry) return;
    if (entry.zoneId !== currentZoneId) {
      selectZone(entry.zoneId);
      // Wait for render then highlight
      setTimeout(() => selectRoom(roomId), 800);
    } else {
      selectRoom(roomId);
      // Pan to room node
      const node = currentNodes.find(n => n.id === roomId);
      if (node) panToNode(node);
    }
  }

  function panToNode(node) {
    const { w, h } = svgDims();
    const t = d3.zoomTransform(svgEl);
    svg.transition().duration(400).call(
      zoom.translateTo,
      node.x,
      node.y
    );
  }

  // ── Path finding (BFS) ────────────────────────────────────────────────
  function findPath(fromId, toId) {
    const queue = [[fromId]];
    const visited = new Set([fromId]);
    let iterations = 0;
    const limit = 5000;

    while (queue.length && iterations++ < limit) {
      const path = queue.shift();
      const curr = path[path.length - 1];
      if (curr === toId) return path;

      const entry = roomIndex[curr];
      if (!entry) continue;

      for (const exit of entry.room.exits) {
        if (!visited.has(exit.t)) {
          visited.add(exit.t);
          queue.push([...path, exit.t]);
        }
      }
    }
    return null;
  }

  function highlightPath(path) {
    const pathSet = new Set(path);

    // Build set of path edges (min-max key)
    const pathEdges = new Set();
    for (let i = 0; i < path.length - 1; i++) {
      pathEdges.add(Math.min(path[i], path[i+1]) + '-' + Math.max(path[i], path[i+1]));
    }

    gNodes.selectAll('.room-node')
      .classed('path-node', d => pathSet.has(d.id) && d.id !== pathStartId)
      .classed('path-start', d => d.id === pathStartId);

    gLinks.selectAll('.exit-line')
      .classed('in-path', d => {
        const src = typeof d.source === 'object' ? d.source.id : d.source;
        const tgt = typeof d.target === 'object' ? d.target.id : d.target;
        return pathEdges.has(Math.min(src, tgt) + '-' + Math.max(src, tgt));
      });
  }

  function resetPath() {
    pathStartId = null;
    pathMode = false;
    $btnPath.classList.remove('active');
    $pathStatus.textContent = 'Click "Set Start" to begin path mode';
    $btnSetStart.textContent = 'Set as path start';
    $btnSetStart.classList.remove('start');
    $btnClearPath.style.display = 'none';
    gNodes.selectAll('.room-node').classed('path-node', false).classed('path-start', false);
    gLinks.selectAll('.exit-line').classed('in-path', false);
  }

  // ── Room search ───────────────────────────────────────────────────────
  function searchRooms(query) {
    const q = query.toLowerCase().trim();
    if (!q || !world) {
      $roomResults.style.display = 'none';
      $roomResults.innerHTML = '';
      return;
    }

    const results = [];
    for (const zone of world.zones) {
      for (const room of zone.rooms) {
        if (room.name.toLowerCase().includes(q)) {
          results.push({ room, zone });
          if (results.length >= 30) break;
        }
      }
      if (results.length >= 30) break;
    }

    if (!results.length) {
      $roomResults.innerHTML = '<div style="font-size:0.65rem;color:#4a5568;padding:0.25rem">No rooms found</div>';
      $roomResults.style.display = 'block';
      return;
    }

    $roomResults.innerHTML = results.map(({ room, zone }) => `
      <div class="room-result" data-room="${room.id}" data-zone="${zone.id}">
        <div>${escHtml(room.name)} <span style="color:#4a5568">#${room.id}</span></div>
        <div class="room-result-zone">${escHtml(zone.name)}</div>
      </div>`).join('');

    $roomResults.querySelectorAll('.room-result').forEach(el => {
      el.addEventListener('click', () => {
        $roomSearch.value = '';
        $roomResults.style.display = 'none';
        $roomResults.innerHTML = '';
        jumpToRoom(+el.dataset.room);
      });
    });

    $roomResults.style.display = 'block';
  }

  // ── Controls ──────────────────────────────────────────────────────────
  function bindControls() {
    // Zone search
    $zoneSearch.addEventListener('input', () => {
      if (world) renderZoneList($zoneSearch.value);
    });

    // Room search
    let searchTimer;
    $roomSearch.addEventListener('input', () => {
      clearTimeout(searchTimer);
      searchTimer = setTimeout(() => searchRooms($roomSearch.value), 150);
    });
    $roomSearch.addEventListener('blur', () => {
      setTimeout(() => {
        $roomResults.style.display = 'none';
      }, 200);
    });

    // Zoom buttons
    document.getElementById('btn-zoom-in').addEventListener('click', () => {
      svg.transition().duration(200).call(zoom.scaleBy, 1.5);
    });
    document.getElementById('btn-zoom-out').addEventListener('click', () => {
      svg.transition().duration(200).call(zoom.scaleBy, 1 / 1.5);
    });
    document.getElementById('btn-reset').addEventListener('click', () => {
      const { w, h } = svgDims();
      svg.transition().duration(300).call(
        zoom.transform,
        d3.zoomIdentity.translate(0, 0).scale(1)
      );
    });

    // Path mode
    $btnPath.addEventListener('click', () => {
      if (!selectedRoomId) {
        $pathStatus.textContent = 'Select a room first, then click Path.';
        return;
      }
      pathMode = !pathMode;
      $btnPath.classList.toggle('active', pathMode);
      if (pathMode) {
        pathStartId = selectedRoomId;
        $pathStatus.textContent = `Start: #${pathStartId}. Click another room.`;
        $btnSetStart.textContent = 'Find path to here';
        $btnSetStart.classList.add('start');
        $btnClearPath.style.display = 'block';
        gNodes.selectAll('.room-node').classed('path-start', d => d.id === pathStartId);
      } else {
        resetPath();
      }
    });

    // Detail panel: set path start / find path
    $btnSetStart.addEventListener('click', () => {
      if (!selectedRoomId) return;
      if (pathStartId !== null && pathStartId !== selectedRoomId) {
        const path = findPath(pathStartId, selectedRoomId);
        if (path) {
          highlightPath(path);
          $pathStatus.textContent = `Path: ${path.length} rooms`;
        } else {
          $pathStatus.textContent = 'No path found.';
        }
        $btnClearPath.style.display = 'block';
      } else {
        pathStartId = selectedRoomId;
        $pathStatus.textContent = `Start: #${pathStartId} — now select destination.`;
        $btnSetStart.textContent = 'Find path to here';
        $btnSetStart.classList.add('start');
        $btnClearPath.style.display = 'block';
        gNodes.selectAll('.room-node').classed('path-start', d => d.id === pathStartId);
      }
    });

    $btnClearPath.addEventListener('click', resetPath);

    // Detail close
    document.getElementById('detail-close').addEventListener('click', deselectRoom);

    // Mobile sidebar toggle
    $btnSidebar.addEventListener('click', () => {
      $sidebar.classList.toggle('open');
    });

    // Touch support: treat tap as click (D3 handles most of this already)
    svgEl.addEventListener('touchstart', (e) => {
      if (e.touches.length === 1) e.preventDefault();
    }, { passive: false });

    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
      if (e.target.tagName === 'INPUT') return;
      if (e.key === 'Escape') { deselectRoom(); resetPath(); }
      if (e.key === '+' || e.key === '=') svg.transition().duration(150).call(zoom.scaleBy, 1.3);
      if (e.key === '-') svg.transition().duration(150).call(zoom.scaleBy, 1 / 1.3);
    });
  }

  // ── Util ──────────────────────────────────────────────────────────────
  function escHtml(str) {
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

})();
