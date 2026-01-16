# Conturs Scoring Service

Lead scoring microservice written in Go. Zero external dependencies - stdlib only.

## Quick Start

```bash
go run main.go
```

```bash
curl -X POST http://localhost:8082/leads \
  -H "Content-Type: application/json" \
  -d '{
    "leads": [{"email": "john@example.com", "company": "Acme Inc"}],
    "api_key": "your_key",
    "email": "you@company.com"
  }'
```

## API

### POST /leads

Score a batch of leads.

**Request:**

```json
{
  "leads": [
    {
      "email": "john@example.com",
      "firstname": "John",
      "lastname": "Doe",
      "company": "Acme Inc",
      "jobtitle": "CEO",
      "industry": "Technology",
      "phone": "+1234567890",
      "city": "San Francisco",
      "country": "USA",
      "lead_status": "qualified",
      "email_open_count": 5,
      "email_click_count": 2,
      "num_deals": 1,
      "deal_amount": 50000,
      "create_date": "2024-01-15",
      "notes_last_updated": "2024-01-20"
    }
  ],
  "api_key": "sk_your_api_key",
  "email": "your@email.com",
  "client_id": "optional_override"
}
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
        {"name": "Lead Source", "weight": 0.10, "value": 1.0, "contribution": 0.10},
        {"name": "Company Match", "weight": 0.15, "value": 1.0, "contribution": 0.15}
      ]
    }
  ],
  "method": "similar_clients",
  "client_id": "abc123"
}
```

### GET /health

```bash
curl http://localhost:8082/health
```

```json
{"status": "healthy", "service": "scoring-service"}
```

## Scoring Factors

| Factor | Weight | Description |
|--------|--------|-------------|
| `lead_source` | 0.10 | Has email address |
| `has_valid_email` | 0.10 | Email contains @ |
| `has_company_match` | 0.15 | Has company name |
| `industry_match` | 0.05 | Has industry |
| `days_since_created` | 0.10 | Lead freshness (0-90 days) |
| `lead_status` | 0.10 | CRM status value |
| `engagement_score` | 0.10 | Email opens + clicks |
| `profile_completeness` | 0.15 | % of fields filled |
| `company_size_bucket` | 0.10 | Job title seniority |
| `recency_score` | 0.05 | Recent activity |

## Score Labels

| Score | Label |
|-------|-------|
| 80-100 | Hot Lead |
| 60-79 | Warm Lead |
| 40-59 | Cool Lead |
| 0-39 | Cold Lead |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CONFIG_API_URL` | `https://api.conturs.com` | Config API endpoint |
| `PORT` | `8082` | Service port |

## Build

```bash
# Local
go build -o scoring main.go

# Linux (from Windows)
set GOOS=linux
set GOARCH=amd64
go build -o scoring main.go
```

## Docker

```bash
docker build -t conturs-scoring .
docker run -p 8082:8082 conturs-scoring
```

## License

MIT
