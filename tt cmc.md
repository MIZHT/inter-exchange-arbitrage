Парсинг CoinMarketCap и построение карты рынков
1. ОБЩЕЕ ОПИСАНИЕ

Разработать модуль для парсинга данных с CoinMarketCap, который:

    Загружает поисковый индекс всех криптовалют

    Построчно обрабатывает каждую монету (через пул прокси)

    Собирает информацию о торговых парах (спот/фьючерсы) с USDT

    Формирует структурированный вывод для использования в арбитражном боте

2. ИСТОЧНИКИ ДАННЫХ
2.1 Поисковый индекс
text

URL: https://s3.coinmarketcap.com/generated/core/crypto/web-search.json
Метод: GET
Формат: JSON
Описание: Статический файл со списком всех криптовалют на CMC
Структура:
{
  "fields": ["id", "name", "symbol", "slug", "type", "rank", "search_score"],
  "values": [
    [1, "Bitcoin", "BTC", "bitcoin", 1, 1, 100.0],
    [2, "Litecoin", "LTC", "litecoin", 1, 20, 85.5],
    // ... до ~3000+ записей
  ]
}

2.2 API маркет-пар
text

URL: https://api.coinmarketcap.com/data-api/v3/cryptocurrency/market-pairs/latest
Метод: GET
Параметры:
  - id: int (ID монеты из поискового индекса)
  - category: "all" (все типы пар)
  - start: int (пагинация)
  - limit: int (макс 200)
  - quoteCurrencyId: 825 (только USDT)

Пример: https://api.coinmarketcap.com/data-api/v3/cryptocurrency/market-pairs/latest?id=1&category=all&start=1&limit=200&quoteCurrencyId=825

3. СТРУКТУРЫ ДАННЫХ
3.1 Внутренние структуры
go

// Маппинг ID -> Symbol из поискового индекса
type SymbolMap map[int]string  // key: id, value: symbol

// Информация о монете с торговыми парами
type CoinMarkets struct {
    CoinID    int64    `json:"coin_id"`
    Symbol    string   `json:"symbol"`
    Name      string   `json:"name"`
    Spot      []string `json:"spot"`      // биржи со спотовыми парами USDT
    Futures   []string `json:"futures"`   // биржи с фьючерсными парами USDT
    FetchedAt string   `json:"fetched_at"`
}

// Промежуточная структура для marketPair
type MarketPair struct {
    ExchangeID   int    `json:"exchangeId"`
    ExchangeSlug string `json:"exchangeSlug"`
    QuoteSymbol  string `json:"quoteSymbol"`
    Category     string `json:"category"`     // "spot", "perpetual", "futures"
    CenterType   string `json:"centerType"`   // "cex" или "dex"
}

// Ответ API
type MarketPairsResponse struct {
    Data struct {
        ID             int          `json:"id"`
        Name           string       `json:"name"`
        Symbol         string       `json:"symbol"`
        NumMarketPairs int          `json:"numMarketPairs"`
        MarketPairs    []MarketPair `json:"marketPairs"`
    } `json:"data"`
    Status struct {
        ErrorCode    string `json:"error_code"`
        ErrorMessage string `json:"error_message"`
    } `json:"status"`
}

3.2 Финальная структура вывода
go

// Основная структура результата
type CMCResult struct {
    Snapshot Snapshot `json:"snapshot"`
    Metadata Metadata `json:"metadata"`
}

// Карта: symbol -> TickerOutput
type Snapshot map[string]TickerOutput

// Вывод для одного тикера
type TickerOutput struct {
    Symbol    string     `json:"symbol"`
    CoinID    int64      `json:"coin_id"`
    Name      string     `json:"name"`
    Spot      [][]string `json:"spot"`      // Группы бирж для спота
    Futures   [][]string `json:"futures"`   // Группы бирж для фьючерсов
    Errors    []string   `json:"errors,omitempty"`
}

// Метаданные о процессе парсинга
type Metadata struct {
    TotalCoins        int       `json:"total_coins"`
    SuccessfullyParsed int      `json:"successfully_parsed"`
    FailedCoins       int       `json:"failed_coins"`
    TotalHTTPRequests int       `json:"total_http_requests"`
    StartTime         time.Time `json:"start_time"`
    EndTime           time.Time `json:"end_time"`
    DurationMs        int64     `json:"duration_ms"`
}

4. АЛГОРИТМ РАБОТЫ
4.1 Блок-схема
text

1. Загрузка поискового индекса
   ↓
2. Создание SymbolMap (id → symbol)
   ↓
3. Инициализация пула прокси (до 40)
   ↓
4. Для каждой монеты из SymbolMap (параллельно):
   ├─ Получение маркет-пар через API
   ├─ Фильтрация пар:
   │  ├─ quoteSymbol == "USDT"
   │  ├─ centerType == "cex"
   │  └─ category in ["spot", "perpetual", "futures"]
   ├─ Сбор уникальных бирж для spot
   ├─ Сбор уникальных бирж для futures
   └─ Сохранение результата
   ↓
5. Группировка результатов (одинаковые тикеры)
   ↓
6. Формирование финального Snapshot

4.2 Детали фильтрации

Для спота:
go

if pair.QuoteSymbol == "USDT" && 
   pair.CenterType == "cex" && 
   pair.Category == "spot" {
    // добавить биржу в spot список
}

Для фьючерсов:
go

if pair.QuoteSymbol == "USDT" && 
   pair.CenterType == "cex" && 
   (pair.Category == "perpetual" || pair.Category == "futures") {
    // добавить биржу в futures список
}

5. ТРЕБОВАНИЯ К РЕАЛИЗАЦИИ
5.1 Прокси

    Использовать до 40 прокси из конфига

    Round-robin распределение запросов

    Каждый запрос к API должен идти через случайный прокси

    При неудаче - ретрай с другим прокси (до 5 попыток)

5.2 Параллелизм

    Параллельная обработка до 24 монет одновременно

    Использовать worker pool

    Контроль нагрузки через семафор или буферизированный канал

5.3 Кэширование

    Кэшировать результаты в памяти на 15 минут

    При повторном вызове использовать кэш, если он актуален

5.4 Обработка ошибок
Тип ошибки	Действие
Таймаут	Ретрай 3 раза с увеличением задержки
429 Too Many Requests	Пауза 1 сек, ретрай
5xx ошибки	Ретрай до 5 раз
Некорректный JSON	Логировать ошибку, пропустить монету
Нет USDT пар	Пропустить монету (не ошибка)
5.5 Пагинация

    Начинать с start=1

    Запрашивать по 200 записей (limit=200)

    Продолжать пока len(marketPairs) == limit и есть еще пары

    Общее количество пар известно из numMarketPairs

6. ГРУППИРОВКА РЕЗУЛЬТАТОВ
6.1 Проблема

У разных монет может быть одинаковый символ (тикер). Например:

    BTC (Bitcoin) - id: 1

    BTCT (Bitcoin Token) - id: 2234

6.2 Решение

Группировать биржи для каждого уникального coin_id:
go

// Входные данные:
BTC (id:1) → spot: [binance, bybit], futures: [binance, okx]
BTC (id:2234) → spot: [kucoin], futures: []

// Результат - массив групп:
Spot: [
    ["binance", "bybit"],  // группа 1 - Bitcoin (id:1)
    ["kucoin"]              // группа 2 - Bitcoin Token (id:2234)
]
Futures: [
    ["binance", "okx"],    // группа 1 - Bitcoin (id:1)
    []                      // группа 2 - Bitcoin Token (id:2234)
]

6.3 Ключ группировки
go

type GroupKey struct {
    Symbol string
    CoinID int64
    Slug   string
}

7. ФОРМАТ ВЫВОДА
json

{
  "snapshot": {
    "BTC": {
      "symbol": "BTC",
      "coin_id": 1,
      "name": "Bitcoin",
      "spot": [
        ["binance", "bybit", "kucoin"],
        ["okx"]
      ],
      "futures": [
        ["binance", "bybit"],
        ["okx"]
      ],
      "errors": []
    },
    "ETH": {
      "symbol": "ETH",
      "coin_id": 1027,
      "name": "Ethereum",
      "spot": [
        ["binance", "bybit", "okx"]
      ],
      "futures": [
        ["binance", "bybit"]
      ],
      "errors": []
    }
  },
  "metadata": {
    "total_coins": 3124,
    "successfully_parsed": 2800,
    "failed_coins": 324,
    "total_http_requests": 5600,
    "start_time": "2026-08-15T10:00:00Z",
    "end_time": "2026-08-15T10:15:30Z",
    "duration_ms": 930000
  }
}

8. ИНТЕРФЕЙСЫ
go

type CMCService interface {
    // Запуск парсинга
    Start(ctx context.Context) error
    
    // Получение текущего снапшота
    GetSnapshot() Snapshot
    
    // Получение данных по конкретному тикеру
    GetTicker(symbol string) (TickerOutput, bool)
    
    // Обновление данных принудительно
    Refresh(ctx context.Context) error
}

type CMCConfig struct {
    ProxyList      []string      // список прокси (до 40)
    UpdateInterval time.Duration // интервал обновления (15 мин)
    RequestTimeout time.Duration // таймаут запроса (25 сек)
    MaxRetries     int           // макс ретраев (5)
    Workers        int           // кол-во воркеров (до 24)
    PageLimit      int           // лимит на страницу (200)
}

9. КОНФИГУРАЦИЯ
yaml

cmc_matcher:
  enabled: true
  update_interval: 15m
  warmup_interval: 30s
  request_timeout: 25s
  max_retries: 5
  max_workers: 24
  page_limit: 200
  proxy_pool_size: 40
  exchange_mapping:
    # mapping exchangeId → exchange name
    1: "binance"
    2: "bybit"
    3: "okx"
    4: "kucoin"
    5: "gate"
    # ... и так далее

10. ТЕСТОВЫЕ СЦЕНАРИИ
Сценарий	Ожидаемый результат
Нет интернета	Ошибка, не крашится
Нет прокси	Работа через прямой IP
Одна монета имеет 0 пар	Spot: [], Futures: []
50 монет одновременно	Не более 24 параллельных запросов
429 ошибка	Ретрай с задержкой
Обновление через 15 минут	Используется кэш, новые данные не дублируются
11. ЛОГИРОВАНИЕ
go

// Информационные логи
logger.Info("CMC parser started", 
    zap.Int("total_coins", len(symbolMap)),
    zap.Int("proxy_pool", len(proxies)))

logger.Info("Processing coin", 
    zap.String("symbol", symbol),
    zap.Int64("id", id),
    zap.Int("attempt", attempt))

// Warnings (не критические)
logger.Warn("Coin has no USDT pairs",
    zap.String("symbol", symbol),
    zap.Int64("id", id))

// Errors (критические)
logger.Error("Failed to fetch market pairs",
    zap.String("symbol", symbol),
    zap.Int64("id", id),
    zap.Error(err))

// Финальная статистика
logger.Info("CMC parser finished",
    zap.Int("processed", processed),
    zap.Int("failed", failed),
    zap.Int("total_pairs", totalPairs),
    zap.Duration("duration", elapsed))

12. ОГРАНИЧЕНИЯ

    Не более 40 прокси одновременно

    Не более 24 параллельных запросов

    Не более 5 ретраев на один запрос

    Кэш живет 15 минут

    Только USDT пары

    Только CEX биржи (centerType == "cex")

Это ТЗ готово к реализации. Нужно ли что-то уточнить или добавить?