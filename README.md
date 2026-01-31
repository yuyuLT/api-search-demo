# api-search-demo

Go + MySQL で、DB を検索して JSON を返す最小 API デモです。  
検索条件は `category` と `material` の 2 つ（任意）で、ページングも備えます。

## エンドポイント

| Method | Path        | 説明                                          |
| ------ | ----------- | --------------------------------------------- |
| GET    | `/healthz`  | DB 疎通確認                                   |
| GET    | `/v1/items` | 検索 API（category/material + page/per_page） |

> デモでは `SELECT *` でレコード全体を返します（本番では返却カラムの固定/制限推奨）。

---

## Requirements

- Go 1.21+
- MySQL（ローカルで起動できること）
- curl
- （任意）jq

---

## セットアップ

### 1. MySQL: 起動確認

```bash
mysql --protocol=TCP -h 127.0.0.1 -P 3306 -uroot
```

### 2. DB / User 作成

```sql
CREATE DATABASE IF NOT EXISTS demo DEFAULT CHARACTER SET utf8mb4;

DROP USER IF EXISTS 'demo'@'%';
CREATE USER 'demo'@'%' IDENTIFIED BY 'demo';

GRANT ALL PRIVILEGES ON demo.* TO 'demo'@'%';
FLUSH PRIVILEGES;
```

### 3. Schema / Seed の投入

```bash
mysql --protocol=TCP -h 127.0.0.1 -P 3306 -udemo -p demo < sql/schema.sql
mysql --protocol=TCP -h 127.0.0.1 -P 3306 -udemo -p demo < sql/seed.sql
```

投入確認:

```bash
mysql --protocol=TCP -h 127.0.0.1 -P 3306 -udemo -p demo -e "SELECT id, category, material, name FROM items;"
```

---

## API 起動

**bash/zsh:**

```bash
DB_DSN="demo:demo@tcp(127.0.0.1:3306)/demo?parseTime=true&charset=utf8mb4&loc=Asia%2FTokyo" \
PORT=8080 \
go run ./cmd/api
```

**fish:**

```fish
env DB_DSN="demo:demo@tcp(127.0.0.1:3306)/demo?parseTime=true&charset=utf8mb4&loc=Asia%2FTokyo" PORT=8080 go run ./cmd/api
```

---

## 動作確認

### Health check

```bash
curl -i http://127.0.0.1:8080/healthz
```

### 検索 API

```bash
# 全件
curl "http://127.0.0.1:8080/v1/items"

# category 指定
curl "http://127.0.0.1:8080/v1/items?category=cat1"

# material 指定
curl "http://127.0.0.1:8080/v1/items?material=wood"

# category + material
curl "http://127.0.0.1:8080/v1/items?category=cat1&material=wood"

# ページング
curl "http://127.0.0.1:8080/v1/items?page=1&per_page=2"
curl "http://127.0.0.1:8080/v1/items?page=2&per_page=2"

# jq で整形
curl -s "http://127.0.0.1:8080/v1/items?category=cat1" | jq .
```
