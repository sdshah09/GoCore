# Argo CD & GitOps — 5-Slide LinkedIn Carousel

A dev-native PDF carousel about Argo CD and GitOps. Follow-up to the "Why you need Helm" carousel.

## Slides

1. **Hook** — Your CI/CD is pushing directly to prod. Right now.
2. **Problem** — The Push Nightmare (configuration drift)
3. **Solution** — GitOps: Git as source of truth, Argo CD pulls
4. **Self-Healing** — kubectl edit → Argo CD reverts
5. **CTA** — Argo CD vs. Flux debate + P.S. Open-Source K8s Guide

## View in Browser

Open `index.html` in a browser. Use the buttons or arrow keys to navigate slides.

## Generate PDF

```bash
cd docs/argocd-carousel
npm install
node generate-pdf.js
# or: npm run pdf
```

Output:
- `argocd-gitops.pdf` — combined PDF for LinkedIn
- `slide-1.png` through `slide-5.png` — individual slide images
