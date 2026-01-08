# Scoring Service (Go)

Мікросервіс для batch скорингу лідів. Без зовнішніх залежностей (тільки stdlib).

## Endpoints

### POST /leads

Скорить масив лідів.

```bash
curl -X POST http://localhost:8080/leads \
  -H "Content-Type: application/json" \
  -d '{
    "leads": [
      {
        "email": "john@example.com",
        "firstname": "John",
        "lastname": "Doe",
        "company": "Acme Inc",
        "jobtitle": "CEO",
        "industry": "Technology",
        "lead_status": "qualified",
        "email_open_count": 5,
        "email_click_count": 2
      }
    ],
    "client_id": "optional-client-id"
  }'
```

**Response:**
```json
{
  "scores": [
    {
      "email": "john@example.com",
      "score": 85,
      "label": "Hot Lead",
      "factors": [
        {"name": "Lead Source", "weight": 0.08, "value": 1, "contribution": 0.08}
      ]
    }
  ],
  "method": "similar_clients",
  "client_id": "abc123"
}
```

### GET /health

Health check.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CONFIG_API_URL` | `https://api.conturs.com` | Python API для конфігу |
| `PORT` | `8080` | Порт сервісу |

## Run

```bash
cd "scoring(GO)"
go run main.go
```

Або з кастомним конфігом:

```bash
CONFIG_API_URL=http://localhost:8000 PORT=9000 go run main.go
```

## Build

```bash
go build -o scoring-service main.go
./scoring-service
```
