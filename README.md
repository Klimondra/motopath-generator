# MotoPath Generator

![Google Gemini](https://img.shields.io/badge/google%20gemini-%238E75B2.svg?style=for-the-badge&logo=google%20gemini&logoColor=white)
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Alpine.js](https://img.shields.io/badge/alpinejs-%238BC0D0.svg?style=for-the-badge&logo=alpine.js&logoColor=white)
![Tailwind CSS](https://img.shields.io/badge/tailwindcss-%2338B2AC.svg?style=for-the-badge&logo=tailwind-css&logoColor=white)

Progressive Web App (PWA) pro automatické generování uzavřených off-roadových okruhů pro DIY motokáry. Aplikace na základě aktuální GPS polohy a dat z OpenStreetMap najde trasu převážně po polních a lesních cestách a umožní její export do formátu GPX pro snadnou navigaci.

*Tento projekt je **vibecoded**.* (Protože tohle neberu tak seriozně a chtěl jsem to mít rychle)

## Spuštění

**Požadavky:** Go 1.22+

Spuštění lokálního serveru:
```bash
go run ./cmd/server