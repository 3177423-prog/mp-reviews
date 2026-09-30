"""
Реестр кабинетов WB.

Одна сессия продавца обслуживает все кабинеты сразу: пользователь seller-portal
видит все юрлица, а кабинет выбирается куками x-supplier-id и x-supplier-id-external.
Поэтому cURL вставляется один раз, а не отдельно на каждое юрлицо.

Токен официального API отзывов — свой на каждый кабинет: WB_FEEDBACKS_TOKEN_<КЛЮЧ>.
"""

import os

ACCOUNTS = [
    ("ms", "Маркетспейс",     "00000000-0000-0000-0000-000000000001"),
    ("cr", "Цифровой Ритейл", "00000000-0000-0000-0000-000000000002"),
    ("hb", "Хоум Брендс",     "00000000-0000-0000-0000-000000000003"),
]

KEYS = [key for key, _, _ in ACCOUNTS]
NAME = {key: name for key, name, _ in ACCOUNTS}
SUPPLIER = {key: sid for key, _, sid in ACCOUNTS}
DEFAULT = "ms"


def valid(acc):
    """Ключ кабинета или значение по умолчанию. Чужое значение молча не пропускаем."""
    return acc if acc in NAME else DEFAULT


def current():
    """Кабинет текущего прогона — из переменной окружения WB_ACCOUNT."""
    return valid(os.environ.get("WB_ACCOUNT"))


def token_var(acc):
    return "WB_FEEDBACKS_TOKEN_%s" % valid(acc).upper()
