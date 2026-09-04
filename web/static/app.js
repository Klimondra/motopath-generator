document.addEventListener('alpine:init', () => {
  Alpine.data('motopathApp', () => ({
    // State
    userLat: 50.0875,
    userLon: 14.4214,
    hasGpsFix: false,
    targetDistKm: 10,
    strictness: 'balanced',
    seed: Math.floor(Math.random() * 1000000),
    loading: false,
    loadingStep: '',
    error: null,
    routeData: null,
    activeLayer: 'osm',
    outdoorMode: false,
    drawerOpen: true,

    // Leaflet references
    map: null,
    tileLayers: {},
    routeLayerGroup: null,
    startMarker: null,

    init() {
      this.initMap();
      this.registerServiceWorker();
      // Auto-locate on mobile
      this.locateMe(false);
    },

    initMap() {
      // Create map
      this.map = L.map('map', {
        center: [this.userLat, this.userLon],
        zoom: 13,
        zoomControl: false,
      });

      // Custom Zoom Control top-right
      L.control.zoom({ position: 'topright' }).addTo(this.map);

      // Tile layers
      this.tileLayers.osm = L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        attribution: '&copy; OpenStreetMap contributors'
      });

      this.tileLayers.topo = L.tileLayer('https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png', {
        maxZoom: 17,
        attribution: 'Map data: &copy; OpenStreetMap contributors, SRTM | Style: &copy; OpenTopoMap (CC-BY-SA)'
      });

      this.tileLayers.sat = L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}', {
        maxZoom: 18,
        attribution: 'Tiles &copy; Esri &mdash; Source: Esri, i-cubed, USDA, USGS, AEX, GeoEye, Getmapping, Aerogrid, IGN, IGP, UPR-EGP, and the GIS User Community'
      });

      // Set default layer
      this.tileLayers.osm.addTo(this.map);

      // Route layers group
      this.routeLayerGroup = L.layerGroup().addTo(this.map);

      // Create start marker
      this.updateStartMarker();

      // Click on map to pick custom start point
      this.map.on('click', (e) => {
        this.userLat = parseFloat(e.latlng.lat.toFixed(6));
        this.userLon = parseFloat(e.latlng.lng.toFixed(6));
        this.updateStartMarker();
      });
    },

    setLayer(layerKey) {
      if (this.tileLayers[this.activeLayer]) {
        this.map.removeLayer(this.tileLayers[this.activeLayer]);
      }
      this.activeLayer = layerKey;
      if (this.tileLayers[layerKey]) {
        this.tileLayers[layerKey].addTo(this.map);
      }
    },

    updateStartMarker() {
      if (this.startMarker) {
        this.startMarker.setLatLng([this.userLat, this.userLon]);
      } else {
        const kartIcon = L.divIcon({
          className: 'custom-start-marker',
          html: `
            <div class="relative flex items-center justify-center">
              <div class="absolute w-8 h-8 bg-orange-500 rounded-full animate-ping opacity-75"></div>
              <div class="w-8 h-8 bg-orange-600 border-2 border-white rounded-full shadow-lg flex items-center justify-center text-white font-bold text-xs">
                🏁
              </div>
            </div>
          `,
          iconSize: [32, 32],
          iconAnchor: [16, 16],
        });

        this.startMarker = L.marker([this.userLat, this.userLon], {
          icon: kartIcon,
          draggable: true,
        }).addTo(this.map);

        this.startMarker.on('dragend', (e) => {
          const pos = e.target.getLatLng();
          this.userLat = parseFloat(pos.lat.toFixed(6));
          this.userLon = parseFloat(pos.lng.toFixed(6));
        });
      }
    },

    locateMe(userInitiated = true) {
      if (!navigator.geolocation) {
        if (userInitiated) alert('Geolokace není vaším prohlížečem podporována.');
        return;
      }

      this.loading = true;
      this.loadingStep = 'Zaměřuji vaši polohu přes GPS...';

      navigator.geolocation.getCurrentPosition(
        (pos) => {
          this.userLat = parseFloat(pos.coords.latitude.toFixed(6));
          this.userLon = parseFloat(pos.coords.longitude.toFixed(6));
          this.hasGpsFix = true;
          this.loading = false;
          this.updateStartMarker();
          this.map.flyTo([this.userLat, this.userLon], 14, { duration: 1.2 });
        },
        (err) => {
          this.loading = false;
          console.warn('Geolocation error:', err.message);
          if (userInitiated) {
            alert('Nepodařilo se získat přesnou polohu GPS. Můžete kliknout do mapy pro ruční výběr startu.');
          }
        },
        {
          enableHighAccuracy: true,
          timeout: 10000,
          maximumAge: 30000,
        }
      );
    },

    async generateRoute(newSeed = false) {
      if (newSeed) {
        this.seed = Math.floor(Math.random() * 1000000);
      }

      this.loading = true;
      this.error = null;
      this.loadingStep = '1/3 Stahuji polní cesty z OpenStreetMap...';

      // Progressive status steps simulation for nice UX
      const stepTimer1 = setTimeout(() => {
        if (this.loading) this.loadingStep = '2/3 Analyzuji síť a stavím graf tras...';
      }, 1500);

      const stepTimer2 = setTimeout(() => {
        if (this.loading) this.loadingStep = '3/3 Počítám optimální uzavřený okruh (Dijkstra)...';
      }, 3000);

      try {
        const response = await fetch('/api/route', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            start_lat: this.userLat,
            start_lon: this.userLon,
            target_dist_km: parseFloat(this.targetDistKm),
            strictness: this.strictness,
            seed: this.seed,
          }),
        });

        clearTimeout(stepTimer1);
        clearTimeout(stepTimer2);

        const data = await response.json();

        if (!response.ok || !data.success) {
          throw new Error(data.error || 'Generování okruhu selhalo');
        }

        this.routeData = data;
        this.renderRoute(data);
        // On mobile, minimize drawer slightly so map is visible
        if (window.innerWidth < 768) {
          this.drawerOpen = false;
        }
      } catch (err) {
        this.error = err.message || 'Chyba při komunikaci se serverem';
        console.error('Route generation failed:', err);
      } finally {
        this.loading = false;
      }
    },

    renderRoute(data) {
      this.routeLayerGroup.clearLayers();

      if (!data.points || data.points.length === 0) return;

      const latlngs = data.points.map((p) => [p.lat, p.lon]);

      // Draw background glow polyline for outdoor high visibility
      L.polyline(latlngs, {
        color: this.outdoorMode ? '#000000' : '#1e293b',
        weight: 9,
        opacity: 0.9,
        lineCap: 'round',
        lineJoin: 'round',
      }).addTo(this.routeLayerGroup);

      // Draw main neon orange/amber off-road route
      const mainPolyline = L.polyline(latlngs, {
        color: this.outdoorMode ? '#ffea00' : '#f97316',
        weight: 5,
        opacity: 1.0,
        dashArray: null,
        lineCap: 'round',
        lineJoin: 'round',
      }).addTo(this.routeLayerGroup);

      // Add waypoints & start/end markers
      const startPt = latlngs[0];
      const startIcon = L.divIcon({
        className: 'start-finish-badge',
        html: `<div class="px-2 py-1 bg-green-600 text-white font-black text-xs rounded-full shadow-lg border border-white whitespace-nowrap">🚩 START / CÍL</div>`,
        iconSize: [80, 24],
        iconAnchor: [40, 28],
      });
      L.marker(startPt, { icon: startIcon }).addTo(this.routeLayerGroup);

      // Fit map bounds to show complete circuit with comfortable padding
      this.map.fitBounds(mainPolyline.getBounds(), {
        padding: [50, 50],
        maxZoom: 16,
      });
    },

    downloadGPX() {
      if (!this.routeData || !this.routeData.points || this.routeData.points.length === 0) {
        alert('Nejprve vygenerujte trasu.');
        return;
      }

      const points = this.routeData.points;
      const stats = this.routeData.stats;
      const nowISO = new Date().toISOString();
      const distKm = stats ? stats.total_distance_km.toFixed(1) : this.targetDistKm;
      const offroadPct = stats ? stats.offroad_percent.toFixed(0) : '0';

      let gpx = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="MotoPath Generator (https://github.com/motopath)" xmlns="http://www.topografix.com/GPX/1/1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.topografix.com/GPX/1/1 http://www.topografix.com/GPX/1/1/gpx.xsd">
  <metadata>
    <name>MotoPath Off-Road Okruh ${distKm} km (${offroadPct}% terén)</name>
    <desc>Vygenerováno v MotoPath aplikaci pro motokáry a terénní jízdu</desc>
    <time>${nowISO}</time>
  </metadata>
  <trk>
    <name>MotoPath Okruh ${distKm}km</name>
    <type>OffRoad_GoKart</type>
    <trkseg>
`;

      for (let i = 0; i < points.length; i++) {
        const pt = points[i];
        gpx += `      <trkpt lat="${pt.lat.toFixed(6)}" lon="${pt.lon.toFixed(6)}">\n`;
        gpx += `        <time>${nowISO}</time>\n`;
        gpx += `      </trkpt>\n`;
      }

      gpx += `    </trkseg>
  </trk>
</gpx>`;

      const blob = new Blob([gpx], { type: 'application/gpx+xml;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      const dateStr = new Date().toISOString().slice(0, 10);
      a.href = url;
      a.download = `motopath_okruh_${distKm}km_${dateStr}.gpx`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    },

    toggleOutdoorMode() {
      this.outdoorMode = !this.outdoorMode;
      if (this.routeData) {
        this.renderRoute(this.routeData);
      }
    },

    registerServiceWorker() {
      if ('serviceWorker' in navigator) {
        window.addEventListener('load', () => {
          navigator.serviceWorker.register('/sw.js').catch((err) => {
            console.log('[PWA] ServiceWorker registration failed:', err);
          });
        });
      }
    }
  }));
});
