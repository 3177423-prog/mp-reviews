CREATE TABLE reviews (
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
    fetched_at        TEXT NOT NULL
, account TEXT NOT NULL DEFAULT 'ms');
CREATE INDEX ix_reviews_status ON reviews(status);
CREATE INDEX ix_reviews_nm ON reviews(nm_id);
CREATE TABLE sku_stats (
    nm_id             INTEGER PRIMARY KEY,
    ratings_count     INTEGER,
    avg_rating        REAL,
    truncated         INTEGER,      -- 1, если считали не по всей истории
    updated_at        TEXT NOT NULL
);
CREATE TABLE complaints (
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
CREATE INDEX ix_complaints_outcome ON complaints(outcome);
CREATE TABLE ai_results (
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
CREATE TABLE meta (
    key               TEXT PRIMARY KEY,
    value             TEXT
);
CREATE INDEX ix_reviews_account ON reviews(account);
