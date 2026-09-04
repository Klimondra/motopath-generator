# 🏎️ MotoPath Generator - PWA Generátor Off-Road Okruhů pro Motokáry

Progressive Web App (PWA) pro automatické generování uzavřených off-roadových okruhů (převážně po polních a lesních cestách) pro DIY motokáry a terénní stroje na základě aktuální GPS polohy uživatele.

---

## 🚀 Hlavní Vlastnosti

* **Čistý Go 1.22+ Backend:** Využívá výhradně standardní knihovnu `net/http` s novým routovacím systémem a čistou architekturou (doménová logika je izolovaná od HTTP handlerů).
* **Inteligentní Off-Road Routing:**
  * Stahuje OpenStreetMap data přes Overpass API (`track`, `path`, `unclassified`, `residential`, `tertiary`).
  * Penalizační váhová funkce preferuje nezpevněné polní/lesní cesty (`track` = 1.0x, asfaltové propojky s penalizací 18x–50x).
  * Generování uzavřených okruhů (Dijkstra) s dynamickou penalizací již navštívených hran (zabraňuje jízdě tam a zpět po stejné cestě a nutí algoritmus uzavřít plnohodnotný okruh).
* **Odlehčený Frontend bez Build Stepů:**
  * **Alpine.js 3** pro reaktivní správu stavu.
  * **Tailwind CSS** přes CDN pro moderní mobilní UI.
  * **Leaflet.js** pro interaktivní mapu s přepínáním vrstev (Běžná mapa, Topografická, Satelitní).
* **PWA & Terénní UX:**
  * Nainstalovatelné jako nativní aplikace na mobil (Service Worker + Manifest).
  * **Režim vysokého kontrastu (☀️ Slunce):** Zvýšený kontrast a neonové vykreslení pro dobrou čitelnost na přímém slunci.
  * **1-Click GPX Export:** Lokální generování standardního `.gpx` XML souboru pro přímý import do aplikací **Mapy.cz**, **OsmAnd**, **Garmin**, **GPX Viewer** atd.

---

## 📁 Struktura Projektu

```
motopath-generator/
├── cmd/
│   └── server/
│       └── main.go          # Vstupní bod, konfigurace, graceful shutdown
├── internal/
│   ├── domain/
│   │   └── models.go        # GeoPoint, RouteRequest, RouteResponse, Stats
│   ├── overpass/
│   │   └── client.go        # Klient pro Overpass API s cachováním a failoverem
│   ├── routing/
│   │   ├── geo.go           # Haversine vzdálenosti, azimuty, bounding box
│   │   ├── cost.go          # Váhové matice a kalkulace cen hran (track vs asfalt)
│   │   ├── graph.go         # In-memory orientovaný graf a BFS reachability
│   │   ├── dijkstra.go      # Dijkstrův algoritmus s dynamickými penalizacemi
│   │   ├── circuit.go       # Algoritmus pro generování uzavřených okruhů
│   │   └── routing_test.go  # Unit testy pro geo výpočty a Dijkstru
│   └── api/
│       ├── handler.go       # HTTP handlery (Go 1.22 mux)
│       └── middleware.go    # Logger, CORS, Recovery middleware
├── web/
│   └── static/
│       ├── index.html       # Mobile-first HTML rozhraní
│       ├── app.js           # Alpine.js logika, Leaflet provázání a GPX export
│       ├── manifest.json    # PWA Web App Manifest
│       ├── sw.js            # PWA Service Worker (offline cache)
│       └── icon.svg         # Vektorová ikona aplikace
├── go.mod
└── README.md
```

---

## 🛠️ Spuštění Aplikace

### Požadavky
* Go 1.22+

### Spuštění lokálního serveru
```bash
go run ./cmd/server
```

Aplikace poběží na adrese **http://localhost:8080** (nebo portu specifikovaném proměnnou `PORT=8080`).

### Spuštění testů
```bash
go test -v ./...
```

---

## 📡 API Endpointy

### `POST /api/route`
Vygeneruje uzavřený okruh.

**Request:**
```json
{
  "start_lat": 49.939,
  "start_lon": 14.188,
  "target_dist_km": 10.0,
  "strictness": "balanced",
  "seed": 12345
}
```
* `strictness`: `"hardcore"` (pouze terén), `"balanced"` (vyvážený), `"relaxed"` (mix cest).
* `seed`: volitelný seed pro deterministické nebo náhodné generování různých tvarů okruhu.

**Response:**
```json
{
  "success": true,
  "message": "Closed loop generated successfully",
  "points": [
    {"lat": 49.941037, "lon": 14.202249},
    ...
  ],
  "stats": {
    "total_distance_km": 9.85,
    "offroad_percent": 94.2,
    "paved_percent": 5.8,
    "estimated_time_min": 24,
    "highway_breakdown_m": {
      "track": 8500,
      "path": 800,
      "unclassified": 550
    }
  },
  "bounding_box": [49.93, 14.18, 49.95, 14.21]
}
```

### `GET /api/health`
Health check status.
