# Conturs Scoring Service

A lightweight, high-performance lead scoring microservice written in Go. Zero external dependencies - uses only the Go standard library.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## Overview

Conturs Scoring Service calculates lead scores based on customizable weights. It integrates with the Conturs Config API to fetch personalized scoring configurations, or can be used standalone with default weights.

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Client    │────▶│ Scoring Service  │────▶│  Config API     │
│  (HubSpot,  │     │   (This repo)    │     │  (Optional)     │
│   CRM, etc) │◀────│                  │◀────│                 │
└─────────────┘     └──────────────────┘     └─────────────────┘
```

## Features

- **Zero Dependencies** - Built with Go stdlib only
- **Fast** - Scores thousands of leads per second
- **Customizable** - 10 scoring factors with configurable weights
- **Simple API** - Single endpoint for batch scoring
- **Self-Hosted** - Run anywhere: Docker, VM, serverless

## Quick Start

### Run Locally

```bash
git clone https://github.com/conturs/scoring-service.git
cd scoring-service
go run main.go
```

### Run with Docker

```bash
docker build -t conturs-scoring .
docker run -p 8082:8082 conturs-scoring
```

### Test the API

```bash
curl -X POST http://localhost:8082/leads \
  -H "Content-Type: application/json" \
  -d '{
    "leads": [{"email": "john@example.com", "company": "Acme Inc", "jobtitle": "CEO"}],
    "api_key": "your_key",
    "email": "you@company.com"
  }'
```

## API Reference

### POST /leads

Score a batch of leads.

#### Request

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

#### Response

```json
{
  "scores": [
    {
      "email": "john@example.com",
      "score": 85,
      "label": "Hot Lead",
      "factors": [
        {"name": "Lead Source", "weight": 0.10, "value": 1.0, "contribution": 0.10},
        {"name": "Company Match", "weight": 0.15, "value": 1.0, "contribution": 0.15},
        {"name": "Company Size", "weight": 0.10, "value": 1.0, "contribution": 0.10}
      ]
    }
  ],
  "method": "similar_clients",
  "client_id": "abc123"
}
```

#### Lead Fields

| Field | Type | Description |
|-------|------|-------------|
| `email` | string | Lead's email address (required for identification) |
| `firstname` | string | First name |
| `lastname` | string | Last name |
| `company` | string | Company name |
| `jobtitle` | string | Job title (used for seniority scoring) |
| `industry` | string | Industry/sector |
| `phone` | string | Phone number |
| `city` | string | City |
| `country` | string | Country |
| `lead_status` | string | CRM status: `new`, `open`, `in_progress`, `qualified`, `unqualified` |
| `email_open_count` | int | Number of email opens |
| `email_click_count` | int | Number of email clicks |
| `num_deals` | int | Number of associated deals |
| `deal_amount` | float | Total deal value |
| `create_date` | string | Lead creation date (ISO 8601 or timestamp) |
| `notes_last_updated` | string | Last activity date |

### GET /health

Health check endpoint.

```bash
curl http://localhost:8082/health
```

```json
{"status": "healthy", "service": "scoring-service"}
```

## Scoring Factors

The service uses 10 factors to calculate lead scores:

| Factor | Default Weight | Description |
|--------|---------------|-------------|
| `lead_source` | 0.10 | Has email address |
| `has_valid_email` | 0.10 | Email contains @ symbol |
| `has_company_match` | 0.15 | Has company name |
| `industry_match` | 0.05 | Has industry specified |
| `days_since_created` | 0.10 | Lead freshness (0-90 days) |
| `lead_status` | 0.10 | CRM status value |
| `engagement_score` | 0.10 | Email opens + clicks |
| `profile_completeness` | 0.15 | % of fields filled |
| `company_size_bucket` | 0.10 | Job title seniority |
| `recency_score` | 0.05 | Recent activity |

### Score Labels

| Score Range | Label |
|-------------|-------|
| 80-100 | 🔥 Hot Lead |
| 60-79 | 🌡️ Warm Lead |
| 40-59 | ❄️ Cool Lead |
| 0-39 | 🧊 Cold Lead |

### Job Title Seniority

Job titles are scored based on decision-making authority:

| Title Contains | Score |
|----------------|-------|
| CEO, Founder, Owner | 1.0 |
| Director, VP, Chief | 0.8 |
| Manager, Head | 0.6 |
| Other | 0.3 |

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CONFIG_API_URL` | `https://api.conturs.com` | Config API endpoint |
| `PORT` | `8082` | Service port |

### Standalone Mode

To run without the Config API, the service will use default weights if the API is unavailable.

```bash
# Use custom config API
CONFIG_API_URL=http://localhost:8000 go run main.go

# Use different port
PORT=9000 go run main.go
```

## Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o scoring main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/scoring .
EXPOSE 8082
CMD ["./scoring"]
```

```bash
docker build -t conturs-scoring .
docker run -d -p 8082:8082 \
  -e CONFIG_API_URL=https://api.conturs.com \
  conturs-scoring
```

### Docker Compose

```yaml
version: '3.8'
services:
  scoring:
    build: .
    ports:
      - "8082:8082"
    environment:
      - CONFIG_API_URL=https://api.conturs.com
    restart: unless-stopped
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: scoring-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: scoring
  template:
    metadata:
      labels:
        app: scoring
    spec:
      containers:
      - name: scoring
        image: conturs/scoring-service:latest
        ports:
        - containerPort: 8082
        env:
        - name: CONFIG_API_URL
          value: "https://api.conturs.com"
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: scoring-service
spec:
  selector:
    app: scoring
  ports:
  - port: 80
    targetPort: 8082
```

## Integration Examples

### HubSpot Workflow

```javascript
// HubSpot Custom Code Action
const response = await fetch('https://scoring.conturs.com/leads', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    leads: [{
      email: contact.email,
      firstname: contact.firstname,
      lastname: contact.lastname,
      company: contact.company,
      jobtitle: contact.jobtitle,
      lead_status: contact.hs_lead_status,
      email_open_count: contact.hs_email_open_count,
      email_click_count: contact.hs_email_click_count
    }],
    api_key: process.env.CONTURS_API_KEY,
    email: 'your@company.com'
  })
});

const result = await response.json();
return { lead_score: result.scores[0].score };
```

### Python

```python
import requests

def score_leads(leads: list, api_key: str, email: str) -> dict:
    response = requests.post(
        'https://scoring.conturs.com/leads',
        json={
            'leads': leads,
            'api_key': api_key,
            'email': email
        }
    )
    return response.json()

# Usage
result = score_leads(
    leads=[{'email': 'john@example.com', 'company': 'Acme'}],
    api_key='sk_xxx',
    email='you@company.com'
)
print(f"Score: {result['scores'][0]['score']}")
```

### JavaScript/Node.js

```javascript
async function scoreLeads(leads, apiKey, email) {
  const response = await fetch('https://scoring.conturs.com/leads', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ leads, api_key: apiKey, email })
  });
  return response.json();
}

// Usage
const result = await scoreLeads(
  [{ email: 'john@example.com', company: 'Acme' }],
  'sk_xxx',
  'you@company.com'
);
console.log(`Score: ${result.scores[0].score}`);
```

### n8n Workflow

Use HTTP Request node with:
- **Method**: POST
- **URL**: `https://scoring.conturs.com/leads`
- **Body**:
```json
{
  "leads": [{{ $json }}],
  "api_key": "{{ $env.CONTURS_API_KEY }}",
  "email": "your@company.com"
}
```

## Development

### Build

```bash
# Local build
go build -o scoring main.go

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o scoring main.go

# With optimizations
go build -ldflags="-s -w" -o scoring main.go
```

### Test

```bash
go test ./...
```

### Lint

```bash
golangci-lint run
```

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) first.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- 📖 [Documentation](https://docs.conturs.com)
- 💬 [Discord Community](https://discord.gg/conturs)
- 🐛 [Issue Tracker](https://github.com/conturs/scoring-service/issues)
- ✉️ [Email](mailto:support@conturs.com)
