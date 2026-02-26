package main

import (
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

// Request and response types

type Lead struct {
	Email                          string  `json:"email"`
	Has_firstname                  bool    `json:"has_firstname,omitempty"`
	Has_lastname                   bool    `json:"has_lastname,omitempty"`
	Company                        string  `json:"company,omitempty"`
	Jobtitle                       string  `json:"jobtitle,omitempty"`
	Industry                       string  `json:"industry,omitempty"`
	Has_phone                      bool    `json:"has_phone,omitempty"`
	Lead_status                    string  `json:"lead_status,omitempty"`
	Dealstage                      string  `json:"dealstage,omitempty"`
	Amount                         float64 `json:"amount,omitempty"`
	Num_meetings_booked            int     `json:"num_meetings_booked,omitempty"`
	Num_calls                      int     `json:"num_calls,omitempty"`
	Hs_email_replied               int     `json:"hs_email_replied,omitempty"`
	Num_completed_tasks            int     `json:"num_completed_tasks,omitempty"`
	Hs_last_sales_activity_timestamp string `json:"hs_last_sales_activity_timestamp,omitempty"`
	Create_date                    string  `json:"create_date,omitempty"`
}

type Leads_request struct {
	Leads     []Lead `json:"leads"`
	Client_id string `json:"client_id,omitempty"`
	Email     string `json:"email"`
}

type Leads_response struct {
	Scores    []Lead_score `json:"scores"`
	Method    string       `json:"method"`
	Client_id string       `json:"client_id"`
}

type Lead_score struct {
	Email   string         `json:"email"`
	Score   int            `json:"score"`
	Label   string         `json:"label"`
	Factors []Score_factor `json:"factors"`
}

type Score_factor struct {
	Name         string  `json:"name"`
	Weight       float64 `json:"weight"`
	Value        float64 `json:"value"`
	Contribution float64 `json:"contribution"`
}

type Config_response struct {
	Weights   map[string]float64 `json:"weights"`
	Client_id string             `json:"client_id"`
	Method    string             `json:"method"`
}

type Error_response struct {
	Error string `json:"error"`
}

// Config

var (
	config_api_url = get_env("CONFIG_API_URL", "https://api.conturs.com")
	port           = get_env("PORT", "8082")
)

func get_env(key, default_value string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return default_value
}

// Scoring logic

func calculate_score(lead Lead, weights map[string]float64) Lead_score {
	var factors []Score_factor
	var total_score float64
	var total_weight_used float64

	add_factor := func(name string, weight, value float64) {
		if weight <= 0 || value <= 0 {
			return
		}
		contribution := value * weight
		total_score += contribution
		total_weight_used += weight
		factors = append(factors, Score_factor{
			Name:         name,
			Weight:       weight,
			Value:        value,
			Contribution: contribution,
		})
	}

	// 1. Deal Stage
	if weights["deal_stage"] > 0 {
		stage_scores := map[string]float64{
			"closedwon":              1.0,
			"contractsent":          0.85,
			"decisionmakerboughtin": 0.75,
			"presentationscheduled": 0.6,
			"qualifiedtobuy":       0.5,
			"appointmentscheduled":  0.4,
			"closedlost":           0.1,
		}
		stage := strings.ToLower(lead.Dealstage)
		value := 0.0
		if stage != "" {
			if v, ok := stage_scores[stage]; ok {
				value = v
			} else {
				value = 0.2
			}
		}
		add_factor("deal_stage", weights["deal_stage"], value)
	}

	// 2. Deal Amount
	if weights["deal_amount"] > 0 && lead.Amount > 0 {
		value := math.Min(1.0, lead.Amount/100000+0.3)
		add_factor("deal_amount", weights["deal_amount"], value)
	}

	// 3. Lead Status
	if weights["lead_status"] > 0 {
		status_scores := map[string]float64{
			"qualified":   1.0,
			"in_progress": 0.7,
			"open":        0.5,
			"new":         0.3,
			"unqualified": 0.1,
			"bad_timing":  0.2,
		}
		status := strings.ToLower(lead.Lead_status)
		value := 0.2
		if status != "" {
			if v, ok := status_scores[status]; ok {
				value = v
			} else {
				value = 0.3
			}
		}
		add_factor("lead_status", weights["lead_status"], value)
	}

	// 4. Days Since Last Activity
	if weights["days_since_last_activity"] > 0 {
		value := 0.05
		if lead.Hs_last_sales_activity_timestamp != "" {
			if t, err := parse_date(lead.Hs_last_sales_activity_timestamp); err == nil {
				days := int(time.Since(t).Hours() / 24)
				switch {
				case days <= 3:
					value = 1.0
				case days <= 7:
					value = 0.8
				case days <= 14:
					value = 0.6
				case days <= 30:
					value = 0.4
				case days <= 60:
					value = 0.2
				}
			}
		}
		add_factor("days_since_last_activity", weights["days_since_last_activity"], value)
	}

	// 5. Meeting Booked Count
	if weights["meeting_booked_count"] > 0 {
		value := 0.0
		if lead.Num_meetings_booked >= 3 {
			value = 1.0
		} else if lead.Num_meetings_booked >= 1 {
			value = 0.7
		}
		add_factor("meeting_booked_count", weights["meeting_booked_count"], value)
	}

	// 6. Call Completed Count
	if weights["call_completed_count"] > 0 {
		value := 0.0
		if lead.Num_calls >= 5 {
			value = 1.0
		} else if lead.Num_calls >= 2 {
			value = 0.6
		} else if lead.Num_calls >= 1 {
			value = 0.4
		}
		add_factor("call_completed_count", weights["call_completed_count"], value)
	}

	// 7. Email Reply Count
	if weights["email_reply_count"] > 0 {
		value := 0.0
		if lead.Hs_email_replied >= 5 {
			value = 1.0
		} else if lead.Hs_email_replied >= 2 {
			value = 0.7
		} else if lead.Hs_email_replied >= 1 {
			value = 0.5
		}
		add_factor("email_reply_count", weights["email_reply_count"], value)
	}

	// 8. Tasks Completed
	if weights["tasks_completed_count"] > 0 {
		value := math.Min(1.0, float64(lead.Num_completed_tasks)/5)
		add_factor("tasks_completed_count", weights["tasks_completed_count"], value)
	}

	// 9. Days Since Create
	if weights["days_since_create"] > 0 {
		value := 0.2
		if lead.Create_date != "" {
			if t, err := parse_date(lead.Create_date); err == nil {
				days := int(time.Since(t).Hours() / 24)
				switch {
				case days <= 7:
					value = 1.0
				case days <= 30:
					value = 0.7
				case days <= 90:
					value = 0.4
				}
			}
		}
		add_factor("days_since_create", weights["days_since_create"], value)
	}

	// 10. Job Title Seniority
	if weights["job_title_seniority"] > 0 && lead.Jobtitle != "" {
		title := strings.ToLower(lead.Jobtitle)
		value := 0.3
		switch {
		case contains_any(title, "ceo", "founder", "owner", "president", "chief"):
			value = 1.0
		case contains_any(title, "vp", "vice president", "director"):
			value = 0.8
		case contains_any(title, "head", "manager", "lead"):
			value = 0.6
		case contains_any(title, "senior", "sr"):
			value = 0.4
		}
		add_factor("job_title_seniority", weights["job_title_seniority"], value)
	}

	// 11. Has Email Valid
	if weights["has_email_valid"] > 0 {
		value := bool_to_float(is_valid_email(lead.Email))
		add_factor("has_email_valid", weights["has_email_valid"], value)
	}

	// 12. Profile Completeness
	if weights["profile_completeness"] > 0 {
		filled := 0
		for _, f := range []string{lead.Email, lead.Company, lead.Jobtitle, lead.Industry} {
			if f != "" {
				filled++
			}
		}
		if lead.Has_firstname {
			filled++
		}
		if lead.Has_lastname {
			filled++
		}
		if lead.Has_phone {
			filled++
		}
		value := float64(filled) / 7.0
		add_factor("profile_completeness", weights["profile_completeness"], value)
	}

	// 13. Has Phone
	if weights["has_phone"] > 0 {
		value := bool_to_float(lead.Has_phone)
		add_factor("has_phone", weights["has_phone"], value)
	}

	// Normalization (same as MCP)
	total_weights := 0.0
	for _, w := range weights {
		total_weights += w
	}
	normalized := 0.0
	if total_weight_used > 0 {
		normalized = total_score / total_weight_used
	}
	completeness := 0.0
	if total_weights > 0 {
		completeness = total_weight_used / total_weights
	}
	completeness_multiplier := 0.5 + 0.5*completeness

	score := int(math.Round(clamp(normalized*completeness_multiplier*100, 0, 100)))

	return Lead_score{
		Email:   lead.Email,
		Score:   score,
		Label:   get_score_label(score),
		Factors: factors,
	}
}

func get_score_label(score int) string {
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

// Helper functions

func bool_to_float(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func is_valid_email(email string) bool {
	return email != "" && strings.Contains(email, "@")
}

func contains_any(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}

func parse_date(date_str string) (time.Time, error) {
	if !strings.Contains(date_str, "-") {
		var timestamp int64
		if _, err := fmt.Sscanf(date_str, "%d", &timestamp); err == nil {
			return time.UnixMilli(timestamp), nil
		}
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02",
		"2006/01/02",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, date_str); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", date_str)
}

// API client

func fetch_config(api_key string) (*Config_response, error) {
	req, err := http.NewRequest(http.MethodGet, config_api_url+"/api/config", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-API-Key", api_key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		resp_body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("config API error %d: %s", resp.StatusCode, string(resp_body))
	}

	var config Config_response
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return &config, nil
}

// HTTP helpers

func write_json(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func write_error(w http.ResponseWriter, status int, message string) {
	write_json(w, status, Error_response{Error: message})
}


// Handlers

func health_handler(w http.ResponseWriter, r *http.Request) {
	write_json(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "scoring-service",
	})
}

func auth_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		write_error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	api_key := r.Header.Get("X-API-Key")
	if api_key == "" {
		write_error(w, http.StatusUnauthorized, "api_key required")
		return
	}

	config, err := fetch_config(api_key)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			write_error(w, http.StatusUnauthorized, "Invalid API key")
			return
		}
		write_error(w, http.StatusInternalServerError, "Failed to validate API key")
		return
	}

	write_json(w, http.StatusOK, map[string]string{
		"status":    "valid",
		"client_id": config.Client_id,
	})
}

func leads_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		write_error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req Leads_request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		write_error(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	api_key := r.Header.Get("X-API-Key")
	if api_key == "" || req.Email == "" {
		write_error(w, http.StatusUnauthorized, "api_key and email required")
		return
	}

	if len(req.Leads) == 0 {
		write_error(w, http.StatusBadRequest, "No leads provided")
		return
	}

	config, err := fetch_config(api_key)
	if err != nil {
		log.Printf("Failed to fetch config: %v", err)
		if strings.Contains(err.Error(), "unauthorized") {
			write_error(w, http.StatusUnauthorized, "Invalid API key")
			return
		}
		write_error(w, http.StatusInternalServerError, "Failed to fetch scoring config")
		return
	}

	var scores []Lead_score
	for _, lead := range req.Leads {
		scores = append(scores, calculate_score(lead, config.Weights))
	}

	write_json(w, http.StatusOK, Leads_response{
		Scores:    scores,
		Method:    config.Method,
		Client_id: config.Client_id,
	})
}

// Main

func main() {
	http.HandleFunc("/health", health_handler)
	http.HandleFunc("/auth", auth_handler)
	http.HandleFunc("/leads", leads_handler)

	addr := ":" + port
	log.Printf("Scoring service starting on %s", addr)
	log.Printf("Config API: %s", config_api_url)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}
