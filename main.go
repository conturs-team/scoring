package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

// ============================================================================
// MODELS
// ============================================================================

type Lead struct {
	Email            string  `json:"email"`
	Firstname        string  `json:"firstname,omitempty"`
	Lastname         string  `json:"lastname,omitempty"`
	Company          string  `json:"company,omitempty"`
	Jobtitle         string  `json:"jobtitle,omitempty"`
	Industry         string  `json:"industry,omitempty"`
	Phone            string  `json:"phone,omitempty"`
	City             string  `json:"city,omitempty"`
	Country          string  `json:"country,omitempty"`
	LeadStatus       string  `json:"lead_status,omitempty"`
	EmailOpenCount   int     `json:"email_open_count,omitempty"`
	EmailClickCount  int     `json:"email_click_count,omitempty"`
	NumDeals         int     `json:"num_deals,omitempty"`
	DealAmount       float64 `json:"deal_amount,omitempty"`
	CreateDate       string  `json:"create_date,omitempty"`
	NotesLastUpdated string  `json:"notes_last_updated,omitempty"`
}

type LeadsRequest struct {
	Leads    []Lead `json:"leads"`
	ClientID string `json:"client_id,omitempty"`
	APIKey   string `json:"api_key"`
	Email    string `json:"email"`
}

type ScoreFactor struct {
	Name         string  `json:"name"`
	Weight       float64 `json:"weight"`
	Value        float64 `json:"value"`
	Contribution float64 `json:"contribution"`
}

type LeadScore struct {
	Email   string        `json:"email"`
	Score   int           `json:"score"`
	Label   string        `json:"label"`
	Factors []ScoreFactor `json:"factors"`
}

type LeadsResponse struct {
	Scores   []LeadScore `json:"scores"`
	Method   string      `json:"method"`
	ClientID string      `json:"client_id"`
}

type ConfigResponse struct {
	Weights  map[string]float64 `json:"weights"`
	ClientID string             `json:"client_id"`
	Method   string             `json:"method"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// ============================================================================
// CONFIG
// ============================================================================

var (
	configAPIURL = getEnv("CONFIG_API_URL", "https://api.conturs.com")
	port         = getEnv("PORT", "8080")
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// ============================================================================
// SCORING LOGIC
// ============================================================================

var weightLabels = map[string]string{
	"lead_source":          "Lead Source",
	"days_since_created":   "Recency",
	"lead_status":          "Lead Status",
	"has_email_valid":      "Valid Email",
	"has_company_match":    "Company Match",
	"engagement_score":     "Engagement",
	"profile_completeness": "Profile Complete",
	"company_size_bucket":  "Company Size",
	"industry_match":       "Industry Match",
	"recency_score":        "Activity Recency",
}

func calculateScore(lead Lead, weights map[string]float64) LeadScore {
	var factors []ScoreFactor
	totalScore := 0.0

	// 1. lead_source
	if w, ok := weights["lead_source"]; ok && w > 0 {
		value := 0.0
		if lead.Email != "" {
			value = 1.0
		}
		contribution := value * w
		totalScore += contribution
		if value > 0 {
			factors = append(factors, ScoreFactor{
				Name:         weightLabels["lead_source"],
				Weight:       w,
				Value:        value,
				Contribution: contribution,
			})
		}
	}

	// 2. days_since_created
	if w, ok := weights["days_since_created"]; ok && w > 0 && lead.CreateDate != "" {
		createTime, err := parseDate(lead.CreateDate)
		if err == nil {
			daysSinceCreated := int(time.Since(createTime).Hours() / 24)
			value := math.Max(0, math.Min(1, 1-float64(daysSinceCreated)/90))
			contribution := value * w
			totalScore += contribution
			factors = append(factors, ScoreFactor{
				Name:         weightLabels["days_since_created"],
				Weight:       w,
				Value:        value,
				Contribution: contribution,
			})
		}
	}

	// 3. lead_status
	if w, ok := weights["lead_status"]; ok && w > 0 && lead.LeadStatus != "" {
		statusScores := map[string]float64{
			"new":         0.3,
			"open":        0.5,
			"in_progress": 0.7,
			"qualified":   1.0,
			"unqualified": 0.1,
		}
		value := 0.5
		if v, ok := statusScores[strings.ToLower(lead.LeadStatus)]; ok {
			value = v
		}
		contribution := value * w
		totalScore += contribution
		factors = append(factors, ScoreFactor{
			Name:         weightLabels["lead_status"],
			Weight:       w,
			Value:        value,
			Contribution: contribution,
		})
	}

	// 4. has_email_valid
	if w, ok := weights["has_email_valid"]; ok && w > 0 {
		value := 0.0
		if lead.Email != "" && strings.Contains(lead.Email, "@") {
			value = 1.0
		}
		contribution := value * w
		totalScore += contribution
		if value > 0 {
			factors = append(factors, ScoreFactor{
				Name:         weightLabels["has_email_valid"],
				Weight:       w,
				Value:        value,
				Contribution: contribution,
			})
		}
	}

	// 5. has_company_match
	if w, ok := weights["has_company_match"]; ok && w > 0 {
		value := 0.0
		if lead.Company != "" {
			value = 1.0
		}
		contribution := value * w
		totalScore += contribution
		if value > 0 {
			factors = append(factors, ScoreFactor{
				Name:         weightLabels["has_company_match"],
				Weight:       w,
				Value:        value,
				Contribution: contribution,
			})
		}
	}

	// 6. engagement_score
	if w, ok := weights["engagement_score"]; ok && w > 0 {
		opens := float64(lead.EmailOpenCount)
		clicks := float64(lead.EmailClickCount)
		value := math.Min(1, (opens/10)*0.4+(clicks/3)*0.6)
		contribution := value * w
		totalScore += contribution
		if value > 0 {
			factors = append(factors, ScoreFactor{
				Name:         weightLabels["engagement_score"],
				Weight:       w,
				Value:        value,
				Contribution: contribution,
			})
		}
	}

	// 7. profile_completeness
	if w, ok := weights["profile_completeness"]; ok && w > 0 {
		fields := []string{
			lead.Email, lead.Firstname, lead.Lastname,
			lead.Company, lead.Jobtitle, lead.Phone,
			lead.City, lead.Country, lead.Industry,
		}
		filledCount := 0
		for _, f := range fields {
			if f != "" {
				filledCount++
			}
		}
		value := float64(filledCount) / float64(len(fields))
		contribution := value * w
		totalScore += contribution
		factors = append(factors, ScoreFactor{
			Name:         weightLabels["profile_completeness"],
			Weight:       w,
			Value:        value,
			Contribution: contribution,
		})
	}

	// 8. company_size_bucket
	if w, ok := weights["company_size_bucket"]; ok && w > 0 && lead.Jobtitle != "" {
		title := strings.ToLower(lead.Jobtitle)
		value := 0.3
		if strings.Contains(title, "ceo") || strings.Contains(title, "founder") || strings.Contains(title, "owner") {
			value = 1.0
		} else if strings.Contains(title, "director") || strings.Contains(title, "vp") || strings.Contains(title, "chief") {
			value = 0.8
		} else if strings.Contains(title, "manager") || strings.Contains(title, "head") {
			value = 0.6
		}
		contribution := value * w
		totalScore += contribution
		factors = append(factors, ScoreFactor{
			Name:         weightLabels["company_size_bucket"],
			Weight:       w,
			Value:        value,
			Contribution: contribution,
		})
	}

	// 9. industry_match
	if w, ok := weights["industry_match"]; ok && w > 0 {
		value := 0.0
		if lead.Industry != "" {
			value = 1.0
		}
		contribution := value * w
		totalScore += contribution
		if value > 0 {
			factors = append(factors, ScoreFactor{
				Name:         weightLabels["industry_match"],
				Weight:       w,
				Value:        value,
				Contribution: contribution,
			})
		}
	}

	// 10. recency_score
	if w, ok := weights["recency_score"]; ok && w > 0 {
		hasDeals := lead.NumDeals > 0
		hasRecentNotes := false
		if lead.NotesLastUpdated != "" {
			if notesTime, err := parseDate(lead.NotesLastUpdated); err == nil {
				hasRecentNotes = time.Since(notesTime).Hours() < 30*24
			}
		}
		value := 0.2
		if hasDeals {
			value = 0.7
		} else if hasRecentNotes {
			value = 0.5
		}
		contribution := value * w
		totalScore += contribution
		factors = append(factors, ScoreFactor{
			Name:         weightLabels["recency_score"],
			Weight:       w,
			Value:        value,
			Contribution: contribution,
		})
	}

	score := int(math.Round(math.Min(100, math.Max(0, totalScore*100))))

	return LeadScore{
		Email:   lead.Email,
		Score:   score,
		Label:   getScoreLabel(score),
		Factors: factors,
	}
}

func parseDate(dateStr string) (time.Time, error) {
	if !strings.Contains(dateStr, "-") {
		var ts int64
		if _, err := fmt.Sscanf(dateStr, "%d", &ts); err == nil {
			return time.UnixMilli(ts), nil
		}
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02",
		"2006/01/02",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

func getScoreLabel(score int) string {
	switch {
	case score >= 80:
		return "Hot Lead"
	case score >= 60:
		return "Warm Lead"
	case score >= 40:
		return "Cool Lead"
	default:
		return "Cold Lead"
	}
}

// ============================================================================
// API CLIENT
// ============================================================================

func fetchConfig(email, apiKey string) (*ConfigResponse, error) {
	reqBody, _ := json.Marshal(map[string]string{"email": email, "api_key": apiKey})

	resp, err := http.Post(configAPIURL+"/config", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("fetch config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("config API error %d: %s", resp.StatusCode, string(body))
	}

	var config ConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return &config, nil
}

// ============================================================================
// HANDLERS
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "scoring-service",
	})
}

func leadsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req LeadsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if req.APIKey == "" || req.Email == "" {
		writeError(w, http.StatusUnauthorized, "api_key and email required")
		return
	}

	if len(req.Leads) == 0 {
		writeError(w, http.StatusBadRequest, "No leads provided")
		return
	}

	// Use validated email for config
	configEmail := req.Email
	if req.ClientID != "" {
		configEmail = req.ClientID
	}

	config, err := fetchConfig(configEmail, req.APIKey)
	if err != nil {
		log.Printf("Failed to fetch config: %v", err)
		if strings.Contains(err.Error(), "unauthorized") {
			writeError(w, http.StatusUnauthorized, "Invalid API key")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to fetch scoring config")
		return
	}

	var scores []LeadScore
	for _, lead := range req.Leads {
		scores = append(scores, calculateScore(lead, config.Weights))
	}

	writeJSON(w, http.StatusOK, LeadsResponse{
		Scores:   scores,
		Method:   config.Method,
		ClientID: config.ClientID,
	})
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	http.HandleFunc("/health", corsMiddleware(healthHandler))
	http.HandleFunc("/leads", corsMiddleware(leadsHandler))

	addr := ":" + port
	log.Printf("Scoring service starting on %s", addr)
	log.Printf("Config API: %s", configAPIURL)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}
