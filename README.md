# tinyd-landing

Source for the landing page of [tinyd](https://github.com/jalonsogo/tinyd) — a tiny Docker Desktop in your terminal.

Live at **[jalonsogo.github.io/tinyd-landing](https://jalonsogo.github.io/tinyd-landing/)**.

## Stack

Plain HTML, CSS and vanilla JS — no build step, no dependencies.

- `index.html` — markup
- `styles.css` — all styling and the responsive breakpoints
- `script.js` — hero terminal animation, demo tabs, copy buttons, clock
- `logo/` — SVG and PNG assets

## Local preview

```sh
python3 -m http.server 8765
# then open http://localhost:8765
```

## Deploy

The source lives here (`jalonsogo/tinyd-landing`); the deployed site is mirrored
to the `gh-pages` branch of [`jalonsogo/tinyd`](https://github.com/jalonsogo/tinyd)
so the URL stays at `jalonsogo.github.io/tinyd`.

Both pushes are wrapped in `deploy.sh`:

```sh
./deploy.sh
```

That runs:

```sh
git push origin main              # source repo
git push tinyd main:gh-pages      # mirror used by GitHub Pages
```

Requires the `tinyd` remote to be configured locally
(`git remote add tinyd https://github.com/jalonsogo/tinyd.git`).

## License

LOLcense — same as tinyd.
