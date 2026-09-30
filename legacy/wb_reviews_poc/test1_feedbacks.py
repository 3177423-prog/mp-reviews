"""
ТЕСТ 1 — proof of concept получения отзывов из официального API Wildberries.

Только чтение (GET). Ничего в WB не меняет.
Токен берётся из переменной окружения WB_FEEDBACKS_TOKEN или из файла .env рядом со скриптом.
Токен нигде не печатается и не сохраняется в результатах.

Запуск:  python3 test1_feedbacks.py
"""

import csv
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime

BASE = "https://feedbacks-api.wildberries.ru"
TAKE = 100
PAUSE = 0.4          # лимит WB — 3 запроса/сек
TIMEOUT = 60

HERE = os.path.dirname(os.path.abspath(__file__))
RAW = os.path.join(HERE, "raw")
NORM = os.path.join(HERE, "normalized")
REPORT = os.path.join(HERE, "report")

# Поля, значения которых не показываем ни в консоли, ни в отчёте, ни в CSV.
SENSITIVE = {"text", "pros", "cons", "userName", "photoLinks", "video",
             "answer.text", "video.link", "video.previewImage"}

TOKEN = ""


def load_env():
    """Подхватывает .env рядом со скриптом, если переменной ещё нет в окружении."""
    path = os.path.join(HERE, ".env")
    if not os.path.exists(path):
        return
    with open(path, encoding="utf-8-sig") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            os.environ.setdefault(key.strip(), value.strip())


def die(message):
    print("ОШИБКА: " + message)
    sys.exit(1)


def save_raw(label, url, params, status, body, elapsed):
    payload = {
        "label": label,
        "url": url,                # токена в URL нет, он идёт заголовком
        "params": params,
        "http_status": status,
        "elapsed_sec": elapsed,
        "fetched_at": datetime.now().isoformat(timespec="seconds"),
        "body": body,
    }
    with open(os.path.join(RAW, label + ".json"), "w", encoding="utf-8") as f:
        json.dump(payload, f, ensure_ascii=False, indent=2)


def get(path, params, label):
    """Один GET-запрос. Возвращает разобранный JSON. Сырой ответ пишет в raw/."""
    url = BASE + path + "?" + urllib.parse.urlencode(params)
    request = urllib.request.Request(url, method="GET")
    request.add_header("Authorization", TOKEN)
    request.add_header("Accept", "application/json")

    started = time.time()
    try:
        with urllib.request.urlopen(request, timeout=TIMEOUT) as response:
            status, body = response.status, response.read().decode("utf-8")
    except urllib.error.HTTPError as e:
        status, body = e.code, e.read().decode("utf-8", "replace")
    except Exception as e:                       # сеть, DNS, таймаут
        die("не удалось выполнить запрос %s: %s" % (label, type(e).__name__))

    elapsed = round(time.time() - started, 2)

    if status == 429:
        print("  лимит запросов (429), ждём 5 секунд и пробуем ещё раз")
        time.sleep(5)
        return get(path, params, label + "_retry")

    explain = {
        401: "токен неверный или истёк",
        403: "у токена нет категории «Вопросы и отзывы»",
        402: "требуется оплата — проверьте баланс в личном кабинете компании WB",
        400: "неправильные параметры запроса",
    }
    if status != 200:
        save_raw(label, url, params, status, body, elapsed)
        die("%s вернул код %s (%s). Сырой ответ сохранён в raw/"
            % (label, status, explain.get(status, "см. raw/")))

    try:
        parsed = json.loads(body)
    except json.JSONDecodeError:
        save_raw(label, url, params, status, body, elapsed)
        die("%s вернул не JSON, сырой ответ сохранён в raw/" % label)

    save_raw(label, url, params, status, body, elapsed)
    time.sleep(PAUSE)
    return parsed


def flatten(feedback):
    """Разворачивает вложенные объекты в плоские ключи вида productDetails.nmId."""
    flat = {}
    for key, value in feedback.items():
        if isinstance(value, dict):
            for subkey, subvalue in value.items():
                flat["%s.%s" % (key, subkey)] = subvalue
        else:
            flat[key] = value
    return flat


def filled(value):
    return value not in (None, "", [], {})


def covered(key, other_keys):
    """Поле считается совпавшим, даже если в одном ответе оно пришло целиком (answer),
    а в другом развёрнуто (answer.text) — иначе получим мнимое расхождение на null."""
    return (key in other_keys
            or any(o.startswith(key + ".") for o in other_keys)
            or any(key.startswith(o + ".") for o in other_keys))


COLUMNS = ["id", "createdDate", "productValuation", "has_text", "text_len",
           "photo_count", "has_video", "nmId", "imtId", "supplierArticle",
           "brandName", "size", "subjectId", "subjectName", "orderStatus",
           "matchingSize", "color", "bables_count", "lastOrderCreatedAt",
           "has_answer", "answer_state", "state", "wasViewed",
           "isAbleSupplierFeedbackValuation", "supplierFeedbackValuation",
           "isAbleSupplierProductValuation", "supplierProductValuation",
           "isAbleReturnProductOrders", "returnProductOrdersDate",
           "parentFeedbackId", "childFeedbackId", "source"]


def write_csv(list_unanswered, list_answered):
    """Плоская таблица для Excel. Без текстов отзывов и имён покупателей."""
    csv_path = os.path.join(NORM, "feedbacks.csv")
    with open(csv_path, "w", encoding="utf-8-sig", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=COLUMNS)
        writer.writeheader()
        for source, batch in (("unanswered", list_unanswered), ("answered", list_answered)):
            for fb in batch:
                details = fb.get("productDetails") or {}
                answer = fb.get("answer") or {}
                text = fb.get("text") or ""
                writer.writerow({
                    "id": fb.get("id"),
                    "createdDate": fb.get("createdDate"),
                    "productValuation": fb.get("productValuation"),
                    "has_text": bool(text),
                    "text_len": len(text),
                    "photo_count": len(fb.get("photoLinks") or []),
                    "has_video": bool(fb.get("video")),
                    "nmId": details.get("nmId"),
                    "imtId": details.get("imtId"),
                    "supplierArticle": details.get("supplierArticle"),
                    "brandName": details.get("brandName"),
                    "size": details.get("size"),
                    "subjectId": fb.get("subjectId"),
                    "subjectName": fb.get("subjectName"),
                    "orderStatus": fb.get("orderStatus"),
                    "matchingSize": fb.get("matchingSize"),
                    "color": fb.get("color"),
                    "bables_count": len(fb.get("bables") or []),
                    "lastOrderCreatedAt": fb.get("lastOrderCreatedAt"),
                    "has_answer": bool(answer.get("text")),
                    "answer_state": answer.get("state"),
                    "state": fb.get("state"),
                    "wasViewed": fb.get("wasViewed"),
                    "isAbleSupplierFeedbackValuation": fb.get("isAbleSupplierFeedbackValuation"),
                    "supplierFeedbackValuation": fb.get("supplierFeedbackValuation"),
                    "isAbleSupplierProductValuation": fb.get("isAbleSupplierProductValuation"),
                    "supplierProductValuation": fb.get("supplierProductValuation"),
                    "isAbleReturnProductOrders": fb.get("isAbleReturnProductOrders"),
                    "returnProductOrdersDate": fb.get("returnProductOrdersDate"),
                    "parentFeedbackId": fb.get("parentFeedbackId"),
                    "childFeedbackId": fb.get("childFeedbackId"),
                    "source": source,
                })
    return csv_path


def write_report(list_unanswered, list_answered, unanswered_raw, single_data, sample_id):
    all_feedbacks = list_unanswered + list_answered
    flat_all = [flatten(fb) for fb in all_feedbacks]
    keys = sorted({k for row in flat_all for k in row})
    # убираем «родителей» вложенных объектов: есть answer.text — сам answer не показываем
    keys = [k for k in keys if not any(other.startswith(k + ".") for other in keys)]

    lines = ["# ТЕСТ 1 — состав и заполненность полей отзыва", ""]
    lines.append("Дата запуска: %s" % datetime.now().isoformat(timespec="seconds"))
    lines.append("")
    lines.append("Отзывов получено: %d (необработанных %d, обработанных %d)"
                 % (len(all_feedbacks), len(list_unanswered), len(list_answered)))
    lines.append("Счётчики из ответа API: countUnanswered=%s, countArchive=%s"
                 % (unanswered_raw.get("data", {}).get("countUnanswered"),
                    unanswered_raw.get("data", {}).get("countArchive")))
    lines.append("")

    ratings = {}
    for fb in all_feedbacks:
        ratings[fb.get("productValuation")] = ratings.get(fb.get("productValuation"), 0) + 1
    lines.append("## Распределение оценок")
    lines.append("")
    for rating in sorted(ratings, key=lambda x: (x is None, x)):
        lines.append("- %s★ — %d" % (rating, ratings[rating]))
    lines.append("")
    lines.append("Из них 1–3★: %d"
                 % sum(v for k, v in ratings.items() if isinstance(k, int) and k <= 3))
    lines.append("")

    lines.append("## Заполненность полей")
    lines.append("")
    lines.append("| Поле | Непустых | % | Тип | Пример |")
    lines.append("|---|---|---|---|---|")
    for key in keys:
        non_empty = [row.get(key) for row in flat_all if filled(row.get(key))]
        types = sorted({type(v).__name__ for v in non_empty}) or ["—"]
        if key in SENSITIVE:
            example = "скрыто"
        elif non_empty:
            example = str(non_empty[0])[:40].replace("|", "/")
        else:
            example = "—"
        lines.append("| %s | %d | %d%% | %s | %s |"
                     % (key, len(non_empty),
                        round(100 * len(non_empty) / len(flat_all)),
                        "/".join(types), example))
    lines.append("")

    flat_single = flatten(single_data)
    only_single = sorted(k for k in flat_single if not covered(k, keys))
    only_list = sorted(k for k in keys if not covered(k, set(flat_single)))
    lines.append("## Сверка: список отзывов против одиночного запроса")
    lines.append("")
    lines.append("Сверяли отзыв id=%s" % sample_id)
    lines.append("")
    lines.append("- Есть только в одиночном запросе: %s" % (", ".join(only_single) or "нет"))
    lines.append("- Есть только в списке: %s" % (", ".join(only_list) or "нет"))
    lines.append("")
    lines.append("Вывод: %s" % ("одиночный запрос данных не добавляет, хватит списка"
                                if not only_single else
                                "одиночный запрос даёт дополнительные поля, см. выше"))

    report_path = os.path.join(REPORT, "field_coverage.md")
    with open(report_path, "w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")
    return report_path


def main():
    global TOKEN

    load_env()
    TOKEN = os.environ.get("WB_FEEDBACKS_TOKEN", "").strip()
    if not TOKEN:
        die("не найдена переменная WB_FEEDBACKS_TOKEN. "
            "Создайте файл .env рядом со скриптом (см. инструкцию).")

    for folder in (RAW, NORM, REPORT):
        os.makedirs(folder, exist_ok=True)
        os.chmod(folder, 0o700)

    print("Шаг 1. Проверка доступа")
    get("/api/v1/feedbacks", {"isAnswered": "false", "take": 1, "skip": 0}, "00_access_check")
    print("  доступ есть")

    print("Шаг 2. Отзывы isAnswered=false (до %d)" % TAKE)
    unanswered = get("/api/v1/feedbacks",
                     {"isAnswered": "false", "take": TAKE, "skip": 0, "order": "dateDesc"},
                     "01_feedbacks_unanswered")
    list_unanswered = unanswered.get("data", {}).get("feedbacks") or []
    print("  получено: %d" % len(list_unanswered))

    print("Шаг 3. Отзывы isAnswered=true (до %d)" % TAKE)
    answered = get("/api/v1/feedbacks",
                   {"isAnswered": "true", "take": TAKE, "skip": 0, "order": "dateDesc"},
                   "02_feedbacks_answered")
    list_answered = answered.get("data", {}).get("feedbacks") or []
    print("  получено: %d" % len(list_answered))

    all_feedbacks = list_unanswered + list_answered
    if not all_feedbacks:
        die("API ответил успешно, но отзывов не вернул. Проверьте аккаунт и период.")

    print("Шаг 4. Один отзыв через /api/v1/feedback для сверки полей")
    sample = next((f for f in all_feedbacks if f.get("photoLinks")), all_feedbacks[0])
    single = get("/api/v1/feedback", {"id": sample["id"]}, "03_feedback_single")

    print("Шаг 5. Нормализованный CSV")
    csv_path = write_csv(list_unanswered, list_answered)
    print("  записано строк: %d" % len(all_feedbacks))

    print("Шаг 6. Отчёт по заполненности полей")
    report_path = write_report(list_unanswered, list_answered, unanswered,
                               single.get("data") or {}, sample["id"])

    print("")
    print("ГОТОВО.")
    print("  сырые ответы:  %s" % RAW)
    print("  таблица:       %s" % csv_path)
    print("  отчёт:         %s" % report_path)


if __name__ == "__main__":
    main()
