"""
Сравнение моделей на задаче «подобрать основание жалобы на отзыв».
Гоняет один и тот же промпт через доступные каналы и сверяет с ручной разметкой.
"""

import json
import os
import subprocess
import sys
import time
import urllib.request

import store

HERE = os.path.dirname(os.path.abspath(__file__))

BASES = ["не относится к товару", "пустой отзыв", "оскорбления",
         "угроза судом", "персональные данные или реклама",
         "опровергается карточкой", "отзыв о другом товаре", "основания нет"]

SYSTEM = """Ты помощник продавца на Wildberries. По отзыву покупателя определяешь,
есть ли формальное основание пожаловаться на отзыв площадке.

Допустимые основания: {bases}.

Правила разбора, в порядке приоритета:
1. Если текст отзыва пустой — основание всегда «пустой отзыв». Исключений нет.
2. Если есть мат, оскорбление в адрес продавца или магазина, угроза судом или жалобой
   в органы — основание «оскорбления» или «угроза судом», даже если рядом есть
   претензия к товару.
3. Если претензия про доставку, повреждение или бой при перевозке, недостачу,
   пересорт, упаковку, пункт выдачи, оплату или процесс возврата — «не относится к товару».
   Товар, пришедший разбитым или некомплектным, относится сюда же.
4. Если претензия про сам товар — брак, плесень, не работает, плохое качество,
   не понравился, не подошёл — «основания нет». Не подгоняй основание под желаемое.

Общие запреты:
- Нельзя выдумывать факты. Ссылаться можно только на то, что дано во входных данных.
- Нельзя приписывать покупателю мотивы, намерения и обстоятельства.
- Черновик жалобы — не более 1000 символов, только проверяемые факты.

Верни ТОЛЬКО JSON-объект, без markdown-обрамления и без пояснений до или после:
{{"основание": "...", "сила": "сильное|слабое|нет", "уверенность": 0.0,
"аргумент": "...", "черновик": "...", "чего_не_хватает": "..."}}"""

PROMPT = """Отзыв:
оценка: {valuation}
текст: {text}
категория товара: {subject}
название товара: {product}
статус заказа: {order_status}
отзыв оставлен через дней после заказа: {days}
фото приложено: {photos}
теги WB: {bables}"""


def build(review):
    return PROMPT.format(
        valuation=review["valuation"],
        text=(review["text"] or "").strip() or "(пусто, только оценка)",
        subject=review["subject_name"] or "-",
        product=(review["product_name"] or "-")[:70],
        order_status=review["order_status"] or "-",
        days=review["days_to_review"] if review["days_to_review"] is not None else "-",
        photos=review["photo_count"], bables=review["bables"] or "-")


def parse_json(raw):
    start, end = raw.find("{"), raw.rfind("}")
    if start < 0 or end < 0:
        return None
    try:
        return json.loads(raw[start:end + 1])
    except json.JSONDecodeError:
        return None


def ask_claude(system, user, model="sonnet"):
    started = time.time()
    result = subprocess.run(
        ["claude", "-p", user, "--model", model, "--append-system-prompt", system],
        capture_output=True, text=True, timeout=180, cwd="/tmp")
    return parse_json(result.stdout), round(time.time() - started, 1)


def zai_key():
    cfg = json.load(open("/root/.openclaw/openclaw.json"))
    return cfg["models"]["providers"]["zai"]["apiKey"]


def ask_glm(system, user, model="glm-5.2"):
    started = time.time()
    payload = json.dumps({
        "model": model,
        "messages": [{"role": "system", "content": system},
                     {"role": "user", "content": user}],
        "temperature": 0.2, "max_tokens": 1200,
    }).encode()
    request = urllib.request.Request(
        "https://api.z.ai/api/coding/paas/v4/chat/completions", data=payload,
        headers={"Authorization": "Bearer " + zai_key(),
                 "Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=180) as response:
        body = json.loads(response.read().decode())
    text = body["choices"][0]["message"]["content"]
    return parse_json(text), round(time.time() - started, 1)


CHANNELS = {"Sonnet (подписка Claude)": lambda s, u: ask_claude(s, u, "sonnet"),
            "GLM-5.2 (подписка Z.AI)": lambda s, u: ask_glm(s, u, "glm-5.2")}

if len(sys.argv) > 1:
    CHANNELS = {k: v for k, v in CHANNELS.items() if sys.argv[1].lower() in k.lower()}


if __name__ == "__main__":
    ids = json.load(open(os.path.join(HERE, "bench_set.json"), encoding="utf-8"))
    db = store.connect()
    system = SYSTEM.format(bases=", ".join(BASES))
    results = {}
    for name, ask in CHANNELS.items():
        rows = []
        print("\n=== %s" % name, flush=True)
        for item in ids:
            review = dict(db.execute("SELECT * FROM reviews WHERE id=?",
                                     (item["id"],)).fetchone())
            try:
                answer, seconds = ask(system, build(review))
            except Exception as e:
                answer, seconds = None, 0
                print("  ошибка: %s" % type(e).__name__, flush=True)
            got = (answer or {}).get("основание", "—")
            ok = "OK " if got.strip().lower() == item["эталон"] else "-- "
            rows.append({"id": item["id"], "эталон": item["эталон"], "ответ": got,
                         "совпало": got.strip().lower() == item["эталон"],
                         "сила": (answer or {}).get("сила"),
                         "уверенность": (answer or {}).get("уверенность"),
                         "черновик": (answer or {}).get("черновик", ""),
                         "сек": seconds})
            print("  %s %-32s → %-32s %4.1f с" % (ok, item["эталон"], got, seconds), flush=True)
        hits = sum(1 for r in rows if r["совпало"])
        print("  ИТОГО: %d/%d, среднее время %.1f с"
              % (hits, len(rows), sum(r["сек"] for r in rows) / len(rows)))
        results[name] = rows
    json.dump(results, open(os.path.join(HERE, "bench_result.json"), "w"),
              ensure_ascii=False, indent=2)
