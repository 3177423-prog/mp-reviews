"""
Автоподача жалоб: три раза в день забирает очередь и отправляет жалобы в кабинет.

Кабинет задаётся переменной окружения WB_ACCOUNT (ms | cr | hb).

Порядок тот же, что в панели: приоритет по ступеням, затем сила основания, затем свежесть.
Отправляются только отзывы с готовым черновиком и известным кодом причины.
Итог каждого прогона уходит в Telegram.
"""

import fcntl
import os
import sys
import time
from datetime import datetime

import accounts
import cabinet
import store
import watchdog

HERE = os.path.dirname(os.path.abspath(__file__))
PER_RUN = 150          # потолок за прогон, 450 в сутки — вдвое выше притока (~225/день).
                       # Лимита на число жалоб у WB нет: 326 за день прошли без последствий.
                       # Барьер оставлен как страховка от разгона при сбое разметки.
PAUSE = 2.0            # пауза между жалобами, секунды
PANEL_URL = "https://reviews.wild-expert.ru"


def log(message):
    print("%s  %s" % (datetime.now().strftime("%H:%M:%S"), message), flush=True)


def rate_text(db, acc, days=30):
    """Одобряемость за последние days дней и сдвиг относительно предыдущих days дней.

    Считаем только по решённым: пока WB не ответил, жалоба не говорит ни за, ни против.
    """
    approved, decided = store.rate_window(db, acc, days)
    if not decided:
        return "за %d дней решённых жалоб нет" % days
    now_rate = 100.0 * approved / decided
    text = "%d%% (%d из %d решённых за %d дней)" % (round(now_rate), approved, decided, days)

    was_approved, was_decided = store.rate_window(db, acc, days, offset_days=days)
    if was_decided:
        was_rate = 100.0 * was_approved / was_decided
        delta = round(now_rate) - round(was_rate)
        sign = "+" if delta > 0 else ("" if delta else "±")
        text += "\nПредыдущие %d дней: %d%% (%d из %d) — %s%d п.п." % (
            days, round(was_rate), was_approved, was_decided, sign, delta)
    else:
        text += "\nПредыдущие %d дней: сравнивать не с чем" % days
    return text


def pick(db, acc):
    """Что подаём: есть черновик, известен код причины, жалоба ещё не подана."""
    ready = []
    for item in store.queue(db, acc):
        draft = (item.get("ai_draft") or "").strip()
        basis = (item.get("ai_basis") or "").strip()
        if not draft or not basis:
            continue
        # названия причин в базе в нижнем регистре, в справочнике — как в кабинете
        label = next((name for name, _ in cabinet.REASONS
                      if name.lower() == basis.lower()), None)
        if not label or cabinet.reason_for(label) is None:
            continue
        ready.append((item, label))
        if len(ready) >= PER_RUN:
            break
    return ready


if __name__ == "__main__":
    acc = accounts.current()
    lock = open(os.path.join(HERE, ".autosubmit.%s.lock" % acc), "w")
    try:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except BlockingIOError:
        sys.exit("подача уже выполняется")

    db = store.connect()
    log("кабинет: %s" % accounts.NAME[acc])

    # Живость сессии проверяем ЯВНО: cabinet.call не бросает исключение на 401,
    # и без этой проверки прогон молотил бы всю очередь об один и тот же отказ.
    try:
        status, _ = cabinet.call(acc, "/feedbacks?take=100&skip=0&isAnswered=false")
    except Exception as e:
        status = type(e).__name__
    if status != 200:
        log("кабинет ответил %s — подачу отменяю" % status)
        watchdog.notify("Автоподача жалоб не состоялась (%s): кабинет ответил %s.\n"
                        "Обновите сессию: %s/session"
                        % (accounts.NAME[acc], status, PANEL_URL))
        sys.exit(1)

    # Сверка с кабинетом до подачи: отзывы с уже поданной жалобой из очереди убираем,
    # иначе потратим запрос впустую и получим invalid-request-input.
    found = cabinet.scan_complaints(acc, pages=3)
    store.mark_already_filed(db, set(found))
    store.sync_outcomes(db, found)

    ready = pick(db, acc)
    log("к подаче: %d" % len(ready))

    sent, failed, last_error = 0, 0, ""
    for number, (item, label) in enumerate(ready):
        ok, error = cabinet.submit_complaint(
            acc, item["id"], item["ai_draft"], cabinet.reason_for(label))

        # Обрыв связи — не отказ кабинета. Записывать такую попытку нельзя:
        # жалоба не подана, а отзыв ушёл бы из очереди навсегда. Останавливаемся,
        # очередь остаётся нетронутой и уйдёт в следующий прогон.
        if not ok and cabinet.NET_ERROR in error:
            log("сеть недоступна, останавливаю прогон: %s" % error[:90])
            watchdog.notify("Автоподача остановлена (%s): пропала связь с кабинетом.\n"
                            "Подано до сбоя: %d\nОсталось в очереди: %d\n"
                            "Очередь не потеряна, подача продолжится в следующем прогоне."
                            % (accounts.NAME[acc], sent, len(ready) - number))
            break

        store.record_submission(db, item["id"], label, cabinet.reason_for(label),
                                item["ai_draft"], item.get("tier") or 0, ok, error)
        if ok:
            sent += 1
        else:
            failed += 1
            last_error = error
            log("  не прошла %s: %s" % (item["id"][:8], error[:80]))
            if "401" in error:      # сессия умерла посреди прогона — дальше бессмысленно
                log("сессия отвалилась, останавливаю прогон")
                watchdog.notify("Автоподача остановлена (%s): сессия кабинета отвалилась "
                                "посреди прогона.\nПодано до сбоя: %d\n"
                                "Обновите сессию: %s/session"
                                % (accounts.NAME[acc], sent, PANEL_URL))
                break
        if number < len(ready) - 1:
            time.sleep(PAUSE)

    # Итог в Telegram больше не шлём после каждого прогона — одна сводка за сутки
    # по всем кабинетам уходит из daily_report.py в 20:30 МСК.
    log("подано %d, не прошло %d%s" % (sent, failed,
        ("; последняя ошибка: %s" % last_error[:150]) if failed else ""))
