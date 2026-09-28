# OpsPilot console

Next.js app for the OpsPilot control plane. Run instructions live in the repository README.

```bash
npm install
npm run dev
```

The dev server uses port 3461 and proxies `/api` and `/health` to `API_PROXY_URL` (default `http://127.0.0.1:8094`).
