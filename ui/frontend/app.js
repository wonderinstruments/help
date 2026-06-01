const searchInput = document.getElementById('search-input');
const sidebar = document.getElementById('sidebar');
const content = document.getElementById('content');

let searchTimeout = null;
let currentDocPath = '';

async function init() {
  // Load theme override if available
  try {
    const themeCSS = await window.go.main.App.GetThemeCSS();
    console.log('[help-ui] theme CSS received, length:', themeCSS?.length);
    console.log('[help-ui] theme CSS:', themeCSS);
    if (themeCSS) {
      const style = document.createElement('style');
      style.textContent = themeCSS;
      document.head.appendChild(style);
    }
  } catch (e) {
    console.error('[help-ui] theme load error:', e);
  }

  // Log computed styles
  const computed = getComputedStyle(document.documentElement);
  console.log('[help-ui] --font-size:', computed.getPropertyValue('--font-size'));
  console.log('[help-ui] --font-body:', computed.getPropertyValue('--font-body'));
  console.log('[help-ui] body font-size:', getComputedStyle(document.body).fontSize);
  console.log('[help-ui] body font-family:', getComputedStyle(document.body).fontFamily);

  const docs = await window.go.main.App.ListDocuments('');
  renderSidebar(docs);

  const initialDoc = await window.go.main.App.GetInitialDoc();
  if (initialDoc) {
    readDoc(initialDoc);
  }
}

function renderSidebar(docs) {
  sidebar.innerHTML = docs.map(d => `
    <div class="doc-item" data-path="${d.Path}">
      ${d.Title}
      ${(d.Tags || []).map(t => `<span class="tag">${t}</span>`).join('')}
    </div>
  `).join('');

  sidebar.querySelectorAll('.doc-item').forEach(el => {
    el.addEventListener('click', () => readDoc(el.dataset.path));
  });
}

async function readDoc(path) {
  currentDocPath = path;
  const html = await window.go.main.App.GetDocument(path);
  content.innerHTML = html;
  bindDocLinks();

  sidebar.querySelectorAll('.doc-item').forEach(el => {
    el.classList.toggle('active', el.dataset.path === path);
  });
}

function bindDocLinks() {
  content.querySelectorAll('a[href]').forEach(el => {
    const href = el.getAttribute('href');
    if (href && href.endsWith('.md') && !href.startsWith('http')) {
      el.addEventListener('click', (e) => {
        e.preventDefault();
        const resolved = resolvePath(currentDocPath, href);
        readDoc(resolved);
      });
    }
  });
}

function resolvePath(from, rel) {
  const dir = from.includes('/') ? from.substring(0, from.lastIndexOf('/') + 1) : '';
  const parts = (dir + rel).split('/');
  const resolved = [];
  for (const p of parts) {
    if (p === '..') resolved.pop();
    else if (p !== '.' && p !== '') resolved.push(p);
  }
  return resolved.join('/');
}

async function doSearch(query) {
  if (!query.trim()) {
    const docs = await window.go.main.App.ListDocuments('');
    renderSidebar(docs);
    content.innerHTML = '';
    return;
  }

  const results = await window.go.main.App.Search(query, '', 10);

  content.innerHTML = (results || []).map(r => `
    <div class="search-result" data-path="${r.DocPath}">
      <span class="score">${r.Score.toFixed(2)}</span>
      <div class="title">${r.DocTitle}</div>
      ${r.Heading ? `<div class="heading">${r.Heading}</div>` : ''}
      <div class="snippet">${(r.Content || '').slice(0, 150)}...</div>
    </div>
  `).join('');

  content.querySelectorAll('.search-result').forEach(el => {
    el.addEventListener('click', () => readDoc(el.dataset.path));
  });
}

searchInput.addEventListener('input', (e) => {
  clearTimeout(searchTimeout);
  searchTimeout = setTimeout(() => doSearch(e.target.value), 300);
});

searchInput.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    searchInput.value = '';
    doSearch('');
  }
});

init();
