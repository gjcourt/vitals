import { initTheme } from './theme.js';

const statusEl = document.getElementById('status');
const refreshBtn = document.getElementById('refreshBtn');
const logoutBtn = document.getElementById('logoutBtn');

const waterCanvas = document.getElementById('waterChart');
const weightCanvas = document.getElementById('weightChart');
const waterNote = document.getElementById('waterNote');
const weightNote = document.getElementById('weightNote');

const unitKgBtn = document.getElementById('unitKg');
const unitLbBtn = document.getElementById('unitLb');

let days = Number(new URLSearchParams(location.search).get('days') || '60');
if (!Number.isFinite(days) || days <= 0) days = 60;
if (days > 366) days = 366;

let unit = localStorage.getItem('weightUnit') || 'lb';

function setUnit(u) {
  unit = u;
  localStorage.setItem('weightUnit', u);
  unitKgBtn.classList.toggle('active', u === 'kg');
  unitLbBtn.classList.toggle('active', u === 'lb');
}

unitKgBtn.addEventListener('click', async () => {
  setUnit('kg');
  await refresh();
});
unitLbBtn.addEventListener('click', async () => {
  setUnit('lb');
  await refresh();
});

document.querySelectorAll('button[data-days]').forEach((btn) => {
  btn.addEventListener('click', async () => {
    const d = Number(btn.getAttribute('data-days'));
    if (Number.isFinite(d) && d > 0) {
      days = d;
      await refresh();
    }
  });
});

logoutBtn.addEventListener('click', async () => {
  await fetch('/api/auth/logout', { method: 'POST' });
  location.reload();
});

refreshBtn.addEventListener('click', refresh);

function drawLineChart(canvas, series, opts) {
  const ctx = canvas.getContext('2d');
  const w = canvas.width;
  const h = canvas.height;

  ctx.clearRect(0, 0, w, h);

  const padL = 44;
  const padR = 10;
  const padT = 12;
  const padB = 28;

  // Background
  const bg = getComputedStyle(document.body).getPropertyValue('--bg').trim();
  ctx.fillStyle = bg;
  ctx.fillRect(0, 0, w, h);

  // Plot area
  const pw = w - padL - padR;
  const ph = h - padT - padB;

  const values = series.map((p) => p.value).filter((v) => Number.isFinite(v));
  if (values.length === 0) {
    ctx.fillStyle = getComputedStyle(document.body).getPropertyValue('--muted').trim();
    ctx.font = '14px system-ui';
    ctx.fillText('No data', padL, padT + 18);
    return;
  }

  let vMin = Math.min(...values);
  let vMax = Math.max(...values);
  if (vMin === vMax) {
    vMin -= 1;
    vMax += 1;
  }

  function xAt(i) {
    if (series.length <= 1) return padL;
    return padL + (i * pw) / (series.length - 1);
  }
  function yAt(v) {
    const t = (v - vMin) / (vMax - vMin);
    return padT + (1 - t) * ph;
  }

  // Grid + y labels (3 lines)
  ctx.strokeStyle = 'rgba(128,128,128,0.2)';
  ctx.fillStyle = getComputedStyle(document.body).getPropertyValue('--muted').trim();
  ctx.font = '12px system-ui';

  for (let g = 0; g <= 2; g++) {
    const t = g / 2;
    const y = padT + t * ph;
    ctx.beginPath();
    ctx.moveTo(padL, y);
    ctx.lineTo(w - padR, y);
    ctx.stroke();

    const val = vMax - t * (vMax - vMin);
    ctx.fillText(val.toFixed(opts.yDecimals ?? 1), 6, y + 4);
  }

  // Line
  ctx.strokeStyle = opts.color;
  ctx.lineWidth = 2;
  ctx.beginPath();
  let started = false;
  for (let i = 0; i < series.length; i++) {
    const p = series[i];
    if (!Number.isFinite(p.value)) {
      started = false;
      continue;
    }
    const x = xAt(i);
    const y = yAt(p.value);
    if (!started) {
      ctx.moveTo(x, y);
      started = true;
    } else {
      ctx.lineTo(x, y);
    }
  }
  ctx.stroke();

  // Draw dots for all points
  ctx.fillStyle = opts.color;
  for (let i = 0; i < series.length; i++) {
    const p = series[i];
    if (!Number.isFinite(p.value)) continue;
    const x = xAt(i);
    const y = yAt(p.value);
    ctx.beginPath();
    ctx.arc(x, y, 3, 0, 2 * Math.PI);
    ctx.fill();
  }

  // X labels: first / middle / last (day)
  ctx.fillStyle = getComputedStyle(document.body).getPropertyValue('--muted').trim();
  const idxs = [0, Math.floor((series.length - 1) / 2), series.length - 1];
  const seen = new Set();
  for (const idx of idxs) {
    if (idx < 0 || idx >= series.length) continue;
    if (seen.has(idx)) continue;
    seen.add(idx);
    const label = series[idx].label;
    const x = xAt(idx);
    ctx.fillText(label, Math.max(0, x - 18), h - 10);
  }
}

async function refresh() {
  statusEl.textContent = 'Loading…';
  const res = await fetch(`/api/charts/daily?days=${encodeURIComponent(days)}&unit=${encodeURIComponent(unit)}`);
  const j = await safeJson(res);
  if (!res.ok) {
    statusEl.textContent = j?.error || 'Failed to load';
    return;
  }

  const items = j?.items || [];

  const waterSeries = items.map((it) => ({
    label: String(it.day).slice(5),
    value: Number(it.waterLiters),
  }));

  const weightSeries = items.map((it) => ({
    label: String(it.day).slice(5),
    value: it.weight ? Number(it.weight.value) : NaN,
  }));

  drawLineChart(waterCanvas, waterSeries, { color: '#22c55e', yDecimals: 1 });
  drawLineChart(weightCanvas, weightSeries, { color: '#60a5fa', yDecimals: unit === 'kg' ? 1 : 1 });

  const waterVals = waterSeries.map((p) => p.value).filter((v) => Number.isFinite(v));
  const weightVals = weightSeries.map((p) => p.value).filter((v) => Number.isFinite(v));

  const waterSum = waterVals.reduce((a, b) => a + b, 0);
  const waterAvg = waterVals.length ? waterSum / waterVals.length : 0;
  waterNote.textContent = `${days} days • avg ${waterAvg.toFixed(2)} L/day`;

  if (weightVals.length) {
    const last = weightVals[weightVals.length - 1];
    weightNote.textContent = `${days} days • latest ${last.toFixed(1)} ${unit}`;
  } else {
    weightNote.textContent = `${days} days • no weight entries`;
  }

  statusEl.textContent = 'Up to date';
}

async function safeJson(res) {
  try {
    return await res.json();
  } catch {
    return null;
  }
}

setUnit(unit);
initTheme();
refresh();
