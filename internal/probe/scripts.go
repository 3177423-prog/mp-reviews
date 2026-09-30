package probe

// Скрипты, которые пробник выполняет в странице. Оба работают в изолированном мире
// (как скрипты расширений): странице не видны ни обработчики, ни функция-привязка.

const (
	bindingName   = "mpProbeAction"
	actionWorld   = "mpProbe"
	snapshotWorld = "mpProbeSnapshot"
)

// actionScript — журнал действий пользователя: клик, начало ввода, изменение поля, отправка формы.
// Значения полей не записываются — только длина (и ни длины, ни значения для паролей и секретных полей);
// текст выбранного пункта <select> и состояние флажков записываются, это основания и настройки жалобы.
const actionScript = `(() => {
  if (globalThis.__mpProbeInstalled) return;
  globalThis.__mpProbeInstalled = true;
  const send = (o) => {
    try { o.ts = Date.now(); o.url = location.href; ` + bindingName + `(JSON.stringify(o)); } catch (e) {}
  };
  const clip = (s, n) => String(s || '').replace(/\s+/g, ' ').trim().slice(0, n);
  const sel = 'button,a,input,select,textarea,label,summary,option,[role],[onclick],[tabindex]';
  const describe = (el) => {
    if (!el || el.nodeType !== 1) return null;
    const d = { tag: el.tagName.toLowerCase() };
    const attr = (n) => el.getAttribute(n);
    if (el.id) d.id = clip(el.id, 80);
    if (attr('name')) d.name = clip(attr('name'), 80);
    if (attr('type')) d.type = clip(attr('type'), 40);
    if (attr('role')) d.role = clip(attr('role'), 40);
    const label = attr('aria-label') || attr('title') || attr('placeholder');
    if (label) d.label = clip(label, 80);
    if (!/^(input|textarea|select)$/.test(d.tag)) {
      const t = clip(el.innerText || el.textContent, 80);
      if (t) d.text = t;
    }
    if (typeof el.className === 'string' && el.className) d.class = clip(el.className, 120);
    if (attr('href')) d.href = clip(el.href || attr('href'), 500);
    const testid = attr('data-testid') || attr('data-test-id') || attr('data-qa');
    if (testid) d.testid = clip(testid, 80);
    return d;
  };
  const origin = (e) => {
    const p = e.composedPath ? e.composedPath() : [];
    let el = p.length ? p[0] : e.target;
    while (el && el.nodeType !== 1) el = el.parentNode;
    return el;
  };
  const actionable = (el) => (el && el.closest && el.closest(sel)) || el;
  const secretField = (el) => el.type === 'password' ||
    /token|auth|session|secret|sign|cookie|csrf|passw/i.test([el.name, el.id, el.autocomplete].join(' '));
  addEventListener('click', (e) => send({ action: 'click', el: describe(actionable(origin(e))) }), true);
  const typing = new WeakSet();
  addEventListener('input', (e) => {
    const el = origin(e);
    if (!el || typing.has(el)) return;
    typing.add(el);
    send({ action: 'input', el: describe(el) });
  }, true);
  addEventListener('change', (e) => {
    const el = origin(e);
    if (!el) return;
    typing.delete(el);
    const o = { action: 'change', el: describe(el) };
    if (el.tagName === 'SELECT') o.selected = Array.from(el.selectedOptions || []).map((x) => clip(x.text, 80));
    else if (el.type === 'checkbox' || el.type === 'radio') o.checked = !!el.checked;
    else if (el.type === 'file') o.files = el.files ? el.files.length : 0;
    else if (!secretField(el) && typeof el.value === 'string') o.value_len = el.value.length;
    send(o);
  }, true);
  addEventListener('submit', (e) => {
    const f = origin(e);
    const o = { action: 'submit', el: describe(f) };
    if (f && f.tagName === 'FORM') {
      o.form_action = f.getAttribute('action') ? f.action : '';
      o.form_method = String(f.method || '').toLowerCase();
    }
    send(o);
  }, true);
})();`

// snapshotScript возвращает копию DOM без содержимого скриптов, с вычищенными секретными
// атрибутами, значениями скрытых и парольных полей и секретными параметрами адресов.
// В копию добавляется CSP, чтобы открытый снимок не обращался в сеть.
const snapshotScript = `(() => {
  const RX = /token|auth|session|secret|sign|cookie|csrf|xsrf|passw|jwt|api[-_]?key|wb-seller-lk/i;
  const H = '<скрыто>';
  const cleanURL = (v) => {
    try {
      const u = new URL(v, location.href);
      let changed = false;
      for (const k of Array.from(u.searchParams.keys())) {
        if (RX.test(k)) { u.searchParams.set(k, H); changed = true; }
      }
      if (u.password || u.username) { u.username = ''; u.password = ''; changed = true; }
      return changed ? u.href : v;
    } catch (e) { return v; }
  };
  const root = document.documentElement.cloneNode(true);
  for (const s of root.querySelectorAll('script')) s.textContent = '';
  for (const el of root.querySelectorAll('*')) {
    for (const a of Array.from(el.attributes)) {
      if (RX.test(a.name)) { el.setAttribute(a.name, H); continue; }
      if (/^(href|src|action|formaction|poster|data-src|data-href)$/i.test(a.name)) el.setAttribute(a.name, cleanURL(a.value));
    }
    const tag = el.tagName.toLowerCase();
    if (tag === 'input' && el.hasAttribute('value')) {
      const t = (el.getAttribute('type') || '').toLowerCase();
      if (t === 'password' || t === 'hidden' || RX.test(el.getAttribute('name') || '')) el.setAttribute('value', H);
    }
    if (tag === 'meta') {
      const n = [el.getAttribute('name'), el.getAttribute('property'), el.getAttribute('http-equiv')].join(' ');
      if (RX.test(n)) el.setAttribute('content', H);
    }
  }
  const head = root.querySelector('head');
  if (head) {
    const csp = document.createElement('meta');
    csp.setAttribute('http-equiv', 'Content-Security-Policy');
    csp.setAttribute('content', "default-src 'none'; img-src data:; style-src 'unsafe-inline'");
    head.insertBefore(csp, head.firstChild);
  }
  return '<!DOCTYPE html>\n' + root.outerHTML;
})()`
