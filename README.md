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

GitHub Pages publishes from `main` automatically. Push and the site updates:

```sh
git push
```

## License

LOLcense — same as tinyd.
