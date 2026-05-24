/* ---------- helpers ---------- */
const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];

const colorize = {
  prompt: (s) => `<span class="c-prompt">${s}</span>`,
  cmd: (s) => `<span class="c-cmd">${s}</span>`,
  dim: (s) => `<span class="c-dim">${s}</span>`,
  green: (s) => `<span class="c-green">${s}</span>`,
  yellow: (s) => `<span class="c-yellow">${s}</span>`,
  red: (s) => `<span class="c-red">${s}</span>`,
  cyan: (s) => `<span class="c-cyan">${s}</span>`,
  magenta: (s) => `<span class="c-magenta">${s}</span>`,
  border: (s) => `<span class="c-border">${s}</span>`,
};

/* ---------- hero terminal animation ---------- */
const termPre = $("#term-pre");

const heroScript = [
  { type: "prompt", text: "$ ", delay: 0 },
  { type: "typing", text: "./tinyd", delay: 50 },
  { type: "newline" },
  { type: "wait", ms: 280 },
  { type: "block", html: buildHeroFrame() },
];

function pad(s, n) {
  s = String(s);
  if (s.length >= n) return s.slice(0, n);
  return s + " ".repeat(n - s.length);
}

function buildHeroFrame() {
  const FRAME_WIDTH = 68;
  const border = colorize.border;

  /* dot helpers */
  const dY = `<span class="c-yellow">●</span>`;
  const dR = `<span class="c-red">●</span>`;
  const dG = `<span class="c-green">●</span>`;
  const dO = `<span class="c-dim">○</span>`;

  /* row format: 2 + 1(dot) + 1 + 16(name) + 1 + 20(img) + 1 + 5(cpu) + 1 + 5(mem) + 1 + 12(ports) + 2 trailing = 68 */
  const padRight = (s, n) => (s.length >= n ? s.slice(0, n) : s + " ".repeat(n - s.length));
  const padLeft = (s, n) => (s.length >= n ? s.slice(0, n) : " ".repeat(n - s.length) + s);

  const fmt = (dot, name, img, cpu, mem, ports) => {
    const n = padRight(name, 16);
    const i = padRight(img, 20);
    const c = padLeft(cpu, 5);
    const m = padLeft(mem, 5);
    const p = padRight(ports, 12);
    return `  ${dot} ${n} ${i} ${c} ${m} ${p}  `;
  };

  /* --- tabs: top corners, label row, bottom corners that merge with the separator --- */
  /* active tab box is 14 chars wide (cols 2..15), label "Containers" is 10 chars padded with 1 space each side */
  const tabTop =
    "  " +
    border("┌────────────┐") +
    " ".repeat(FRAME_WIDTH - 16);
  const tabRow =
    "  " +
    border("│") +
    ' <span class="c-tab-active">Containers</span> ' +
    border("│") +
    "  " +
    colorize.dim("Images  Models  Volumes  Networks") +
    " ".repeat(Math.max(1, FRAME_WIDTH - 50));
  /* merge row sits under the tabs */
  const tabMerge =
    "  " +
    border("┘") +
    " ".repeat(12) +
    border("└") +
    border("─".repeat(FRAME_WIDTH - 17));

  const headersText = padRight("  NAME             IMAGE                CPU   MEM   PORTS", FRAME_WIDTH);
  const headers = colorize.dim(headersText);

  const rows = [
    { sel: true, dot: dY, name: "focused_jemison", img: "mlinarik/min…",      cpu: "--",    mem: "--",   ports: "19132/udp" },
    {            dot: dR, name: "infallible_sam…", img: "payments-service",   cpu: "--",    mem: "--",   ports: "—" },
    {            dot: dO, name: "user-api-demo",   img: "python:3.12-slim",   cpu: "--",    mem: "--",   ports: "8000→8000" },
    {            dot: dO, name: "redis-demo",      img: "redis:7-alpine",     cpu: "--",    mem: "--",   ports: "6379→6379" },
    {            dot: dG, name: "nginx-proxy",     img: "nginx:latest",       cpu: "2.3%",  mem: "128M", ports: "80→8080" },
    {            dot: dG, name: "api-server",      img: "node:18-alpine",     cpu: "15.1%", mem: "512M", ports: "3000→3000" },
  ];

  const rowLines = rows
    .map((r) => {
      const inner = fmt(r.dot, r.name, r.img, r.cpu, r.mem, r.ports);
      return r.sel ? `<span class="c-bg-sel">${inner}</span>` : inner;
    })
    .join("\n");

  const pagerLeft = "  Showing 1-6 of 6";
  const pagerRight = "F1 help · q quit  ";
  const pager = colorize.dim(pagerLeft) + " ".repeat(Math.max(1, FRAME_WIDTH - pagerLeft.length - pagerRight.length)) + colorize.dim(pagerRight);

  return [tabTop, tabRow, tabMerge, headers, rowLines, pager].join("\n");
}

async function typeOut() {
  if (!termPre) return;
  let out = "";

  for (const step of heroScript) {
    if (step.type === "prompt") {
      out += colorize.prompt(step.text);
      termPre.innerHTML = out + '<span class="blink"></span>';
    } else if (step.type === "typing") {
      for (const ch of step.text) {
        out += colorize.cmd(ch);
        termPre.innerHTML = out + '<span class="blink"></span>';
        await sleep(step.delay);
      }
    } else if (step.type === "newline") {
      out += "\n";
      termPre.innerHTML = out + '<span class="blink"></span>';
    } else if (step.type === "wait") {
      await sleep(step.ms);
    } else if (step.type === "block") {
      out += step.html;
      termPre.innerHTML = out;
      const mascot = document.querySelector(".hero-mascot");
      if (mascot) {
        await sleep(180);
        mascot.classList.add("is-visible");
      }
    }
  }
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

/* ---------- demo data ---------- */
/* dot states: yellow = warming/recent, red = stopped, hollow = idle, green = healthy */
const demoData = {
  containers: {
    headers: ["", "NAME", "IMAGE", "CPU", "MEM", "PORTS"],
    align: ["", "l", "l", "r", "r", "l"],
    rows: [
      ["yellow", "focused_jemison", "mlinarik/minecraft-bedrock-server", "--", "--", "19132/udp"],
      ["red", "infallible_sammet", "payments-service:2.4.1", "--", "--", "—"],
      ["hollow", "user-api-demo", "python:3.12-slim", "--", "--", "8000→8000"],
      ["hollow", "redis-demo", "redis:7-alpine", "--", "--", "6379→6379"],
      ["green", "nginx-proxy", "nginx:latest", "2.3%", "128MB", "80→8080, 443→8443"],
      ["green", "api-server", "node:18-alpine", "15.1%", "512MB", "3000→3000"],
    ],
    selected: 0,
  },
  images: {
    headers: ["", "REPOSITORY", "TAG", "SIZE", "CREATED"],
    align: ["", "l", "l", "r", "l"],
    rows: [
      ["green", "node", "18-alpine", "1.2GB", "2d ago"],
      ["green", "nginx", "latest", "142MB", "5d ago"],
      ["green", "postgres", "15", "412MB", "1w ago"],
      ["yellow", "<none>", "<none>", "318MB", "2w ago"],
      ["hollow", "redis", "7-alpine", "32MB", "3w ago"],
      ["hollow", "alpine", "edge", "5MB", "1mo ago"],
    ],
    selected: 0,
  },
  models: {
    headers: ["", "NAME", "FAMILY", "PARAMETERS", "QUANT", "SIZE"],
    align: ["", "l", "l", "r", "l", "r"],
    rows: [
      ["green", "llama3.1:latest", "llama", "8B", "Q4_K_M", "4.7GB"],
      ["green", "qwen2.5-coder:7b", "qwen2", "7B", "Q4_0", "4.2GB"],
      ["hollow", "phi3:mini", "phi3", "3.8B", "Q4_K_M", "2.3GB"],
      ["hollow", "mistral:7b", "mistral", "7B", "Q4_0", "4.1GB"],
    ],
    selected: 0,
  },
  volumes: {
    headers: ["", "NAME", "DRIVER", "ATTACHED TO", "CREATED"],
    align: ["", "l", "l", "l", "l"],
    rows: [
      ["green", "app-data", "local", "nginx-proxy, api-server", "2d ago"],
      ["green", "postgres-vol", "local", "payments-service", "1w ago"],
      ["green", "redis-cache-data", "local", "redis-demo", "1w ago"],
      ["hollow", "tmp-backup", "local", "—", "3w ago"],
    ],
    selected: 0,
  },
  networks: {
    headers: ["", "NAME", "DRIVER", "SUBNET", "SCOPE"],
    align: ["", "l", "l", "l", "l"],
    rows: [
      ["green", "bridge", "bridge", "172.17.0.0/16", "local"],
      ["green", "app-network", "bridge", "172.18.0.0/16", "local"],
      ["hollow", "host", "host", "—", "local"],
      ["hollow", "none", "null", "—", "local"],
    ],
    selected: 1,
  },
};

function renderDemo(tab) {
  const data = demoData[tab];
  const body = $("#demo-body");
  if (!data || !body) return;

  const gridTemplates = {
    containers: "24px minmax(180px, 1.6fr) minmax(220px, 2fr) 80px 90px minmax(160px, 1.4fr)",
    images: "24px minmax(180px, 1.4fr) minmax(120px, 1fr) 100px minmax(140px, 1.2fr)",
    models: "24px minmax(180px, 1.5fr) minmax(100px, 1fr) 120px minmax(100px, 0.8fr) 90px",
    volumes: "24px minmax(160px, 1.2fr) 90px minmax(220px, 2fr) minmax(100px, 1fr)",
    networks: "24px minmax(140px, 1fr) 100px minmax(180px, 1.4fr) 100px",
  };
  const gridStyle = `grid-template-columns: ${gridTemplates[tab] || "18px repeat(" + (data.headers.length - 1) + ", 1fr)"};`;

  const alignClass = (i) => {
    const a = data.align && data.align[i];
    return a === "r" ? " align-right" : "";
  };

  let html = "";
  html += `<div class="row header" style="${gridStyle}">`;
  data.headers.forEach((h, i) => {
    html += `<div class="cell${alignClass(i)}">${h}</div>`;
  });
  html += `</div>`;

  data.rows.forEach((row, i) => {
    const sel = i === data.selected ? " selected" : "";
    html += `<div class="row${sel}" style="${gridStyle}">`;
    row.forEach((cell, idx) => {
      if (idx === 0) {
        const dotHtml = cell === "hollow"
          ? `<span class="status-dot hollow" aria-label="idle"></span>`
          : `<span class="status-dot ${cell}" aria-label="${cell}"></span>`;
        html += `<div class="cell">${dotHtml}</div>`;
      } else {
        html += `<div class="cell${alignClass(idx)}">${cell || ""}</div>`;
      }
    });
    html += `</div>`;
  });

  body.innerHTML = html;
  const pager = $("#sb-pager");
  if (pager) pager.textContent = `Showing 1-${data.rows.length} of ${data.rows.length}`;
}

function bindDemoTabs() {
  $$(".demo-tab").forEach((btn) => {
    btn.addEventListener("click", () => {
      $$(".demo-tab").forEach((b) => {
        b.classList.remove("is-active");
        b.setAttribute("aria-selected", "false");
      });
      btn.classList.add("is-active");
      btn.setAttribute("aria-selected", "true");
      renderDemo(btn.dataset.tab);
    });
  });
}

/* ---------- keyboard shortcuts for demo tabs ---------- */
function bindKeyboard() {
  document.addEventListener("keydown", (e) => {
    if (e.target.tagName === "INPUT" || e.target.tagName === "TEXTAREA") return;
    const buttons = $$(".demo-tab");
    const count = buttons.length;
    const key = e.key;
    if (["1", "2", "3", "4", "5"].includes(key)) {
      const idx = parseInt(key, 10) - 1;
      if (buttons[idx]) buttons[idx].click();
    } else if (key === "ArrowRight" || key === "l") {
      const active = buttons.findIndex((b) => b.classList.contains("is-active"));
      buttons[(active + 1) % count].click();
    } else if (key === "ArrowLeft" || key === "h") {
      const active = buttons.findIndex((b) => b.classList.contains("is-active"));
      buttons[(active - 1 + count) % count].click();
    }
  });
}

/* ---------- copy buttons ---------- */
function bindCopy() {
  const toast = $("#toast");
  const showToast = (msg) => {
    toast.textContent = msg;
    toast.classList.add("is-visible");
    clearTimeout(showToast._t);
    showToast._t = setTimeout(() => toast.classList.remove("is-visible"), 1600);
  };

  $$("[data-copy]").forEach((el) => {
    el.addEventListener("click", async () => {
      const text = el.getAttribute("data-copy").replace(/&#10;/g, "\n");
      try {
        await navigator.clipboard.writeText(text);
        el.classList.add("is-copied");
        showToast("copied ✓ " + (text.length > 36 ? text.slice(0, 36) + "…" : text));
        setTimeout(() => el.classList.remove("is-copied"), 1800);
      } catch {
        showToast("press ⌘C to copy");
      }
    });
  });
}

/* ---------- clock + uptime ---------- */
function bindClock() {
  const clock = $("#clock");
  const uptime = $("#uptime");
  if (!clock || !uptime) return;
  const start = Date.now();
  const fmt = (n) => String(n).padStart(2, "0");
  const tick = () => {
    const d = new Date();
    clock.textContent = `${fmt(d.getHours())}:${fmt(d.getMinutes())}:${fmt(d.getSeconds())}`;
    const s = Math.floor((Date.now() - start) / 1000);
    if (s < 60) uptime.textContent = `uptime ${s}s`;
    else if (s < 3600) uptime.textContent = `uptime ${Math.floor(s / 60)}m${s % 60}s`;
    else uptime.textContent = `uptime ${Math.floor(s / 3600)}h${Math.floor((s % 3600) / 60)}m`;
  };
  tick();
  setInterval(tick, 1000);
}

/* ---------- subtle live tweak: pulse one container's CPU ---------- */
function startLivePulse() {
  setInterval(() => {
    const active = $(".demo-tab.is-active");
    if (!active || active.dataset.tab !== "containers") return;
    const rows = $$("#demo-body .row:not(.header)");
    rows.forEach((row, i) => {
      const data = demoData.containers.rows[i];
      if (!data || data[0] !== "green") return;
      const cells = row.querySelectorAll(".cell");
      if (!cells[3]) return;
      const base = parseFloat(data[3]);
      if (Number.isNaN(base)) return;
      const jitter = (Math.random() - 0.5) * 1.2;
      const v = Math.max(0.1, base + jitter);
      cells[3].textContent = v.toFixed(1) + "%";
    });
  }, 1400);
}

/* ---------- boot ---------- */
window.addEventListener("DOMContentLoaded", () => {
  typeOut();
  bindDemoTabs();
  bindKeyboard();
  bindCopy();
  bindClock();
  renderDemo("containers");
  startLivePulse();
});
