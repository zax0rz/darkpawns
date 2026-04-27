# Dark Pawns Mudlet Map Implementation Plan

## Quick Start

### 1. Fork the Template
```bash
# Go to: https://github.com/Delwing/online-mudlet-map-template
# Click "Use this template" → "Create a new repository"
# Name: darkpawns-map
```

### 2. Generate Your Map
```bash
# Install dependencies
npm install mudlet-map-binary-reader

# Run the example generator
node map-generation-example.js

# This creates darkpawns-map.dat (simulated)
```

### 3. Deploy the Map
1. Replace `map.dat` in the template with your `darkpawns-map.dat`
2. Update `page/i18n/en.json`:
   ```json
   {
     "title": "Dark Pawns World Map",
     "description": "Interactive map of the Dark Pawns game world"
   }
   ```
3. Add NPC data to `page/data/npc.json`
4. Push to GitHub
5. Go to Settings → Pages → Source: GitHub Actions

### 4. Access Your Map
Visit: `https://your-username.github.io/darkpawns-map/`

## Integration with Dark Pawns Website

### Option A: Iframe Embed (Simplest)
```html
<iframe 
  src="https://your-username.github.io/darkpawns-map/"
  width="100%" 
  height="600px"
  frameborder="0"
  allowfullscreen>
</iframe>
```

### Option B: JavaScript Integration
```html
<!-- In your Dark Pawns website -->
<div id="darkpawns-map-container"></div>

<script type="module">
  import { MudletMapBrowser } from 'https://cdn.jsdelivr.net/npm/mudlet-map-browser-script/dist/index.min.js';
  
  const map = new MudletMapBrowser({
    container: document.getElementById('darkpawns-map-container'),
    mapUrl: 'https://your-username.github.io/darkpawns-map/map.dat',
    theme: 'dark',
    language: 'en'
  });
  
  map.load();
</script>
```

## Advanced Features

### Real-time Agent Tracking
```javascript
// Connect to Dark Pawns WebSocket
const ws = new WebSocket('wss://darkpawns-server.com/agents');

ws.onmessage = (event) => {
  const agentData = JSON.parse(event.data);
  
  // Update map with agent positions
  map.updateMarker('agent-' + agentData.id, {
    roomId: agentData.roomId,
    label: agentData.name,
    color: agentData.color,
    icon: 'person'
  });
};
```

### Quest Integration
```javascript
// Highlight quest locations
map.highlightRooms([1001, 1002, 1003], {
  color: '#ff9900',
  label: 'Active Quest: Find the Lost Artifact'
});

// Add click handlers for quest info
map.onRoomClick = (roomId) => {
  const room = getRoomData(roomId); // Your room data
  if (room.quests) {
    showQuestDialog(room.quests);
  }
};
```

## Data Structure

### Room Data Format
```json
{
  "id": 1001,
  "name": "Grand Hall",
  "coordinates": { "x": 0, "y": 0, "z": 0 },
  "areaId": 1,
  "environment": 3,
  "exits": {
    "north": 1002,
    "east": 1003
  },
  "metadata": {
    "description": "The main gathering place...",
    "npcs": ["Guard", "Merchant"],
    "quests": ["Starting Quest"],
    "danger": "low",
    "loot": ["Common Items"],
    "requires_key": false
  }
}
```

### NPC Data Format
```json
{
  "npcs": [
    {
      "id": "blacksmith",
      "name": "Grimbold the Blacksmith",
      "roomId": 1002,
      "description": "Master weaponsmith with 30 years experience",
      "quests": ["Repair Sword", "Find Rare Ore"],
      "shop": ["Iron Sword", "Leather Armor", "Repair Kit"]
    }
  ]
}
```

## Automation Scripts

### Map Update Script
```javascript
// update-map.js
const { MudletMapReader } = require('mudlet-map-binary-reader');
const fs = require('fs');

// Load existing map
const map = MudletMapReader.read('map.dat');

// Add new room from game data
const newRoom = {
  id: map.nextRoomId++,
  name: 'New Discovery',
  x: 10,
  y: 5,
  z: 0,
  areaId: 1,
  environment: 2
};

map.rooms[newRoom.id] = newRoom;

// Save updated map
MudletMapReader.write(map, 'map.dat');

// Commit and push to GitHub
// (Add Git automation here)
```

### CI/CD Pipeline (.github/workflows/deploy.yml)
```yaml
name: Deploy Map

on:
  push:
    branches: [main]
  workflow_dispatch:

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Generate Map
        run: |
          npm install mudlet-map-binary-reader
          node scripts/generate-map.js
          
      - name: Deploy to GitHub Pages
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./dist
```

## Performance Considerations

### For Large Maps:
1. **Area-based Loading:** Load only visible areas
2. **Level of Detail:** Simplify distant rooms
3. **Caching:** Cache map data locally
4. **Compression:** Use gzip for .dat files

### Optimization Tips:
```javascript
// Only load areas within viewport
map.setViewportBounds({
  minX: -100,
  maxX: 100,
  minY: -100,
  maxY: 100,
  z: 0
});

// Use web workers for pathfinding
const pathfinder = new Worker('pathfinder-worker.js');
```

## Testing

### Test Suite:
```javascript
// test-map.js
const assert = require('assert');
const { MudletMapReader } = require('mudlet-map-binary-reader');

describe('Dark Pawns Map', () => {
  it('should load successfully', () => {
    const map = MudletMapReader.read('map.dat');
    assert.ok(map.version === 20);
  });
  
  it('should have valid room connections', () => {
    const map = MudletMapReader.read('map.dat');
    
    for (const room of Object.values(map.rooms)) {
      // Check that exit destinations exist
      if (room.north !== -1) {
        assert.ok(map.rooms[room.north], `Room ${room.id} north exit invalid`);
      }
    }
  });
});
```

## Maintenance

### Regular Tasks:
1. **Backup Maps:** Daily backups of map.dat
2. **Version Control:** Tag map versions with game updates
3. **User Feedback:** Collect map usage analytics
4. **Bug Reports:** GitHub Issues template for map problems

### Update Schedule:
- **Weekly:** Minor fixes and NPC updates
- **Monthly:** New area additions
- **Quarterly:** Major map expansions

## Support Resources

1. **Mudlet Documentation:** https://wiki.mudlet.org/
2. **Template Issues:** https://github.com/Delwing/online-mudlet-map-template/issues
3. **Dark Pawns Discord:** #maps channel
4. **Example Maps:**
   - https://delwing.github.io/arkadia-mapa/
   - https://ire-mudlet-mapping.github.io/AchaeaCrowdmap/

## Success Metrics

- **Adoption Rate:** % of players using the map
- **Usage Time:** Average time spent on map
- **Feature Usage:** Which map features are most used
- **Bug Reports:** Number and severity of issues
- **Player Feedback:** Survey results

---

**Next Action:** Fork the template and generate your first map!

**Estimated Time:** 2 hours for basic map, 1 day for full integration

**Required Skills:** Basic Git, JavaScript, understanding of coordinate systems