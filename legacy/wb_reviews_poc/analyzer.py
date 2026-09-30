"""
AI-разметка очереди: по каждому неразмеченному отзыву определяет основание жалобы,
силу, уверенность и готовит черновик текста.

Модель — Sonnet через подписку Claude (CLI). Запускается по таймеру systemd.
"""

import fcntl
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime

import accounts
import store

MODEL = "sonnet"
GLM_MODEL = "glm-5.2"
GLM_URL = "https://api.z.ai/api/coding/paas/v4/chat/completions"
GLM_CONFIG = "/root/.openclaw/openclaw.json"
DEEPSEEK_MODEL = "deepseek-chat"
DEEPSEEK_URL = "https://api.deepseek.com/chat/completions"
ALICE_MODEL = "aliceai-llm-flash"
ALICE_URL = "https://llm.api.cloud.yandex.net/v1/chat/completions"
SHARED_ENV = "/root/.env.shared"
PROMPT_VERSION = "v11"       # 18.09.2026: Sonnet основной, теги WB в правилах, маршрутизация кодом
LIMIT_PER_RUN = 250         # весь поток за прогон; сверка первой партии пройдена
CALL_TIMEOUT = 180

# Ровно те причины, что есть в форме жалобы кабинета WB. Своих названий не выдумываем.
BASES = ["отзыв оставили конкуренты", "отзыв не относится к товару",
         "спам-реклама в тексте", "нецензурная лексика",
         "отзыв с политическим контекстом", "угрозы, оскорбления", "другое"]

SYSTEM = """Ты помощник продавца на Wildberries. По отзыву покупателя определяешь,
есть ли формальное основание пожаловаться на отзыв площадке.

Допустимые основания: {bases}.

Выбирай ТОЛЬКО из этих семи причин — это список формы кабинета, других нет.

Как выбирать, в порядке приоритета:
1. Нецензурная лексика — есть мат, в том числе замаскированный площадкой звёздочками
   (***, ****). Ставь эту причину, даже если остальной отзыв спокойный и по делу.
2. Угрозы, оскорбления — сюда же любая негативная характеристика самого продавца,
   а не товара. Признаки: «ужасный продавец», «недобросовестный продавец»,
   «не рекомендую продавца», «не советую продавца», «продавец обманщик», «мошенники»,
   «не покупайте у этого продавца», «больше у него ничего не куплю», а также угроза
   судом, жалобой в органы, «засужу», «напишу куда следует».
   Эта причина ГЛАВНЕЕ, чем «отзыв не относится к товару»: если покупатель ругает
   доставку или возврат и при этом называет продавца ужасным или недобросовестным —
   выбирай «угрозы, оскорбления».
   В черновике указывай, что отзыв содержит оценку личности и деловой репутации
   продавца, а не описание товара.
3. Спам-реклама в тексте — ссылки, никнеймы, приглашения в каналы, реклама других
   магазинов, предложения писать отзывы за деньги.
4. Отзыв с политическим контекстом — политика, события, лозунги, не относящиеся к товару.
5. Отзыв не относится к товару — претензия про доставку, повреждение при перевозке,
   недостачу, пересорт, упаковку, пункт выдачи, оплату или процесс возврата.
   Товар, пришедший разбитым или некомплектным, сюда же. Это правило ВАЖНЕЕ
   правила 7: если в отзыве не хватает детали, чего-то не доложили, пришло не то
   или коробка повреждена — это причина 5, а не «другое», даже когда покупатель
   ругает и сам товар тоже.

ТЕГИ WB — это то, что покупатель выбрал из списка площадки при написании отзыва,
и модератор их видит. Учитывай их:
- тег «неполный комплект» → причина 5, «отзыв не относится к товару»;
- теги «плохое качество», «нет эффекта», «низкая мощность», «неудобно пользоваться»,
  «не соответствует описанию» при причине «другое» — сила «нет»: покупатель
  описал претензию к самому товару структурированно, такие отзывы площадка
  не снимает. Сила «слабое» здесь допустима только когда текст отзыва
  противоречит тегу (текст положительный, тег негативный).
6. Отзыв оставили конкуренты — есть прямые признаки: сравнение с конкретным магазином
   в его пользу, шаблонный текст, реклама чужого товара. Домыслы не годятся.
7. Другое — всё остальное: пустой отзыв без текста, оценка не соответствует
   положительному тексту, претензия к самому товару, эмоции без конкретики.

ГЛАВНОЕ: черновик жалобы пишется ВСЕГДА, для любого отзыва, без исключений.
Пустым черновик не оставляй никогда. В поле «сила» честно ставь «сильное», «слабое»
или «нет» — это подсказка сотруднику, а не повод не писать текст.

Отдельно: если текст отзыва положительный, товар покупателя устроил, а оценка 1-3 —
причина «другое», и в черновике проси не удалить отзыв, а ИСКЛЮЧИТЬ ЕГО ИЗ РЕЙТИНГА.

Общие запреты — действуют всегда, даже когда основание слабое:
- Нельзя выдумывать факты. Ссылаться можно только на то, что дано во входных данных.
- Нельзя приписывать покупателю мотивы, намерения и обстоятельства.
- Нельзя утверждать то, чего в отзыве нет.

Как писать черновик жалобы — его читает живой модератор:
- Начни с «Здравствуйте!».
- Пиши обычным человеческим языком, короткими предложениями, от лица продавца.
- Название товара целиком НЕ переписывай: в карточках оно длинное, с перечислением
  для поиска, живые люди так не пишут. Называй товар коротко и по смыслу:
  «набор тарелок», «подгузники», «зонт». Можно вообще сказать «товар».
- Одним-двумя предложениями объясни, почему отзыв не должен оставаться на карточке.
- Закончи просьбой: «Просим удалить отзыв» или «Просим скрыть отзыв».
  Исключение — случай «положительный текст, низкая оценка»: там проси
  «Просим исключить отзыв из рейтинга товара».
- Не пиши «оснований для жалобы нет» и не извиняйся — текст читает модератор площадки.
- Не более 1000 символов, обычно хватает трёх-четырёх предложений.
- Без канцелярита, без «данный», «осуществляется», «в связи с вышеизложенным».
- Не цитируй отзыв целиком; короткая цитата уместна, только если она сама по себе
  доказывает нарушение (мат, угроза).

ЗАПРЕЩЕНО ПРИЗНАВАТЬ ПРЕТЕНЗИЮ ОБОСНОВАННОЙ. Это жалоба, а не объяснительная.
Никогда не пиши фразы вроде «касается свойств самого товара», «относится к качеству
товара», «это претензия к товару», «покупатель прав». Такой текст модератор читает
как согласие с отзывом и отклоняет жалобу.

Отдельно запрещено рассуждать о том, нарушает отзыв правила или нет: никаких
«это не нарушение правил размещения отзывов», «формально правила не нарушены».
Решает модератор, а не мы. Наше дело — назвать причину, по которой отзыв не должен
влиять на карточку, и попросить его убрать.

Когда сказать по существу нечего, аргументируй только тем, чего в отзыве НЕ хватает,
и выбирай одну-две линии:
- нет описания конкретного дефекта, только оценочное суждение;
- утверждение невозможно проверить и соотнести с товаром;
- текст отзыва не объясняет выставленную оценку;
- претензия к посадке или размеру субъективна, размеры указаны в карточке;
- описанное относится к личным ощущениям, а не к характеристикам товара.

НЕ упоминай фотографии и не пиши, что покупатель их не приложил. Фото есть у единиц,
а на части товаров такая просьба выглядит неуместно. Единственное исключение —
фото к отзыву действительно приложено и на нём заявленного дефекта не видно.
Пример: «Здравствуйте! В отзыве нет описания конкретного дефекта — только общая
оценка. Проверить утверждение и соотнести его с товаром невозможно.
Просим скрыть отзыв.»

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


def log(message):
    print("%s  %s" % (datetime.now().strftime("%H:%M:%S"), message), flush=True)


def build_user_prompt(review):
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


def ask_claude(system, user):
    """Основной канал: Sonnet через подписку Claude."""
    result = subprocess.run(
        ["claude", "-p", user, "--model", MODEL, "--append-system-prompt", system],
        capture_output=True, text=True, timeout=CALL_TIMEOUT, cwd="/tmp")
    return parse_json(result.stdout), result.stdout


def ask_glm(system, user):
    """Запасной канал: GLM-5.2 по подписке Z.AI. max_tokens с запасом —
    модель тратит часть лимита на рассуждение: при малом max_tokens ответ приходит
    пустым или JSON обрывается на середине."""
    key = json.load(open(GLM_CONFIG))["models"]["providers"]["zai"]["apiKey"]
    payload = json.dumps({
        "model": GLM_MODEL, "temperature": 0.2, "max_tokens": 4000,
        "messages": [{"role": "system", "content": system},
                     {"role": "user", "content": user}]}).encode()
    request = urllib.request.Request(
        GLM_URL, data=payload,
        headers={"Authorization": "Bearer " + key, "Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=CALL_TIMEOUT) as response:
        body = json.loads(response.read().decode("utf-8"))
    message = body["choices"][0]["message"]
    text = message.get("content") or ""
    answer = parse_json(text)
    if not answer:                      # изредка ответ уезжает в поле рассуждений
        answer = parse_json(message.get("reasoning_content") or "")
    return answer, text


def ask_deepseek(system, user):
    """Запасной канал: DeepSeek по API-ключу из .env (платный, но копеечный)."""
    key = ""
    path = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env")
    for line in open(path, encoding="utf-8-sig"):
        if line.startswith("DEEPSEEK_API_KEY="):
            key = line.split("=", 1)[1].strip()
    if not key:
        return None, "нет DEEPSEEK_API_KEY"
    payload = json.dumps({
        "model": DEEPSEEK_MODEL, "temperature": 0.2, "max_tokens": 1500,
        "messages": [{"role": "system", "content": system},
                     {"role": "user", "content": user}]}).encode()
    request = urllib.request.Request(
        DEEPSEEK_URL, data=payload,
        headers={"Authorization": "Bearer " + key, "Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=CALL_TIMEOUT) as response:
        body = json.loads(response.read().decode("utf-8"))
    text = body["choices"][0]["message"].get("content") or ""
    return parse_json(text), text


def shared_env(name):
    for line in open(SHARED_ENV, encoding="utf-8-sig"):
        if line.startswith(name + "="):
            return line.split("=", 1)[1].strip()
    return ""


def ask_alice(system, user):
    """Запасной канал: Alice AI Flash (Yandex AI Studio).
    Ходит только по OpenAI-совместимому пути, modelUri обязателен целиком."""
    key = shared_env("YANDEX_AISTUDIO_API_KEY")
    folder = shared_env("YANDEX_AISTUDIO_FOLDER_ID")
    if not key or not folder:
        return None, "нет ключа Yandex AI Studio"
    payload = json.dumps({
        "model": "gpt://%s/%s" % (folder, ALICE_MODEL),
        "temperature": 0.2, "max_tokens": 1500,
        "messages": [{"role": "system", "content": system},
                     {"role": "user", "content": user}]}).encode()
    request = urllib.request.Request(
        ALICE_URL, data=payload,
        headers={"Authorization": "Api-Key " + key, "Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=CALL_TIMEOUT) as response:
        body = json.loads(response.read().decode("utf-8"))
    text = body["choices"][0]["message"].get("content") or ""
    return parse_json(text), text


def ask_codex(system, user):
    """Запасной канал: GPT через подписку codex. Работает только после `codex login`."""
    result = subprocess.run(
        ["codex", "exec", "--skip-git-repo-check", system + "\n\n" + user],
        capture_output=True, text=True, timeout=CALL_TIMEOUT, cwd="/tmp",
        stdin=subprocess.DEVNULL)
    return parse_json(result.stdout), result.stdout


# Sonnet основной — решение заказчика по аудиту 18.09.2026: на 12 жалобах,
# размеченных Sonnet, одобрено 80% против 42,6% у DeepSeek на 7 000. Выборка
# мала, поэтому это проверка, а не вывод: через неделю сравнить по prompt_version.
# ~10 с на отзыв, дневной поток (~225) укладывается в один прогон.
# Alice: её контент-фильтр отказывается разбирать часть отзывов («Я не могу
# обсуждать эту тему») — отказ распознаём и сразу уходим на следующий канал.
PROVIDERS = [(MODEL, ask_claude), (DEEPSEEK_MODEL, ask_deepseek), (ALICE_MODEL, ask_alice),
             (GLM_MODEL, ask_glm), ("codex", ask_codex)]

REFUSAL_MARKERS = ("не могу обсуждать", "поговорим о чём-нибудь ещё")


def ask(system, user, photo_count=0, tags=""):
    """Идём по каналам, пока кто-то не вернёт разбираемый JSON.
    Возвращает (ответ, имя канала, последний сырой текст)."""
    last_raw = ""
    for name, call in PROVIDERS:
        for attempt in (1, 2):
            try:
                answer, raw = call(system, user)
            except Exception as e:
                answer, raw = None, "%s: %s" % (name, type(e).__name__)
            last_raw = raw
            if answer and not draft_is_sound(str(answer.get("черновик") or ""), photo_count, tags):
                log("    канал %s написал негодный текст (самоподрыв или ссылка на фото)" % name)
                answer = None
            if answer:
                if name != PROVIDERS[0][0] or attempt > 1:
                    log("    ответил канал %s (попытка %d)" % (name, attempt))
                return answer, name, raw
            if any(m in (raw or "").lower() for m in REFUSAL_MARKERS):
                log("    канал %s отказался разбирать отзыв" % name)
                break
            if attempt == 1:
                time.sleep(2)
        log("    канал %s не ответил" % name)
    return None, "", last_raw


# Фразы, которыми модель признаёт претензию обоснованной — такой черновик подавать нельзя.
SELF_DEFEATING = (
    "касается свойств самого товара", "касаются свойств самого товара",
    "относится к качеству товара", "относятся к качеству товара",
    "относится к свойствам товара", "относятся к свойствам товара",
    "это претензия к товару", "претензия к качеству товара",
    "претензия к самому товару", "покупатель прав", "недостаток товара подтверж",
    # признание, что нарушения нет — модератор читает это как «жалоба необоснованна»
    "а не нарушение", "не нарушение правил", "не является нарушением",
    "не нарушает правил", "претензия к характеристикам товара",
    "относится к характеристикам товара", "касается характеристик товара",
    "не противоречит правилам", "формально не нарушает",
)


def draft_is_sound(draft, photo_count=0, tags=""):
    """Черновик не должен подтверждать претензию и не должен ссылаться на фотографии,
    которых к отзыву не прикладывали.

    Исключение — тег покупателя «не как на фото»: тут речь о фото карточки, а не
    об отзыве, и черновик обязан его упомянуть. Без исключения Sonnet проваливал
    такие отзывы на всех каналах подряд (18.09.2026, E7uvks9q)."""
    low = (draft or "").lower()
    if any(marker in low for marker in SELF_DEFEATING):
        return False
    if not photo_count and "фото" not in (tags or "").lower() \
            and ("фото" in low or "снимк" in low or "изображен" in low):
        return False
    return True


# Аудит 7 457 жалоб (18.09.2026): некомплект, недостача и повреждение при доставке
# уходили в WB как «другое» и проходили на 52%, а под «отзыв не относится
# к товару» такие же проходят на 70%. Промпт это правило содержит с v10, но модель
# его нарушает — поэтому маршрутизация продублирована кодом.
MISROUTED = ("недостач", "не хватает", "не доложил", "недоложил", "не положил",
             "некомплект", "неполный комплект", "не полный комплект", "пересорт",
             "прислали не то", "пришёл не тот", "пришел не тот", "пришла не та",
             "другой товар", "вместо заказанного", "помят", "разбит", "повреждена упаковка",
             "повреждённая упаковка", "повреждена коробка", "коробка порвана")
# «доставка», «курьер», «ПВЗ» сюда намеренно не входят: они встречаются и в нейтральном
# контексте («доставка быстрая, а товар сломался»), а ошибочная причина сжигает
# единственную попытку жалобы на отзыв.
# Теги площадки, при которых претензия к самому товару описана структурированно:
# на аудите одобрено 13–21% против 54% у отзывов без тегов. Такую жалобу
# честно считаем безосновательной — фильтр подачи (когда будет включён) её отложит.
WEAK_TAGS = ("плохое качество", "нет эффекта", "низкая мощность",
             "неудобно пользоваться", "не соответствует описанию")
# Причины сильнее любой маршрутизации: мат и оскорбления модератор снимает сам.
HARD_BASES = ("нецензурная лексика", "угрозы, оскорбления", "спам-реклама в тексте",
              "отзыв с политическим контекстом")


def route(basis, strength, review):
    """Правит основание и силу по тексту и тегам отзыва, независимо от модели."""
    text = (review.get("text") or "").lower()
    tags = [t.strip().lower() for t in (review.get("bables") or "").split(",")]
    if basis in HARD_BASES:
        return basis, strength
    if "неполный комплект" in tags or any(m in text for m in MISROUTED):
        return "отзыв не относится к товару", strength
    if basis == "другое" and any(t in WEAK_TAGS for t in tags):
        return basis, "нет"
    return basis, strength


def normalize(answer):
    """Приводим ответ к нашим справочникам, чтобы мусор не попал в базу."""
    basis = str(answer.get("основание", "")).strip().lower()
    if basis not in BASES:
        basis = "другое"          # причины вне справочника кабинета не принимаем
    strength = str(answer.get("сила", "")).strip().lower()
    if strength not in ("сильное", "слабое", "нет"):
        strength = "слабое"
    try:
        confidence = max(0.0, min(1.0, float(answer.get("уверенность") or 0)))
    except (TypeError, ValueError):
        confidence = 0.0
    draft = str(answer.get("черновик") or "")[:1000]
    return basis, strength, confidence, draft


if __name__ == "__main__":
    # один запуск за раз: таймер не должен наложиться на ручной прогон
    lock = open(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".analyzer.lock"), "w")
    try:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except BlockingIOError:
        sys.exit("разметка уже выполняется, второй запуск не нужен")

    db = store.connect()
    acc = accounts.current()
    system = SYSTEM.format(bases=", ".join(BASES))
    limit = int(sys.argv[1]) if len(sys.argv) > 1 else LIMIT_PER_RUN
    pending = store.unanalyzed(db, acc, limit)
    log("кабинет %s, к разметке: %d" % (accounts.NAME[acc], len(pending)))
    done, failed = 0, 0
    for review in pending:
        started = time.time()
        answer, channel, raw = ask(system, build_user_prompt(review),
                                   review.get("photo_count") or 0, review.get("bables") or "")
        if not answer:
            failed += 1
            store.save_ai_failure(db, review["id"], MODEL, PROMPT_VERSION, raw[:2000])
            log("  %s — ответ не разобран" % review["id"][:8])
            continue
        basis, strength, confidence, draft = normalize(answer)
        routed, strength = route(basis, strength, review)
        if routed != basis:
            # Основание сменилось кодом — черновик модели писался под старое
            # и с новым не сходится. Просим переписать именно под эту причину.
            hint = ("\n\nОСНОВАНИЕ УЖЕ ВЫБРАНО: «%s» — в отзыве недостача, некомплект "
                    "или повреждение при доставке. Верни JSON с этим основанием "
                    "и черновиком, который аргументирует именно его." % routed)
            again, channel2, raw2 = ask(system, build_user_prompt(review) + hint,
                                        review.get("photo_count") or 0, review.get("bables") or "")
            if again:
                answer, channel = again, channel2
                _, _, confidence, draft = normalize(again)
            log("  %s маршрутизация: %s → %s" % (review["id"][:8], basis, routed))
            basis = routed
        store.save_ai_result(db, review["id"], basis, strength, confidence,
                             str(answer.get("аргумент") or "")[:2000], draft,
                             str(answer.get("чего_не_хватает") or "")[:500],
                             channel, PROMPT_VERSION, round(time.time() - started, 1))
        done += 1
        log("  %s %-24s %-8s %.2f  %.0f с"
            % (review["id"][:8], basis, strength, confidence, time.time() - started))
    log("размечено: %d, не разобрано: %d" % (done, failed))
