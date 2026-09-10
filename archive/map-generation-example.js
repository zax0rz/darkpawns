// Dark Pawns Map Generation Example
// Using mudlet-map-binary-reader pattern

// This is a conceptual example showing how to generate a Mudlet map for Dark Pawns
// Actual implementation would require the mudlet-map-binary-reader npm package

class DarkPawnsMapGenerator {
  constructor() {
    this.map = {
      version: 20,
      areas: {},
      rooms: {},
      areaNames: {},
      labels: {},
      colors: {},
      environments: {}
    };
    
    this.nextRoomId = 1;
    this.nextAreaId = 1;
  }
  
  // Create a new area in the map
  createArea(name) {
    const areaId = this.nextAreaId++;
    this.map.areas[areaId] = {
      id: areaId,
      name: name,
      rooms: []
    };
    this.map.areaNames[areaId] = name;
    return areaId;
  }
  
  // Create a room in the map
  createRoom(name, x, y, z = 0, areaId = 1, environment = 1) {
    const roomId = this.nextRoomId++;
    
    this.map.rooms[roomId] = {
      id: roomId,
      name: name,
      x: x,
      y: y,
      z: z,
      areaId: areaId,
      environment: environment,
      north: -1,
      south: -1,
      east: -1,
      west: -1,
      up: -1,
      down: -1,
      northeast: -1,
      northwest: -1,
      southeast: -1,
      southwest: -1,
      in: -1,
      out: -1,
      mSpecialExits: {},
      mSpecialExitLocks: [],
      userData: {},
      weight: 1
    };
    
    // Add room to area
    if (this.map.areas[areaId]) {
      this.map.areas[areaId].rooms.push(roomId);
    }
    
    return roomId;
  }
  
  // Connect two rooms with a standard exit
  connectRooms(fromRoomId, toRoomId, direction) {
    const validDirections = [
      'north', 'south', 'east', 'west', 'up', 'down',
      'northeast', 'northwest', 'southeast', 'southwest',
      'in', 'out'
    ];
    
    if (!validDirections.includes(direction)) {
      throw new Error(`Invalid direction: ${direction}`);
    }
    
    if (this.map.rooms[fromRoomId] && this.map.rooms[toRoomId]) {
      this.map.rooms[fromRoomId][direction] = toRoomId;
      
      // Add reverse connection for bidirectional paths
      const reverseDirections = {
        north: 'south',
        south: 'north',
        east: 'west',
        west: 'east',
        up: 'down',
        down: 'up',
        northeast: 'southwest',
        southwest: 'northeast',
        northwest: 'southeast',
        southeast: 'northwest',
        in: 'out',
        out: 'in'
      };
      
      if (reverseDirections[direction]) {
        this.map.rooms[toRoomId][reverseDirections[direction]] = fromRoomId;
      }
    }
  }
  
  // Connect rooms with a special exit (portal, door, etc.)
  connectSpecialExit(fromRoomId, toRoomId, exitName, locked = false) {
    if (this.map.rooms[fromRoomId] && this.map.rooms[toRoomId]) {
      this.map.rooms[fromRoomId].mSpecialExits[exitName] = toRoomId;
      
      if (locked) {
        this.map.rooms[fromRoomId].mSpecialExitLocks.push(toRoomId);
      }
    }
  }
  
  // Add user data to a room (quest info, NPCs, etc.)
  addRoomUserData(roomId, key, value) {
    if (this.map.rooms[roomId]) {
      this.map.rooms[roomId].userData[key] = value;
    }
  }
  
  // Generate a simple Dark Pawns starting area
  generateStartingArea() {
    console.log('Generating Dark Pawns starting area...');
    
    // Create the main area
    const darkPawnsArea = this.createArea('Dark Pawns World');
    
    // Create key locations
    const startingVillage = this.createRoom('Starting Village', 0, 0, 0, darkPawnsArea, 3);
    const villageSquare = this.createRoom('Village Square', 0, 1, 0, darkPawnsArea, 3);
    const blacksmith = this.createRoom('Blacksmith Shop', 1, 1, 0, darkPawnsArea, 3);
    const tavern = this.createRoom('Rusty Tankard Tavern', -1, 1, 0, darkPawnsArea, 3);
    const forestPath = this.createRoom('Forest Path', 0, 2, 0, darkPawnsArea, 2);
    const ancientRuins = this.createRoom('Ancient Ruins', 1, 3, 0, darkPawnsArea, 4);
    const darkCave = this.createRoom('Dark Cave Entrance', -1, 3, 0, darkPawnsArea, 5);
    
    // Connect rooms
    this.connectRooms(startingVillage, villageSquare, 'north');
    this.connectRooms(villageSquare, blacksmith, 'east');
    this.connectRooms(villageSquare, tavern, 'west');
    this.connectRooms(villageSquare, forestPath, 'north');
    this.connectRooms(forestPath, ancientRuins, 'east');
    this.connectRooms(forestPath, darkCave, 'west');
    
    // Add special exits (portals, secret doors, etc.)
    this.connectSpecialExit(ancientRuins, darkCave, 'shadow_portal', true);
    
    // Add user data (quest info, NPCs, etc.)
    this.addRoomUserData(startingVillage, 'description', 'A quiet village where new adventurers begin their journey.');
    this.addRoomUserData(startingVillage, 'npcs', ['Village Elder', 'Training Master']);
    this.addRoomUserData(startingVillage, 'quests', ['Starting Equipment', 'Learn Basics']);
    
    this.addRoomUserData(blacksmith, 'description', 'The sound of hammer on anvil fills the air. Weapons and armor line the walls.');
    this.addRoomUserData(blacksmith, 'npcs', ['Grimbold the Blacksmith']);
    this.addRoomUserData(blacksmith, 'shop', ['Iron Sword', 'Leather Armor', 'Repair Kit']);
    
    this.addRoomUserData(tavern, 'description', 'A rowdy tavern filled with adventurers sharing tales over ale.');
    this.addRoomUserData(tavern, 'npcs', ['Barkeep', 'Mysterious Stranger']);
    this.addRoomUserData(tavern, 'quests', ['Find Lost Keg', 'Deliver Message']);
    
    this.addRoomUserData(ancientRuins, 'description', 'Crumbling stone structures hint at a forgotten civilization.');
    this.addRoomUserData(ancientRuins, 'danger', 'medium');
    this.addRoomUserData(ancientRuins, 'loot', ['Ancient Artifact', 'Rune Stones']);
    
    this.addRoomUserData(darkCave, 'description', 'A foreboding entrance to the underground realms.');
    this.addRoomUserData(darkCave, 'danger', 'high');
    this.addRoomUserData(darkCave, 'warning', 'Recommended level: 5+');
    
    console.log(`Generated ${Object.keys(this.map.rooms).length} rooms in area "${this.map.areaNames[darkPawnsArea]}"`);
    
    return this.map;
  }
  
  // Export map to JSON (for debugging/visualization)
  exportToJSON() {
    return JSON.stringify(this.map, null, 2);
  }
  
  // Simulate saving to .dat file (would use mudlet-map-binary-reader in production)
  saveToDat(filename) {
    console.log(`[SIMULATED] Saving map to ${filename}`);
    console.log(`  Version: ${this.map.version}`);
    console.log(`  Areas: ${Object.keys(this.map.areas).length}`);
    console.log(`  Rooms: ${Object.keys(this.map.rooms).length}`);
    
    // In production: MudletMapReader.write(this.map, filename);
    return true;
  }
}

// Example usage
if (require.main === module) {
  const generator = new DarkPawnsMapGenerator();
  
  // Generate the map
  const map = generator.generateStartingArea();
  
  // Export to JSON for inspection
  const json = generator.exportToJSON();
  console.log('\nMap structure (first 2000 chars):');
  console.log(json.substring(0, 2000) + '...');
  
  // Simulate saving to .dat file
  generator.saveToDat('darkpawns-map.dat');
  
  console.log('\n=== MAP GENERATION COMPLETE ===');
  console.log('Next steps:');
  console.log('1. Install: npm install mudlet-map-binary-reader');
  console.log('2. Replace simulated save with: MudletMapReader.write(map, "darkpawns-map.dat")');
  console.log('3. Use template: https://github.com/Delwing/online-mudlet-map-template');
  console.log('4. Deploy to GitHub Pages');
}

module.exports = DarkPawnsMapGenerator;