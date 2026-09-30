const $ = id => document.getElementById(id);
const api = async (path, opts) => {
  const r = await fetch('/api/' + path, opts);
  const data = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(data.error || ('erro ' + r.status));
  return data;
};
const store = { get(k, d) { try { return JSON.parse(localStorage.getItem(k)) ?? d; } catch { return d; } }, set(k, v) { try { localStorage.setItem(k, JSON.stringify(v)); } catch {} } };
const showErr = e => { $('err').textContent = e ? (e.message || e) : ''; };
const fillSelect = (el, items, val, label, placeholder) => {
  el.innerHTML = '';
  const o = document.createElement('option'); o.value = ''; o.textContent = placeholder; el.appendChild(o);
  for (const it of items) { const op = document.createElement('option'); op.value = val(it); op.textContent = label(it); el.appendChild(op); }
  el.disabled = false;
};
const fmtDate = d => { const [y, m, dd] = d.split('-'); const w = ['dom','seg','ter','qua','qui','sex','sáb'][new Date(+y, m - 1, +dd).getDay()]; return `${w} ${dd}/${m}`; };
const chips = (el, items, name, val, label, checked) => {
  el.innerHTML = '';
  if (!items.length) { el.innerHTML = '<span class="muted">—</span>'; return; }
  for (const it of items) {
    const l = document.createElement('label'); l.className = 'chip';
    const i = document.createElement('input'); i.type = 'checkbox'; i.name = name; i.value = val(it); i.checked = checked(val(it));
    l.append(i, document.createTextNode(label(it)));
    if (it.available === false) { l.style.opacity = '.55'; l.title = 'Sem sessões deste formato para este filme'; }
    el.appendChild(l);
  }
};
const checked = name => [...document.querySelectorAll(`input[name=${name}]:checked`)].map(i => i.value);

let job = null, timer = null, rows = [], sortKey = 'when', sortDir = 1, saved = store.get('filters', {});

async function init() {
  setInterval(() => fetch('/api/ping').catch(() => {}), 5000); fetch('/api/ping').catch(() => {});
  api('version').then(v => $('ver').textContent = '· ' + v).catch(() => {});
  try {
    fillSelect($('movie'), await api('movies'), m => m.id, m => m.name, 'Escolha…');
    if (saved.movie) { $('movie').value = saved.movie; if ($('movie').value) await loadOptions(); }
  } catch (e) { showErr(e); }
}
async function loadOptions() {
  $('filters').hidden = true; $('results').hidden = true;
  if (!$('movie').value) return;
  $('status').textContent = 'Carregando opções…';
  const o = await api(`options?movieId=${$('movie').value}`);
  $('status').textContent = '';
  const f = saved.f || {};
  chips($('dates'), o.dates, 'dates', d => d, fmtDate, v => !f.dates || f.dates.includes(String(v)));
  chips($('theaters'), o.theaters, 'theaters', t => t.id, t => t.name, v => !f.theaters || f.theaters.includes(String(v)));
  chips($('features'), o.features, 'features', t => t.id, t => t.name, v => !f.features || f.features.includes(String(v)));
  chips($('audios'), o.audios, 'audios', t => t.id, t => t.name, v => !f.audios || f.audios.includes(String(v)));
  for (const [id, k] of [['from', 'from'], ['to', 'to'], ['group', 'group'], ['minfree', 'minfree']]) if (f[k] !== undefined) $(id).value = f[k];
  $('rowfrom').value = f.rowfrom || ''; $('rowto').value = f.rowto || '';
  $('rowhint').textContent = '';
  $('onlygroups').checked = !!f.onlygroups; $('vertical').checked = f.vertical !== false; $('special').checked = !!f.special;
  $('filters').hidden = false;
  const movie = $('movie').value;
  api(`rows?movieId=${movie}`).then(r => {
    if (movie !== $('movie').value || !r.max) return; // usuário já trocou de filme
    $('rowto').placeholder = `última (${r.max})`;
    $('rowhint').textContent = r.min === r.max ? `As salas têm ${r.max} fileiras.` : `As salas têm de ${r.min} a ${r.max} fileiras (a maior tem ${r.max}).`;
  }).catch(() => {});
  if (!o.dates.length) $('status').textContent = 'Sem sessões à venda para este filme nestes cinemas.';
}
$('movie').onchange = () => { saved = { movie: $('movie').value }; store.set('filters', saved); loadOptions().catch(showErr); };

function collect() {
  const num = id => parseInt($(id).value, 10) || 0;
  return {
    dates: checked('dates'), theaterIds: checked('theaters').map(Number), features: checked('features').map(Number), audios: checked('audios').map(Number),
    timeFrom: $('from').value, timeTo: $('to').value, rowFrom: Math.max(0, num('rowfrom')), rowTo: Math.max(0, num('rowto')), groupSize: Math.max(1, num('group')), minFree: num('minfree'),
    onlyWithGroups: $('onlygroups').checked, vertical: $('vertical').checked, includeSpecial: $('special').checked,
  };
}
function persist() {
  // Só guarda a lista quando algo foi desmarcado; "tudo marcado" é o padrão e acompanha datas novas.
  const partial = name => {
    const all = [...document.querySelectorAll(`input[name=${name}]`)];
    return all.some(i => !i.checked) ? all.filter(i => i.checked).map(i => i.value) : undefined;
  };
  saved.f = { dates: partial('dates'), theaters: partial('theaters'), features: partial('features'), audios: partial('audios'),
    from: $('from').value, to: $('to').value, rowfrom: $('rowfrom').value, rowto: $('rowto').value, group: $('group').value, minfree: $('minfree').value, onlygroups: $('onlygroups').checked, vertical: $('vertical').checked, special: $('special').checked };
  store.set('filters', saved);
}

$('go').onclick = async () => {
  showErr(null); persist();
  const f = collect();
  $('go').disabled = true; $('stop').hidden = false; $('barwrap').hidden = false; $('bar').style.width = '0'; $('status').textContent = 'Buscando sessões…';
  try {
    const { jobId } = await api('search', { method: 'POST', body: JSON.stringify({ movieId: $('movie').value, filters: f }) });
    job = jobId;
    timer = setInterval(poll, 700); poll();
  } catch (e) { showErr(e); done(); }
};
$('stop').onclick = () => { if (job) api('cancel?id=' + job).catch(() => {}); };
function done() { clearInterval(timer); job = null; $('go').disabled = false; $('stop').hidden = true; $('barwrap').hidden = true; }
async function poll() {
  if (!job) return;
  try {
    const j = await api('job?id=' + job);
    if (j.total) { $('bar').style.width = (100 * j.done / j.total) + '%'; $('status').textContent = `Consultando mapas de assentos: ${j.done}/${j.total}`; }
    if (j.finished) {
      done(); $('status').textContent = '';
      if (j.error) return showErr(j.error);
      rows = j.rows; render();
    }
  } catch (e) { showErr(e); done(); }
}

const cols = [
  ['when', 'Quando', r => r.date + ' ' + r.time, r => `${fmtDate(r.date)} ${r.time}`],
  ['theater', 'Cinema', r => r.theaterName, r => r.theaterName],
  ['room', 'Sala', r => r.room, r => r.room],
  ['type', 'Tipo', r => (r.features || []).join(', '), r => (r.features || []).join(', ') || '—'],
  ['audio', 'Áudio', r => r.audio, r => r.audio || '—'],
  ['free', 'Livres', r => r.stats.free, r => r.error ? `<span class="msg" title="${esc(r.error)}">erro</span>` : cell(r.stats.free, r.stats.total)],
  ['groups', 'Grupos de N', r => r.stats.groups, r => r.error ? '' : `<span class="${r.stats.groups ? 'ok' : 'bad'}">${r.stats.groups}</span>`],
  ['run', 'Maior bloco', r => r.stats.bestRun, r => r.error ? '' : r.stats.bestRun],
];
const esc = s => String(s).replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
const cell = (free, total) => { const p = total ? free / total : 0; return `<span class="${p > .4 ? 'ok' : p > .15 ? 'warn' : 'bad'}">${free}</span> <span class="muted">/ ${total}</span>`; };
function render() {
  $('results').hidden = false;
  const n = Math.max(1, parseInt($('group').value, 10) || 1);
  $('thead').innerHTML = cols.map(([k, t]) => `<th data-k="${k}">${k === 'groups' ? `Grupos de ${n}` : t}${sortKey === k ? (sortDir > 0 ? ' ▲' : ' ▼') : ''}</th>`).join('');
  document.querySelectorAll('#thead th').forEach(th => th.onclick = () => { sortDir = sortKey === th.dataset.k ? -sortDir : 1; sortKey = th.dataset.k; render(); });
  const col = cols.find(c => c[0] === sortKey) || cols[0];
  const sorted = [...rows].sort((a, b) => { const x = col[2](a), y = col[2](b); return (x < y ? -1 : x > y ? 1 : 0) * sortDir; });
  $('rows').innerHTML = sorted.map(r => '<tr>' + cols.map(c => `<td>${c[3](r)}</td>`).join('') + '</tr>').join('');
  const free = rows.reduce((s, r) => s + (r.stats.free || 0), 0);
  $('summary').textContent = rows.length ? `${rows.length} sessões · ${free} assentos livres no total` : 'Nenhuma sessão atende aos filtros.';
}
$('csv').onclick = () => {
  const head = ['data', 'hora', 'cinema', 'sala', 'tipo', 'audio', 'livres', 'total', 'grupos', 'maior_sequencia'];
  const lines = [head.join(';')].concat(rows.map(r => [r.date, r.time, r.theaterName, r.room, (r.features || []).join('/'), r.audio, r.stats.free, r.stats.total, r.stats.groups, r.stats.bestRun].map(v => `"${String(v).replace(/"/g, '""')}"`).join(';')));
  const a = document.createElement('a'); a.href = URL.createObjectURL(new Blob(['﻿' + lines.join('\r\n')], { type: 'text/csv;charset=utf-8' })); a.download = 'sessoes.csv'; a.click();
};
init();
