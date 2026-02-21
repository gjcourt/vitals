export function initTheme() {
  const themeSelect = document.getElementById('themeSelect');
  if (!themeSelect) return;

  const saved = localStorage.getItem('theme') || 'dark';
  document.documentElement.setAttribute('data-theme', saved);
  themeSelect.value = saved;

  themeSelect.addEventListener('change', () => {
    const val = themeSelect.value;
    document.documentElement.setAttribute('data-theme', val);
    localStorage.setItem('theme', val);

    // Update theme-color meta tag
    const meta = document.querySelector('meta[name="theme-color"]');
    if(meta) {
        const bg = getComputedStyle(document.body).getPropertyValue('--bg').trim();
        meta.setAttribute('content', bg);
    }
  });

  // Initial meta tag update
  const pkg = getComputedStyle(document.body).getPropertyValue('--bg').trim();
  const meta = document.querySelector('meta[name="theme-color"]');
  if(meta) meta.setAttribute('content', pkg);
}
