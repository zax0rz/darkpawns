// Client-side Omnisearch Engine for Dark Pawns Astro site

export interface SearchEntry {
  t: string; // Title
  c: string; // Category: help | items | mobs | world | docs | pages
  s: string; // Subtype badge
  u: string; // URL
  k: string; // Keywords
  d: string; // Description
  v: number; // VNUM
}

interface ScoredEntry {
  item: SearchEntry;
  score: number;
}

export class OmnisearchClient {
  private modal: HTMLElement | null = null;
  private backdrop: HTMLElement | null = null;
  private input: HTMLInputElement | null = null;
  private clearBtn: HTMLElement | null = null;
  private closeBtn: HTMLElement | null = null;
  private filterPills: NodeListOf<Element> = document.querySelectorAll('.filter-pill');
  private resultsList: HTMLElement | null = null;
  private emptyState: HTMLElement | null = null;
  private countEl: HTMLElement | null = null;

  private index: SearchEntry[] = [];
  private activeCategory = 'all';
  private activeIndex = -1;
  private isIndexLoaded = false;
  private isLoading = false;
  private debounceTimer: number | null = null;
  private lastFocusedElement: HTMLElement | null = null;

  constructor() {
    this.modal = document.getElementById('dp-search-modal');
    this.backdrop = document.getElementById('dp-search-backdrop');
    this.input = document.getElementById('dp-search-input') as HTMLInputElement;
    this.clearBtn = document.getElementById('dp-search-clear');
    this.closeBtn = document.getElementById('dp-search-close');
    this.filterPills = document.querySelectorAll('.filter-pill');
    this.resultsList = document.getElementById('dp-search-list');
    this.emptyState = document.getElementById('dp-search-empty');
    this.countEl = document.getElementById('dp-search-count');

    this.initEventListeners();
  }

  private initEventListeners() {
    // Global hotkeys: Cmd+K, Ctrl+K, /
    window.addEventListener('keydown', (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        this.toggle();
      } else if (e.key === '/' && !this.isTypingInInput(e.target as HTMLElement)) {
        e.preventDefault();
        this.open();
      } else if (e.key === 'Escape' && this.isOpen()) {
        e.preventDefault();
        this.close();
      }
    });

    // Triggers with [data-search-trigger]
    document.querySelectorAll('[data-search-trigger]').forEach((el) => {
      el.addEventListener('click', (e) => {
        e.preventDefault();
        this.open();
      });
      // Pre-fetch on hover
      el.addEventListener('mouseenter', () => this.loadIndex());
    });

    // Quick query suggestion buttons
    document.querySelectorAll('.quick-link').forEach((btn) => {
      btn.addEventListener('click', (e) => {
        const q = (e.target as HTMLElement).getAttribute('data-query');
        if (q && this.input) {
          this.input.value = q;
          this.handleInput();
        }
      });
    });

    // Backdrop & Close button
    this.backdrop?.addEventListener('click', () => this.close());
    this.closeBtn?.addEventListener('click', () => this.close());
    this.clearBtn?.addEventListener('click', () => {
      if (this.input) {
        this.input.value = '';
        this.input.focus();
        this.handleInput();
      }
    });

    // Category filter pills
    this.filterPills.forEach((pill) => {
      pill.addEventListener('click', () => {
        this.filterPills.forEach((p) => {
          p.classList.remove('active');
          p.setAttribute('aria-selected', 'false');
        });
        pill.classList.add('active');
        pill.setAttribute('aria-selected', 'true');
        this.activeCategory = pill.getAttribute('data-category') || 'all';
        this.handleInput();
      });
    });

    // Input event listener (debounced)
    this.input?.addEventListener('input', () => {
      if (this.debounceTimer) window.clearTimeout(this.debounceTimer);
      this.debounceTimer = window.setTimeout(() => this.handleInput(), 80);
    });

    // Keyboard navigation inside results
    this.input?.addEventListener('keydown', (e) => this.handleKeydown(e));
  }

  private isTypingInInput(target: HTMLElement | null): boolean {
    if (!target) return false;
    const tag = target.tagName.toLowerCase();
    return tag === 'input' || tag === 'textarea' || target.isContentEditable;
  }

  public isOpen(): boolean {
    return this.modal ? !this.modal.hidden : false;
  }

  public open() {
    if (!this.modal || this.isOpen()) return;
    this.lastFocusedElement = document.activeElement as HTMLElement;
    this.modal.hidden = false;
    document.body.style.overflow = 'hidden';
    this.loadIndex();
    setTimeout(() => {
      this.input?.focus();
      this.input?.select();
    }, 30);
  }

  public close() {
    if (!this.modal || !this.isOpen()) return;
    this.modal.hidden = true;
    document.body.style.overflow = '';
    if (this.lastFocusedElement) {
      this.lastFocusedElement.focus();
    }
  }

  public toggle() {
    if (this.isOpen()) {
      this.close();
    } else {
      this.open();
    }
  }

  private async loadIndex() {
    if (this.isIndexLoaded || this.isLoading) return;
    this.isLoading = true;
    if (this.countEl) this.countEl.textContent = 'Loading index...';

    try {
      const res = await fetch('/data/search-index.json');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      this.index = await res.json();
      this.isIndexLoaded = true;
      if (this.countEl) this.countEl.textContent = `${this.index.length.toLocaleString()} entries loaded`;
      if (this.input && this.input.value.trim().length > 0) {
        this.handleInput();
      }
    } catch (err) {
      console.error('Failed to load search index:', err);
      if (this.countEl) this.countEl.textContent = 'Failed to load index';
    } finally {
      this.isLoading = false;
    }
  }

  private handleInput() {
    const query = (this.input?.value || '').trim().toLowerCase();
    if (this.clearBtn) {
      this.clearBtn.hidden = query.length === 0;
    }

    if (!query) {
      if (this.emptyState) this.emptyState.style.display = 'block';
      if (this.resultsList) this.resultsList.innerHTML = '';
      if (this.countEl) {
        this.countEl.textContent = this.isIndexLoaded
          ? `${this.index.length.toLocaleString()} entries`
          : 'Ready';
      }
      this.activeIndex = -1;
      return;
    }

    if (!this.isIndexLoaded) {
      this.loadIndex();
      return;
    }

    if (this.emptyState) this.emptyState.style.display = 'none';

    const results = this.search(query, this.activeCategory);
    this.renderResults(results, query);
  }

  private search(query: string, category: string): SearchEntry[] {
    const tokens = query.split(/\s+/).filter(Boolean);
    const isVnumQuery = query.startsWith('#') || /^\d+$/.test(query);
    const rawNum = query.replace('#', '');

    const scored: ScoredEntry[] = [];

    for (let i = 0; i < this.index.length; i++) {
      const item = this.index[i];
      if (category !== 'all' && item.c !== category) {
        continue;
      }

      let score = 0;
      const titleLower = item.t.toLowerCase();
      const descLower = item.d.toLowerCase();
      const keyLower = item.k.toLowerCase();
      const subtypeLower = item.s.toLowerCase();

      // Exact title match
      if (titleLower === query) score += 150;
      else if (titleLower.startsWith(query)) score += 80;

      // VNUM exact match
      if (isVnumQuery && item.v && String(item.v) === rawNum) {
        score += 200;
      }

      // Token matching
      let matchesAllTokens = true;
      for (const token of tokens) {
        if (titleLower.includes(token)) score += 35;
        else if (keyLower.includes(token)) score += 20;
        else if (subtypeLower.includes(token)) score += 15;
        else if (descLower.includes(token)) score += 5;
        else {
          matchesAllTokens = false;
        }
      }

      if (matchesAllTokens && score > 0) {
        scored.push({ item, score });
      }
    }

    scored.sort((a, b) => b.score - a.score);
    return scored.slice(0, 50).map((s) => s.item);
  }

  private highlight(text: string, query: string): string {
    if (!query || !text) return text;
    const tokens = query.split(/\s+/).filter(Boolean).map((t) => t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));
    if (tokens.length === 0) return text;
    const regex = new RegExp(`(${tokens.join('|')})`, 'gi');
    return text.replace(regex, '<mark>$1</mark>');
  }

  private renderResults(results: SearchEntry[], query: string) {
    if (!this.resultsList) return;

    if (results.length === 0) {
      this.resultsList.innerHTML = `
        <div class="search-state-msg">
          <p class="empty-title">No matches found for "${this.escapeHtml(query)}"</p>
          <p class="empty-sub">Try searching by category, spell name, item type, or VNUM.</p>
        </div>
      `;
      if (this.countEl) this.countEl.textContent = '0 matches';
      this.activeIndex = -1;
      return;
    }

    if (this.countEl) {
      this.countEl.textContent = `${results.length} match${results.length === 1 ? '' : 'es'}`;
    }

    // Grouping logic when category == 'all'
    if (this.activeCategory === 'all') {
      const groups: Record<string, { label: string; icon: string; items: SearchEntry[] }> = {
        help: { label: 'Help & Spells', icon: '📜', items: [] },
        items: { label: 'Items & Equipment', icon: '⚔️', items: [] },
        mobs: { label: 'Monsters & NPCs', icon: '👹', items: [] },
        world: { label: 'World & Map', icon: '🗺️', items: [] },
        docs: { label: 'Docs & Lore', icon: '📖', items: [] },
        pages: { label: 'Core Pages', icon: '🏛️', items: [] },
      };

      for (const item of results) {
        if (groups[item.c]) {
          groups[item.c].items.push(item);
        }
      }

      let html = '';
      let globalIndex = 0;

      for (const [key, group] of Object.entries(groups)) {
        if (group.items.length === 0) continue;
        html += `<div class="search-group-header"><span>${group.icon} ${group.label}</span><span>${group.items.length}</span></div>`;
        
        for (const item of group.items) {
          html += this.renderItemHtml(item, query, globalIndex++);
        }

        if (key === 'items' && group.items.length >= 4) {
          html += `<a href="/database/?tab=items&q=${encodeURIComponent(query)}" class="search-handoff-link">View all matching items in Codex &rarr;</a>`;
        } else if (key === 'mobs' && group.items.length >= 4) {
          html += `<a href="/database/?tab=mobs&q=${encodeURIComponent(query)}" class="search-handoff-link">View all matching mobs in Codex &rarr;</a>`;
        }
      }

      this.resultsList.innerHTML = html;
    } else {
      // Flat filtered list
      let html = '';
      results.forEach((item, index) => {
        html += this.renderItemHtml(item, query, index);
      });
      this.resultsList.innerHTML = html;
    }

    this.activeIndex = 0;
    this.updateActiveItem();
  }

  private renderItemHtml(item: SearchEntry, query: string, index: number): string {
    const vnumHtml = item.v ? `<span class="search-vnum">#${item.v}</span>` : '';
    return `
      <a href="${item.u}" class="search-item" data-index="${index}" role="option" id="search-opt-${index}" aria-selected="false">
        <div class="search-item-main">
          <div class="search-item-title-row">
            <span class="search-item-title">${this.highlight(item.t, query)}</span>
            ${vnumHtml}
          </div>
          <p class="search-item-desc">${this.highlight(item.d, query)}</p>
        </div>
        <div class="search-item-meta">
          <span class="search-badge ${item.c}">${item.s}</span>
        </div>
      </a>
    `;
  }

  private escapeHtml(str: string): string {
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  private handleKeydown(e: KeyboardEvent) {
    const items = this.resultsList?.querySelectorAll('.search-item');
    if (!items || items.length === 0) return;

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      this.activeIndex = (this.activeIndex + 1) % items.length;
      this.updateActiveItem();
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      this.activeIndex = (this.activeIndex - 1 + items.length) % items.length;
      this.updateActiveItem();
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (this.activeIndex >= 0 && this.activeIndex < items.length) {
        const activeEl = items[this.activeIndex] as HTMLAnchorElement;
        if (activeEl && activeEl.href) {
          window.location.href = activeEl.href;
        }
      }
    }
  }

  private updateActiveItem() {
    const items = this.resultsList?.querySelectorAll('.search-item');
    if (!items) return;

    items.forEach((item, index) => {
      if (index === this.activeIndex) {
        item.classList.add('active');
        item.setAttribute('aria-selected', 'true');
        item.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
        if (this.input) {
          this.input.setAttribute('aria-activedescendant', item.id);
        }
      } else {
        item.classList.remove('active');
        item.setAttribute('aria-selected', 'false');
      }
    });
  }
}

// Auto-initialize when loaded
if (typeof document !== 'undefined') {
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
      (window as any).__dp_search = new OmnisearchClient();
    });
  } else {
    (window as any).__dp_search = new OmnisearchClient();
  }
}
