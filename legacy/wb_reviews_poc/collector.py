"""
Сборщик: тянет новые отзывы 1-3★ из API Wildberries в базу
и раз в сутки пересчитывает рейтинг и число оценок по артикулам очереди.

Кабинет задаётся переменной окружения WB_ACCOUNT (ms | cr | hb).

Только GET. Ничего в WB не меняет. Запускается по таймеру systemd.
"""

import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime

import accounts
import store

BASE = "https://feedbacks-api.wildberries.ru"
PAUSE = 0.4                  # лимит WB — 3 запроса/сек
TIMEOUT = 120
MAX_VALUATION = 3
SKU_PER_RUN = 15             # сколько артикулов пересчитывать за один прогон
SKU_MAX_PAGES = 5            # потолок страниц на артикул (5 x 5000 отзывов)

HERE = os.path.dirname(os.path.abspath(__file__))
TOKEN = ""


def log(message):
    print("%s  %s" % (datetime.now().strftime("%H:%M:%S"), message), flush=True)


def load_token(acc):
    """У каждого кабинета свой токен API отзывов: WB_FEEDBACKS_TOKEN_MS и т.д."""
    path = os.path.join(HERE, ".env")
    name = accounts.token_var(acc)
    if not os.path.exists(path):
        sys.exit("ОШИБКА: нет файла .env с %s" % name)
    for line in open(path, encoding="utf-8-sig"):
        if line.startswith(name + "="):
            value = line.split("=", 1)[1].strip()
            if value:
                return value
    sys.exit("ОШИБКА: %s не задан — введите токен в панели" % name)


# Обрыв связи и перегрузка WB — временные помехи, а не повод падать.
# 15.09.2026 сбор Маркетспейса и Цифрового Ритейла умер на TimeoutError
# и URLError, и оба кабинета простояли 18 часов: sys.exit убивал скрипт,
# а разметка и подача идут следом по цепочке && и вообще не запускались.
NET_RETRIES = 4
RETRY_CODES = (429, 500, 502, 503, 504)


def get(path, params, attempt=1):
    url = BASE + path + "?" + urllib.parse.urlencode(params)
    last = ""
    for number in range(NET_RETRIES):
        request = urllib.request.Request(url)
        request.add_header("Authorization", TOKEN)
        try:
            with urllib.request.urlopen(request, timeout=TIMEOUT) as response:
                data = json.loads(response.read().decode("utf-8"))
            time.sleep(PAUSE)
            return (data.get("data") or {})
        except urllib.error.HTTPError as e:
            last = "HTTP %s" % e.code
            if e.code not in RETRY_CODES:
                sys.exit("ОШИБКА: %s вернул %s" % (path, e.code))
        except Exception as e:
            last = type(e).__name__
        if number < NET_RETRIES - 1:
            pause = 5 * (number + 1)
            log("  %s: %s, повтор через %d с" % (path, last, pause))
            time.sleep(pause)
    sys.exit("ОШИБКА: запрос %s не выполнен после %d попыток (%s)"
             % (path, NET_RETRIES, last))


def collect_new(db, acc):
    """Новые необработанные отзывы с оценкой не выше 3."""
    total, added = 0, 0
    for skip in range(0, 20000, 5000):
        data = get("/api/v1/feedbacks",
                   {"isAnswered": "false", "take": 5000, "skip": skip, "order": "dateDesc"})
        batch = data.get("feedbacks") or []
        if not batch:
            break
        total += len(batch)
        low = [f for f in batch if (f.get("productValuation") or 5) <= MAX_VALUATION]
        added += store.upsert_reviews(db, acc, low)
        if len(batch) < 5000:
            break
    log("просмотрено отзывов: %d, новых в очередь: %d" % (total, added))
    return added


def refresh_sku_stats(db, acc):
    """Рейтинг и число оценок по артикулу — из архива отзывов этого артикула."""
    ids = store.stale_sku_ids(db, acc, SKU_PER_RUN)
    for nm_id in ids:
        values, truncated = [], False
        for page in range(SKU_MAX_PAGES):
            batch = get("/api/v1/feedbacks/archive",
                        {"take": 5000, "skip": page * 5000, "order": "dateDesc",
                         "nmId": nm_id}).get("feedbacks") or []
            values += [f["productValuation"] for f in batch
                       if isinstance(f.get("productValuation"), int)]
            if len(batch) < 5000:
                break
            truncated = page == SKU_MAX_PAGES - 1
        if values:
            store.save_sku_stats(db, nm_id, len(values),
                                 round(sum(values) / len(values), 3), truncated)
    log("пересчитано артикулов: %d" % len(ids))


def sync_cabinet(db, acc):
    """Один заход в раздел жалоб кабинета решает две задачи:
    чистит очередь от уже обжалованных и подтягивает исходы жалоб."""
    try:
        import cabinet
        found = cabinet.scan_complaints(acc, pages=3)
    except Exception as e:
        log("кабинет недоступен, сверку пропускаю (%s)" % type(e).__name__)
        return
    removed = store.mark_already_filed(db, set(found))
    updated = store.sync_outcomes(db, found)
    log("сверка с кабинетом: убрано из очереди %d, обновлено исходов %d"
        % (removed, updated))


if __name__ == "__main__":
    ACC = accounts.current()
    log("кабинет: %s" % accounts.NAME[ACC])
    TOKEN = load_token(ACC)
    db = store.connect()
    collect_new(db, ACC)
    sync_cabinet(db, ACC)
    refresh_sku_stats(db, ACC)
    store.mark_run(db, "collector", ACC)
    log("итог: %s" % store.counters(db, ACC))
