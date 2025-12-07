document.addEventListener('DOMContentLoaded', () => {
  // --- CONFIGURATION & CONSTANTS ---
  const CONFIG = {
    tileSize: 256,
    minZoom: 0,
    extraZoom: 3,
    baseBounds: [[0, 0], [-256, 256]],
    maxBounds: [[100, -100], [-356, 356]],
    errorTile: 'data:image/webp;base64,UklGRkIAAABXRUJQVlA4WAoAAAAQAAAAAAAAAAAAQUxQSAgAAAAAAFYAMQAAQUYDNQAA'
  };

  const DOM = {
    map: document.getElementById('map'),
    // New Select Elements
    selectContainer: document.getElementById('mapSelect'),
    selectTrigger: document.getElementById('selectTrigger'),
    selectText: document.getElementById('selectedText'),
    selectOptions: document.getElementById('selectOptions'),

    steamLink: document.getElementById('steamLink'),
    btnTop: document.getElementById('btn-top'),
    btnSat: document.getElementById('btn-sat'),
    btnLoc: document.getElementById('btn-loc'),
    valGame: document.getElementById('val-game'),
    valWorld: document.getElementById('val-world')
  };

  // --- STATE MANAGEMENT ---
  const state = {
    maps: {},
    currentMap: null,
    selectedMapName: null,
    layerType: 'topographic',
    showLocations: true,
    tileLayer: null,
    locationsLayer: L.layerGroup()
  };

  // --- MAP INITIALIZATION ---
  const map = L.map(DOM.map, {
    crs: L.CRS.Simple,
    minZoom: CONFIG.minZoom,
    maxBounds: CONFIG.maxBounds,
    maxBoundsViscosity: 0.8,
    zoomControl: false,
    attributionControl: false
  });

  L.control.zoom({ position: 'topleft' }).addTo(map);
  L.control.attribution({ position: 'bottomright', prefix: false }).addTo(map);
  state.locationsLayer.addTo(map);

  // Size Control
  const SizeControl = L.Control.extend({
    options: { position: 'bottomleft' },
    onAdd: function () {
      this._div = L.DomUtil.create('div', 'map-size-control');
      this._div.style.display = 'none';
      return this._div;
    },
    update: function (text) {
      if (!this._div) return;
      this._div.innerHTML = text || '';
      this._div.style.display = text ? 'block' : 'none';
    }
  });
  const sizeControl = new SizeControl();
  map.addControl(sizeControl);

  // --- MATH HELPERS ---
  const MathUtils = {
    gameToWorld: (gameX, gameZ, mapSize) => {
      if (!mapSize) return { lat: 0, lon: 0 };
      const PI = Math.PI;
      const lon = (gameX * (360.0 / mapSize)) - 180.0;
      const mercatorY = (gameZ * ((2.0 * PI) / mapSize)) - PI;
      const latRad = (2.0 * Math.atan(Math.exp(mercatorY))) - (PI * 0.5);
      return { lat: latRad * (180.0 / PI), lon };
    },

    worldToLeaflet: (lon, lat, mapSize) => {
      if (!mapSize) return [0, 0];
      const PI = Math.PI;
      const gameX = (lon + 180.0) / (360.0 / mapSize);
      const latRad = lat * (PI / 180.0);
      const mercatorY = Math.log(Math.tan((PI * 0.25) + (latRad * 0.5)));
      const gameZ = (mercatorY + PI) / ((2.0 * PI) / mapSize);
      const ratio = mapSize / 256;
      return [(gameZ / ratio) - 256, gameX / ratio];
    },

    leafletToGame: (latlng, mapSize) => {
      if (!mapSize) return { x: 0, z: 0 };
      const ratio = mapSize / 256;
      let x = latlng.lng * ratio;
      let z = (256 + latlng.lat) * ratio;
      x = Math.max(0, Math.min(mapSize, x));
      z = Math.max(0, Math.min(mapSize, z));
      return { x, z };
    }
  };

  // --- CORE LOGIC ---

  function loadActiveMap(updateHistory = true) {
    const mapName = state.selectedMapName;
    if (!mapName || !state.maps[mapName]) return;

    const config = state.maps[mapName];
    state.currentMap = config;

    // 1. Update UI Selector Text
    DOM.selectText.textContent = formatMapName(config.name);

    // Highlight active option in custom select
    document.querySelectorAll('.option').forEach(opt => {
      opt.classList.toggle('selected', opt.dataset.value === mapName);
    });

    // 2. Validate & Switch Layers
    validateLayerAvailability(config);

    // 3. UI Info Updates
    updateUI(config);
    updateURL(mapName, state.layerType, updateHistory);

    // 4. Load Tile Layer
    const actualLimit = config.zoom || 8;
    const displayLimit = actualLimit + CONFIG.extraZoom;
    map.setMaxZoom(displayLimit);

    if (state.tileLayer) map.removeLayer(state.tileLayer);

    const url = `/maps/${mapName}/${state.layerType}/{z}/{x}/{y}.webp`;
    state.tileLayer = L.tileLayer(url, {
      tileSize: CONFIG.tileSize,
      noWrap: true,
      tms: false,
      minZoom: CONFIG.minZoom,
      maxNativeZoom: actualLimit,
      maxZoom: displayLimit,
      bounds: CONFIG.baseBounds,
      attribution: config.attribution,
      errorTileUrl: CONFIG.errorTile
    }).addTo(map);

    if (!state.tileLayer._url_changed_only) {
      map.fitBounds(CONFIG.baseBounds);
    }
    updateZoomVisibility();

    // 5. Load Locations
    handleLocations(mapName, config);
  }

  function validateLayerAvailability(config) {
    const hasTopo = !config.no_topographic;
    const hasSat = !config.no_satellite;

    DOM.btnTop.classList.toggle('disabled', !hasTopo);
    DOM.btnSat.classList.toggle('disabled', !hasSat);

    if (state.layerType === 'topographic' && !hasTopo && hasSat) {
      state.layerType = 'satellite';
    } else if (state.layerType === 'satellite' && !hasSat && hasTopo) {
      state.layerType = 'topographic';
    }

    DOM.btnTop.classList.toggle('active', state.layerType === 'topographic');
    DOM.btnSat.classList.toggle('active', state.layerType === 'satellite');
  }

  function handleLocations(mapName, config) {
    state.locationsLayer.clearLayers();
    DOM.btnLoc.classList.remove('disabled');

    if (!config.size || config.size <= 0) {
      disableLocations();
      return;
    }

    fetch(`/maps/${mapName}/locations.geojson`)
      .then(res => {
        if (!res.ok) throw new Error("No locations");
        return res.json();
      })
      .then(geojson => {
        if (!geojson.features || geojson.features.length === 0) {
          disableLocations();
          return;
        }

        geojson.features.forEach(feature => {
          const [lon, lat] = feature.geometry.coordinates;
          const latlng = MathUtils.worldToLeaflet(lon, lat, config.size);
          const type = feature.properties.type || 'village';
          const name = feature.properties.name;

          const icon = L.divIcon({
            className: `map-label type-${type}`,
            html: `<span>${name}</span>`,
            iconSize: [200, 30],
            iconAnchor: [100, 15]
          });

          L.marker(latlng, { icon: icon, interactive: false }).addTo(state.locationsLayer);
        });

        if (state.showLocations) {
          if (!map.hasLayer(state.locationsLayer)) map.addLayer(state.locationsLayer);
          DOM.btnLoc.classList.add('active');
        } else {
          if (map.hasLayer(state.locationsLayer)) map.removeLayer(state.locationsLayer);
          DOM.btnLoc.classList.remove('active');
        }
      })
      .catch(() => disableLocations());
  }

  function disableLocations() {
    state.locationsLayer.clearLayers();
    DOM.btnLoc.classList.remove('active');
    DOM.btnLoc.classList.add('disabled');
  }

  function updateUI(config) {
    const steamUrl = getSteamUrl(config.id);
    if (steamUrl) {
      DOM.steamLink.href = steamUrl;
      DOM.steamLink.classList.remove('disabled');
    } else {
      DOM.steamLink.removeAttribute('href');
      DOM.steamLink.classList.add('disabled');
    }

    if (config.size && config.size > 0) {
      const km = Math.round(config.size / 1000);
      const area = Math.round((config.size / 1000) ** 2);
      sizeControl.update(`${km}x${km}km (${area}km²)`);
    } else {
      sizeControl.update('');
    }
  }

  function updateURL(mapName, layerType, doPushState) {
    const path = `/${mapName}/${layerType}`;
    if (doPushState && window.location.pathname !== path) {
      window.history.pushState({ map: mapName, type: layerType }, "", path);
    }
  }

  function updateZoomVisibility() {
    const zoom = Math.floor(map.getZoom());
    const container = map.getContainer();
    container.className = container.className.replace(/\bview-level-\d+\b/g, '');
    const level = Math.min(zoom, 4);
    container.classList.add(`view-level-${level}`);
  }

  function getSteamUrl(id) {
    if (!id) return null;
    return id < 100000000
      ? `https://store.steampowered.com/app/${id}/`
      : `https://steamcommunity.com/sharedfiles/filedetails/?id=${id}`;
  }

  function formatMapName(name) {
    return name.charAt(0).toUpperCase() + name.slice(1);
  }

  // --- CUSTOM SELECT LOGIC ---

  function toggleSelect() {
    DOM.selectContainer.classList.toggle('open');
  }

  function closeSelect() {
    DOM.selectContainer.classList.remove('open');
  }

  function renderMapOptions(data) {
    DOM.selectOptions.innerHTML = '';

    if (!data || data.length === 0) {
      DOM.selectText.textContent = "No Maps";
      return;
    }

    data.forEach(m => {
      const el = document.createElement('div');
      el.className = 'option';
      el.dataset.value = m.name;
      el.textContent = formatMapName(m.name);

      if (m.index !== undefined && m.index !== null) {
        el.classList.add('primary');
      }

      el.addEventListener('click', (e) => {
        e.stopPropagation(); // Prevents bubbling to container click
        if (state.selectedMapName !== m.name) {
          state.selectedMapName = m.name;
          loadActiveMap(true);
        }
        closeSelect();
      });

      DOM.selectOptions.appendChild(el);
    });
  }

  // Event Listener for clicking outside
  document.addEventListener('click', (e) => {
    if (!DOM.selectContainer.contains(e.target)) {
      closeSelect();
    }
  });

  DOM.selectTrigger.addEventListener('click', toggleSelect);

  // --- GLOBAL EXPORTS ---

  window.setLayerType = (type) => {
    if (state.currentMap) {
      if (type === 'topographic' && state.currentMap.no_topographic) return;
      if (type === 'satellite' && state.currentMap.no_satellite) return;
    }
    state.layerType = type;
    loadActiveMap(true);
  };

  window.toggleLocations = () => {
    if (DOM.btnLoc.classList.contains('disabled')) return;
    state.showLocations = !state.showLocations;
    DOM.btnLoc.classList.toggle('active', state.showLocations);
    if (state.showLocations) map.addLayer(state.locationsLayer);
    else map.removeLayer(state.locationsLayer);
  };

  // --- MAP EVENTS ---

  map.on('zoomend', updateZoomVisibility);

  map.on('mousemove', (e) => {
    if (state.currentMap && state.currentMap.size) {
      const g = MathUtils.leafletToGame(e.latlng, state.currentMap.size);
      const w = MathUtils.gameToWorld(g.x, g.z, state.currentMap.size);
      DOM.valGame.textContent = `<${g.x.toFixed(1)}, 0, ${g.z.toFixed(1)}>`;
      DOM.valWorld.textContent = `[${w.lon.toFixed(6)}, ${w.lat.toFixed(6)}]`;
    } else {
      DOM.valGame.textContent = "N/A";
      DOM.valWorld.textContent = "N/A";
    }
  });

  map.on('click', (e) => {
    if (!state.currentMap || !state.currentMap.size) return;
    const g = MathUtils.leafletToGame(e.latlng, state.currentMap.size);
    const w = MathUtils.gameToWorld(g.x, g.z, state.currentMap.size);
    const mapName = formatMapName(state.currentMap.name);

    const textToCopy = `<${g.x.toFixed(2)}, 0, ${g.z.toFixed(2)}>\n[${w.lon.toFixed(6)}, ${w.lat.toFixed(6)}]\n${mapName}`;

    navigator.clipboard.writeText(textToCopy).then(() => {
      const popup = L.popup({
        className: 'copied-popup',
        closeButton: false,
        autoClose: true,
        closeOnClick: true
      }).setLatLng(e.latlng).setContent("Copied!").openOn(map);
      setTimeout(() => map.closePopup(), 2000);
    });
  });

  window.addEventListener('popstate', (event) => {
    if (event.state && event.state.map) {
      state.selectedMapName = event.state.map;
      if (event.state.type) state.layerType = event.state.type;
      loadActiveMap(false);
    } else {
      parseUrlAndLoad();
    }
  });

  // --- INITIALIZATION ---

  function parseUrlAndLoad() {
    const parts = window.location.pathname.split('/').filter(p => p.length > 0);
    let mapToLoad = Object.keys(state.maps)[0]; // Default first map

    if (parts.length > 0 && state.maps[parts[0].toLowerCase()]) {
      mapToLoad = parts[0].toLowerCase();
    }

    if (parts.length > 1) {
      const t = parts[1].toLowerCase();
      if (t === 'satellite' || t === 'topographic') state.layerType = t;
    }

    if (mapToLoad) {
      state.selectedMapName = mapToLoad;
      loadActiveMap(false);
    }
  }

  // Fetch Config
  fetch('/api/maps')
    .then(res => res.json())
    .then(data => {
      state.maps = {};
      if (!data || data.length === 0) return;

      data.forEach(m => state.maps[m.name] = m);

      // Init Custom Select
      renderMapOptions(data);

      parseUrlAndLoad();
    })
    .catch(e => console.error("Error loading maps:", e));
});
