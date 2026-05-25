(function() {
  let dbData = null;
  let activeTab = 'mobs'; // 'mobs' or 'items'
  let activeId = null;

  // DOM Elements
  const loadingEl = document.getElementById('db-loading');
  const searchInput = document.getElementById('db-search');
  const resultsContainer = document.getElementById('db-results-container');
  const emptyState = document.getElementById('db-empty');
  const detailContainer = document.getElementById('db-detail-container');
  
  // Tab Elements
  const tabMobs = document.getElementById('tab-mobs');
  const tabItems = document.getElementById('tab-items');
  const filterMobsSection = document.getElementById('filter-mobs-section');
  const filterItemsSection = document.getElementById('filter-items-section');

  // Filter Elements
  const mobLevelSlider = document.getElementById('filter-mob-level');
  const mobLevelVal = document.getElementById('filter-mob-level-val');
  const mobAlignSelect = document.getElementById('filter-mob-align');
  const itemTypeSelect = document.getElementById('filter-item-type');
  const itemWearSelect = document.getElementById('filter-item-wear');

  // Dice roll helper to get average value (e.g. "3d5+10" -> 3*3 + 10 = 19)
  function calcDiceAvg(diceStr) {
    if (!diceStr) return 0;
    const match = diceStr.match(/^(\d+)d(\d+)\+(\d+)$/);
    if (!match) return 0;
    const num = parseInt(match[1]);
    const sides = parseInt(match[2]);
    const plus = parseInt(match[3]);
    const avgSides = (1 + sides) / 2;
    return Math.round(num * avgSides + plus);
  }

  // 1. Fetch MUD database
  async function loadDatabase() {
    try {
      const response = await fetch('/data/database.json');
      if (!response.ok) throw new Error('Network response was not ok');
      dbData = await response.json();
      
      // Hide loading
      if (loadingEl) loadingEl.style.display = 'none';
      
      // Initial render & check Hash
      initRouting();
      renderList();
    } catch (error) {
      console.error('Failed to load database.json:', error);
      if (loadingEl) {
        loadingEl.innerHTML = '<div style="color:var(--accent);font-weight:bold;">Failed to load MUD database. Please try reloading.</div>';
      }
    }
  }

  // 2. State & Hash routing (e.g., #mob-3001, #item-1205)
  function initRouting() {
    window.addEventListener('hashchange', handleHash);
    handleHash();
  }

  function handleHash() {
    const hash = window.location.hash;
    if (!hash) {
      clearDetail();
      return;
    }

    const match = hash.match(/^#(mob|item)-(\d+)$/);
    if (match) {
      const type = match[1] === 'mob' ? 'mobs' : 'items';
      const id = match[2];
      
      if (dbData && dbData[type] && dbData[type][id]) {
        switchTab(type);
        selectRecord(type, id);
      }
    }
  }

  function selectRecord(type, id) {
    activeId = id;
    
    // Highlight list item
    const items = resultsContainer.querySelectorAll('.result-item');
    items.forEach(el => {
      if (el.getAttribute('data-id') === id) {
        el.classList.add('active');
        el.scrollIntoView({ block: 'nearest' });
      } else {
        el.classList.remove('active');
      }
    });

    renderDetail(type, id);
  }

  function clearDetail() {
    activeId = null;
    if (emptyState) emptyState.style.display = 'flex';
    if (detailContainer) detailContainer.style.display = 'none';
    const items = resultsContainer.querySelectorAll('.result-item');
    items.forEach(el => el.classList.remove('active'));
  }

  // 3. Tab Management
  function switchTab(tab) {
    if (activeTab === tab) return;
    activeTab = tab;

    if (activeTab === 'mobs') {
      tabMobs.classList.add('active');
      tabItems.classList.remove('active');
      filterMobsSection.style.display = 'block';
      filterItemsSection.style.display = 'none';
      searchInput.placeholder = 'Search mobs by name or VNUM…';
    } else {
      tabMobs.classList.remove('active');
      tabItems.classList.add('active');
      filterMobsSection.style.display = 'none';
      filterItemsSection.style.display = 'block';
      searchInput.placeholder = 'Search items by name or VNUM…';
    }
    
    searchInput.value = '';
    renderList();
  }

  // Tab click listeners
  if (tabMobs && tabItems) {
    tabMobs.addEventListener('click', () => switchTab('mobs'));
    tabItems.addEventListener('click', () => switchTab('items'));
  }

  // 4. Render Sidebar List with Filters
  function renderList() {
    if (!dbData) return;
    
    const query = searchInput.value.toLowerCase().trim();
    const dataList = dbData[activeTab];
    const results = [];

    // Filter Logic
    for (const id in dataList) {
      const record = dataList[id];
      const name = record.s.toLowerCase();
      const keywords = record.k.toLowerCase();
      const vnumStr = String(record.v);

      // Search match
      const textMatch = !query || name.includes(query) || keywords.includes(query) || vnumStr.includes(query);
      if (!textMatch) continue;

      // Dynamic criteria matching
      if (activeTab === 'mobs') {
        const minLvl = parseInt(mobLevelSlider.value);
        if (record.lvl < minLvl) continue;

        const alignType = mobAlignSelect.value;
        if (alignType === 'good' && record.alg <= 300) continue;
        if (alignType === 'neutral' && (record.alg > 300 || record.alg < -300)) continue;
        if (alignType === 'evil' && record.alg >= -300) continue;
      } else {
        const itemType = itemTypeSelect.value;
        if (itemType !== 'all' && record.type !== itemType) continue;

        const wearPos = itemWearSelect.value;
        if (wearPos !== 'all' && !record.wear.includes(wearPos)) continue;
      }

      results.push(record);
    }

    // Sort by VNUM ascending
    results.sort((a, b) => a.v - b.v);

    // Build DOM
    resultsContainer.innerHTML = '';
    
    if (results.length === 0) {
      resultsContainer.innerHTML = '<div style="padding: 1.5rem; text-align: center; color: var(--ink-muted); font-size: 0.8rem;">No records found.</div>';
      return;
    }

    // Cap displayed elements at 150 to keep DOM super responsive
    const displayResults = results.slice(0, 150);

    displayResults.forEach(r => {
      const item = document.createElement('div');
      item.className = 'result-item';
      if (String(r.v) === activeId) {
        item.classList.add('active');
      }
      item.setAttribute('data-id', r.v);
      
      let metaLeft = '';
      let metaRight = '';
      
      if (activeTab === 'mobs') {
        metaLeft = `Lvl ${r.lvl} ${r.sex === 1 ? 'Male' : r.sex === 2 ? 'Female' : 'Neutral'}`;
        metaRight = `Align: ${r.alg}`;
      } else {
        metaLeft = r.type;
        metaRight = `${r.wt} lbs | ${r.cst} gp`;
      }

      item.innerHTML = `
        <span class="result-name">${r.s}</span>
        <div class="result-meta">
          <span>${metaLeft}</span>
          <span class="result-vnum">#${r.v}</span>
        </div>
      `;

      item.addEventListener('click', () => {
        const hashType = activeTab === 'mobs' ? 'mob' : 'item';
        window.location.hash = `#${hashType}-${r.v}`;
      });

      resultsContainer.appendChild(item);
    });

    if (results.length > 150) {
      const more = document.createElement('div');
      more.style.padding = '0.5rem';
      more.style.textAlign = 'center';
      more.style.fontSize = '0.65rem';
      more.style.color = 'var(--ink-muted)';
      more.style.borderTop = '1px solid rgba(26, 22, 20, 0.05)';
      more.innerText = `Showing 150 of ${results.length} records. Refine search to see more.`;
      resultsContainer.appendChild(more);
    }
  }

  // Filter listener events
  if (searchInput) searchInput.addEventListener('input', renderList);
  if (mobLevelSlider) {
    mobLevelSlider.addEventListener('input', () => {
      mobLevelVal.innerText = mobLevelSlider.value;
      renderList();
    });
  }
  if (mobAlignSelect) mobAlignSelect.addEventListener('change', renderList);
  if (itemTypeSelect) itemTypeSelect.addEventListener('change', renderList);
  if (itemWearSelect) itemWearSelect.addEventListener('change', renderList);

  // 5. Render Detail Card
  function renderDetail(type, id) {
    if (!dbData || !detailContainer || !emptyState) return;

    const record = dbData[type][id];
    if (!record) return;

    emptyState.style.display = 'none';
    detailContainer.style.display = 'block';

    if (type === 'mobs') {
      renderMobDetail(record);
    } else {
      renderItemDetail(record);
    }
  }

  function renderMobDetail(m) {
    let sexStr = 'Neutral';
    if (m.sex === 1) sexStr = 'Male';
    else if (m.sex === 2) sexStr = 'Female';

    // Spawn Locations List
    let spawnsHtml = '<li>This mobile does not spawn naturally in the world.</li>';
    if (m.spw && m.spw.length > 0) {
      spawnsHtml = m.spw.map(s => `
        <li class="rel-item">
          <a class="rel-link" href="/map?room=${s.room}">Room ${s.room}: ${s.name}</a>
          <span class="rel-meta">Zone ${s.zone}</span>
        </li>
      `).join('');
    }

    // Equipment & Inventory drops list
    let dropsHtml = '<li>This mobile carries no items.</li>';
    if (m.drp && m.drp.length > 0) {
      dropsHtml = m.drp.map(d => `
        <li class="rel-item">
          <a class="rel-link" href="#item-${d.obj_vnum}">${d.name}</a>
          <span class="rel-meta">${d.slot}</span>
        </li>
      `).join('');
    }

    let shopHtml = '';
    if (m.shop) {
      const shopItems = m.shop.items_sold.map(itm => `
        <li class="rel-item">
          <a class="rel-link" href="#item-${itm.vnum}">${itm.name}</a>
          <span class="rel-meta">#${itm.vnum}</span>
        </li>
      `).join('');
      shopHtml = `
        <div class="detail-section">
          <h3>Shopkeeper Inventory</h3>
          <p style="font-size:0.75rem;margin:0 0 0.5rem;color:var(--ink-muted);">Runs shop #${m.shop.shop_vnum} open ${m.shop.open_hours}. Markup: x${m.shop.sell_mult}</p>
          <ul class="rel-list">${shopItems}</ul>
        </div>
      `;
    }

    detailContainer.innerHTML = `
      <article class="detail-card">
        <header class="detail-header">
          <h1 class="detail-short">${m.s}</h1>
          <p class="detail-long">${m.l}</p>
        </header>

        <div class="detail-grid">
          <!-- Left Column: Core Stats -->
          <div class="detail-section">
            <h3>Character Stats</h3>
            <table class="stats-table">
              <tr><td class="label">VNUM</td><td class="val">#${m.v}</td></tr>
              <tr><td class="label">Level</td><td class="val">${m.lvl}</td></tr>
              <tr><td class="label">Race</td><td class="val">${m.rc}</td></tr>
              <tr><td class="label">Sex</td><td class="val">${sexStr}</td></tr>
              <tr><td class="label">Alignment</td><td class="val">${m.alg}</td></tr>
              <tr><td class="label">Base AC</td><td class="val">${m.ac}</td></tr>
              <tr><td class="label">Gold</td><td class="val">${m.gld}</td></tr>
              <tr><td class="label">Experience</td><td class="val">${m.exp}</td></tr>
            </table>
          </div>

          <!-- Middle Column: Combat & Dice visualizer -->
          <div class="detail-section">
            <h3>Power Index Visuals</h3>
            <div class="stat-vis-wrap" id="mob-visualizer">
              <div style="width:100%;">
                <div class="vis-bar-row">
                  <div class="vis-bar-label"><span>Base Level</span><span>${m.lvl}/40</span></div>
                  <div class="vis-bar-bg"><div class="vis-bar-fill" style="width:${Math.min(100, (m.lvl / 40) * 100)}%;"></div></div>
                </div>
                <div class="vis-bar-row">
                  <div class="vis-bar-label"><span>Avg Hit Points</span><span>${calcDiceAvg(m.hp)} hp</span></div>
                  <div class="vis-bar-bg"><div class="vis-bar-fill" style="width:${Math.min(100, (calcDiceAvg(m.hp) / 1000) * 100)}%;"></div></div>
                </div>
                <div class="vis-bar-row">
                  <div class="vis-bar-label"><span>Armor Rating (AC)</span><span>${m.ac} ac</span></div>
                  <div class="vis-bar-bg"><div class="vis-bar-fill" style="width:${Math.min(100, (Math.max(0, 1000 - m.ac) / 1000) * 100)}%;"></div></div>
                </div>
                <div class="vis-bar-row">
                  <div class="vis-bar-label"><span>Avg Attack Power</span><span>${calcDiceAvg(m.dmg)} dmg</span></div>
                  <div class="vis-bar-bg"><div class="vis-bar-fill" style="width:${Math.min(100, (calcDiceAvg(m.dmg) / 50) * 100)}%;"></div></div>
                </div>
              </div>
            </div>
          </div>
        </div>

        ${m.d ? `
        <div class="detail-section">
          <h3>Detailed Observation</h3>
          <div class="detail-desc-block">${m.d}</div>
        </div>
        ` : ''}

        <div class="detail-grid" style="margin-top:0.5rem;">
          <div class="detail-section">
            <h3>Spawn Points</h3>
            <ul class="rel-list">${spawnsHtml}</ul>
          </div>
          <div class="detail-section">
            <h3>Equipment & Drops</h3>
            <ul class="rel-list">${dropsHtml}</ul>
          </div>
        </div>
        
        ${shopHtml}
      </article>
    `;
  }

  function renderItemDetail(o) {
    // Wear Slots Text
    const wearPosStr = o.wear.length > 0 ? o.wear.join(', ') : 'None';
    const extraFlagsStr = o.extra.length > 0 ? o.extra.join(', ') : 'None';

    // Stats / Affects list
    let affectsHtml = '<p style="font-size:0.75rem;color:var(--ink-muted);margin:0;">No magical affects on this item.</p>';
    if (o.aff && o.aff.length > 0) {
      affectsHtml = o.aff.map(a => `
        <div style="font-size:0.75rem;padding:0.35rem 0.5rem;background:var(--paper-deep);border-radius:2px;display:flex;justify-content:space-between;margin-bottom:0.25rem;">
          <span style="font-weight:700;color:var(--accent);">${a.location}</span>
          <span style="font-family:var(--font-mono);font-weight:700;">${a.modifier > 0 ? '+' : ''}${a.modifier}</span>
        </div>
      `).join('');
    }

    // Drop list (mobs carrying it)
    let dropsHtml = '<li>No mobs load naturally with this item.</li>';
    if (o.mobs && o.mobs.length > 0) {
      dropsHtml = o.mobs.map(l => `
        <li class="rel-item">
          <a class="rel-link" href="#mob-${l.mob_vnum}">${l.name}</a>
          <span class="rel-meta">${l.slot}</span>
        </li>
      `).join('');
    }

    // Spawn in rooms / containers
    let roomsHtml = '';
    if (o.rms && o.rms.length > 0) {
      const roomItems = o.rms.map(p => {
        if (p.container_vnum) {
          return `
            <li class="rel-item">
              <a class="rel-link" href="#item-${p.container_vnum}">Inside ${p.name}</a>
              <span class="rel-meta">#${p.container_vnum}</span>
            </li>
          `;
        } else {
          return `
            <li class="rel-item">
              <a class="rel-link" href="/map?room=${p.room}">Room ${p.room}: ${p.name}</a>
              <span class="rel-meta">Zone ${p.zone}</span>
            </li>
          `;
        }
      }).join('');
      roomsHtml = `
        <div class="detail-section">
          <h3>Placed In Rooms / Containers</h3>
          <ul class="rel-list">${roomItems}</ul>
        </div>
      `;
    }

    // Shopkeepers selling
    let shopsHtml = '';
    if (o.shp && o.shp.length > 0) {
      const shopItems = o.shp.map(s => `
        <li class="rel-item">
          <a class="rel-link" href="#mob-${s.keeper_vnum}">${s.keeper_name}</a>
          <span class="rel-meta" style="color:var(--accent);font-weight:700;">${s.price} gp</span>
        </li>
      `).join('');
      shopsHtml = `
        <div class="detail-section">
          <h3>Sold by Merchants</h3>
          <ul class="rel-list">${shopItems}</ul>
        </div>
      `;
    }

    // Extra descriptions list
    let edescHtml = '';
    if (o.edesc && o.edesc.length > 0) {
      const edescItems = o.edesc.map(ed => `
        <div style="margin-bottom:0.75rem;">
          <h4 style="font-family:var(--font-display);font-size:0.7rem;text-transform:uppercase;color:var(--ink-muted);margin:0 0 0.25rem 0;border-bottom:1px solid rgba(26,22,20,0.06);padding-bottom:0.1rem;">Keywords: ${ed.keywords}</h4>
          <p style="font-size:0.8rem;line-height:1.5;margin:0;white-space:pre-wrap;">${ed.desc}</p>
        </div>
      `).join('');
      edescHtml = `
        <div class="detail-section">
          <h3>Extra Details</h3>
          <div class="detail-desc-block" style="font-family:var(--font-body);">${edescItems}</div>
        </div>
      `;
    }

    detailContainer.innerHTML = `
      <article class="detail-card">
        <header class="detail-header">
          <h1 class="detail-short">${o.s}</h1>
          <p class="detail-long">${o.l}</p>
        </header>

        <div class="detail-grid">
          <!-- Left Column: Core Stats -->
          <div class="detail-section">
            <h3>Object Specs</h3>
            <table class="stats-table">
              <tr><td class="label">VNUM</td><td class="val">#${o.v}</td></tr>
              <tr><td class="label">Item Type</td><td class="val">${o.type}</td></tr>
              <tr><td class="label">Wear Slot</td><td class="val" style="font-size:0.65rem;">${wearPosStr}</td></tr>
              <tr><td class="label">Weight</td><td class="val">${o.wt} lbs</td></tr>
              <tr><td class="label">Standard Cost</td><td class="val">${o.cst} gp</td></tr>
              <tr><td class="label">Load Probability</td><td class="val">${o.load}%</td></tr>
              <tr><td class="label">Active Flags</td><td class="val" style="font-size:0.65rem;">${extraFlagsStr}</td></tr>
            </table>
          </div>

          <!-- Middle Column: Affects & Values -->
          <div class="detail-section">
            <h3>Magical Affects</h3>
            ${affectsHtml}
            
            ${o.script ? `
            <div style="margin-top:0.5rem;padding:0.4rem 0.5rem;border:1px solid var(--accent);border-radius:2px;font-size:0.7rem;background:rgba(168, 32, 26, 0.03);">
              <strong style="color:var(--accent);font-family:var(--font-display);text-transform:uppercase;">Special Script</strong><br>
              <code style="font-family:var(--font-mono);">${o.script}</code>
            </div>
            ` : ''}
          </div>
        </div>

        ${edescHtml}

        <div class="detail-grid" style="margin-top:0.5rem;">
          <div class="detail-section">
            <h3>Spawned / Carried By Mobs</h3>
            <ul class="rel-list">${dropsHtml}</ul>
          </div>
          ${roomsHtml || shopsHtml ? `
          <div class="detail-section" style="display:flex;flex-direction:column;gap:1.5rem;">
            ${roomsHtml}
            ${shopsHtml}
          </div>
          ` : ''}
        </div>
      </article>
    `;
  }

  // Init loading MUD Database
  document.addEventListener('DOMContentLoaded', loadDatabase);

})();
