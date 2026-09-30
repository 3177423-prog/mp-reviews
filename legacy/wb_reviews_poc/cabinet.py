"""
Работа с кабинетами продавца WB по сохранённой сессии.

Сессия вводится через панель (вставкой «Copy as cURL»), хранится в файле с правами 600
и в интерфейсе показывается только маской. Сессия одна на все юрлица: кабинет
выбирается подстановкой supplier-id в куки — см. accounts.py.
"""

import json
import os
import re
import time
import urllib.error
import urllib.request

import accounts
import store

HERE = os.path.dirname(os.path.abspath(__file__))
SESSION_FILE = os.path.join(HERE, "data", "cabinet_session.json")

API = "https://seller-reviews.wildberries.ru/ns/fa-seller-api/reviews-ext-seller-portal/api/v2"
NEEDED = ("authorizev3", "wb-seller-lk", "cookie", "x-supplier-id")
UA = ("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "
      "(KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36")


def parse_curl(text):
    """Достаёт нужные заголовки из строки, скопированной как cURL."""
    found = {}
    for name, value in re.findall(r"-H\s+'([^:]+):\s*([^']*)'", text) + \
                       re.findall(r'-H\s+"([^:]+):\s*([^"]*)"', text):
        key = name.strip().lower()
        if key in NEEDED:
            found[key] = value.strip()
    match = re.search(r"-b\s+'([^']*)'", text) or re.search(r'-b\s+"([^"]*)"', text)
    if match and "cookie" not in found:
        found["cookie"] = match.group(1).strip()
    if "x-supplier-id" not in found and found.get("cookie"):
        m = re.search(r"x-supplier-id=([0-9a-f-]{36})", found["cookie"])
        if m:
            found["x-supplier-id"] = m.group(1)
    return found


def parse_cookies(text):
    """Принимает либо JSON-выгрузку расширения Cookie-Editor, либо строку name=value; ..."""
    text = (text or "").strip()
    if not text:
        return ""
    if text.startswith("["):
        try:
            items = json.loads(text)
        except json.JSONDecodeError:
            return ""
        return "; ".join("%s=%s" % (c.get("name"), c.get("value"))
                         for c in items if c.get("name"))
    return re.sub(r"^\s*(-b\s*|cookie:\s*)", "", text, flags=re.I).strip().strip("'\"")


def save_session(headers):
    os.makedirs(os.path.dirname(SESSION_FILE), exist_ok=True)
    with open(SESSION_FILE, "w", encoding="utf-8") as f:
        json.dump(headers, f)
    os.chmod(SESSION_FILE, 0o600)


def load_session():
    if not os.path.exists(SESSION_FILE):
        return None
    return json.load(open(SESSION_FILE, encoding="utf-8"))


def mask():
    session = load_session()
    if not session:
        return None
    return "поставщик %s, ключ %s…" % (session.get("x-supplier-id", "?"),
                                       (session.get("authorizev3") or "")[:8])


def switch_cookie(cookie, supplier):
    """Переставляет сессию на нужное юрлицо. Кабинет определяют обе куки:
    x-supplier-id-external — главная, без неё список отзывов приходит пустым."""
    cookie, external = re.subn(r"x-supplier-id-external=[0-9a-f-]{36}",
                               "x-supplier-id-external=" + supplier, cookie)
    cookie, plain = re.subn(r"x-supplier-id=[0-9a-f-]{36}",
                            "x-supplier-id=" + supplier, cookie)
    # Молча оставить чужой кабинет нельзя: жалоба ушла бы не тому юрлицу и сгорела бы,
    # подать её второй раз WB уже не даст.
    if not external or not plain:
        raise RuntimeError("в куках нет x-supplier-id — сессию нужно вставить заново")
    return cookie


def headers(acc):
    session = load_session()
    if not session:
        raise RuntimeError("сессия кабинета не задана")
    supplier = accounts.SUPPLIER[accounts.valid(acc)]
    head = {"accept": "*/*", "content-type": "application/json",
            "origin": "https://seller.wildberries.ru",
            "referer": "https://seller.wildberries.ru/feedbacks/feedbacks-tab/not-answered",
            "user-agent": UA}
    head.update({k: v for k, v in session.items() if k in NEEDED})
    head["cookie"] = switch_cookie(session.get("cookie") or "", supplier)
    head["x-supplier-id"] = supplier
    return head


# Сетевой сбой — не ошибка кабинета, а обрыв связи. Раньше он вылетал наружу
# и убивал весь прогон: 10.09.2026 подача по Маркетспейсу оборвалась на середине
# очереди из 150 жалоб, и отчёт не ушёл вообще.
NET_RETRIES = 3
NET_ERROR = "сеть недоступна"


def call(acc, path, method="GET", body=None):
    last = ""
    for attempt in range(NET_RETRIES):
        request = urllib.request.Request(
            API + path, method=method, headers=headers(acc),
            data=json.dumps(body).encode() if body is not None else None)
        try:
            with urllib.request.urlopen(request, timeout=60) as response:
                return response.status, json.loads(response.read().decode("utf-8"))
        except urllib.error.HTTPError as e:
            return e.code, {"errorText": e.read(300).decode("utf-8", "replace")}
        except (urllib.error.URLError, TimeoutError, OSError) as e:
            last = str(e)[:150]
            if attempt < NET_RETRIES - 1:
                time.sleep(5 * (attempt + 1))
    return 0, {"errorText": "%s: %s" % (NET_ERROR, last)}


def reasons(acc):
    """Справочник причин жалобы на отзыв для этого кабинета."""
    status, data = call(acc, "/feedbacks/complaints/reasons")
    if status != 200 or data.get("error"):
        return []
    return (data.get("data") or {}).get("feedbackComplaints") or []


REASON_DEFAULT = 19          # «Другое» — когда конкретная причина не подходит

# Справочник причин ровно как в форме кабинета WB.
# Коды подтверждены на живых жалобах: 11, 12, 16, 19, 20.
# Два кода пока не известны — их узнаем, когда такую жалобу подадут руками.
REASONS = [
    ("Отзыв оставили конкуренты",        12),
    ("Отзыв не относится к товару",      11),
    ("Спам-реклама в тексте",          None),
    ("Нецензурная лексика",              16),
    ("Отзыв с политическим контекстом", None),
    ("Угрозы, оскорбления",              20),
    ("Другое",                           19),
]
REASON_BY_NAME = {name: code for name, code in REASONS}


def reason_for(name):
    """Код причины по её названию из формы кабинета. None — код ещё не известен."""
    key = (name or "").strip().lower()
    for label, code in REASONS:
        if label.lower() == key:
            return code
    return REASON_DEFAULT


def submit_complaint(acc, feedback_id, explanation, reason_id=REASON_DEFAULT, attempt=1):
    """Подаёт одну жалобу. Возвращает (успех, текст ошибки).
    На 429 (слишком часто) один раз ждём 10 секунд и пробуем снова."""
    if reason_id is None:
        return False, ("код этой причины ещё не известен — подайте одну такую жалобу "
                       "руками из кабинета, и код прочитается автоматически")
    explanation = (explanation or "").strip()
    if not explanation:
        return False, "пустое пояснение — жалоба не отправлена"
    if len(explanation) > 1000:
        return False, "пояснение длиннее 1000 символов"
    status, data = call(acc, "/feedbacks/complaints", method="PATCH", body={
        "feedbackId": feedback_id,
        "feedbackComplaint": {"id": reason_id, "explanation": explanation}})
    if status == 429 and attempt == 1:
        time.sleep(10)
        return submit_complaint(acc, feedback_id, explanation, reason_id, attempt + 1)
    if status != 200:
        return False, "кабинет ответил %s: %s" % (status, str(data.get("errorText"))[:200])
    if data.get("error"):
        return False, str(data.get("errorText") or "кабинет отклонил запрос")[:200]
    return True, ""


def complaint_state(acc, feedback_id):
    """Состояние жалобы по одному отзыву, если он ещё виден в кабинете."""
    for answered in ("false", "true"):
        for skip in range(0, 500, 100):
            status, data = call(
                acc, "/feedbacks?take=100&skip=%d&isAnswered=%s" % (skip, answered))
            feedbacks = (data.get("data") or {}).get("feedbacks") or []
            if not feedbacks:
                break
            for f in feedbacks:
                if f["id"] == feedback_id:
                    return (f.get("supplierComplaints") or {}).get("feedbackComplaint") or {}
    return {}


def scan_complaints(acc, pages=15):
    """Раздел жалоб кабинета: id отзыва -> {status, id, explanation}.

    isAnswered=false — жалобы на рассмотрении, true — уже обработанные.
    """
    found = {}
    for answered in ("false", "true"):
        for skip in range(0, pages * 100, 100):
            status, data = call(acc, "/feedbacks/complaints?take=100&skip=%d&isAnswered=%s"
                                % (skip, answered))
            feedbacks = (data.get("data") or {}).get("feedbacks") or []
            if not feedbacks:
                break
            before = len(found)
            for f in feedbacks:
                complaint = (f.get("supplierComplaints") or {}).get("feedbackComplaint") or {}
                if complaint.get("id"):
                    found[f["id"]] = complaint
            if len(found) == before:      # страницы пошли по кругу — дальше смысла нет
                break
    return found


def verify(acc):
    """Проверяет, что сессия живая и открывает именно тот кабинет, отзывы которого
    лежат в базе под этим ключом."""
    status, data = call(acc, "/feedbacks?take=100&skip=0&isAnswered=false")
    if status != 200 or data.get("error"):
        return {"ok": False, "problem": "кабинет не ответил (%s)" % status}
    feedbacks = (data.get("data") or {}).get("feedbacks") or []
    db = store.connect()
    ours = {r["id"] for r in db.execute(
        "SELECT id FROM reviews WHERE account=?", (accounts.valid(acc),))}
    matched = len({f["id"] for f in feedbacks} & ours)
    problem = None
    if not feedbacks:
        problem = "кабинет вернул пустой список отзывов"
    elif not ours:
        problem = "по этому кабинету в базе ещё нет отзывов — сверять не с чем"
    elif not matched:
        problem = "ни один отзыв не совпал с нашей базой — это другой кабинет"
    return {"ok": bool(feedbacks) and (matched > 0 or not ours),
            "seen": len(feedbacks), "matched": matched,
            "reasons": reasons(acc), "problem": problem}
