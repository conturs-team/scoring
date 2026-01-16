# Conturs Scoring Service

Lead scoring microservice written in Go. Zero external dependencies - stdlib only.

## How It Works

```
┌─────────────┐         ┌──────────────────┐         ┌─────────────────┐
│   Client    │────────▶│ Scoring Service  │────────▶│  Config API     │
│             │         │   POST /leads    │         │  POST /config   │
│             │◀────────│                  │◀────────│                 │
└─────────────┘         └──────────────────┘         └─────────────────┘
     leads[]                  scores[]                   weights{}
```

1. Client sends leads array to Scoring Service
2. Scoring Service fetches personalized weights from Config API
3. Each lead is scored using the weights
4. Scores with breakdown are returned

## Quick Start

```bash
# Run
go run main.go

# Build
go build -o scoring main.go
./scoring
```

Service starts on `http://localhost:8082`

---

## API Reference

### POST /leads

Score a batch of leads.

#### Request

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `leads` | array | Yes | Array of lead objects to score |
| `api_key` | string | Yes | Your API key |
| `email` | string | Yes | Your account email |
| `client_id` | string | No | Override email for config lookup |

#### Lead Object

| Field | Type | Description |
|-------|------|-------------|
| `email` | string | Lead's email address |
| `firstname` | string | First name |
| `lastname` | string | Last name |
| `company` | string | Company name |
| `jobtitle` | string | Job title (used for seniority scoring) |
| `industry` | string | Industry/sector |
| `phone` | string | Phone number |
| `city` | string | City |
| `country` | string | Country |
| `lead_status` | string | Status: `new`, `open`, `in_progress`, `qualified`, `unqualified` |
| `email_open_count` | int | Number of email opens |
| `email_click_count` | int | Number of email clicks |
| `num_deals` | int | Number of associated deals |
| `deal_amount` | float | Total deal value |
| `create_date` | string | Creation date (ISO 8601 or Unix ms) |
| `notes_last_updated` | string | Last activity date |

#### Example Request

```bash
curl -X POST http://localhost:8082/leads \
  -H "Content-Type: application/json" \
  -d '{
    "leads": [
      {
        "email": "john.doe@techcorp.com",
        "firstname": "John",
        "lastname": "Doe",
        "company": "TechCorp Inc",
        "jobtitle": "CEO",
        "industry": "Technology",
        "phone": "+1-555-123-4567",
        "city": "San Francisco",
        "country": "USA",
        "lead_status": "qualified",
        "email_open_count": 12,
        "email_click_count": 5,
        "num_deals": 2,
        "create_date": "2024-12-01",
        "notes_last_updated": "2025-01-10"
      },
      {
        "email": "jane@startup.io",
        "company": "Startup.io",
        "jobtitle": "Marketing Manager",
        "lead_status": "open",
        "email_open_count": 3
      }
    ],
    "api_key": "sk_your_api_key",
    "email": "sales@yourcompany.com"
  }'
```

#### Example Response

```json
{
  "scores": [
    {
      "email": "john.doe@techcorp.com",
      "score": 87,
      "label": "Hot Lead",
      "factors": [
        {
          "name": "Lead Source",
          "weight": 0.10,
          "value": 1.0,
          "contribution": 0.10
        },
        {
          "name": "Valid Email",
          "weight": 0.10,
          "value": 1.0,
          "contribution": 0.10
        },
        {
          "name": "Company Match",
          "weight": 0.15,
          "value": 1.0,
          "contribution": 0.15
        },
        {
          "name": "Industry Match",
          "weight": 0.05,
          "value": 1.0,
          "contribution": 0.05
        },
        {
          "name": "Recency",
          "weight": 0.10,
          "value": 0.48,
          "contribution": 0.048
        },
        {
          "name": "Lead Status",
          "weight": 0.10,
          "value": 1.0,
          "contribution": 0.10
        },
        {
          "name": "Engagement",
          "weight": 0.10,
          "value": 1.0,
          "contribution": 0.10
        },
        {
          "name": "Profile Complete",
          "weight": 0.15,
          "value": 1.0,
          "contribution": 0.15
        },
        {
          "name": "Company Size",
          "weight": 0.10,
          "value": 1.0,
          "contribution": 0.10
        },
        {
          "name": "Activity Recency",
          "weight": 0.05,
          "value": 0.7,
          "contribution": 0.035
        }
      ]
    },
    {
      "email": "jane@startup.io",
      "score": 48,
      "label": "Cool Lead",
      "factors": [
        {
          "name": "Lead Source",
          "weight": 0.10,
          "value": 1.0,
          "contribution": 0.10
        },
        {
          "name": "Valid Email",
          "weight": 0.10,
          "value": 1.0,
          "contribution": 0.10
        },
        {
          "name": "Company Match",
          "weight": 0.15,
          "value": 1.0,
          "contribution": 0.15
        },
        {
          "name": "Lead Status",
          "weight": 0.10,
          "value": 0.5,
          "contribution": 0.05
        },
        {
          "name": "Engagement",
          "weight": 0.10,
          "value": 0.12,
          "contribution": 0.012
        },
        {
          "name": "Profile Complete",
          "weight": 0.15,
          "value": 0.33,
          "contribution": 0.05
        },
        {
          "name": "Company Size",
          "weight": 0.10,
          "value": 0.6,
          "contribution": 0.06
        },
        {
          "name": "Activity Recency",
          "weight": 0.05,
          "value": 0.2,
          "contribution": 0.01
        }
      ]
    }
  ],
  "method": "similar_clients",
  "client_id": "123456"
}
```

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `scores` | array | Array of scored leads |
| `scores[].email` | string | Lead's email |
| `scores[].score` | int | Score 0-100 |
| `scores[].label` | string | Human-readable label |
| `scores[].factors` | array | Breakdown of scoring factors |
| `method` | string | Config method used (`similar_clients`, `industry_prior`) |
| `client_id` | string | Client ID from Config API |

---

### GET /health

Health check endpoint.

#### Example Request

```bash
curl http://localhost:8082/health
```

#### Example Response

```json
{
  "status": "healthy",
  "service": "scoring-service"
}
```

---

## Scoring Logic

### Score Calculation

Final score is calculated as:

```
score = Σ(factor_value × factor_weight) × 100
```

Score is clamped to 0-100 range.

### Score Labels

| Score Range | Label |
|-------------|-------|
| 80-100 | Hot Lead |
| 60-79 | Warm Lead |
| 40-59 | Cool Lead |
| 0-39 | Cold Lead |

### Scoring Factors

#### 1. Lead Source (`lead_source`)

Checks if lead has an email address.

| Condition | Value |
|-----------|-------|
| Has email | 1.0 |
| No email | 0.0 |

#### 2. Valid Email (`has_valid_email`)

Validates email contains `@`.

| Condition | Value |
|-----------|-------|
| Contains @ | 1.0 |
| Invalid | 0.0 |

#### 3. Company Match (`has_company_match`)

Checks if company name is provided.

| Condition | Value |
|-----------|-------|
| Has company | 1.0 |
| No company | 0.0 |

#### 4. Industry Match (`industry_match`)

Checks if industry is provided.

| Condition | Value |
|-----------|-------|
| Has industry | 1.0 |
| No industry | 0.0 |

#### 5. Days Since Created (`days_since_created`)

Lead freshness based on creation date. Newer leads score higher.

```
value = max(0, 1 - days/90)
```

| Days | Value |
|------|-------|
| 0 | 1.0 |
| 45 | 0.5 |
| 90+ | 0.0 |

#### 6. Lead Status (`lead_status`)

CRM status value.

| Status | Value |
|--------|-------|
| qualified | 1.0 |
| in_progress | 0.7 |
| open | 0.5 |
| new | 0.3 |
| unqualified | 0.1 |
| other | 0.5 |

#### 7. Engagement Score (`engagement_score`)

Based on email opens and clicks.

```
value = min(1, (opens/10 × 0.4) + (clicks/3 × 0.6))
```

| Opens | Clicks | Value |
|-------|--------|-------|
| 10+ | 3+ | 1.0 |
| 5 | 2 | 0.6 |
| 0 | 0 | 0.0 |

#### 8. Profile Completeness (`profile_completeness`)

Percentage of filled profile fields.

Fields counted: email, firstname, lastname, company, jobtitle, phone, city, country, industry

```
value = filled_fields / 9
```

| Filled Fields | Value |
|---------------|-------|
| 9/9 | 1.0 |
| 5/9 | 0.56 |
| 1/9 | 0.11 |

#### 9. Company Size / Seniority (`company_size_bucket`)

Based on job title keywords.

| Job Title Contains | Value |
|--------------------|-------|
| ceo, founder, owner | 1.0 |
| director, vp, chief | 0.8 |
| manager, head | 0.6 |
| other | 0.3 |

#### 10. Activity Recency (`recency_score`)

Recent activity indicator.

| Condition | Value |
|-----------|-------|
| Has deals (`num_deals > 0`) | 0.7 |
| Notes updated < 30 days | 0.5 |
| No recent activity | 0.2 |

---

## Error Responses

### 400 Bad Request

```json
{
  "error": "Invalid request body: ..."
}
```

```json
{
  "error": "No leads provided"
}
```

### 401 Unauthorized

```json
{
  "error": "api_key and email required"
}
```

```json
{
  "error": "Invalid API key"
}
```

### 405 Method Not Allowed

```json
{
  "error": "Method not allowed"
}
```

### 500 Internal Server Error

```json
{
  "error": "Failed to fetch scoring config"
}
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CONFIG_API_URL` | `https://api.conturs.com` | Config API base URL |
| `PORT` | `8082` | Service port |

### Example

```bash
CONFIG_API_URL=http://localhost:8000 PORT=9000 go run main.go
```

---

## CORS

Allowed origins:
- `https://conturs.com`
- `https://www.conturs.com`
- `https://app.conturs.com`

---

## Build & Deploy

### Local Build

```bash
go build -o scoring main.go
./scoring
```

### Cross-Compile for Linux

```bash
# From Windows
set GOOS=linux
set GOARCH=amd64
go build -o scoring main.go

# From macOS/Linux
GOOS=linux GOARCH=amd64 go build -o scoring main.go
```

### Docker

```bash
docker build -t conturs-scoring .
docker run -p 8082:8082 -e CONFIG_API_URL=https://api.conturs.com conturs-scoring
```

---

## License

MIT
