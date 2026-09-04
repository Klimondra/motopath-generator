# 🏎️ MotoPath Generator (Vercel Serverless + PWA)

Progressive Web App (PWA) pro automatické generování uzavřených off-roadových okruhů (převážně po polních a lesních cestách) pro DIY motokáry a terénní stroje na základě aktuální GPS polohy uživatele.

Projekt je optimalizován pro nasazení na **Vercel** pomocí **Serverless Functions (Go Runtime)** a **Vercel Static Output** pro frontend.

---

## 🏗️ Architektura a Struktura Projektu

Aplikace využívá čistou architekturu s izolací doménové logiky od serverless handlerů.

```
motopath-generator/
├── api/
│   ├── route.go          # Vercel Serverless Function entry point (Handler) pro /api/route
│   └── route_test.go     # Testy pro API handler (GET/POST/OPTIONS/CORS)
├── internal/
│   ├── domain/
│   │   └── models.go     # Datové modely (GeoPoint, RouteRequest, RouteStats, OSM)
│   ├── overpass/
│   │   └── client.go     # OSM Overpass API klient s paměťovou cache a failoverem
│   └── routing/
│       ├── geo.go        # Sférická trigonometrie (Haversine, azimuty, cílové body)
│       ├── cost.go       # Váhování terénu (track 1.0x, penalizace asfaltu 18x–50x)
│       ├── graph.go      # In-memory orientovaný graf a BFS analýza dostupnosti
│       ├── dijkstra.go   # Dijkstrův algoritmus s dynamickou penalizací navštívených hran
│       ├── circuit.go    # Algoritmus pro generování uzavřených okruhů
│       └── routing_test.go # Unit testy pro routing
├── public/               # Statický frontend (Vercel servíruje automaticky na rootu /)
│   ├── index.html        # Mobile-first HTML rozhraní s Tailwind CSS CDN
│   ├── app.js            # Alpine.js reaktivita, Leaflet mapa a lokální GPX export
│   ├── manifest.json     # PWA Manifest
│   ├── sw.js             # PWA Service Worker (offline cache)
│   └── icon.svg          # Ikona aplikace
├── vercel.json           # Vercel konfigurace a URL rewrites
├── go.mod                # Go modul
└── README.md
```

---

## 🚀 Nasazení na Vercel

### 1. Nasazení přes Vercel CLI
```bash
# Instalace Vercel CLI (pokud nemáte)
npm i -g vercel

# Lokální vývoj a testování (spustí serverless funkce i statický frontend)
vercel dev

# Přímé nasazení do produkce
vercel --prod
```

### 2. Nasazení přes Git (GitHub / GitLab / Bitbucket)
1. Pushněte repozitář na GitHub.
2. V administraci Vercelu zvolte **"Add New Project"** -> **"Import Git Repository"**.
3. Vercel automaticky detekuje statický obsah v `public/` a Go serverless funkci v `api/route.go`.
4. Klikněte na **Deploy**.

---

## 🧪 Lokální Testování a Spuštění Unit Testů

Pro spuštění všech unit testů v Go:
```bash
go test -v ./...
```

---

## 📡 API Endpointy

### `POST /api/route`
Vygeneruje uzavřený okruh.

**Request Payload:**
```json
{
  "start_lat": 49.939,
  "start_lon": 14.188,
  "target_dist_km": 10.0,
  "strictness": "balanced",
  "seed": 12345
}
```
* `strictness`: `"hardcore"` (pouze polní/lesní cesty), `"balanced"` (vyvážený režim), `"relaxed"` (mix cest).
* `seed`: volitelný integer pro náhodné/deterministické varianty tvaru okruhu.

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

### `GET /api/route` nebo `GET /api/health`
Vrací stav zdraví API:
```json
{
  "status": "ok",
  "service": "motopath-generator",
  "time": "2026-09-04T22:15:00Z"
}
```
