(function() {
  let dbData = null;
  let activeTab = 'mobs'; // 'mobs' or 'items'
  let activeId = null;

  // Virtualization / Progressive Scroll State
  let filteredRecords = [];
  let renderedCount = 0;
  const BATCH_SIZE = 50;

  // Active Filter state
  let filters = {
    mobs: {
      search: '',
      levelMin: 0,
      align: 'all',
      zones: new Set(),
      shopkeeper: false,
      aggressive: false,
      unique: false,
      sort: 'zone'
    },
    items: {
      search: '',
      type: 'all',
      wear: 'all',
      affect: 'all',
      zones: new Set(),
      magic: false,
      script: false,
      sort: 'zone'
    }
  };

  // DOM Elements
  const loadingEl = document.getElementById('db-loading');
  const searchInput = document.getElementById('db-search');
  const resultsContainer = document.getElementById('db-results-container');
  const resultsInner = document.getElementById('db-results-inner');
  const emptyState = document.getElementById('db-empty');
  const detailContainer = document.getElementById('db-detail-container');
  
  // Skyline Overview Strip
  const skylineContainer = document.getElementById('db-skyline');

  // Tab Elements
  const tabMobs = document.getElementById('tab-mobs');
  const tabItems = document.getElementById('tab-items');
  const filterMobsSection = document.getElementById('filter-mobs-section');
  const filterItemsSection = document.getElementById('filter-items-section');

  // Mobile navigation
  const dbMain = document.getElementById('db-main');
  const dbMobileClose = document.getElementById('db-mobile-close');
  const mobileFiltersTrigger = document.getElementById('mobile-filters-trigger');
  const dbFiltersDrawer = document.getElementById('db-filters');
  const filterSheetCloseBtn = document.getElementById('filter-sheet-close-btn');
  const activeChipsBar = document.getElementById('active-chips-bar');
  const sortSelect = document.getElementById('db-sort');

  // Filter Elements
  const mobLevelSlider = document.getElementById('filter-mob-level');
  const mobLevelVal = document.getElementById('filter-mob-level-val');
  const mobAlignSelect = document.getElementById('filter-mob-align');
  const mobZoneContainer = document.getElementById('filter-mob-zone');
  
  const chkMobShop = document.getElementById('chk-mob-shop');
  const chkMobAgg = document.getElementById('chk-mob-agg');
  const chkMobUniq = document.getElementById('chk-mob-uniq');

  const itemTypeSelect = document.getElementById('filter-item-type');
  const itemWearSelect = document.getElementById('filter-item-wear');
  const itemAffectSelect = document.getElementById('filter-item-affect');
  const itemZoneContainer = document.getElementById('filter-item-zone');

  const chkItmMag = document.getElementById('chk-itm-mag');
  const chkItmScr = document.getElementById('chk-itm-scr');

  // Scroll Shadows
  const shadowTop = document.getElementById('shadow-top');
  const shadowBottom = document.getElementById('shadow-bottom');

  // Helper: Dice Average
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

  // Load Database
  let zoneNames = {}; // zone id → display name (from the world map index)
  async function loadDatabase() {
    try {
      const [dbResp, mapResp] = await Promise.all([
        fetch('/data/database.json'),
        fetch('/map/map-index.json')
      ]);
      if (!dbResp.ok) throw new Error('Network response not ok');
      dbData = await dbResp.json();
      if (mapResp.ok) {
        const mapIndex = await mapResp.json();
        for (const z of mapIndex.zones || []) zoneNames[z.id] = z.name;
      }

      // Setup counts
      const mobCount = document.getElementById('mob-count');
      const itemCount = document.getElementById('item-count');
      if (mobCount) mobCount.textContent = dbData.mobs ? Object.keys(dbData.mobs).length.toLocaleString() : '';
      if (itemCount) itemCount.textContent = dbData.items ? Object.keys(dbData.items).length.toLocaleString() : '';

      // Initialize filter UI (zones list)
      populateZoneFilters();

      // Hide loading
      if (loadingEl) loadingEl.style.display = 'none';

      // Setup Event Listeners
      setupFilterListeners();
      setupScrollShadows();
      setupVirtualScroll();

      // Initial render, routing, skyline
      initRouting();
      renderSkyline();
      applyFiltersAndRender();
    } catch (error) {
      console.error('Failed to load database.json:', error);
      if (loadingEl) {
        loadingEl.innerHTML = '<div style="color:var(--accent);font-weight:bold;padding:2rem;text-align:center;">Failed to load MUD database. Please try reloading.</div>';
      }
    }
  }

  // Populate dynamic multi-select Zone lists
  function populateZoneFilters() {
    // Mobs Zones
    const mobZones = new Set();
    for (const id in dbData.mobs) {
      const m = dbData.mobs[id];
      if (m.spw && m.spw.length > 0) {
        mobZones.add(m.spw[0].zone);
      }
    }
    const sortedMobZones = Array.from(mobZones).sort((a,b) => a - b);
    mobZoneContainer.innerHTML = sortedMobZones.map(zoneId => `
      <label class="filter-checkbox-row" data-zone="${zoneId}">
        <input type="checkbox" class="chk-mob-zone-item" value="${zoneId}">
        ${getZoneName(zoneId)}
      </label>
    `).join('');

    // Items Zones
    const itemZones = new Set();
    for (const id in dbData.items) {
      const o = dbData.items[id];
      if (o.rms && o.rms.length > 0) {
        const primarySpawn = o.rms[0];
        if (primarySpawn.zone) {
          itemZones.add(primarySpawn.zone);
        }
      }
    }
    const sortedItemZones = Array.from(itemZones).sort((a,b) => a - b);
    itemZoneContainer.innerHTML = sortedItemZones.map(zoneId => `
      <label class="filter-checkbox-row" data-zone="${zoneId}">
        <input type="checkbox" class="chk-itm-zone-item" value="${zoneId}">
        ${getZoneName(zoneId)}
      </label>
    `).join('');
  }

  // Event Listeners for Filters
  function setupFilterListeners() {
    // Search input
    searchInput.addEventListener('input', () => {
      filters[activeTab].search = searchInput.value;
      applyFiltersAndRender();
    });

    // Sort select
    sortSelect.addEventListener('change', () => {
      filters[activeTab].sort = sortSelect.value;
      applyFiltersAndRender();
    });

    // Mob Level slider
    mobLevelSlider.addEventListener('input', () => {
      mobLevelVal.innerText = mobLevelSlider.value;
      filters.mobs.levelMin = parseInt(mobLevelSlider.value);
      applyFiltersAndRender();
    });

    // Mob Alignment
    mobAlignSelect.addEventListener('change', () => {
      filters.mobs.align = mobAlignSelect.value;
      applyFiltersAndRender();
    });

    // Mob Zone checkboxes
    mobZoneContainer.addEventListener('change', () => {
      const checked = mobZoneContainer.querySelectorAll('.chk-mob-zone-item:checked');
      filters.mobs.zones = new Set(Array.from(checked).map(c => parseInt(c.value)));
      applyFiltersAndRender();
    });

    // Mob Flags
    chkMobShop.addEventListener('change', () => {
      filters.mobs.shopkeeper = chkMobShop.checked;
      toggleFlagClass('switch-mob-shop', chkMobShop.checked);
      applyFiltersAndRender();
    });
    chkMobAgg.addEventListener('change', () => {
      filters.mobs.aggressive = chkMobAgg.checked;
      toggleFlagClass('switch-mob-agg', chkMobAgg.checked);
      applyFiltersAndRender();
    });
    chkMobUniq.addEventListener('change', () => {
      filters.mobs.unique = chkMobUniq.checked;
      toggleFlagClass('switch-mob-uniq', chkMobUniq.checked);
      applyFiltersAndRender();
    });

    // Item Type
    itemTypeSelect.addEventListener('change', () => {
      filters.items.type = itemTypeSelect.value;
      applyFiltersAndRender();
    });

    // Item Wear
    itemWearSelect.addEventListener('change', () => {
      filters.items.wear = itemWearSelect.value;
      applyFiltersAndRender();
    });

    // Item Affect
    itemAffectSelect.addEventListener('change', () => {
      filters.items.affect = itemAffectSelect.value;
      applyFiltersAndRender();
    });

    // Item Zone checkboxes
    itemZoneContainer.addEventListener('change', () => {
      const checked = itemZoneContainer.querySelectorAll('.chk-itm-zone-item:checked');
      filters.items.zones = new Set(Array.from(checked).map(c => parseInt(c.value)));
      applyFiltersAndRender();
    });

    // Item Flags
    chkItmMag.addEventListener('change', () => {
      filters.items.magic = chkItmMag.checked;
      toggleFlagClass('switch-itm-mag', chkItmMag.checked);
      applyFiltersAndRender();
    });
    chkItmScr.addEventListener('change', () => {
      filters.items.script = chkItmScr.checked;
      toggleFlagClass('switch-itm-scr', chkItmScr.checked);
      applyFiltersAndRender();
    });

    // Tab buttons
    tabMobs.addEventListener('click', () => switchTab('mobs'));
    tabItems.addEventListener('click', () => switchTab('items'));

    // Mobile specific controls
    mobileFiltersTrigger.addEventListener('click', () => {
      dbFiltersDrawer.classList.add('open');
    });
    filterSheetCloseBtn.addEventListener('click', () => {
      dbFiltersDrawer.classList.remove('open');
    });

    dbMobileClose.addEventListener('click', () => {
      dbMain.classList.remove('open');
      window.location.hash = ''; // clear hash routing
    });
  }

  function toggleFlagClass(id, checked) {
    const el = document.getElementById(id);
    if (el) {
      if (checked) el.classList.add('active');
      else el.classList.remove('active');
    }
  }

  // Active filter chip management
  function renderActiveChips() {
    activeChipsBar.innerHTML = '';
    let chips = [];
    const active = filters[activeTab];

    if (activeTab === 'mobs') {
      if (active.levelMin > 0) {
        chips.push({ label: `Lvl ≥ ${active.levelMin}`, clear: () => {
          mobLevelSlider.value = 0;
          mobLevelVal.innerText = 0;
          active.levelMin = 0;
        }});
      }
      if (active.align !== 'all') {
        chips.push({ label: `Align: ${active.align}`, clear: () => {
          mobAlignSelect.value = 'all';
          active.align = 'all';
        }});
      }
      active.zones.forEach(z => {
        chips.push({ label: getZoneName(z), clear: () => {
          const chk = mobZoneContainer.querySelector(`.chk-mob-zone-item[value="${z}"]`);
          if (chk) chk.checked = false;
          active.zones.delete(z);
        }});
      });
      if (active.shopkeeper) {
        chips.push({ label: 'Shopkeeper', clear: () => {
          chkMobShop.checked = false;
          toggleFlagClass('switch-mob-shop', false);
          active.shopkeeper = false;
        }});
      }
      if (active.aggressive) {
        chips.push({ label: 'Aggressive', clear: () => {
          chkMobAgg.checked = false;
          toggleFlagClass('switch-mob-agg', false);
          active.aggressive = false;
        }});
      }
      if (active.unique) {
        chips.push({ label: 'Unique', clear: () => {
          chkMobUniq.checked = false;
          toggleFlagClass('switch-mob-uniq', false);
          active.unique = false;
        }});
      }
    } else {
      if (active.type !== 'all') {
        chips.push({ label: `Type: ${active.type}`, clear: () => {
          itemTypeSelect.value = 'all';
          active.type = 'all';
        }});
      }
      if (active.wear !== 'all') {
        chips.push({ label: `Wear: ${active.wear}`, clear: () => {
          itemWearSelect.value = 'all';
          active.wear = 'all';
        }});
      }
      if (active.affect !== 'all') {
        chips.push({ label: `Affect: ${active.affect}`, clear: () => {
          itemAffectSelect.value = 'all';
          active.affect = 'all';
        }});
      }
      active.zones.forEach(z => {
        chips.push({ label: getZoneName(z), clear: () => {
          const chk = itemZoneContainer.querySelector(`.chk-itm-zone-item[value="${z}"]`);
          if (chk) chk.checked = false;
          active.zones.delete(z);
        }});
      });
      if (active.magic) {
        chips.push({ label: 'Magic', clear: () => {
          chkItmMag.checked = false;
          toggleFlagClass('switch-itm-mag', false);
          active.magic = false;
        }});
      }
      if (active.script) {
        chips.push({ label: 'Scripted', clear: () => {
          chkItmScr.checked = false;
          toggleFlagClass('switch-itm-scr', false);
          active.script = false;
        }});
      }
    }

    if (chips.length > 0) {
      activeChipsBar.style.display = 'flex';
      // Mobile indicator counts
      const countLabel = document.getElementById('active-filters-count');
      if (countLabel) {
        countLabel.innerText = chips.length;
        countLabel.style.display = 'inline-block';
      }

      chips.forEach(c => {
        const item = document.createElement('div');
        item.className = 'filter-chip';
        item.innerHTML = `${c.label} <span class="close-x">✕</span>`;
        item.addEventListener('click', () => {
          c.clear();
          applyFiltersAndRender();
        });
        activeChipsBar.appendChild(item);
      });
    } else {
      activeChipsBar.style.display = 'none';
      const countLabel = document.getElementById('active-filters-count');
      if (countLabel) countLabel.style.display = 'none';
    }
  }

  // Scroll shadows
  function setupScrollShadows() {
    resultsContainer.addEventListener('scroll', updateScrollShadows);
    window.addEventListener('resize', updateScrollShadows);
  }

  function updateScrollShadows() {
    const el = resultsContainer;
    if (el.scrollHeight > el.clientHeight) {
      if (el.scrollTop > 5) shadowTop.style.display = 'block';
      else shadowTop.style.display = 'none';

      if (el.scrollTop + el.clientHeight < el.scrollHeight - 5) shadowBottom.style.display = 'block';
      else shadowBottom.style.display = 'none';
    } else {
      shadowTop.style.display = 'none';
      shadowBottom.style.display = 'none';
    }
  }

  // State & Hash Routing
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

    // List item highlight
    const items = resultsInner.querySelectorAll('.result-item');
    items.forEach(el => {
      if (el.getAttribute('data-id') === id) {
        el.classList.add('active');
        el.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
      } else {
        el.classList.remove('active');
      }
    });

    renderDetail(type, id);

    // Slide up bottom-sheet on Mobile
    if (window.innerWidth <= 900) {
      dbMain.classList.add('open');
    }
  }

  function clearDetail() {
    activeId = null;
    if (emptyState) emptyState.style.display = 'flex';
    if (detailContainer) detailContainer.style.display = 'none';
    const items = resultsInner.querySelectorAll('.result-item');
    items.forEach(el => el.classList.remove('active'));
    dbMain.classList.remove('open');
  }

  // Tab Management
  function switchTab(tab) {
    if (activeTab === tab) return;
    activeTab = tab;

    // Reset sort select
    sortSelect.value = filters[activeTab].sort;

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

    searchInput.value = filters[activeTab].search;
    renderSkyline();
    applyFiltersAndRender();
  }

  // Progressive infinite scroll loader (Virtualization)
  function setupVirtualScroll() {
    resultsContainer.addEventListener('scroll', () => {
      if (renderedCount >= filteredRecords.length) return;
      const el = resultsContainer;
      // If we scroll within 60px of the bottom, load the next block
      if (el.scrollTop + el.clientHeight >= el.scrollHeight - 60) {
        appendNextBatch();
      }
    });
  }

  // Core Filtering and Sorting calculations
  function applyFiltersAndRender() {
    if (!dbData) return;
    renderActiveChips();

    const query = filters[activeTab].search.toLowerCase().trim();
    const dataList = dbData[activeTab];
    filteredRecords = [];

    for (const id in dataList) {
      const record = dataList[id];
      const name = record.s.toLowerCase();
      const keywords = record.k.toLowerCase();
      const vnumStr = String(record.v);

      // Search match
      const textMatch = !query || name.includes(query) || keywords.includes(query) || vnumStr.includes(query);
      if (!textMatch) continue;

      if (activeTab === 'mobs') {
        // Level Min
        if (record.lvl < filters.mobs.levelMin) continue;

        // Alignment
        const alignType = filters.mobs.align;
        if (alignType === 'good' && record.alg <= 300) continue;
        if (alignType === 'neutral' && (record.alg > 300 || record.alg < -300)) continue;
        if (alignType === 'evil' && record.alg >= -300) continue;

        // Spawns in Zone (Multi-select)
        if (filters.mobs.zones.size > 0) {
          const spawnZones = record.spw ? record.spw.map(s => s.zone) : [];
          const matchesZone = spawnZones.some(z => filters.mobs.zones.has(z));
          if (!matchesZone) continue;
        }

        // Shopkeeper Flag
        if (filters.mobs.shopkeeper && !record.shop) continue;

        // Aggressive Flag (alignment <= -700, or keywords/action flags contain aggressive)
        const isAgg = record.alg <= -700 || record.k.includes('aggressive') || record.k.includes('aggro');
        if (filters.mobs.aggressive && !isAgg) continue;

        // Unique Mob Flag (derived from spawns count == 1)
        const isUniq = record.spw && record.spw.length === 1;
        if (filters.mobs.unique && !isUniq) continue;

      } else {
        // Item Type
        const itemType = filters.items.type;
        if (itemType !== 'all' && record.type !== itemType) continue;

        // Wear position
        const wearPos = filters.items.wear;
        if (wearPos !== 'all' && !record.wear.includes(wearPos)) continue;

        // Magical affects locations
        if (filters.items.affect !== 'all') {
          const hasAffect = record.aff && record.aff.some(a => a.location === filters.items.affect);
          if (!hasAffect) continue;
        }

        // Spawns in Zone (Multi-select)
        if (filters.items.zones.size > 0) {
          const spawnZones = record.rms ? record.rms.map(r => r.zone).filter(Boolean) : [];
          const matchesZone = spawnZones.some(z => filters.items.zones.has(z));
          if (!matchesZone) continue;
        }

        // Magic Quality Flag
        if (filters.items.magic && (!record.aff || record.aff.length === 0)) continue;

        // Scripted Flag
        if (filters.items.script && !record.script) continue;
      }

      filteredRecords.push(record);
    }

    // Sort Layer
    sortFilteredRecords();

    // Clear list and progressive render first batch
    resultsInner.innerHTML = '';
    renderedCount = 0;
    appendNextBatch();
  }

  // Sort logic layer
  function sortFilteredRecords() {
    const order = filters[activeTab].sort;

    if (order === 'name') {
      filteredRecords.sort((a,b) => a.s.localeCompare(b.s));
    } else if (order === 'level') {
      if (activeTab === 'mobs') {
        filteredRecords.sort((a,b) => b.lvl - a.lvl || a.v - b.v); // Level descending, VNUM ascending
      } else {
        filteredRecords.sort((a,b) => b.cst - a.cst || a.v - b.v); // Cost descending for items
      }
    } else if (order === 'vnum') {
      filteredRecords.sort((a,b) => a.v - b.v);
    } else {
      // Default: Sort by primary zone, then level/cost, then name
      filteredRecords.sort((a,b) => {
        const zoneA = getPrimaryZone(a);
        const zoneB = getPrimaryZone(b);
        if (zoneA !== zoneB) return zoneA - zoneB;

        if (activeTab === 'mobs') {
          if (a.lvl !== b.lvl) return b.lvl - a.lvl;
        } else {
          if (a.cst !== b.cst) return b.cst - a.cst;
        }
        return a.s.localeCompare(b.s);
      });
    }
  }

  function getPrimaryZone(r) {
    if (activeTab === 'mobs') {
      return r.spw && r.spw.length > 0 ? r.spw[0].zone : 999;
    } else {
      return r.rms && r.rms.length > 0 && r.rms[0].zone ? r.rms[0].zone : 999;
    }
  }

  function getZoneName(zoneId) {
    if (zoneId === 999) return "Isolated Items / Objects";
    return zoneNames[zoneId] || `Zone ${zoneId}`;
  }

  // progressive progressive appending to list
  function appendNextBatch() {
    if (filteredRecords.length === 0) {
      resultsInner.innerHTML = '<div style="padding:2rem;text-align:center;color:var(--ink-muted);font-size:0.8rem;">No records matched your filters.</div>';
      updateScrollShadows();
      return;
    }

    const start = renderedCount;
    const end = Math.min(start + BATCH_SIZE, filteredRecords.length);
    const batch = filteredRecords.slice(start, end);

    let lastZone = start > 0 ? getPrimaryZone(filteredRecords[start - 1]) : null;
    let lastLevelBand = start > 0 ? Math.floor(filteredRecords[start - 1].lvl / 10) : null;

    const isZoneSorted = filters[activeTab].sort === 'zone';
    const isLevelSorted = filters[activeTab].sort === 'level' && activeTab === 'mobs';

    batch.forEach((r, idx) => {
      const currentZone = getPrimaryZone(r);

      // Zone Category headers in default sort
      if (isZoneSorted && currentZone !== lastZone) {
        const groupHeader = document.createElement('div');
        groupHeader.className = 'zone-header';
        const zoneCount = filteredRecords.filter(rec => getPrimaryZone(rec) === currentZone).length;
        groupHeader.innerText = `// ${getZoneName(currentZone).toUpperCase()} · ${zoneCount} entries`;
        resultsInner.appendChild(groupHeader);
        lastZone = currentZone;
      }

      // Level band tier dividers
      if (isLevelSorted && activeTab === 'mobs') {
        const currentBand = Math.floor(r.lvl / 10);
        if (currentBand !== lastLevelBand) {
          const tierDiv = document.createElement('div');
          tierDiv.className = 'tier-divider';
          const minL = currentBand * 10 || 1;
          const maxL = currentBand * 10 + 9;
          tierDiv.innerText = `TIER ${getRomanNumeral(4 - currentBand)} (LVL ${minL}-${maxL})`;
          resultsInner.appendChild(tierDiv);
          lastLevelBand = currentBand;
        }
      }

      // Result Row
      const item = document.createElement('div');
      item.className = 'result-item';
      if (String(r.v) === activeId) {
        item.classList.add('active');
      }
      item.setAttribute('data-id', r.v);

      // Gutter flags
      let gutterHtml = '';
      if (activeTab === 'mobs') {
        const isShop = !!r.shop;
        const isAgg = r.alg <= -700 || r.k.includes('aggressive') || r.k.includes('aggro');
        const isUniq = r.spw && r.spw.length === 1;

        if (isShop) gutterHtml = `<span class="gutter-flag" title="Shopkeeper">§</span>`;
        else if (isUniq) gutterHtml = `<span class="gutter-flag" title="Unique Mob">◆</span>`;
        else if (isAgg) gutterHtml = `<span class="gutter-flag" title="Aggressive candidate">‼</span>`;
        else gutterHtml = `<span class="gutter-flag"></span>`;
      } else {
        const isMag = r.aff && r.aff.length > 0;
        const isScr = !!r.script;

        if (isScr) gutterHtml = `<span class="gutter-flag" title="Special Scripted Object">📜</span>`;
        else if (isMag) gutterHtml = `<span class="gutter-flag" title="Magical Affects Object">✦</span>`;
        else gutterHtml = `<span class="gutter-flag"></span>`;
      }

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
        ${gutterHtml}
        <div class="result-item-content">
          <span class="result-name">${r.s}</span>
          <div class="result-meta">
            <span>${metaLeft}</span>
            <span class="result-vnum">#${r.v}</span>
          </div>
        </div>
      `;

      item.addEventListener('click', () => {
        const hashType = activeTab === 'mobs' ? 'mob' : 'item';
        window.location.hash = `#${hashType}-${r.v}`;
      });

      resultsInner.appendChild(item);
    });

    renderedCount = end;
    updateScrollShadows();
  }

  function getRomanNumeral(n) {
    const map = { 4: 'IV', 3: 'III', 2: 'II', 1: 'I' };
    return map[n] || 'I';
  }

  // Skyline analytics rendering
  function renderSkyline() {
    if (!dbData) return;
    const contentEl = skylineContainer.querySelector('.skyline-content');
    const skeletonEl = skylineContainer.querySelector('.skyline-skeleton');

    skeletonEl.style.display = 'none';
    contentEl.style.display = 'block';
    contentEl.innerHTML = '';

    const list = dbData[activeTab];
    
    // Panel 1: Level histogram
    let panel1Html = '';
    if (activeTab === 'mobs') {
      const bands = [0, 0, 0, 0, 0, 0, 0, 0]; // 8 bands of 5 levels: 1-5, 6-10, 11-15, 16-20, 21-25, 26-30, 31-35, 36-40
      for (const id in list) {
        const m = list[id];
        const bandIdx = Math.min(7, Math.floor((m.lvl - 1) / 5));
        if (bandIdx >= 0) bands[bandIdx]++;
      }
      const maxBand = Math.max(...bands);
      const barsHtml = bands.map((cnt, idx) => {
        const minL = idx * 5 + 1;
        const maxL = idx * 5 + 5;
        const bandLabel = idx === 7 ? `${minL}+` : `${minL}-${maxL}`;
        const heightPct = maxBand > 0 ? (cnt / maxBand) * 100 : 0;
        const isActive = filters.mobs.levelMin >= minL && filters.mobs.levelMin <= maxL;
        return `
          <div class="skyline-bar-wrap ${isActive ? 'active' : ''}" data-min="${minL}">
            <div class="skyline-bar" style="height: ${heightPct}%;"></div>
            <div class="skyline-tooltip">${cnt} mobs (Lvl ${bandLabel})</div>
          </div>
        `;
      }).join('');
      panel1Html = `
        <div class="skyline-panel">
          <h4>Level Curve Density</h4>
          <div class="skyline-chart">${barsHtml}</div>
        </div>
      `;
    } else {
      // Items by Weight Curves
      const bands = [0, 0, 0, 0, 0, 0, 0, 0]; // 8 bands: 0-1, 2-5, 6-10, 11-20, 21-50, 51-100, 101-200, 201+
      const bandLabels = ['0-1', '2-5', '6-10', '11-20', '21-50', '51-100', '101-200', '201+'];
      for (const id in list) {
        const o = list[id];
        let idx = 7;
        if (o.wt <= 1) idx = 0;
        else if (o.wt <= 5) idx = 1;
        else if (o.wt <= 10) idx = 2;
        else if (o.wt <= 20) idx = 3;
        else if (o.wt <= 50) idx = 4;
        else if (o.wt <= 100) idx = 5;
        else if (o.wt <= 200) idx = 6;
        bands[idx]++;
      }
      const maxBand = Math.max(...bands);
      const barsHtml = bands.map((cnt, idx) => {
        const heightPct = maxBand > 0 ? (cnt / maxBand) * 100 : 0;
        return `
          <div class="skyline-bar-wrap" data-weight-idx="${idx}">
            <div class="skyline-bar" style="height: ${heightPct}%;"></div>
            <div class="skyline-tooltip">${cnt} items (${bandLabels[idx]} lbs)</div>
          </div>
        `;
      }).join('');
      panel1Html = `
        <div class="skyline-panel">
          <h4>Weight Spectrum (lbs)</h4>
          <div class="skyline-chart">${barsHtml}</div>
        </div>
      `;
    }

    // Panel 2: Density by Zone (Top 3 active zones)
    const zoneCounts = {};
    for (const id in list) {
      const r = list[id];
      const zoneId = getPrimaryZone(r);
      if (zoneId !== 999) {
        zoneCounts[zoneId] = (zoneCounts[zoneId] || 0) + 1;
      }
    }
    const topZones = Object.entries(zoneCounts)
      .sort((a,b) => b[1] - a[1])
      .slice(0, 3);

    const zonesListHtml = topZones.map(([zoneId, cnt]) => {
      const isSelected = activeTab === 'mobs' ? filters.mobs.zones.has(parseInt(zoneId)) : filters.items.zones.has(parseInt(zoneId));
      return `
        <div class="skyline-row ${isSelected ? 'active' : ''}" data-zone="${zoneId}">
          <span>${getZoneName(parseInt(zoneId))}</span>
          <span style="font-weight:700;">${cnt}</span>
        </div>
      `;
    }).join('');
    
    const panel2Html = `
      <div class="skyline-panel">
        <h4>Top Spawn Areas</h4>
        <div class="skyline-list" style="margin-top:0.25rem;">
          ${zonesListHtml || '<div style="font-size:0.65rem;color:var(--ink-muted);">No zones spawned.</div>'}
        </div>
      </div>
    `;

    // Panel 3: Flag breakdowns
    let panel3Html = '';
    if (activeTab === 'mobs') {
      let shops = 0, aggs = 0, uniqs = 0;
      for (const id in list) {
        const m = list[id];
        if (m.shop) shops++;
        if (m.alg <= -700 || m.k.includes('aggressive') || m.k.includes('aggro')) aggs++;
        if (m.spw && m.spw.length === 1) uniqs++;
      }
      panel3Html = `
        <div class="skyline-panel">
          <h4>Observable Temperaments</h4>
          <div class="skyline-chips">
            <span class="skyline-chip ${filters.mobs.shopkeeper ? 'active' : ''}" id="sky-chip-shop">§ Shopkeepers (${shops})</span>
            <span class="skyline-chip ${filters.mobs.aggressive ? 'active' : ''}" id="sky-chip-agg">‼ Aggressive (${aggs})</span>
            <span class="skyline-chip ${filters.mobs.unique ? 'active' : ''}" id="sky-chip-uniq">◆ Uniques (${uniqs})</span>
          </div>
        </div>
      `;
    } else {
      let magical = 0, scripted = 0, standard = 0;
      for (const id in list) {
        const o = list[id];
        if (o.aff && o.aff.length > 0) magical++;
        if (o.script) scripted++;
        if ((!o.aff || o.aff.length === 0) && !o.script) standard++;
      }
      panel3Html = `
        <div class="skyline-panel">
          <h4>Object Attributes</h4>
          <div class="skyline-chips">
            <span class="skyline-chip ${filters.items.magic ? 'active' : ''}" id="sky-chip-mag">🪄 Magic (${magical})</span>
            <span class="skyline-chip ${filters.items.script ? 'active' : ''}" id="sky-chip-scr">📜 Scripted (${scripted})</span>
            <span class="skyline-chip" style="pointer-events:none;">✦ Standard (${standard})</span>
          </div>
        </div>
      `;
    }

    const grid = document.createElement('div');
    grid.className = 'skyline-grid';
    grid.innerHTML = panel1Html + panel2Html + panel3Html;
    contentEl.appendChild(grid);

    // Bind skyline click listeners
    // 1. Level chart click
    const bars = grid.querySelectorAll('.skyline-bar-wrap');
    bars.forEach(bar => {
      bar.addEventListener('click', () => {
        if (activeTab === 'mobs') {
          const minL = parseInt(bar.getAttribute('data-min'));
          mobLevelSlider.value = minL;
          mobLevelVal.innerText = minL;
          filters.mobs.levelMin = minL;
          applyFiltersAndRender();
          renderSkyline();
        }
      });
    });

    // 2. Zone rows click
    const rows = grid.querySelectorAll('.skyline-row');
    rows.forEach(row => {
      row.addEventListener('click', () => {
        const zoneId = parseInt(row.getAttribute('data-zone'));
        if (activeTab === 'mobs') {
          if (filters.mobs.zones.has(zoneId)) filters.mobs.zones.delete(zoneId);
          else filters.mobs.zones.add(zoneId);
          // Sync checkbox UI
          const chk = mobZoneContainer.querySelector(`.chk-mob-zone-item[value="${zoneId}"]`);
          if (chk) chk.checked = filters.mobs.zones.has(zoneId);
        } else {
          if (filters.items.zones.has(zoneId)) filters.items.zones.delete(zoneId);
          else filters.items.zones.add(zoneId);
          // Sync checkbox UI
          const chk = itemZoneContainer.querySelector(`.chk-itm-zone-item[value="${zoneId}"]`);
          if (chk) chk.checked = filters.items.zones.has(zoneId);
        }
        applyFiltersAndRender();
        renderSkyline();
      });
    });

    // 3. Flag chip click
    const shopChip = grid.querySelector('#sky-chip-shop');
    const aggChip = grid.querySelector('#sky-chip-agg');
    const uniqChip = grid.querySelector('#sky-chip-uniq');

    if (shopChip) {
      shopChip.addEventListener('click', () => {
        chkMobShop.checked = !chkMobShop.checked;
        filters.mobs.shopkeeper = chkMobShop.checked;
        toggleFlagClass('switch-mob-shop', chkMobShop.checked);
        applyFiltersAndRender();
        renderSkyline();
      });
    }
    if (aggChip) {
      aggChip.addEventListener('click', () => {
        chkMobAgg.checked = !chkMobAgg.checked;
        filters.mobs.aggressive = chkMobAgg.checked;
        toggleFlagClass('switch-mob-agg', chkMobAgg.checked);
        applyFiltersAndRender();
        renderSkyline();
      });
    }
    if (uniqChip) {
      uniqChip.addEventListener('click', () => {
        chkMobUniq.checked = !chkMobUniq.checked;
        filters.mobs.unique = chkMobUniq.checked;
        toggleFlagClass('switch-mob-uniq', chkMobUniq.checked);
        applyFiltersAndRender();
        renderSkyline();
      });
    }

    const magChip = grid.querySelector('#sky-chip-mag');
    const scrChip = grid.querySelector('#sky-chip-scr');

    if (magChip) {
      magChip.addEventListener('click', () => {
        chkItmMag.checked = !chkItmMag.checked;
        filters.items.magic = chkItmMag.checked;
        toggleFlagClass('switch-itm-mag', chkItmMag.checked);
        applyFiltersAndRender();
        renderSkyline();
      });
    }
    if (scrChip) {
      scrChip.addEventListener('click', () => {
        chkItmScr.checked = !chkItmScr.checked;
        filters.items.script = chkItmScr.checked;
        toggleFlagClass('switch-itm-scr', chkItmScr.checked);
        applyFiltersAndRender();
        renderSkyline();
      });
    }
  }

  // Render Detail Card
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

    // Spawn Locations
    let spawnsHtml = '<li>This mobile does not spawn naturally in the world.</li>';
    if (m.spw && m.spw.length > 0) {
      spawnsHtml = m.spw.map(s => `
        <li class="rel-item">
          <a class="rel-link" href="/map?room=${s.room}">Room ${s.room}: ${s.name}</a>
          <span class="rel-meta">Zone ${s.zone}</span>
        </li>
      `).join('');
    }

    // Equipment & Inventory drops (enriched with lookup values)
    let dropsHtml = '<li>This mobile carries no items.</li>';
    if (m.drp && m.drp.length > 0) {
      dropsHtml = m.drp.map(d => {
        const itemLookup = dbData.items[d.obj_vnum];
        let priceText = '';
        let magicalHtml = '';
        if (itemLookup) {
          priceText = ` · <span style="color:var(--accent);font-weight:700;">${itemLookup.cst} gp</span> · ${itemLookup.type}`;
          
          // Magical affects chips preview
          if (itemLookup.aff && itemLookup.aff.length > 0) {
            const chips = itemLookup.aff.map(a => `
              <span class="aff-micro-chip">${a.location} ${a.modifier > 0 ? '+' : ''}${a.modifier}</span>
            `).join('');
            magicalHtml = `<div class="aff-chip-row">${chips}</div>`;
          }
        }

        return `
          <li class="rel-item" style="flex-direction: column; align-items: flex-start; gap: 0.25rem;">
            <div style="width: 100%; display: flex; justify-content: space-between; align-items: center;">
              <a class="rel-link" href="#item-${d.obj_vnum}">${d.name}</a>
              <span class="rel-meta">${d.slot}${priceText}</span>
            </div>
            ${magicalHtml}
          </li>
        `;
      }).join('');
    }

    // Shopkeeper details
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
          <p style="font-size:0.75rem;margin:0 0 0.5rem;color:var(--ink-muted);line-height:1.4;">
            Runs shop #${m.shop.shop_vnum} open ${m.shop.open_hours}. Markup: x${m.shop.sell_mult}
          </p>
          <ul class="rel-list">${shopItems}</ul>
        </div>
      `;
    }

    const totalCount = Object.keys(dbData.mobs).length;

    detailContainer.innerHTML = `
      <article class="detail-card">
        <header class="detail-header">
          <div class="detail-stamp">№ ${m.v} · Mob record of ${totalCount}</div>
          <h2 class="detail-short">${m.s}</h2>
          <p class="detail-long">${m.l}</p>
          <p style="font-family:var(--font-mono);font-size:0.65rem;margin:0.5rem 0 0;"><a href="/mobs/${m.v}/">Permanent record</a></p>
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
              <tr><td class="label">Base AC</td><td class="val">${m.ac} ac</td></tr>
              <tr><td class="label">Gold</td><td class="val">${m.gld} gp</td></tr>
              <tr><td class="label">Experience</td><td class="val">${m.exp.toLocaleString()}</td></tr>
            </table>
          </div>

          <!-- Middle Column: Combat & Power visuals -->
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
                  <div class="vis-bar-label"><span>Armor Rating</span><span>${m.ac} ac</span></div>
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

        <div class="detail-grid" style="margin-top:0.5rem;">
          <div class="detail-section">
            <h3 style="border-bottom:none;">Vital Stats</h3>
            <table class="stats-table">
              <tr><td class="label">Hit Points (HP)</td><td class="val">${calcDiceAvg(m.hp)} hp (${m.hp})</td></tr>
              <tr><td class="label">Damage Range</td><td class="val">${calcDiceAvg(m.dmg)} avg (${m.dmg})</td></tr>
            </table>
          </div>
          <div class="detail-section">
            <h3 style="border-bottom:none;">Base Attributes</h3>
            <div style="display:grid;grid-template-columns:repeat(3, 1fr);gap:0.5rem;font-size:0.7rem;text-align:center;">
              <div style="background:var(--paper-deep);padding:0.4rem;border-radius:2px;"><strong style="display:block;color:var(--accent);">STR</strong> ${m.stat[0]}</div>
              <div style="background:var(--paper-deep);padding:0.4rem;border-radius:2px;"><strong style="display:block;color:var(--accent);">INT</strong> ${m.stat[1]}</div>
              <div style="background:var(--paper-deep);padding:0.4rem;border-radius:2px;"><strong style="display:block;color:var(--accent);">WIS</strong> ${m.stat[2]}</div>
              <div style="background:var(--paper-deep);padding:0.4rem;border-radius:2px;"><strong style="display:block;color:var(--accent);">DEX</strong> ${m.stat[3]}</div>
              <div style="background:var(--paper-deep);padding:0.4rem;border-radius:2px;"><strong style="display:block;color:var(--accent);">CON</strong> ${m.stat[4]}</div>
              <div style="background:var(--paper-deep);padding:0.4rem;border-radius:2px;"><strong style="display:block;color:var(--accent);">CHA</strong> ${m.stat[5]}</div>
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
    const wearPosStr = o.wear.length > 0 ? o.wear.join(', ') : 'None';
    const extraFlagsStr = o.extra.length > 0 ? o.extra.join(', ') : 'None';

    // Magical Affects location list
    let affectsHtml = '<p style="font-size:0.75rem;color:var(--ink-muted);margin:0;">No magical affects on this item.</p>';
    if (o.aff && o.aff.length > 0) {
      affectsHtml = o.aff.map(a => `
        <div style="font-size:0.75rem;padding:0.45rem 0.6rem;background:var(--paper-deep);border:1px solid rgba(26,22,20,0.15);border-radius:2px;display:flex;justify-content:space-between;margin-bottom:0.25rem;">
          <span style="font-weight:700;color:var(--accent);">${a.location}</span>
          <span style="font-family:var(--font-mono);font-weight:700;">${a.modifier > 0 ? '+' : ''}${a.modifier}</span>
        </div>
      `).join('');
    }

    // Spawn / Loaded by Mob details (enriched with mob lookup stats)
    let dropsHtml = '<li>No mobs load naturally with this item.</li>';
    if (o.mobs && o.mobs.length > 0) {
      dropsHtml = o.mobs.map(l => {
        const mobLookup = dbData.mobs[l.mob_vnum];
        let mobMeta = '';
        if (mobLookup) {
          const isAgg = mobLookup.alg <= -700 || mobLookup.k.includes('aggressive');
          mobMeta = ` · Lvl ${mobLookup.lvl} ${isAgg ? '‼' : ''} ${mobLookup.alg >= 300 ? 'Good' : mobLookup.alg <= -300 ? 'Evil' : 'Neutral'}`;
        }
        return `
          <li class="rel-item">
            <a class="rel-link" href="#mob-${l.mob_vnum}">${l.name}</a>
            <span class="rel-meta">${l.slot}${mobMeta}</span>
          </li>
        `;
      }).join('');
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

    // Sold by merchants
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

    // Extra observation text
    let edescHtml = '';
    if (o.edesc && o.edesc.length > 0) {
      const edescItems = o.edesc.map(ed => `
        <div style="margin-bottom:0.75rem;">
          <h4 style="font-family:var(--font-display);font-size:0.7rem;text-transform:uppercase;color:var(--ink-muted);margin:0 0 0.25rem 0;border-bottom:1px solid rgba(26,22,20,0.1);padding-bottom:0.15rem;">Keywords: ${ed.keywords}</h4>
          <p style="font-size:0.85rem;line-height:1.55;margin:0;white-space:pre-wrap;font-family:var(--font-body);">${ed.desc}</p>
        </div>
      `).join('');
      edescHtml = `
        <div class="detail-section">
          <h3>Extra Details</h3>
          <div class="detail-desc-block">${edescItems}</div>
        </div>
      `;
    }

    const totalCount = Object.keys(dbData.items).length;

    detailContainer.innerHTML = `
      <article class="detail-card">
        <header class="detail-header">
          <div class="detail-stamp">№ ${o.v} · Item record of ${totalCount}</div>
          <h2 class="detail-short">${o.s}</h2>
          <p class="detail-long">${o.l}</p>
          <p style="font-family:var(--font-mono);font-size:0.65rem;margin:0.5rem 0 0;"><a href="/items/${o.v}/">Permanent record</a></p>
        </header>

        <div class="detail-grid">
          <!-- Left Column: Core Specs -->
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

          <!-- Middle Column: Affects & Scripts -->
          <div class="detail-section">
            <h3>Magical Affects</h3>
            ${affectsHtml}
            
            ${o.script ? `
            <div style="margin-top:0.75rem;padding:0.5rem 0.75rem;border:1px solid var(--accent);border-radius:2px;font-size:0.75rem;background:rgba(168, 32, 26, 0.04);box-shadow: 2px 2px 0px var(--accent);">
              <strong style="color:var(--accent);font-family:var(--font-display);text-transform:uppercase;font-size:0.6rem;letter-spacing:0.05em;display:block;margin-bottom:0.15rem;">Special Script</strong>
              <code style="font-family:var(--font-mono);background:none;color:var(--ink);padding:0;">${o.script}</code>
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

  // Launch on DOM ready
  document.addEventListener('DOMContentLoaded', loadDatabase);

})();
