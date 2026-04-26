// Dark Pawns Interactive Map
// Initial implementation - will be replaced with full Mudlet map browser integration

class DarkPawnsMap {
  constructor() {
    this.container = document.getElementById('map-container');
    this.canvas = document.getElementById('map-canvas');
    this.ctx = this.canvas.getContext('2d');
    this.loading = document.getElementById('map-loading');
    
    // Map state
    this.zoom = 1.0;
    this.panX = 0;
    this.panY = 0;
    this.isDragging = false;
    this.lastX = 0;
    this.lastY = 0;
    
    // Map data
    this.areas = [];
    this.rooms = [];
    this.selectedRoom = null;
    
    // Initialize
    this.initCanvas();
    this.loadMapData();
    this.setupEventListeners();
    this.setupControls();
  }
  
  initCanvas() {
    // Set canvas size to match container
    this.canvas.width = this.container.clientWidth;
    this.canvas.height = this.container.clientHeight;
    this.canvas.style.display = 'block';
    this.loading.style.display = 'none';
  }
  
  async loadMapData() {
    try {
      // Try to load from API first
      const response = await fetch('/docs/api/map.json');
      if (response.ok) {
        const data = await response.json();
        this.areas = data.areas || [];
        this.rooms = data.rooms || [];
      } else {
        // Fallback to sample data
        this.loadSampleData();
      }
    } catch (error) {
      console.warn('Could not load map data, using sample data:', error);
      this.loadSampleData();
    }
    
    this.render();
    this.populateAreasList();
  }
  
  loadSampleData() {
    // Sample map data for demonstration
    this.areas = [
      { id: 1, name: 'Starting Village', description: 'A quiet village where new adventurers begin their journey.', color: '#4CAF50' },
      { id: 2, name: 'Dark Forest', description: 'A mysterious forest filled with danger and secrets.', color: '#2E7D32' },
      { id: 3, name: 'Ancient Ruins', description: 'Crumbling structures of a forgotten civilization.', color: '#795548' },
      { id: 4, name: 'Underground Caverns', description: 'Dark, winding tunnels beneath the surface.', color: '#37474F' }
    ];
    
    this.rooms = [
      { id: 1, name: 'Village Square', x: 100, y: 100, areaId: 1, description: 'The central gathering place of the village.', connections: [2, 3] },
      { id: 2, name: 'Blacksmith Shop', x: 200, y: 100, areaId: 1, description: 'Weapons and armor line the walls.', connections: [1] },
      { id: 3, name: 'Rusty Tankard Tavern', x: 100, y: 200, areaId: 1, description: 'A rowdy tavern filled with adventurers.', connections: [1, 4] },
      { id: 4, name: 'Forest Path', x: 300, y: 200, areaId: 2, description: 'A narrow path through the dark woods.', connections: [3, 5] },
      { id: 5, name: 'Ancient Temple', x: 400, y: 300, areaId: 3, description: 'A temple dedicated to forgotten gods.', connections: [4, 6] },
      { id: 6, name: 'Cave Entrance', x: 500, y: 400, areaId: 4, description: 'The entrance to the underground caverns.', connections: [5] }
    ];
  }
  
  setupEventListeners() {
    // Mouse events for panning
    this.canvas.addEventListener('mousedown', (e) => {
      this.isDragging = true;
      this.lastX = e.clientX;
      this.lastY = e.clientY;
      this.canvas.style.cursor = 'grabbing';
    });
    
    this.canvas.addEventListener('mousemove', (e) => {
      if (this.isDragging) {
        const dx = e.clientX - this.lastX;
        const dy = e.clientY - this.lastY;
        this.panX += dx;
        this.panY += dy;
        this.lastX = e.clientX;
        this.lastY = e.clientY;
        this.render();
      }
      
      // Check for room hover
      const rect = this.canvas.getBoundingClientRect();
      const x = (e.clientX - rect.left - this.panX) / this.zoom;
      const y = (e.clientY - rect.top - this.panY) / this.zoom;
      
      const hoveredRoom = this.getRoomAt(x, y);
      this.canvas.style.cursor = hoveredRoom ? 'pointer' : 'grab';
    });
    
    this.canvas.addEventListener('mouseup', () => {
      this.isDragging = false;
      this.canvas.style.cursor = 'grab';
    });
    
    this.canvas.addEventListener('mouseleave', () => {
      this.isDragging = false;
      this.canvas.style.cursor = 'default';
    });
    
    // Click event for room selection
    this.canvas.addEventListener('click', (e) => {
      const rect = this.canvas.getBoundingClientRect();
      const x = (e.clientX - rect.left - this.panX) / this.zoom;
      const y = (e.clientY - rect.top - this.panY) / this.zoom;
      
      const clickedRoom = this.getRoomAt(x, y);
      if (clickedRoom) {
        this.selectRoom(clickedRoom);
      }
    });
    
    // Zoom with mouse wheel
    this.canvas.addEventListener('wheel', (e) => {
      e.preventDefault();
      const zoomFactor = 0.1;
      const oldZoom = this.zoom;
      
      if (e.deltaY < 0) {
        this.zoom *= (1 + zoomFactor);
      } else {
        this.zoom *= (1 - zoomFactor);
      }
      
      // Clamp zoom
      this.zoom = Math.max(0.1, Math.min(5, this.zoom));
      
      // Adjust pan to zoom toward cursor
      const rect = this.canvas.getBoundingClientRect();
      const mouseX = e.clientX - rect.left;
      const mouseY = e.clientY - rect.top;
      
      this.panX = mouseX - (mouseX - this.panX) * (this.zoom / oldZoom);
      this.panY = mouseY - (mouseY - this.panY) * (this.zoom / oldZoom);
      
      this.render();
    });
    
    // Handle window resize
    window.addEventListener('resize', () => {
      this.initCanvas();
      this.render();
    });
  }
  
  setupControls() {
    // Zoom controls
    document.getElementById('zoom-in').addEventListener('click', () => {
      this.zoom *= 1.2;
      this.zoom = Math.min(5, this.zoom);
      this.render();
    });
    
    document.getElementById('zoom-out').addEventListener('click', () => {
      this.zoom *= 0.8;
      this.zoom = Math.max(0.1, this.zoom);
      this.render();
    });
    
    document.getElementById('reset-view').addEventListener('click', () => {
      this.zoom = 1.0;
      this.panX = 0;
      this.panY = 0;
      this.render();
    });
    
    // Search functionality
    const searchInput = document.getElementById('room-search');
    const searchBtn = document.getElementById('search-btn');
    
    searchBtn.addEventListener('click', () => this.performSearch(searchInput.value));
    searchInput.addEventListener('keypress', (e) => {
      if (e.key === 'Enter') {
        this.performSearch(searchInput.value);
      }
    });
    
    // Keyboard navigation
    document.addEventListener('keydown', (e) => {
      const panSpeed = 20;
      switch(e.key) {
        case 'ArrowUp':
          this.panY += panSpeed;
          this.render();
          break;
        case 'ArrowDown':
          this.panY -= panSpeed;
          this.render();
          break;
        case 'ArrowLeft':
          this.panX += panSpeed;
          this.render();
          break;
        case 'ArrowRight':
          this.panX -= panSpeed;
          this.render();
          break;
        case '+':
        case '=':
          this.zoom *= 1.2;
          this.render();
          break;
        case '-':
          this.zoom *= 0.8;
          this.render();
          break;
        case ' ':
          this.zoom = 1.0;
          this.panX = 0;
          this.panY = 0;
          this.render();
          break;
        case 'Escape':
          this.clearSearch();
          break;
      }
    });
  }
  
  getRoomAt(x, y) {
    const roomRadius = 15;
    for (const room of this.rooms) {
      const dx = room.x - x;
      const dy = room.y - y;
      const distance = Math.sqrt(dx * dx + dy * dy);
      if (distance < roomRadius) {
        return room;
      }
    }
    return null;
  }
  
  selectRoom(room) {
    this.selectedRoom = room;
    this.showRoomDetails(room);
    this.render();
  }
  
  showRoomDetails(room) {
    const area = this.areas.find(a => a.id === room.areaId);
    const detailsContainer = document.getElementById('room-details');
    const title = document.getElementById('room-title');
    const content = document.getElementById('room-content');
    
    detailsContainer.style.display = 'block';
    title.textContent = room.name;
    
    content.innerHTML = `
      <p><strong>Description:</strong> ${room.description || 'No description available.'}</p>
      <p><strong>Area:</strong> ${area ? area.name : 'Unknown'}</p>
      <p><strong>Coordinates:</strong> (${room.x}, ${room.y})</p>
      <p><strong>Connections:</strong> ${room.connections ? room.connections.length : 0} connected rooms</p>
      ${area ? `<p><strong>Area Description:</strong> ${area.description}</p>` : ''}
    `;
  }
  
  performSearch(query) {
    if (!query.trim()) {
      this.clearSearch();
      return;
    }
    
    const searchTerm = query.toLowerCase();
    const results = this.rooms.filter(room => 
      room.name.toLowerCase().includes(searchTerm) || 
      (room.description && room.description.toLowerCase().includes(searchTerm))
    );
    
    this.showSearchResults(results, query);
  }
  
  showSearchResults(results, query) {
    const resultsContainer = document.getElementById('search-results');
    const resultsList = document.getElementById('results-list');
    
    if (results.length === 0) {
      resultsList.innerHTML = `<p>No rooms found matching "${query}"</p>`;
    } else {
      resultsList.innerHTML = `
        <p>Found ${results.length} room(s) matching "${query}":</p>
        <ul style="margin-top: 10px;">
          ${results.map(room => `
            <li style="margin-bottom: 5px;">
              <a href="#" class="search-result" data-room-id="${room.id}" 
                 style="color: #4fc3f7; text-decoration: none;">
                ${room.name}
              </a>
              ${this.areas.find(a => a.id === room.areaId) ? ` (${this.areas.find(a => a.id === room.areaId).name})` : ''}
            </li>
          `).join('')}
        </ul>
      `;
      
      // Add click handlers to search results
      setTimeout(() => {
        document.querySelectorAll('.search-result').forEach(link => {
          link.addEventListener('click', (e) => {
            e.preventDefault();
            const roomId = parseInt(link.dataset.roomId);
            const room = this.rooms.find(r => r.id === roomId);
            if (room) {
              this.selectRoom(room);
              this.centerOnRoom(room);
              document.getElementById('room-search').value = '';
              resultsContainer.style.display = 'none';
            }
          });
        });
      }, 0);
    }
    
    resultsContainer.style.display = 'block';
  }
  
  clearSearch() {
    document.getElementById('room-search').value = '';
    document.getElementById('search-results').style.display = 'none';
  }
  
  centerOnRoom(room) {
    // Center the view on the selected room
    const centerX = this.canvas.width / 2;
    const centerY = this.canvas.height / 2;
    
    this.panX = centerX - room.x * this.zoom;
    this.panY = centerY - room.y * this.zoom;
    this.render();
  }
  
  populateAreasList() {
    const areasList = document.getElementById('areas-list');
    if (!areasList) return;
    
    areasList.innerHTML = this.areas.map(area => `
      <div class="box" style="background: ${area.color}20; border-left: 4px solid ${area.color};">
        <h4 style="color: ${area.color}; margin-bottom: 10px;">${area.name}</h4>
        <p style="color: #ccc; margin-bottom: 10px;">${area.description}</p>
        <p style="color: #999; font-size: 0.9em;">
          Rooms: ${this.rooms.filter(r => r.areaId === area.id).length}
        </p>
        <button class="button is-small is-outlined" 
                style="border-color: ${area.color}; color: ${area.color}; margin-top: 10px;"
                onclick="map.filterByArea(${area.id})">
          Show on Map
        </button>
      </div>
    `).join('');
  }
  
  filterByArea(areaId) {
    // For now, just highlight rooms in this area
    this.render();
  }
  
  render() {
    // Clear canvas
    this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
    
    // Save context
    this.ctx.save();
    
    // Apply transformations
    this.ctx.translate(this.panX, this.panY);
    this.ctx.scale(this.zoom, this.zoom);
    
    // Draw connections first (so they're behind rooms)
    this.drawConnections();
    
    // Draw rooms
    this.drawRooms();
    
    // Draw selected room highlight
    if (this.selectedRoom) {
      this.drawRoomHighlight(this.selectedRoom);
    }
    
    // Restore context
    this.ctx.restore();
    
    // Draw zoom level indicator
    this.drawZoomIndicator();
  }
  
  drawConnections() {
    this.ctx.strokeStyle = '#555';
    this.ctx.lineWidth = 2;
    
    for (const room of this.rooms) {
      if (room.connections) {
        for (const targetId of room.connections) {
          const targetRoom = this.rooms.find(r => r.id === targetId);
          if (targetRoom) {
            this.ctx.beginPath();
            this.ctx.moveTo(room.x, room.y);
            this.ctx.lineTo(targetRoom.x, targetRoom.y);
            this.ctx.stroke();
          }
        }
      }
    }
  }
  
  drawRooms() {
    for (const room of this.rooms) {
      const area = this.areas.find(a => a.id === room.areaId);
      const color = area ? area.color : '#666';
      
      // Draw room circle
      this.ctx.fillStyle = color;
      this.ctx.beginPath();
      this.ctx.arc(room.x, room.y, 10, 0, Math.PI * 2);
      this.ctx.fill();
      
      // Draw room border
      this.ctx.strokeStyle = '#fff';
      this.ctx.lineWidth = 2;
      this.ctx.stroke();
      
      // Draw room name (only if zoomed in enough)
      if (this.zoom > 0.5) {
        this.ctx.fillStyle = '#fff';
        this.ctx.font = '12px Arial';
        this.ctx.textAlign = 'center';
        this.ctx.fillText(room.name, room.x, room.y + 25);
      }
    }
  }
  
  drawRoomHighlight(room) {
    this.ctx.strokeStyle = '#FFD700';
    this.ctx.lineWidth = 3;
    this.ctx.beginPath();
    this.ctx.arc(room.x, room.y, 15, 0, Math.PI * 2);
    this.ctx.stroke();
    
    // Draw glow effect
    this.ctx.strokeStyle = '#FFD70040';
    this.ctx.lineWidth = 8;
    this.ctx.beginPath();
    this.ctx.arc(room.x, room.y, 18, 0, Math.PI * 2);
    this.ctx.stroke();
  }
  
  drawZoomIndicator() {
    this.ctx.fillStyle = 'rgba(0, 0, 0, 0.7)';
    this.ctx.fillRect(10, 10, 120, 40);
    
    this.ctx.fillStyle = '#fff';
    this.ctx.font = '14px Arial';
    this.ctx.textAlign = 'left';
    this.ctx.fillText(`Zoom: ${this.zoom.toFixed(1)}x`, 20, 30);
    this.ctx.fillText(`Pan: (${Math.round(this.panX)}, ${Math.round(this.panY)})`, 20, 50);
  }
}

// Initialize map when page loads
let map;
document.addEventListener('DOMContentLoaded', () => {
  map = new DarkPawnsMap();
  
  // Expose map to global scope for area filter buttons
  window.map = map;
});