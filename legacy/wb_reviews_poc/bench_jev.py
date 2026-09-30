"""Офлайн-тест Jev: отделяет ли его вероятность одобренные жалобы от отклонённых.

Берёт решённые жалобы (outcome approved/rejected) с текстом отзыва, по три основания,
задаёт Jev несколько вопросов да/нет и сравнивает AUC с уверенностью текущей LLM
(ai_results.confidence). В WB ничего не отправляет, БД только читает.
Запуск: /root/scripts/jev/jevgrep/.venv/bin/python bench_jev.py [N_на_класс]
"""
import json, random, sqlite3, sys
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

from typesafe_sdk import Noul, RetryPolicy, TypeSafeClient

N = int(sys.argv[1]) if len(sys.argv) > 1 else 150
BASES = ["Другое", "Отзыв не относится к товару", "Угрозы, оскорбления"]
QUESTIONS = {
    "violates": "Would a Wildberries marketplace moderator hide `review` as violating the rules "
                "for reviews (not about this product, insults, spam, no concrete claim about the product)?",
    "not_product": "Is `review` not about this product itself (it is about delivery, pickup point, "
                   "courier, seller, packaging from the courier, price, or a different item)?",
    "no_defect": "Does `review` state only a general negative opinion without any concrete, "
                 "verifiable defect of the product?",
    "insult": "Does `review` contain insults, threats or obscene language?",
    "approve": "Given `complaint`, would a Wildberries moderator approve the complaint and hide `review`?",
}

key = next(l.split("=", 1)[1].strip().strip('"') for l in Path("/root/.env.shared").read_text().splitlines()
           if l.startswith("TYPESAFE_API_KEY="))
client = TypeSafeClient(api_key=key, model="jev-latest", timeout=30.0,
                        retry=RetryPolicy(max_retries=4, backoff_initial=0.5, backoff_max=8.0,
                                          http_statuses={408, 429, *range(500, 600)}, timeout=60.0))

db = sqlite3.connect("file:data/reviews.db?mode=ro", uri=True)
rows = []
for basis in BASES:
    for outcome in ("approved", "rejected"):
        got = db.execute("""
            select c.id, c.basis, c.outcome, r.text, r.valuation, r.product_name, r.subject_name,
                   c.text_sent, a.confidence, a.strength
            from complaints c join reviews r on r.id = c.review_id
            left join ai_results a on a.review_id = c.review_id
            where c.basis = ? and c.outcome = ? and length(r.text) > 0""", (basis, outcome)).fetchall()
        random.Random(42).shuffle(got)
        rows += got[:N]

usage = {"tokens": 0}


def ask(row):
    cid, basis, outcome, text, val, product, subject, complaint, conf, strength = row
    state = {"product": f"{subject}: {product}", "rating": f"{val} из 5", "review": text,
             "complaint": complaint or ""}
    r = client.system_one(state=state, questions={k: Noul(instructions=q) for k, q in QUESTIONS.items()})
    usage["tokens"] += r.usage.input_tokens or 0
    return {"id": cid, "basis": basis, "approved": outcome == "approved", "llm_conf": conf,
            "strength": strength, **{k: r.nouls[k].noul for k in QUESTIONS}}


def auc(pairs):
    """Вероятность, что случайная одобренная получит балл выше случайной отклонённой."""
    pos = [s for s, y in pairs if y and s is not None]
    neg = [s for s, y in pairs if not y and s is not None]
    if not pos or not neg:
        return None
    wins = sum((p > n) + 0.5 * (p == n) for p in pos for n in neg)
    return round(wins / (len(pos) * len(neg)), 3)


with ThreadPoolExecutor(8) as ex:
    res = list(ex.map(ask, rows))
Path("bench_jev_result.json").write_text(json.dumps(res, ensure_ascii=False, indent=1))

print(f"отзывов {len(res)}, токенов {usage['tokens']}, ~${usage['tokens'] * 0.042 / 1e6:.4f}")
for basis in BASES + ["ВСЕ"]:
    sub = [x for x in res if basis == "ВСЕ" or x["basis"] == basis]
    line = {k: auc([(x[k], x["approved"]) for x in sub]) for k in [*QUESTIONS, "llm_conf"]}
    print(f"{basis:30} n={len(sub):4}  " + "  ".join(f"{k}={v}" for k, v in line.items()))
