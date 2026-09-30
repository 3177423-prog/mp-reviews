"""
Панель: очередь негативных отзывов, решения человека, внесение исходов жалоб.

Flask на 127.0.0.1:5096, снаружи — https://reviews.wild-expert.ru
Логин и пароль — в panel.env. Токен WB — в .env (права 600), в интерфейсе только маска.
"""

import html
import os
import re
import subprocess
import time
from datetime import datetime
from functools import wraps

from flask import Flask, Response, request, redirect, send_file, url_for

import accounts
import store
import cabinet

HERE = os.path.dirname(os.path.abspath(__file__))
ENV_FILE = os.path.join(HERE, ".env")
LOG_FILE = os.path.join(HERE, "run.log")
SCRIPT = os.path.join(HERE, "test1_feedbacks.py")
REPORT = os.path.join(HERE, "report", "field_coverage.md")
CSV = os.path.join(HERE, "normalized", "feedbacks.csv")

BASES = [name for name, _ in cabinet.REASONS]   # ровно как в форме кабинета WB

BATCH_LIMIT = 50       # потолок на одну пачку
BATCH_PAUSE = 2.0      # пауза между жалобами: на 1 с кабинет отдавал 429
PER_PAGE = (25, 50, 100)

TIER_LABEL = {0: "—", 1: "оценок < 100", 2: "рейтинг < 4,7", 3: "обычный"}

CSS = """<style>
 body{font:15px/1.5 -apple-system,Segoe UI,sans-serif;max-width:1180px;margin:26px auto;padding:0 16px}
 h1{font-size:21px} h2{font-size:16px;margin-top:26px}
 a{color:#0a58ca}
 table{border-collapse:collapse;width:100%}
 td,th{border-bottom:1px solid #e6e6e6;padding:9px;vertical-align:top;text-align:left}
 th{background:#fafafa;font-size:13px;font-weight:600}
 .meta{color:#666;font-size:13px} .low{color:#b00;font-weight:600}
 .ok{color:#0a7a2f} .warn{color:#a86400} .no{color:#b00}
 ol{padding-left:20px}
 .draft{font-size:13px;color:#333;background:#f7f7f7;border-radius:6px;padding:7px;margin-top:6px;max-width:330px}
 .t1{background:#fff2f2} .t2{background:#fffbea}
 .alarm{background:#b00;color:#fff;padding:11px 14px;border-radius:8px;margin-bottom:14px;font-weight:600}
 .alarm a{color:#fff;text-decoration:underline}
 .box{border:1px solid #ddd;border-radius:8px;padding:14px;margin:12px 0}
 .nums b{font-size:20px} .nums span{display:inline-block;margin-right:22px}
 button{font-size:14px;padding:6px 12px;border-radius:6px;border:1px solid #888;cursor:pointer}
 select,input[type=text]{font-size:14px;padding:5px;max-width:200px}
 pre{background:#111;color:#eee;padding:12px;border-radius:8px;overflow:auto;max-height:300px;white-space:pre-wrap}
</style>"""

PAGES = (("index", "Сводка"), ("queue", "Очередь"), ("outcomes", "Исходы"),
         ("session_page", "Сессия кабинета"), ("tools", "Служебное"))

app = Flask(__name__)


def panel_credentials():
    user, password = "", ""
    path = os.path.join(HERE, "panel.env")
    if os.path.exists(path):
        for line in open(path, encoding="utf-8-sig"):
            if line.startswith("PANEL_USER="):
                user = line.split("=", 1)[1].strip()
            elif line.startswith("PANEL_PASS="):
                password = line.split("=", 1)[1].strip()
    return user, password


def auth_required(view):
    @wraps(view)
    def wrapper(*args, **kwargs):
        user, password = panel_credentials()
        given = request.authorization
        if not given or given.username != user or given.password != password:
            return Response("Требуется вход", 401,
                            {"WWW-Authenticate": 'Basic realm="WB reviews"'})
        return view(*args, **kwargs)
    return wrapper


def acc():
    """Кабинет, с которым сейчас работает страница. Живёт в адресе: ?acc=cr."""
    return accounts.valid(request.args.get("acc") or request.form.get("acc"))


def token_mask(account):
    if not os.path.exists(ENV_FILE):
        return None
    name = accounts.token_var(account)
    for line in open(ENV_FILE, encoding="utf-8-sig"):
        if line.startswith(name + "="):
            value = line.split("=", 1)[1].strip()
            if value:
                return "%s…%s" % (value[:6], value[-6:])
    return None


SESSION_CACHE = {"checked": 0.0, "alive": True}


def session_alive():
    """Живость сессии. Сессия одна на все кабинеты, поэтому проверяем её на одном
    и кешируем на 3 минуты, чтобы каждая страница не ходила в WB заново."""
    if time.time() - SESSION_CACHE["checked"] < 180:
        return SESSION_CACHE["alive"]
    alive = False
    try:
        if cabinet.load_session():
            status, _ = cabinet.call(accounts.DEFAULT,
                                     "/feedbacks?take=100&skip=0&isAnswered=false")
            alive = status == 200
    except Exception:
        alive = False
    SESSION_CACHE.update(checked=time.time(), alive=alive)
    return alive


def page(title, body, account=None):
    account = account or accounts.DEFAULT
    alarm = ""
    if not session_alive():
        alarm = ('<div class=alarm>Сессия кабинета не работает — жалобы не подаются. '
                 '<a href="%s">Обновить сессию</a></div>'
                 % url_for("session_page", acc=account))
    nav = " · ".join('<a href="%s">%s</a>' % (url_for(endpoint, acc=account), label)
                     for endpoint, label in PAGES)
    # Соседний сервис на том же домене: вопросы покупателей (проксируется Caddy на /qa)
    nav += ' &nbsp;|&nbsp; <a href="/qa/">Вопросы покупателей</a>'
    tabs = " · ".join(
        ("<b>%s</b>" % accounts.NAME[k]) if k == account else
        '<a href="%s">%s</a>' % (url_for(request.endpoint, acc=k), accounts.NAME[k])
        for k in accounts.KEYS)
    return ("<!doctype html><html lang=ru><meta charset=utf-8><title>%s — %s</title>%s%s"
            "<p>%s</p><p class=meta>Кабинет: %s</p><h1>%s <span class=meta>· %s</span></h1>%s</html>"
            % (title, accounts.NAME[account], CSS, alarm, nav, tabs, title,
               esc(accounts.NAME[account]), body))


def esc(value):
    return html.escape(str(value if value is not None else ""))


def wb_link(nm_id):
    """Артикул WB со ссылкой на карточку — чтобы можно было открыть товар в один клик."""
    if not nm_id:
        return ""
    return ('<a href="https://www.wildberries.ru/catalog/%s/detail.aspx" target="_blank">'
            'Арт. WB %s</a>' % (esc(nm_id), esc(nm_id)))


@app.route("/")
@auth_required
def index():
    account = acc()
    db = store.connect()
    c = store.counters(db, account)
    mask = token_mask(account)
    body = """<div class="box nums">
      <span>В очереди<br><b>%d</b></span>
      <span>Подано<br><b>%d</b></span>
      <span>Пропущено<br><b>%d</b></span>
      <span>Ждут исхода<br><b>%d</b></span>
      <span>Одобрено<br><b>%d</b></span>
      <span>Отклонено<br><b>%d</b></span>
      <span>Артикулов с рейтингом<br><b>%d</b></span>
      <span>Размечено AI<br><b>%d</b></span>
      <span>Ждут разметки<br><b>%d</b></span>
     </div>
     <p class=meta>Токен WB: %s. Отзывы с уже поданной жалобой в очередь не попадают (%d шт.).</p>
     <p><a href="%s">Перейти к очереди →</a></p>""" % (
        c["queue"], c["submitted"], c["skipped"], c["pending"], c["approved"],
        c["rejected"], c["sku_stats"], c["analyzed"], c["to_analyze"],
        ("задан " + mask) if mask else "НЕ ЗАДАН", c["prefiled"],
        url_for("queue", acc=account))
    return page("Жалобы на отзывы — сводка", body, account)


@app.route("/queue")
@auth_required
def queue():
    account = acc()
    db = store.connect()
    items = store.queue(db, account)

    f_basis = request.args.get("basis") or ""
    f_state = request.args.get("state") or ""
    if f_basis:
        items = [i for i in items if (i.get("ai_basis") or "") == f_basis]
    if f_state == "analyzed":
        items = [i for i in items if i.get("ai_basis")]
    elif f_state == "pending":
        items = [i for i in items if not i.get("ai_basis") and not i.get("ai_failed")]
    elif f_state == "drafted":
        items = [i for i in items if (i.get("ai_draft") or "").strip()]

    all_bases = sorted({i["ai_basis"] for i in store.queue(db, account) if i.get("ai_basis")})
    try:
        per = int(request.args.get("per") or PER_PAGE[0])
    except ValueError:
        per = PER_PAGE[0]
    if per not in PER_PAGE:
        per = PER_PAGE[0]

    def link(state, basis, label):
        active = (state == f_state and basis == f_basis)
        url = url_for("queue", state=state, basis=basis, per=per, acc=account)
        return ("<b>%s</b>" % esc(label)) if active else '<a href="%s">%s</a>' % (url, esc(label))
    filters = " · ".join(
        [link("", "", "все"), link("drafted", "", "написан текст жалобы"),
         link("analyzed", "", "размеченные"), link("pending", "", "без разметки")]
        + [link("", b, b) for b in all_bases])
    sizes = " · ".join(
        ("<b>%d</b>" % n) if n == per else
        '<a href="%s">%d</a>' % (url_for("queue", state=f_state, basis=f_basis,
                                         per=n, acc=account), n)
        for n in PER_PAGE)

    rows = []
    for it in items[:per]:
        rid = it["id"]
        picked = (it.get("ai_basis") or "").strip()
        ordered = ([b for b in BASES if b.lower() == picked.lower()]
                   + [b for b in BASES if b.lower() != picked.lower()])
        options = "".join("<option>%s</option>" % esc(b) for b in ordered)
        if it.get("ai_failed"):
            ai_block = "<span class=meta>AI: ответ не разобран</span>"
        elif not it.get("ai_basis"):
            ai_block = "<span class=meta>AI: ещё не размечен</span>"
        else:
            colour = {"сильное": "ok", "слабое": "warn", "нет": "meta"}.get(
                it.get("ai_strength"), "meta")
            ai_block = ('<b class=%s>%s</b> <span class=meta>· %s · уверенность %s'
                        '<br>%s · промпт %s</span>'
                        % (colour, esc(it["ai_basis"]), esc(it["ai_strength"]),
                           esc(it["ai_confidence"]), esc(it.get("ai_model")),
                           esc(it.get("ai_version"))))
        stats = ("%s оценок, рейтинг %s" % (it["ratings_count"], it["avg_rating"])
                 if it["ratings_count"] is not None else "рейтинг ещё не считали")
        extra = []
        if it["order_status"] in ("rejected", "returned"):
            extra.append("статус заказа: " + it["order_status"])
        if it["days_to_review"] is not None:
            extra.append("отзыв через %d дн." % it["days_to_review"])
        if it["parent_id"]:
            extra.append("дополнение к отзыву")
        if it["photo_count"]:
            extra.append("фото: %d" % it["photo_count"])
        draft = it.get("ai_draft") or ""
        rows.append("""<tr class="t%s">
          <td><input type=checkbox name=pick value="%s" %s></td>
          <td class=meta>%s<br><span class=low>%s★</span></td>
          <td class=meta>%s<br>%s<br>%s<br>%s</td>
          <td>%s<div class=meta>%s</div></td>
          <td>%s<br><select name="basis_%s">%s</select></td>
          <td><textarea name="text_%s" rows=4 style="width:290px;font-size:13px">%s</textarea><br>
            <button name=action value="one:%s">Подать</button>
            <button name=action value="skip:%s">Пропустить</button></td></tr>""" % (
            it["tier"] if it["tier"] in (1, 2) else "0",
            esc(rid), "disabled" if not draft.strip() else "",
            esc((it["created_date"] or "")[:16].replace("T", " ")), esc(it["valuation"]),
            esc(it["supplier_article"]), esc(it["subject_name"]),
            wb_link(it["nm_id"]), esc(stats),
            esc(it["text"]) or "<i>без текста</i>", esc(" · ".join(extra)),
            ai_block, esc(rid), options,
            esc(rid), esc(draft), esc(rid), esc(rid)))

    c = store.counters(db, account)
    msg = request.args.get("msg") or ""
    banner = ""
    if msg.startswith("ok"):
        banner = "<p class=ok>%s</p>" % esc(msg[3:] or "Жалоба подана.")
    elif msg.startswith("err:"):
        banner = "<p class=no>Подача не прошла: %s</p>" % esc(msg[4:])

    head = (banner
            + "<p class=meta>Фильтр: %s</p>" % filters
            + "<p class=meta>На странице: %s</p>" % sizes
            + "<p class=meta>Показано %d из %d по фильтру, всего в очереди %d. "
              "Размечено AI: %d, ждут разметки: %d. Ступени приоритета %s.</p>" % (
                  min(len(items), per), len(items), c["queue"], c["analyzed"],
                  c["to_analyze"], "включены" if c["queue"] > 100 else "выключены"))
    if not rows:
        return page("Очередь", head + "<p>Под фильтр ничего не попало.</p>", account)
    controls = (
        '<input type=hidden name=acc value="%s">'
        '<input type=hidden name=state value="%s">'
        '<input type=hidden name=basis_filter value="%s">'
        '<input type=hidden name=per value="%d">'
        '<p><button type=button onclick="pickAll(true)">Выделить все</button> '
        '<button type=button onclick="pickAll(false)">Снять выделение</button> '
        '<button name=action value="batch">Подать выбранные</button>'
        '<span class=meta> — по одной с паузой, не больше %d за раз. '
        'Галочка доступна только там, где есть черновик.</span></p>'
        % (esc(account), esc(f_state), esc(f_basis), per, BATCH_LIMIT))
    script = ("<script>function pickAll(v){document.querySelectorAll("
              "'input[name=pick]:not([disabled])').forEach(function(c){c.checked=v})}</script>")
    table = ('<form method=post action="/submit">' + controls +
             '<table><tr><th></th><th>Отзыв</th><th>Товар</th><th>Текст</th>'
             '<th>Разметка AI</th><th>Текст жалобы</th></tr>%s</table>'
             % "".join(rows) + controls + '</form>' + script)
    return page("Очередь", head + table, account)


@app.route("/submit", methods=["POST"])
@auth_required
def submit():
    account = acc()
    db = store.connect()
    action = request.form.get("action") or ""

    # фильтры и размер страницы, чтобы вернуться туда же, откуда отправляли
    keep = {"acc": account,
            "state": request.form.get("state") or None,
            "basis": request.form.get("basis_filter") or None,
            "per": request.form.get("per") or None}
    keep = {k: v for k, v in keep.items() if v}

    if action.startswith("skip:"):
        rid = action[5:]
        store.decide(db, rid, "skipped", request.form.get("basis_" + rid), None, 0)
        return redirect(url_for("queue", **keep))

    if action.startswith("one:"):
        ids = [action[4:]]
    elif action == "batch":
        picked = request.form.getlist("pick")
        ids = picked[:BATCH_LIMIT]
    else:
        picked, ids = [], []
    left = len(picked) - len(ids) if action == "batch" else 0

    # Перед отправкой сверяемся с кабинетом: на отзыв можно пожаловаться один раз,
    # повторная попытка возвращает invalid-request-input и засоряет журнал.
    already = 0
    if ids:
        filed = set(cabinet.scan_complaints(account, pages=3))
        skip_ids = [r for r in ids if r in filed]
        if skip_ids:
            already = store.mark_already_filed(db, skip_ids)
            ids = [r for r in ids if r not in filed]

    # Жалоба необратима, поэтому подаём только те отзывы, которые в базе числятся
    # за открытым сейчас кабинетом: подставленный чужой id дальше не пройдёт.
    if ids:
        marks = ",".join("?" * len(ids))
        mine = {r["id"] for r in db.execute(
            "SELECT id FROM reviews WHERE account=? AND id IN (%s)" % marks,
            [account] + ids)}
        ids = [r for r in ids if r in mine]

    sent, failed, last_error = 0, 0, ""
    for number, rid in enumerate(ids):
        text = (request.form.get("text_" + rid) or "").strip()
        basis = request.form.get("basis_" + rid)
        reason = cabinet.reason_for(basis)   # причина — та, что выбрана в списке
        ok, error = cabinet.submit_complaint(account, rid, text, reason)
        store.record_submission(db, rid, basis, reason, text, 0, ok, error)
        if ok:
            sent += 1
        else:
            failed += 1
            last_error = error
        if number < len(ids) - 1:
            time.sleep(BATCH_PAUSE)

    if not ids and already:
        msg = "ok:Все выбранные отзывы уже обжалованы (%d) — убрал их из очереди." % already
    elif not ids:
        msg = "err:не выбрано ни одного отзыва"
    elif failed and sent:
        msg = "ok:Подано %d, не прошло %d. Последняя ошибка: %s" % (sent, failed, last_error)
    elif failed:
        msg = "err:%s" % last_error
    else:
        msg = "ok:Подано жалоб: %d. Статус — на рассмотрении." % sent
    if already:
        msg += " Пропущено, потому что жалоба уже была подана: %d." % already
    if left:
        msg += " Осталось выбрано %d — нажмите «Подать выбранные» ещё раз." % left
    return redirect(url_for("queue", msg=msg, **keep))


@app.route("/outcomes")
@auth_required
def outcomes():
    account = acc()
    db = store.connect()
    items = store.pending_outcomes(db, account)
    rows = []
    for it in items:
        rows.append("""<tr><td class=meta>%s</td><td class=meta>%s<br>%s<br>%s★</td>
          <td>%s</td><td class=meta>%s</td>
          <td class=meta>%s</td></tr>""" % (
            esc(it["decided_at"][:16].replace("T", " ")), esc(it["supplier_article"]),
            wb_link(it.get("nm_id")), esc(it["valuation"]), esc(it["text"]) or "<i>без текста</i>",
            esc(it["basis"]), {"review":"на рассмотрении","approved":"ОДОБРЕНА",
             "rejected":"отклонена"}.get(it["outcome"], it["outcome"] or "—")))
    report = store.daily_report(db, account)
    report_rows = []
    for day in report:
        done = (day["approved"] or 0) + (day["rejected"] or 0)
        share = ("%d%%" % round(100 * (day["approved"] or 0) / done)) if done else "—"
        report_rows.append(
            "<tr><td>%s</td><td>%d</td><td class=ok>%d</td><td class=no>%d</td>"
            "<td class=meta>%d</td><td><b>%s</b></td></tr>" % (
                esc(day["day"]), day["submitted"], day["approved"] or 0,
                day["rejected"] or 0, day["review"] or 0, share))
    totals = {k: sum(d[k] or 0 for d in report)
              for k in ("submitted", "approved", "rejected", "review")}
    done = totals["approved"] + totals["rejected"]
    total_share = ("%d%%" % round(100 * totals["approved"] / done)) if done else "—"
    # Какие основания WB одобряет чаще — по этому видно, куда смещать разметку AI.
    basis_rows = []
    for b in store.basis_report(db, account, 30):
        done = (b["approved"] or 0) + (b["rejected"] or 0)
        share = round(100 * (b["approved"] or 0) / done) if done else None
        colour = "ok" if (share or 0) >= 60 else ("no" if share is not None and share < 35
                                                  else "warn")
        basis_rows.append(
            "<tr><td>%s</td><td>%d</td><td class=ok>%d</td><td class=no>%d</td>"
            "<td class=meta>%d</td><td class=%s><b>%s</b></td></tr>"
            % (esc(b["basis"] or "—"), b["submitted"], b["approved"] or 0,
               b["rejected"] or 0, b["review"] or 0, colour,
               ("%d%%" % share) if share is not None else "—"))
    basis_block = (
        "<h2>Эффективность оснований за 30 дней</h2>"
        "<table><tr><th>Основание</th><th>Подано</th><th>Одобрено</th><th>Отклонено</th>"
        "<th>На рассмотрении</th><th>Конверсия</th></tr>%s</table>"
        "<p class=meta>Конверсия считается только по решённым жалобам. Основания с низкой "
        "конверсией стоит применять реже: на отзыв можно пожаловаться лишь один раз, и "
        "неудачное основание сжигает эту попытку.</p>"
        % ("".join(basis_rows) or "<tr><td colspan=6>За 30 дней жалоб нет</td></tr>"))

    report_block = (
        basis_block +
        "<h2>Отчёт по дням</h2>"
        "<table><tr><th>Дата</th><th>Подано</th><th>Одобрено</th><th>Отклонено</th>"
        "<th>На рассмотрении</th><th>Конверсия</th></tr>%s"
        "<tr><td><b>Итого</b></td><td><b>%d</b></td><td class=ok><b>%d</b></td>"
        "<td class=no><b>%d</b></td><td class=meta><b>%d</b></td><td><b>%s</b></td></tr>"
        "</table><p class=meta>Дата московская. Конверсия считается только по решённым: "
        "одобренные к сумме одобренных и отклонённых.</p>" % (
            "".join(report_rows) or "<tr><td colspan=6>Жалоб пока нет</td></tr>",
            totals["submitted"], totals["approved"], totals["rejected"],
            totals["review"], total_share))

    body = (report_block +
            '<h2>Поданные жалобы</h2>'
            '<form method=post action="/sync"><input type=hidden name=acc value="%s">'
            '<button>Обновить статусы из кабинета</button></form>'
            '<p class=meta>Поданные жалобы: %d. Статус приходит из кабинета, вручную ничего '
            'отмечать не нужно.</p>' % (esc(account), len(items))) + (
        "<table><tr><th>Подана</th><th>Товар</th><th>Текст отзыва</th>"
        "<th>Основание</th><th>Исход</th></tr>%s</table>" % "".join(rows)
        if rows else "<p>Нет жалоб, ожидающих исхода.</p>")
    return page("Исходы жалоб", body, account)


@app.route("/sync", methods=["POST"])
@auth_required
def sync():
    account = acc()
    db = store.connect()
    store.sync_outcomes(db, cabinet.scan_complaints(account, pages=10))
    return redirect(url_for("outcomes", acc=account))


@app.route("/session", methods=["GET", "POST"])
@auth_required
def session_page():
    account = acc()
    message = ""
    if request.method == "POST":
        found = cabinet.parse_curl(request.form.get("curl") or "")
        cookies = cabinet.parse_cookies(request.form.get("cookies") or "")
        if cookies:
            found["cookie"] = cookies
            supplier = re.search(r"x-supplier-id=([0-9a-f-]{36})", cookies)
            if supplier:
                found["x-supplier-id"] = supplier.group(1)
        if "authorizev3" not in found:
            message = ("<p class=no>В первом поле не нашёл заголовок authorizev3. "
                       "Возьмите запрос к <b>seller.wildberries.ru</b> "
                       "через «Copy as cURL (bash)».</p>")
        elif not found.get("cookie"):
            message = ("<p class=no>Нет куки. Chrome вырезает их из «Copy as cURL» — "
                       "заполните второе поле.</p>")
        else:
            cabinet.save_session(found)
            message = "<p class=ok>Сессия сохранена. Проверяю кабинет…</p>"

    body = message
    current = cabinet.mask()
    if current:
        # Сессия одна на все юрлица, поэтому сразу показываем, что видно в каждом:
        # так сразу заметно, если WB перестал отдавать какой-то кабинет.
        checks = []
        for key in accounts.KEYS:
            try:
                status, data = cabinet.call(
                    key, "/feedbacks?take=100&skip=0&isAnswered=false")
                seen = len((data.get("data") or {}).get("feedbacks") or [])
                checks.append("<li>%s — %s</li>" % (
                    esc(accounts.NAME[key]),
                    ("видно отзывов: %d" % seen) if status == 200 and seen
                    else "<span class=no>не отвечает (%s)</span>" % status))
            except Exception as e:
                checks.append("<li>%s — <span class=no>%s</span></li>"
                              % (esc(accounts.NAME[key]), esc(e)))
        body += ("<div class=box><b>Кабинеты, доступные этой сессии:</b><ul>%s</ul>"
                 "<p class=meta>Сессия вставляется один раз — она открывает все юрлица "
                 "сразу.</p></div>" % "".join(checks))

        check = cabinet.verify(account)
        if check["ok"]:
            rows = "".join("<li><b>%s</b> — %s%s</li>" % (
                esc(r.get("id")), esc(r.get("label")),
                " (пояснение обязательно)" if r.get("explanationRequired") else "")
                for r in check["reasons"])
            verdict = ("Кабинет тот самый: из %d отзывов совпало с нашей базой %d."
                       % (check["seen"], check["matched"]) if check["matched"]
                       else "Кабинет отвечает (отзывов видно %d), но сверить пока не с чем: "
                            "%s." % (check["seen"], check["problem"] or "база пуста"))
            body += ("<div class=box><p class=ok>%s</p><p class=meta>%s</p>"
                     "<b>Причины жалобы, доступные в этом кабинете:</b><ul>%s</ul></div>"
                     % (esc(verdict), esc(current), rows))
        else:
            body += ("<div class=box><p class=no>Сессия задана (%s), но проверку не прошла: "
                     "%s</p></div>" % (esc(current), esc(check["problem"])))
    else:
        body += "<div class=box><p class=no>Сессия кабинета не задана.</p></div>"

    body += """<div class=box>
      <form method=post>
      <input type=hidden name=acc value="%s">
      <b>1. Заголовки</b>""" % esc(account) + """
      <ol>
        <li>Кабинет WB, раздел отзывов. F12 → <b>Network</b> → фильтр <b>Fetch/XHR</b>.</li>
        <li>Ctrl+Shift+R, выберите любой запрос к <b>seller.wildberries.ru</b>.</li>
        <li>Правой кнопкой → <b>Copy</b> → <b>Copy as cURL (bash)</b>, вставьте ниже.</li>
      </ol>
      <textarea name=curl rows=5 style="width:100%;font-family:monospace;font-size:12px"
        placeholder="curl 'https://seller.wildberries.ru/...' -H 'AuthorizeV3: ...' ..."></textarea>

      <b>2. Куки</b>
      <p class=meta>Chrome вырезает куки из «Copy as cURL», поэтому их нужно взять отдельно.
      Проще всего расширением <b>Cookie-Editor</b>: открыть его на странице кабинета →
      <b>Export</b> → <b>Export as JSON</b> → вставить сюда. Подойдёт и строка вида
      <code>name=value; name=value</code> из вкладки Application → Cookies.</p>
      <textarea name=cookies rows=5 style="width:100%;font-family:monospace;font-size:12px"
        placeholder='[{"name":"...","value":"..."}, ...]   или   name=value; name=value'></textarea>

      <button type=submit style="margin-top:8px">Сохранить сессию</button>
      </form>
      <p class=meta>Берутся только authorizev3, wb-seller-lk, cookie и x-supplier-id.
      Файл получает права 600, значения обратно не показываются.</p></div>"""
    return page("Сессия кабинета", body, account)


# ---------------------------------------------------------------- служебное

@app.route("/tools")
@auth_required
def tools():
    account = acc()
    running = subprocess.run(["pgrep", "-f", SCRIPT], capture_output=True).returncode == 0
    log = open(LOG_FILE, encoding="utf-8", errors="replace").read() \
        if os.path.exists(LOG_FILE) else "запусков не было"
    files = ""
    for path, label, endpoint in ((REPORT, "Отчёт по полям", "get_report"),
                                  (CSV, "Таблица отзывов", "get_csv")):
        files += ('<li><a href="%s">%s</a></li>' % (url_for(endpoint), label)
                  if os.path.exists(path) else "<li>%s — нет</li>" % label)

    # Токен API отзывов свой у каждого юрлица — вводятся тут же, по одному на кабинет.
    tokens = ""
    for key in accounts.KEYS:
        mask = token_mask(key)
        tokens += ("""<p><b>%s</b> — %s<br>
          <form method=post action="/token">
           <input type=hidden name=acc value="%s">
           <input type=password name=token placeholder="вставьте токен" autocomplete=off
             style="width:340px;padding:7px">
           <button type=submit>Сохранить</button></form></p>"""
          % (esc(accounts.NAME[key]),
             ("задан " + mask) if mask else "<span class=no>НЕ ЗАДАН</span>", esc(key)))

    body = ("<div class=box><b>Токены Wildberries (API отзывов)</b>%s</div>"
            """<div class=box><b>ТЕСТ 1</b> — разовая проверка состава полей API.<br>
      <form method=post action="/run" style="margin-top:8px">
       <button type=submit %s>%s</button></form></div>
     <ul>%s</ul><h2>Журнал</h2><pre>%s</pre>%s""") % (
        tokens,
        "disabled" if running or not token_mask(account) else "",
        "Выполняется…" if running else "Запустить ТЕСТ 1",
        files, esc(log), '<meta http-equiv=refresh content=3>' if running else "")
    return page("Служебное", body, account)


@app.route("/token", methods=["POST"])
@auth_required
def set_token():
    account = acc()
    value = (request.form.get("token") or "").strip()
    if value:
        name = accounts.token_var(account)
        lines = []
        if os.path.exists(ENV_FILE):
            lines = [l for l in open(ENV_FILE, encoding="utf-8-sig")
                     if not l.startswith(name + "=")]
        lines.append("%s=%s\n" % (name, value))
        with open(ENV_FILE, "w", encoding="utf-8") as f:
            f.writelines(lines)
        os.chmod(ENV_FILE, 0o600)
    return redirect(url_for("tools", acc=account))


@app.route("/run", methods=["POST"])
@auth_required
def run():
    if not token_mask():
        return redirect(url_for("tools"))
    log = open(LOG_FILE, "w", encoding="utf-8")
    log.write("=== запуск %s ===\n" % datetime.now().strftime("%d.%m.%Y %H:%M:%S"))
    log.flush()
    subprocess.Popen(["python3", "-u", SCRIPT], cwd=HERE, stdout=log, stderr=subprocess.STDOUT)
    time.sleep(1)
    return redirect(url_for("tools"))


@app.route("/report")
@auth_required
def get_report():
    return send_file(REPORT, mimetype="text/plain; charset=utf-8")


@app.route("/csv")
@auth_required
def get_csv():
    return send_file(CSV, as_attachment=True, download_name="feedbacks.csv")


if __name__ == "__main__":
    app.run(host="127.0.0.1", port=5096)
