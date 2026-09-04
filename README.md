# 🏎️ MotoPath Generator (Vercel + PWA)

Progressive Web App (PWA) pro automatické generování uzavřených off-roadových okruhů (převážně po polních a lesních cestách) pro DIY motokáry a terénní stroje na základě aktuální GPS polohy uživatele.

Projekt je optimalizován pro nasazení na **Vercel** pomocí **Vercel Go Framework** a **Vercel Static Output** pro frontend.

---

## 🏗️ Struktura Projektu

Aplikace využívá čistou architekturu s izolací doménové logiky od serverless/API handlerů.

```
motopath-generator/
├── main.go               # Vercel Go Framework entrypoint (spouští API server na os.Getenv("PORT"))
├── api/
│   ├── route.go          # API Handler pro /api/route (POST/GET/OPTIONS)
│   └── route_test.go     # Testy pro API handler
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
├── vercel.json           # Vercel konfigurace (zero-config)
├── go.mod                # Go modul
└── README.md
```

---

## 🚀 Nasazení na Vercel

1. Pushněte změny na GitHub:
```bash
git add .
git commit -m "Fix Vercel Go framework entrypoint"
git push origin main
```
2. Vercel automaticky zdetekuje Go projekt pomocí [`main.go`](file:///mnt/data_d/PrivateDokumenty/GoProjekty/motopath-generator/main.go) a statické soubory v [`public/`](file:///mnt/data_d/PrivateDokumenty/GoProjekty/motopath-generator/public).

---

## 🧪 Lokální Vývoj a Testování

```bash
# Spuštění lokálního serveru
go run main.go

# Spuštění všech testů
go test -v ./...
```
