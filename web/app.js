import { initTheme } from './theme.js';

const statusEl = document.getElementById('status');
const refreshBtn = document.getElementById('refreshBtn');
const logoutBtn = document.getElementById('logoutBtn');

const weightValueEl = document.getElementById('weightValue');
const unitKgBtn = document.getElementById('unitKg');
const unitLbBtn = document.getElementById('unitLb');
const saveWeightBtn = document.getElementById('saveWeightBtn');
const undoWeightBtn = document.getElementById('undoWeightBtn');
const weightMetaEl = document.getElementById('weightMeta');
const weightListEl = document.getElementById('weightList');

const waterTotalEl = document.getElementById('waterTotal');
const waterMetaEl = document.getElementById('waterMeta');
const addWaterBtn = document.getElementById('addWaterBtn');
const decWaterBtn = document.getElementById('decWaterBtn');
const undoWaterBtn = document.getElementById('undoWaterBtn');
const waterListEl = document.getElementById('waterList');

let unit = localStorage.getItem('weightUnit') || 'lb';
let addDelta = Number(localStorage.getItem('waterAddDelta') || '1.0');

function setUnit(u){
  unit = u;
  localStorage.setItem('weightUnit', u);
  unitKgBtn.classList.toggle('active', u==='kg');
  unitLbBtn.classList.toggle('active', u==='lb');
}

function setAddDelta(d){
  addDelta = d;
  localStorage.setItem('waterAddDelta', String(d));
  addWaterBtn.textContent = `+ ${d} L`;
}

unitKgBtn.addEventListener('click', ()=>setUnit('kg'));
unitLbBtn.addEventListener('click', ()=>setUnit('lb'));

addWaterBtn.addEventListener('click', async ()=>{
  await postWater(addDelta);
  await refresh();
});

decWaterBtn.addEventListener('click', async ()=>{
  await postWater(-0.25);
  await refresh();
});

undoWaterBtn.addEventListener('click', async ()=>{
  await fetch('/api/water/undo-last', {method:'POST'});
  await refresh();
});

saveWeightBtn.addEventListener('click', async ()=>{
  const value = Number(String(weightValueEl.value).replace(',', '.'));
  if(!Number.isFinite(value) || value<=0){
    toast('Enter a valid weight');
    return;
  }
  const res = await fetch('/api/weight/today', {
    method:'PUT',
    headers:{'Content-Type':'application/json'},
    body: JSON.stringify({value, unit})
  });
  if(!res.ok){
    const j = await safeJson(res);
    toast(j?.error || 'Failed to save');
    return;
  }
  await refresh();
  toast('Saved');
});

undoWeightBtn.addEventListener('click', async ()=>{
  const res = await fetch('/api/weight/undo-last', {method:'POST'});
  const j = await safeJson(res);
  await refresh();
  if(j?.deleted){
    toast('Undid last weight');
  }else{
    toast('No weight entry to undo');
  }
});
logoutBtn.addEventListener('click', async () => {
    await fetch('/logout', { method: 'POST' });
    location.reload();
});


refreshBtn.addEventListener('click', refresh);

document.querySelectorAll('button[data-delta]').forEach(btn=>{
  btn.addEventListener('click', async ()=>{
    const d = Number(btn.getAttribute('data-delta'));
    setAddDelta(d);
    toast(`Set + button to ${d} L`);
  });
});

async function postWater(deltaLiters){
  const res = await fetch('/api/water/event', {
    method:'POST',
    headers:{'Content-Type':'application/json'},
    body: JSON.stringify({deltaLiters})
  });
  if(!res.ok){
    const j = await safeJson(res);
    toast(j?.error || 'Failed');
  }
}

function fmtTime(iso){
  try{
    const d = new Date(iso);
    return d.toLocaleString([], {month:'short', day:'numeric', hour:'2-digit', minute:'2-digit'});
  }catch{ return iso; }
}

function renderList(el, items, render){
  el.innerHTML = '';
  if(!items || items.length===0){
    const div = document.createElement('div');
    div.className='muted';
    div.textContent='No entries yet';
    el.appendChild(div);
    return;
  }
  items.forEach(it=>el.appendChild(render(it)));
}

function weightItem(it){
  const row = document.createElement('div');
  row.className='item';
  const left = document.createElement('div');
  left.className='left';
  const top = document.createElement('div');
  top.className='top';
  top.textContent = `${it.day}: ${it.value} ${it.unit}`;
  const sub = document.createElement('div');
  sub.className='sub';
  sub.textContent = `Saved ${fmtTime(it.createdAt)}`;
  left.appendChild(top);
  left.appendChild(sub);
  row.appendChild(left);
  return row;
}

function waterItem(it){
  const row = document.createElement('div');
  row.className='item';
  const left = document.createElement('div');
  left.className='left';
  const top = document.createElement('div');
  top.className='top';
  const sign = it.deltaLiters>0?'+':'';
  top.textContent = `${sign}${it.deltaLiters} L`;
  const sub = document.createElement('div');
  sub.className='sub';
  sub.textContent = fmtTime(it.createdAt);
  left.appendChild(top);
  left.appendChild(sub);
  row.appendChild(left);
  return row;
}

async function refresh(){
  statusEl.textContent = 'Syncing…';
  const [wToday, wRecent, waterToday, waterRecent] = await Promise.all([
    fetch('/api/weight/today').then(safeJson),
    fetch('/api/weight/recent?limit=14').then(safeJson),
    fetch('/api/water/today').then(safeJson),
    fetch('/api/water/recent?limit=20').then(safeJson),
  ]);

  if(wToday?.entry){
    weightMetaEl.textContent = `Today (${wToday.today}) saved ${fmtTime(wToday.entry.createdAt)}`;
    weightValueEl.value = String(wToday.entry.value);
    setUnit(wToday.entry.unit);
  }else{
    weightMetaEl.textContent = `No entry yet for today (${wToday?.today || ''})`;
    setUnit(unit);
  }

  const total = Number(waterToday?.totalLiters || 0);
  waterTotalEl.textContent = `${total.toFixed(2)} L`;
  waterMetaEl.textContent = `Today (${waterToday?.today || ''})`;

  renderList(waterListEl, waterRecent?.items || [], waterItem);
  renderList(weightListEl, wRecent?.items || [], weightItem);

  statusEl.textContent = 'Up to date';
}

async function safeJson(res){
  try{ return await res.json(); }catch{ return null; }
}

let toastTimer;
function toast(msg){
  statusEl.textContent = msg;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(()=>{statusEl.textContent='';}, 2000);
}

setUnit(unit);
initTheme();
setAddDelta(addDelta);
refresh();
