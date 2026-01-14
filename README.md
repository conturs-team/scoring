# Scoring Service

Lead scoring microservice built with Go. No external dependencies (stdlib only).

## Endpoints

### POST /leads

Score an array of leads.

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
      "lead_status": "qualified",
      "email_open_count": 5,
      "email_click_count": 2,
      "num_deals": 1,
      "create_date": "2024-01-15",
      "notes_last_updated": "2024-01-20"
    }
  ],
  "api_key": "your_api_key",
  "email": "your@email.com",
  "client_id": "optional_client_id"
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
        {
          "name": "Lead Source",
          "weight": 0.10,
          "value": 1.0,
          "contribution": 0.10
        }
      ]
    }
  ],
  "method": "similar_clients",
  "client_id": "abc123"
}
```

**Example:**
```bash
curl -X POST http://localhost:8082/leads \
  -H "Content-Type: application/json" \
  -d '{"leads":[{"email":"john@example.com"}],"api_key":"key","email":"you@email.com"}'
```

### POST /workflow

HubSpot workflow integration endpoint.

**Request:**
```json
{
  "origin": {
    "portalId": 123456,
    "actionDefinitionId": "abc123"
  },
  "object": {
    "objectId": 789,
    "objectType": "contact",
    "properties": {
      "email": "john@example.com",
      "firstname": "John",
      "lastname": "Doe",
      "company": "Acme Inc"
    }
  }
}
```

**Response:**
```json
{
  "outputFields": {
    "score": 85,
    "status": "success"
  }
}
```

### GET /health

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "service": "scoring-service"
}
```

## Scoring Factors

| factor | weight | description |
|--------|--------|-------------|
| lead_source | 0.10 | has email address |
| has_valid_email | 0.10 | email contains @ |
| has_company_match | 0.15 | has company name |
| industry_match | 0.05 | has industry |
| days_since_created | 0.10 | days since created (0-90) |
| lead_status | 0.10 | status value (new/open/qualified) |
| engagement_score | 0.10 | email opens and clicks |
| profile_completeness | 0.15 | filled profile fields |
| company_size_bucket | 0.10 | job title seniority |
| recency_score | 0.05 | recent deals or notes |

## Score Labels

| score | label |
|-------|-------|
| 80-100 | Hot Lead |
| 60-79 | Warm Lead |
| 40-59 | Cool Lead |
| 0-39 | Cold Lead |

## Environment Variables

| variable | default | description |
|----------|---------|-------------|
| `CONFIG_API_URL` | `https://api.conturs.com/config` | config api url |
| `PORT` | `8082` | service port |

## Run

```bash
go run main.go
```

With custom config:

```bash
CONFIG_API_URL=http://localhost:8000 PORT=9000 go run main.go
```

## Build

Local:
```bash
go build -o scoring main.go
./scoring
```

For Linux (from Windows):
```bash
set GOOS=linux
set GOARCH=amd64
go build -o scoring main.go
```
