"""
Итог автоподачи жалоб за сутки — одним сообщением по всем кабинетам.

Заменяет отчёт после каждого прогона: три прогона на три кабинета давали до
девяти сообщений в день. Сбои (сессия, сеть) по-прежнему приходят сразу из
autosubmit.py — это тревога, а не отчёт.

Запуск: wb-reviews-report.timer, 20:30 МСК — после вечернего прогона в 19:00,
который идёт 40–50 минут. `--dry` печатает без отправки.
"""

import sys
from datetime import datetime, time, timedelta, timezone
from zoneinfo import ZoneInfo

import accounts
import store
import watchdog
from autosubmit import PANEL_URL, rate_text

MSK = ZoneInfo("Europe/Moscow")


def build(db, day):
    start = datetime.combine(day, time(), MSK).astimezone(timezone.utc)
    # decided_at хранится ISO-строкой в UTC — сравниваем строками того же формата
    period = tuple(t.isoformat(timespec="seconds") for t in (start, start + timedelta(days=1)))

    parts = ["Автоподача жалоб WB — итог за %s" % day.strftime("%d.%m")]
    total_sent = total_failed = 0
    for acc in accounts.KEYS:
        done = {r["decision"]: r["n"] for r in db.execute("""
            SELECT c.decision, count(*) n FROM complaints c
            JOIN reviews r ON r.id = c.review_id
            WHERE r.account = ? AND c.basis_source = 'panel'
              AND c.decided_at >= ? AND c.decided_at < ?
            GROUP BY c.decision""", (acc,) + period)}
        sent, failed = done.get("submitted", 0), done.get("failed", 0)
        total_sent, total_failed = total_sent + sent, total_failed + failed
        counters = store.counters(db, acc)
        parts.append("%s\nПодано: %d · Не прошло: %d · В очереди: %d\n"
                     "Одобряемость: %s\n"
                     "За всё время: одобрено %d, отклонено %d, на рассмотрении %d"
                     % (accounts.NAME[acc], sent, failed, counters["queue"],
                        rate_text(db, acc), counters["approved"],
                        counters["rejected"], counters["pending"]))
    parts.append("Итого за сутки: подано %d, не прошло %d\n%s"
                 % (total_sent, total_failed, PANEL_URL))
    return "\n\n".join(parts)


if __name__ == "__main__":
    text = build(store.connect(), datetime.now(MSK).date())
    print(text)
    if "--dry" not in sys.argv:
        watchdog.notify(text)
