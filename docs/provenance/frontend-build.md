# Frontend build provenance

Implement F02 (#52): Vue 3/TypeScript/Vite, reproducible checked-in Go embed assets, same UI for served and offline reports. First prove a self-contained IIFE + CSS bundle can retain offline HTML with only packaging differences. No CDN, dynamic imports or separate report UI. Migrate progressively; legacy source remains only until components replace each area.

Evaluate production build/typecheck, Go report tests, actual embedded browser baseline, payload escaping and reproducible generated files. Runtime must work without Node.
