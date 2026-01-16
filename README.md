# Conturs Scoring Service

Lead scoring microservice written in Go. Zero external dependencies - stdlib only.

---

## What is Conturs?

Conturs is a **personalized lead scoring system** that generates customized scoring weights for each client instead of using one-size-fits-all rules.

### The Problem

Traditional lead scoring systems use fixed rules:
- "CEO = +50 points"
- "Enterprise company = +30 points"
- "Email opened = +10 points"

But these rules don't work equally for everyone:
- A B2B SaaS company cares more about engagement metrics
- A recruitment agency prioritizes job title seniority
- An e-commerce business focuses on recency and activity

### The Solution

Conturs learns from successful conversions of **similar clients** to generate personalized weights:

```
Your Business Profile          Similar Successful Clients         Your Personalized Weights
─────────────────────    →    ──────────────────────────    →    ────────────────────────
• SaaS / Technology           • Client A: engagement=0.20        • engagement_score: 0.18
• 50 employees                • Client B: engagement=0.15        • profile_completeness: 0.14
• B2B sales                   • Client C: engagement=0.18        • company_size: 0.12
                                                                  • ...
```

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CONTURS SYSTEM                                 │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────┐      ┌─────────────────┐      ┌─────────────────┐
│   Config    │      │    Scoring      │      │   Your CRM /    │
│    API      │      │    Service      │      │   Data Source   │
│             │      │   (this repo)   │      │                 │
└──────┬──────┘      └────────┬────────┘      └────────┬────────┘
       │                      │                        │
       │  weights{}           │  scores[]              │  leads[]
       │                      │                        │
       └──────────────────────┼────────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │   Your Workflow   │
                    │   (n8n, Zapier,   │
                    │    custom code)   │
                    └───────────────────┘
```

### Config API (`api.conturs.com`)

The brain of the system:

- **Profile Enrichment** — Automatically enriches your business profile based on email (industry, company size, job title, skills)
- **Vector Similarity Search** — Finds clients similar to you using ML embeddings
- **Weight Generation** — Creates personalized scoring weights based on what worked for similar clients
- **Feedback Loop** — Collects conversion data to continuously improve recommendations

### Scoring Service (this repository)

Lightweight scoring engine:

- **Single Binary** — One Go executable, ~10MB, no dependencies
- **Batch Processing** — Score hundreds of leads in one request
- **Transparent Scoring** — Full breakdown of how each score was calculated
- **Offline Capable** — Falls back to default weights if Config API is unavailable

---

## How Scoring Works

### Step 1: Register Your Account

First, register with the Config API to get personalized weights:

```bash
curl -X POST https://api.conturs.com/config \
  -H "Content-Type: application/json" \
  -d '{"email": "sales@yourcompany.com"}'
```

Response:
```json
{
  "client_id": "12345",
  "weights": {
    "lead_source": 0.10,
    "engagement_score": 0.18,
    "profile_completeness": 0.14,
    ...
  },
  "method": "similar_clients",
  "is_new_client": true
}
```

The `method` field tells you how weights were generated:
- `similar_clients` — Based on similar successful businesses (best)
- `industry_prior` — Based on your industry defaults (good)
- `default` — Generic weights (fallback)

### Step 2: Score Your Leads

Send leads to the Scoring Service:

```bash
curl -X POST https://scoring.conturs.com/leads \
  -H "Content-Type: application/json" \
  -d '{
    "leads": [...],
    "api_key": "sk_your_api_key",
    "email": "sales@yourcompany.com"
  }'
```

The service:
1. Fetches your personalized weights from Config API
2. Calculates score for each lead using weighted formula
3. Returns scores with full factor breakdown

### Step 3: Send Feedback (Optional)

Improve recommendations by reporting outcomes:

```bash
curl -X POST https://api.conturs.com/feedback \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "12345",
    "lead_id": "lead_001",
    "action": "converted"
  }'
```

Actions and their rewards:
| Action | Reward | Meaning |
|--------|--------|---------|
| `converted` | +1.0 | Lead became a customer |
| `replied` | +0.5 | Lead responded positively |
| `clicked` | +0.2 | Lead engaged with content |
| `ignored` | -0.5 | Lead didn't respond |
| `bounced` | -1.0 | Invalid contact |

Feedback improves weight recommendations for **all similar clients** — the more feedback in the system, the better everyone's scoring becomes.

---

## Why Self-Hosted?

The Scoring Service is designed to run on your infrastructure:

| Benefit | Description |
|---------|-------------|
| **Speed** | No network latency — scoring happens locally in microseconds |
| **Privacy** | Lead PII never leaves your servers |
| **Reliability** | Works offline with cached/default weights |
| **Cost** | No per-lead pricing — score unlimited leads |
| **Compliance** | Easier GDPR/SOC2 compliance when data stays internal |

Config API is only called **once per session** to fetch weights. All scoring calculations happen locally.

---

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
| `first_name` | string | First name |
| `last_name` | string | Last name |
| `company` | string | Company name |
| `job_title` | string | Job title (used for seniority scoring) |
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
        "first_name": "John",
        "last_name": "Doe",
        "company": "TechCorp Inc",
        "job_title": "CEO",
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
        "job_title": "Marketing Manager",
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

Fields counted: email, first_name, last_name, company, job_title, phone, city, country, industry

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

## SDK Examples

### Python

```python
import requests

def score_leads(leads: list, api_key: str, email: str, base_url: str = "https://scoring.conturs.com") -> dict:
    """Score a batch of leads."""
    response = requests.post(
        f"{base_url}/leads",
        json={
            "leads": leads,
            "api_key": api_key,
            "email": email
        },
        headers={"Content-Type": "application/json"}
    )
    response.raise_for_status()
    return response.json()


# Example usage
leads = [
    {
        "email": "john.doe@techcorp.com",
        "first_name": "John",
        "last_name": "Doe",
        "company": "TechCorp Inc",
        "job_title": "CEO",
        "industry": "Technology",
        "lead_status": "qualified",
        "email_open_count": 12,
        "email_click_count": 5
    },
    {
        "email": "jane@startup.io",
        "company": "Startup.io",
        "job_title": "Marketing Manager"
    }
]

result = score_leads(
    leads=leads,
    api_key="sk_your_api_key",
    email="sales@yourcompany.com"
)

for score in result["scores"]:
    print(f"{score['email']}: {score['score']} ({score['label']})")

# Output:
# john.doe@techcorp.com: 87 (Hot Lead)
# jane@startup.io: 48 (Cool Lead)
```

---

### JavaScript / Node.js

```javascript
async function scoreLeads(leads, apiKey, email, baseUrl = 'https://scoring.conturs.com') {
  const response = await fetch(`${baseUrl}/leads`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      leads,
      api_key: apiKey,
      email,
    }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to score leads');
  }

  return response.json();
}


// Example usage
const leads = [
  {
    email: 'john.doe@techcorp.com',
    first_name: 'John',
    last_name: 'Doe',
    company: 'TechCorp Inc',
    job_title: 'CEO',
    industry: 'Technology',
    lead_status: 'qualified',
    email_open_count: 12,
    email_click_count: 5,
  },
  {
    email: 'jane@startup.io',
    company: 'Startup.io',
    job_title: 'Marketing Manager',
  },
];

scoreLeads(leads, 'sk_your_api_key', 'sales@yourcompany.com')
  .then((result) => {
    result.scores.forEach((score) => {
      console.log(`${score.email}: ${score.score} (${score.label})`);
    });
  })
  .catch(console.error);

// Output:
// john.doe@techcorp.com: 87 (Hot Lead)
// jane@startup.io: 48 (Cool Lead)
```

---

### PHP

```php
<?php

function scoreLeads(array $leads, string $apiKey, string $email, string $baseUrl = 'https://scoring.conturs.com'): array
{
    $payload = json_encode([
        'leads' => $leads,
        'api_key' => $apiKey,
        'email' => $email,
    ]);

    $ch = curl_init("{$baseUrl}/leads");
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => $payload,
        CURLOPT_HTTPHEADER => [
            'Content-Type: application/json',
            'Content-Length: ' . strlen($payload),
        ],
        CURLOPT_RETURNTRANSFER => true,
    ]);

    $response = curl_exec($ch);
    $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    curl_close($ch);

    if ($httpCode !== 200) {
        $error = json_decode($response, true);
        throw new Exception($error['error'] ?? 'Failed to score leads');
    }

    return json_decode($response, true);
}


// Example usage
$leads = [
    [
        'email' => 'john.doe@techcorp.com',
        'first_name' => 'John',
        'last_name' => 'Doe',
        'company' => 'TechCorp Inc',
        'job_title' => 'CEO',
        'industry' => 'Technology',
        'lead_status' => 'qualified',
        'email_open_count' => 12,
        'email_click_count' => 5,
    ],
    [
        'email' => 'jane@startup.io',
        'company' => 'Startup.io',
        'job_title' => 'Marketing Manager',
    ],
];

try {
    $result = scoreLeads($leads, 'sk_your_api_key', 'sales@yourcompany.com');

    foreach ($result['scores'] as $score) {
        echo "{$score['email']}: {$score['score']} ({$score['label']})\n";
    }
} catch (Exception $e) {
    echo "Error: " . $e->getMessage() . "\n";
}

// Output:
// john.doe@techcorp.com: 87 (Hot Lead)
// jane@startup.io: 48 (Cool Lead)
```

---

### Java

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import com.google.gson.Gson;
import com.google.gson.JsonArray;
import com.google.gson.JsonObject;

public class ContursScoring {

    private final String baseUrl;
    private final String apiKey;
    private final String email;
    private final HttpClient client;
    private final Gson gson;

    public ContursScoring(String apiKey, String email) {
        this(apiKey, email, "https://scoring.conturs.com");
    }

    public ContursScoring(String apiKey, String email, String baseUrl) {
        this.apiKey = apiKey;
        this.email = email;
        this.baseUrl = baseUrl;
        this.client = HttpClient.newHttpClient();
        this.gson = new Gson();
    }

    public JsonObject scoreLeads(JsonArray leads) throws Exception {
        JsonObject payload = new JsonObject();
        payload.add("leads", leads);
        payload.addProperty("api_key", apiKey);
        payload.addProperty("email", email);

        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create(baseUrl + "/leads"))
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(gson.toJson(payload)))
            .build();

        HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());

        if (response.statusCode() != 200) {
            JsonObject error = gson.fromJson(response.body(), JsonObject.class);
            throw new Exception(error.has("error") ? error.get("error").getAsString() : "Failed to score leads");
        }

        return gson.fromJson(response.body(), JsonObject.class);
    }

    public static void main(String[] args) {
        try {
            ContursScoring scoring = new ContursScoring("sk_your_api_key", "sales@yourcompany.com");

            JsonArray leads = new JsonArray();

            JsonObject lead1 = new JsonObject();
            lead1.addProperty("email", "john.doe@techcorp.com");
            lead1.addProperty("first_name", "John");
            lead1.addProperty("last_name", "Doe");
            lead1.addProperty("company", "TechCorp Inc");
            lead1.addProperty("job_title", "CEO");
            lead1.addProperty("industry", "Technology");
            lead1.addProperty("lead_status", "qualified");
            lead1.addProperty("email_open_count", 12);
            lead1.addProperty("email_click_count", 5);
            leads.add(lead1);

            JsonObject lead2 = new JsonObject();
            lead2.addProperty("email", "jane@startup.io");
            lead2.addProperty("company", "Startup.io");
            lead2.addProperty("job_title", "Marketing Manager");
            leads.add(lead2);

            JsonObject result = scoring.scoreLeads(leads);
            JsonArray scores = result.getAsJsonArray("scores");

            for (int i = 0; i < scores.size(); i++) {
                JsonObject score = scores.get(i).getAsJsonObject();
                System.out.printf("%s: %d (%s)%n",
                    score.get("email").getAsString(),
                    score.get("score").getAsInt(),
                    score.get("label").getAsString()
                );
            }

        } catch (Exception e) {
            System.err.println("Error: " + e.getMessage());
        }
    }
}

// Output:
// john.doe@techcorp.com: 87 (Hot Lead)
// jane@startup.io: 48 (Cool Lead)
```

**Maven dependency for Gson:**

```xml
<dependency>
    <groupId>com.google.code.gson</groupId>
    <artifactId>gson</artifactId>
    <version>2.10.1</version>
</dependency>
```

---

## License

MIT
