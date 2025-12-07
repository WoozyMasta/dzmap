<!DOCTYPE html>
<html lang="en">

<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>DayZ Web Maps</title>
  <link rel="icon" type="image/x-icon" href="/favicon.ico">
  <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
  <style>{{ .CSS }}</style>
</head>

<body>

  <div id="hud">
    <div class="custom-select" id="mapSelect">
      <button class="select-trigger" id="selectTrigger">
        <span class="map-icon">🗺️</span>
        <span class="selected-text" id="selectedText">Loading...</span>
        <span class="arrow">▼</span>
      </button>
      <div class="select-options" id="selectOptions"></div>
    </div>

    <a id="steamLink" href="#" target="_blank" class="steam-btn" title="Open in Steam">{{ .SVG }}</a>
    <div class="divider"></div>

    <div class="layer-switch">
      <button class="layer-btn active" onclick="setLayerType('topographic')" id="btn-top">Topo</button>
      <button class="layer-btn" onclick="setLayerType('satellite')" id="btn-sat">Sat</button>
      <div style="width:1px; background:var(--border); margin: 0 4px;"></div>
      <button class="layer-btn active" onclick="toggleLocations()" id="btn-loc">Loc</button>
    </div>
  </div>

  <div id="coords-panel">
    <div class="coord-row"><span class="coord-label">Game</span> <span id="val-game">0, 0, 0</span></div>
    <div class="coord-row"><span class="coord-label">World</span> <span id="val-world">[0, 0]</span></div>
  </div>

  <div id="map"></div>

  <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>

  <script>{{ .JS }}</script>
</body>

</html>
