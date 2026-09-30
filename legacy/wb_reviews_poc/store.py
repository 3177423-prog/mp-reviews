"""Хранилище: SQLite. Снимок отзыва на момент попадания в очередь + решения человека.

Кабинет («Маркетспейс», «Цифровой Ритейл», «Хоум Брендс») хранится в reviews.account;
жалобы и разметка привязаны к отзыву, поэтому свою копию ключа не держат.
"""

import os
import sqlite3
from datetime import datetime, timezone

import accounts

DB_PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), "data", "reviews.db")

SCHEMA = """
PRAGMA journal_mode = WAL;

-- Снимок отзыва на момент попадания в очередь
CREATE TABLE IF NOT EXISTS reviews (
    id                TEXT PRIMARY KEY,
    nm_id             INTEGER,
    imt_id            INTEGER,
    supplier_article  TEXT,
    brand_name        TEXT,
    subject_name      TEXT,
    product_name      TEXT,
    valuation         INTEGER,
    text              TEXT,
    has_text          INTEGER,
    photo_count       INTEGER,
    has_video         INTEGER,
    created_date      TEXT,
    order_created_at  TEXT,
    days_to_review    INTEGER,
    order_status      TEXT,
    parent_id         TEXT,
    bables            TEXT,
    complaint_filed   INTEGER,      -- 1, если жалоба уже подана (isAble=false)
    reason_key        INTEGER,      -- supplierFeedbackValuation
    status            TEXT NOT NULL DEFAULT 'new',  -- new | submitted | skipped | prefiled
    fetched_at        TEXT NOT NULL,
    account           TEXT NOT NULL DEFAULT 'ms'    -- ключ кабинета, см. accounts.py
);
CREATE INDEX IF NOT EXISTS ix_reviews_status ON reviews(status);
CREATE INDEX IF NOT EXISTS ix_reviews_nm ON reviews(nm_id);
-- индекс по account создаётся в connect(), после досоздания самой колонки

-- Рейтинг и число оценок по артикулу, пересчёт раз в сутки
CREATE TABLE IF NOT EXISTS sku_stats (
    nm_id             INTEGER PRIMARY KEY,
    ratings_count     INTEGER,
    avg_rating        REAL,
    truncated         INTEGER,      -- 1, если считали не по всей истории
    updated_at        TEXT NOT NULL
);

-- Жалоба: решение человека, подача, исход
CREATE TABLE IF NOT EXISTS complaints (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    review_id         TEXT NOT NULL REFERENCES reviews(id),
    decision          TEXT NOT NULL,     -- submitted | skipped
    basis             TEXT,              -- основание, выбранное человеком
    basis_source      TEXT,              -- human | ai
    note              TEXT,
    text_sent         TEXT,
    text_edited       INTEGER DEFAULT 0,
    tier              INTEGER,
    decided_at        TEXT NOT NULL,
    outcome           TEXT,              -- pending | approved | rejected
    outcome_at        TEXT,
    reason_key_seen   INTEGER            -- ключ причины, подтянутый позже из API
);
CREATE INDEX IF NOT EXISTS ix_complaints_outcome ON complaints(outcome);

-- Служебные отметки: когда что последний раз отрабатывало
CREATE TABLE IF NOT EXISTS meta (
    key               TEXT PRIMARY KEY,
    value             TEXT
);

-- Разметка AI по отзыву
CREATE TABLE IF NOT EXISTS ai_results (
    review_id         TEXT PRIMARY KEY REFERENCES reviews(id),
    basis             TEXT,
    strength          TEXT,          -- сильное | слабое | нет
    confidence        REAL,
    argument          TEXT,
    draft             TEXT,
    missing           TEXT,
    model             TEXT,
    prompt_version    TEXT,
    seconds           REAL,
    failed_raw        TEXT,          -- сырой ответ, если разобрать не удалось
    created_at        TEXT NOT NULL
);
"""


def now():
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def connect():
    os.makedirs(os.path.dirname(DB_PATH), exist_ok=True)
    os.chmod(os.path.dirname(DB_PATH), 0o700)
    db = sqlite3.connect(DB_PATH, timeout=20)
    db.row_factory = sqlite3.Row
    db.executescript(SCHEMA)
    # База заводилась одним кабинетом: всё, что в ней уже есть, — Маркетспейс.
    columns = {r["name"] for r in db.execute("PRAGMA table_info(reviews)")}
    if "account" not in columns:
        db.execute("ALTER TABLE reviews ADD COLUMN account TEXT NOT NULL DEFAULT 'ms'")
    db.execute("CREATE INDEX IF NOT EXISTS ix_reviews_account ON reviews(account)")
    db.commit()
    return db


def days_between(order_iso, review_iso):
    try:
        a = datetime.fromisoformat(order_iso.replace("Z", "+00:00"))
        b = datetime.fromisoformat(review_iso.replace("Z", "+00:00"))
        return (b - a).days
    except Exception:
        return None


COLUMNS = (
    "id, nm_id, imt_id, supplier_article, brand_name, subject_name, product_name, "
    "valuation, text, has_text, photo_count, has_video, created_date, order_created_at, "
    "days_to_review, order_status, parent_id, bables, complaint_filed, reason_key, "
    "status, fetched_at, account")


def upsert_reviews(db, acc, feedbacks):
    """Кладёт новые отзывы. Уже известные не трогает, чтобы не затирать снимок."""
    acc = accounts.valid(acc)
    added = 0
    for f in feedbacks:
        details = f.get("productDetails") or {}
        text = f.get("text") or ""
        filed = 0 if f.get("isAbleSupplierFeedbackValuation") else 1
        row = (
            f.get("id"), details.get("nmId"), details.get("imtId"),
            details.get("supplierArticle"), details.get("brandName"),
            f.get("subjectName"), details.get("productName"),
            f.get("productValuation"), text, 1 if text.strip() else 0,
            len(f.get("photoLinks") or []), 1 if f.get("video") else 0,
            f.get("createdDate"), f.get("lastOrderCreatedAt"),
            days_between(f.get("lastOrderCreatedAt") or "", f.get("createdDate") or ""),
            f.get("orderStatus"), f.get("parentFeedbackId"),
            ", ".join(f.get("bables") or []), filed,
            f.get("supplierFeedbackValuation"),
            "prefiled" if filed else "new", now(), acc,
        )
        cur = db.execute(
            "INSERT OR IGNORE INTO reviews (%s) VALUES (%s)"
            % (COLUMNS, ",".join("?" * len(row))), row)
        added += cur.rowcount
    db.commit()
    return added


def save_sku_stats(db, nm_id, count, avg, truncated):
    db.execute("INSERT OR REPLACE INTO sku_stats VALUES (?,?,?,?,?)",
               (nm_id, count, avg, 1 if truncated else 0, now()))
    db.commit()


def stale_sku_ids(db, acc, limit):
    """Артикулы из очереди, по которым статистики нет или она старше суток."""
    rows = db.execute("""
        SELECT r.nm_id FROM reviews r
        LEFT JOIN sku_stats s ON s.nm_id = r.nm_id
        WHERE r.status = 'new' AND r.account = ? AND r.nm_id IS NOT NULL
          AND (s.updated_at IS NULL OR s.updated_at < datetime('now', '-1 day'))
        GROUP BY r.nm_id LIMIT ?""", (accounts.valid(acc), limit)).fetchall()
    return [r["nm_id"] for r in rows]


def unanalyzed(db, acc, limit):
    """Отзывы очереди без разметки. Неудачные попытки повторяем через полчаса,
    иначе один сбой канала оставил бы отзыв без разметки навсегда."""
    return [dict(r) for r in db.execute("""
        SELECT r.* FROM reviews r LEFT JOIN ai_results a ON a.review_id = r.id
        WHERE r.status = 'new' AND r.account = ?
          AND (a.review_id IS NULL
               OR (a.basis IS NULL AND a.created_at < datetime('now', '-30 minutes')))
        ORDER BY r.created_date DESC LIMIT ?""",
        (accounts.valid(acc), limit)).fetchall()]


def save_ai_result(db, review_id, basis, strength, confidence, argument,
                   draft, missing, model, version, seconds):
    db.execute("""INSERT OR REPLACE INTO ai_results
        (review_id, basis, strength, confidence, argument, draft, missing,
         model, prompt_version, seconds, failed_raw, created_at)
        VALUES (?,?,?,?,?,?,?,?,?,?,NULL,?)""",
               (review_id, basis, strength, confidence, argument, draft, missing,
                model, version, seconds, now()))
    db.commit()


def save_ai_failure(db, review_id, model, version, raw):
    db.execute("""INSERT OR REPLACE INTO ai_results
        (review_id, basis, strength, confidence, argument, draft, missing,
         model, prompt_version, seconds, failed_raw, created_at)
        VALUES (?,NULL,NULL,NULL,NULL,NULL,NULL,?,?,NULL,?,?)""",
               (review_id, model, version, raw, now()))
    db.commit()


def queue(db, acc):
    """Очередь с приоритетом. Ступени включаются, когда в очереди больше 100."""
    rows = db.execute("""
        SELECT r.*, s.ratings_count, s.avg_rating,
               a.basis AS ai_basis, a.strength AS ai_strength,
               a.confidence AS ai_confidence, a.draft AS ai_draft,
               a.argument AS ai_argument, a.failed_raw AS ai_failed,
               a.model AS ai_model, a.prompt_version AS ai_version
        FROM reviews r
        LEFT JOIN sku_stats s ON s.nm_id = r.nm_id
        LEFT JOIN ai_results a ON a.review_id = r.id
        WHERE r.status = 'new' AND r.account = ?""", (accounts.valid(acc),)).fetchall()
    items = [dict(r) for r in rows]
    apply_tiers = len(items) > 100
    for it in items:
        count, rating = it.get("ratings_count"), it.get("avg_rating")
        if not apply_tiers:
            it["tier"] = 0
        elif count is not None and count < 100:
            it["tier"] = 1
        elif rating is not None and rating < 4.7:
            it["tier"] = 2
        else:
            it["tier"] = 3
    rank = {"сильное": 0, "слабое": 1, "нет": 2}
    items.sort(key=lambda i: i["created_date"] or "", reverse=True)   # свежие выше
    items.sort(key=lambda i: rank.get(i.get("ai_strength"), 3))       # сила основания
    items.sort(key=lambda i: i["tier"])                               # ступень главнее
    return items


def decide(db, review_id, decision, basis, note, tier):
    db.execute("""INSERT INTO complaints
        (review_id, decision, basis, basis_source, note, tier, decided_at, outcome)
        VALUES (?,?,?,?,?,?,?,?)""",
               (review_id, decision, basis, "human", note, tier, now(),
                "pending" if decision == "submitted" else None))
    db.execute("UPDATE reviews SET status = ? WHERE id = ?",
               ("submitted" if decision == "submitted" else "skipped", review_id))
    db.commit()


def mark_already_filed(db, review_ids):
    """Отзывы, по которым жалоба в кабинете уже есть, убираем из очереди."""
    if not review_ids:
        return 0
    marks = ",".join("?" * len(review_ids))
    cur = db.execute(
        "UPDATE reviews SET status='prefiled', complaint_filed=1 "
        "WHERE status='new' AND id IN (%s)" % marks, list(review_ids))
    db.commit()
    return cur.rowcount


def record_submission(db, review_id, basis, reason_id, text, tier, ok, error):
    """Фиксирует попытку подачи через кабинет."""
    db.execute("""INSERT INTO complaints
        (review_id, decision, basis, basis_source, note, text_sent, text_edited,
         tier, decided_at, outcome)
        VALUES (?,?,?,?,?,?,?,?,?,?)""",
               (review_id, "submitted" if ok else "failed", basis, "panel",
                error or None, text, 0, tier, now(), "review" if ok else None))
    if ok:
        db.execute("UPDATE reviews SET status='submitted', complaint_filed=1, reason_key=? "
                   "WHERE id=?", (reason_id, review_id))
    db.commit()


def sync_outcomes(db, found):
    """found: id отзыва -> состояние жалобы из кабинета. Возвращает число обновлённых."""
    updated = 0
    for review_id, complaint in found.items():
        status = complaint.get("status")
        if status not in ("review", "approved", "rejected"):
            continue
        cur = db.execute("""UPDATE complaints SET outcome=?, outcome_at=?, reason_key_seen=?
            WHERE review_id=? AND decision='submitted'
              AND (outcome IS NULL OR outcome != ?)""",
                         (status, now(), complaint.get("id"), review_id, status))
        updated += cur.rowcount
    db.commit()
    return updated


def daily_report(db, acc, days=14):
    """Сводка по дням подачи. Дата — московская: в базе время в UTC."""
    return [dict(r) for r in db.execute("""
        SELECT date(datetime(c.decided_at, '+3 hours')) AS day,
               COUNT(*)                                  AS submitted,
               SUM(c.outcome = 'approved')               AS approved,
               SUM(c.outcome = 'rejected')               AS rejected,
               SUM(c.outcome = 'review')                 AS review
        FROM complaints c JOIN reviews r ON r.id = c.review_id
        WHERE c.decision = 'submitted' AND r.account = ?
        GROUP BY day ORDER BY day DESC LIMIT ?""",
        (accounts.valid(acc), days)).fetchall()]


def rate_window(db, acc, days=30, offset_days=0):
    """(одобрено, решено) за окно длиной days, сдвинутое на offset_days назад.

    Считаем по outcome_at — когда исход стал известен. По дате подачи считать нельзя:
    свежие жалобы ещё висят на рассмотрении и занижали бы процент.
    """
    row = db.execute("""
        SELECT SUM(c.outcome = 'approved') AS approved,
               SUM(c.outcome IN ('approved','rejected')) AS decided
        FROM complaints c JOIN reviews r ON r.id = c.review_id
        WHERE r.account = ? AND c.decision = 'submitted' AND c.outcome_at IS NOT NULL
          AND c.outcome_at >= datetime('now', ?) AND c.outcome_at < datetime('now', ?)""",
        (accounts.valid(acc), "-%d days" % (days + offset_days),
         "-%d days" % offset_days)).fetchone()
    return (row["approved"] or 0), (row["decided"] or 0)


def basis_report(db, acc, days=30):
    """Какие основания WB одобряет чаще. Только решённые: у висящих исхода нет."""
    return [dict(r) for r in db.execute("""
        SELECT c.basis,
               COUNT(*)                                  AS submitted,
               SUM(c.outcome = 'approved')               AS approved,
               SUM(c.outcome = 'rejected')               AS rejected,
               SUM(c.outcome = 'review')                 AS review
        FROM complaints c JOIN reviews r ON r.id = c.review_id
        WHERE r.account = ? AND c.decision = 'submitted'
          AND c.decided_at >= datetime('now', ?)
        GROUP BY c.basis ORDER BY submitted DESC""",
        (accounts.valid(acc), "-%d days" % days)).fetchall()]


def pending_outcomes(db, acc):
    return [dict(r) for r in db.execute("""
        SELECT c.*, r.supplier_article, r.nm_id, r.valuation, r.text, r.created_date
        FROM complaints c JOIN reviews r ON r.id = c.review_id
        WHERE c.decision='submitted' AND r.account = ?
        ORDER BY c.decided_at DESC LIMIT 300""", (accounts.valid(acc),)).fetchall()]


def set_outcome(db, complaint_id, outcome):
    db.execute("UPDATE complaints SET outcome = ?, outcome_at = ? WHERE id = ?",
               (outcome, now(), complaint_id))
    db.commit()


def mark_run(db, name, acc):
    """Отмечает успешный прогон: сбор может не найти новых отзывов, и это не поломка."""
    db.execute("INSERT OR REPLACE INTO meta VALUES (?,?)",
               ("%s:%s" % (name, accounts.valid(acc)), now()))
    db.commit()


def last_run(db, name, acc):
    row = db.execute("SELECT value FROM meta WHERE key=?",
                     ("%s:%s" % (name, accounts.valid(acc)),)).fetchone()
    return row[0] if row else None


def counters(db, acc):
    acc = accounts.valid(acc)

    def one(sql):
        return db.execute(sql, (acc,)).fetchone()[0]

    # Жалобы и разметка своего ключа кабинета не хранят — приходим к нему через отзыв.
    def by_complaint(where):
        return one("SELECT COUNT(*) FROM complaints c JOIN reviews r ON r.id = c.review_id "
                   "WHERE r.account = ? AND " + where)

    return {
        "queue": one("SELECT COUNT(*) FROM reviews WHERE account=? AND status='new'"),
        "prefiled": one("SELECT COUNT(*) FROM reviews WHERE account=? AND status='prefiled'"),
        "submitted": by_complaint("c.decision='submitted'"),
        "skipped": by_complaint("c.decision='skipped'"),
        "pending": by_complaint("c.outcome='review'"),
        "failed": by_complaint("c.decision='failed'"),
        "approved": by_complaint("c.outcome='approved'"),
        "rejected": by_complaint("c.outcome='rejected'"),
        "sku_stats": one("""SELECT COUNT(DISTINCT s.nm_id) FROM sku_stats s
            JOIN reviews r ON r.nm_id = s.nm_id WHERE r.account = ?"""),
        "analyzed": one("""SELECT COUNT(*) FROM ai_results a
            JOIN reviews r ON r.id = a.review_id
            WHERE r.account = ? AND a.basis IS NOT NULL"""),
        "ai_failed": one("""SELECT COUNT(*) FROM ai_results a
            JOIN reviews r ON r.id = a.review_id
            WHERE r.account = ? AND a.failed_raw IS NOT NULL"""),
        "to_analyze": one("""SELECT COUNT(*) FROM reviews r
            LEFT JOIN ai_results a ON a.review_id = r.id
            WHERE r.account = ? AND r.status='new' AND a.review_id IS NULL"""),
    }
