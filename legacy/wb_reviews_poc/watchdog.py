"""
Сторож: раз в час проверяет живость системы и пишет в Telegram, если что-то встало.

Проверяет по каждому кабинету:
  1. сессию кабинета — читающим запросом;
  2. токен API отзывов;
  3. разметку — не копится ли очередь без разметки;
  4. сбор — отрабатывал ли он.

Повторно об одной и той же поломке не пишет: состояние хранится в data/watchdog.json.
"""

import json
import os
import urllib.parse
import urllib.request
from datetime import datetime, timezone

import accounts
import cabinet
import store

HERE = os.path.dirname(os.path.abspath(__file__))
STATE_FILE = os.path.join(HERE, "data", "watchdog.json")
CHAT_ID = "-1003890737169"
PANEL_URL = "https://reviews.wild-expert.ru"

UNANALYZED_LIMIT = 150      # столько неразмеченных считаем затором
STALE_HOURS = 16            # столько часов без прогона сбора считаем остановкой.
                            # Цикл ходит в 09:00, 15:00 и 19:00 МСК — ночной
                            # перерыв 14 часов штатный, порог должен быть выше.


def bot_token():
    for line in open("/root/claude-bot/.env", encoding="utf-8-sig"):
        if line.startswith("BOT_TOKEN="):
            return line.split("=", 1)[1].strip()
    return ""


def notify(text):
    token = bot_token()
    if not token:
        print("нет BOT_TOKEN — уведомление не отправлено")
        return
    data = urllib.parse.urlencode({
        "chat_id": CHAT_ID, "text": text, "disable_web_page_preview": "true"}).encode()
    request = urllib.request.Request(
        "https://api.telegram.org/bot%s/sendMessage" % token, data=data)
    try:
        urllib.request.urlopen(request, timeout=30).read()
    except Exception as e:
        print("не удалось отправить уведомление: %s" % type(e).__name__)


def check_cabinet(acc):
    if not cabinet.load_session():
        return "сессия кабинета не задана — подача жалоб не работает"
    try:
        status, data = cabinet.call(acc, "/feedbacks?take=100&skip=0&isAnswered=false")
    except RuntimeError as e:
        return str(e)
    if status == 401:
        return "сессия кабинета протухла (401) — подача жалоб не работает"
    if status != 200 or data.get("error"):
        return "кабинет не отвечает (%s) — подача жалоб не работает" % status
    return None


def check_token(acc):
    path = os.path.join(HERE, ".env")
    name = accounts.token_var(acc)
    token = ""
    for line in open(path, encoding="utf-8-sig"):
        if line.startswith(name + "="):
            token = line.split("=", 1)[1].strip()
    if not token:
        return "токен API отзывов не задан (%s) — сбор не работает" % name
    request = urllib.request.Request(
        "https://feedbacks-api.wildberries.ru/api/v1/feedbacks/count-unanswered")
    request.add_header("Authorization", token)
    try:
        urllib.request.urlopen(request, timeout=40).read()
    except Exception as e:
        code = getattr(e, "code", None)
        return "API отзывов не отвечает (%s) — сбор не работает" % (code or type(e).__name__)
    return None


def check_pipeline(db, acc):
    problems = []
    counters = store.counters(db, acc)
    if counters["to_analyze"] > UNANALYZED_LIMIT:
        problems.append("разметка не успевает: без разметки %d отзывов"
                        % counters["to_analyze"])
    if counters["ai_failed"] > 20:
        problems.append("много неразобранных ответов AI: %d" % counters["ai_failed"])
    # Смотрим на факт прогона сбора, а не на появление новых отзывов: пустой прогон —
    # нормальная ситуация, а вот отсутствие прогонов означает, что цикл встал.
    last = store.last_run(db, "collector", acc)
    if last:
        stamp = datetime.fromisoformat(last)
        if stamp.tzinfo is None:
            stamp = stamp.replace(tzinfo=timezone.utc)
        hours = (datetime.now(timezone.utc) - stamp).total_seconds() / 3600
        if hours > STALE_HOURS:
            problems.append("сбор не отрабатывал уже %.0f ч — проверьте цикл" % hours)
    return problems


def load_state():
    if os.path.exists(STATE_FILE):
        try:
            return json.load(open(STATE_FILE, encoding="utf-8"))
        except json.JSONDecodeError:
            pass
    return {"active": []}


def save_state(state):
    os.makedirs(os.path.dirname(STATE_FILE), exist_ok=True)
    json.dump(state, open(STATE_FILE, "w", encoding="utf-8"), ensure_ascii=False)


if __name__ == "__main__":
    db = store.connect()

    # Сессия одна на все кабинеты: если она мертва, сообщаем об этом один раз,
    # а не тремя одинаковыми строками.
    session_problem = check_cabinet(accounts.DEFAULT)
    problems = [session_problem] if session_problem else []
    for acc in accounts.KEYS:
        local = check_pipeline(db, acc)
        token = check_token(acc)
        if token:
            local.append(token)
        if not session_problem and acc != accounts.DEFAULT:
            cabinet_problem = check_cabinet(acc)
            if cabinet_problem:
                local.append(cabinet_problem)
        problems += ["%s: %s" % (accounts.NAME[acc], p) for p in local]

    state = load_state()
    was = set(state.get("active") or [])
    now = set(problems)

    # о незакрытой поломке напоминаем каждые 3 часа, иначе одно сообщение легко пропустить
    last_reminder = state.get("reminded_at") or ""
    remind = False
    if now and last_reminder:
        try:
            gap = (datetime.now(timezone.utc)
                   - datetime.fromisoformat(last_reminder)).total_seconds()
            remind = gap > 3 * 3600
        except ValueError:
            remind = True
    if remind:
        was = set()          # покажем всё заново

    for problem in sorted(now - was):
        notify("Жалобы на отзывы WB — сбой\n\n%s\n\n%s" % (problem, PANEL_URL))
        print("сообщил о поломке: %s" % problem)
    if was and not now:
        notify("Жалобы на отзывы WB — всё восстановилось.\n%s" % PANEL_URL)
        print("сообщил о восстановлении")

    reminded = store.now() if (remind or (now - set(state.get("active") or []))) \
        else (state.get("reminded_at") or store.now())
    save_state({"active": sorted(now), "checked_at": store.now(),
                "reminded_at": reminded if now else ""})
    print("%s  проблем: %d" % (datetime.now().strftime("%H:%M:%S"), len(problems)))
    for p in problems:
        print("   " + p)
