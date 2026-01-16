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
	"slices"
	"strings"
	"time"
)

// Request and response types

type Lead struct {
	Email             string  `json:"email"`
	Firstname         string  `json:"firstname,omitempty"`
	Lastname          string  `json:"lastname,omitempty"`
	Company           string  `json:"company,omitempty"`
	Jobtitle          string  `json:"jobtitle,omitempty"`
	Industry          string  `json:"industry,omitempty"`
	Phone             string  `json:"phone,omitempty"`
	City              string  `json:"city,omitempty"`
	Country           string  `json:"country,omitempty"`
	Lead_status       string  `json:"lead_status,omitempty"`
	Email_open_count  int     `json:"email_open_count,omitempty"`
	Email_click_count int     `json:"email_click_count,omitempty"`
	Num_deals         int     `json:"num_deals,omitempty"`
	Deal_amount       float64 `json:"deal_amount,omitempty"`
	Create_date       string  `json:"create_date,omitempty"`
	Notes_last_update string  `json:"notes_last_updated,omitempty"`
}

type Leads_request struct {
	Leads     []Lead `json:"leads"`
	Client_id string `json:"client_id,omitempty"`
	Api_key   string `json:"api_key"`
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

	add_factor := func(name string, weight, value float64) {
		if weight <= 0 || value <= 0 {
			return
		}
		contribution := value * weight
		total_score += contribution
		factors = append(factors, Score_factor{
			Name:         name,
			Weight:       weight,
			Value:        value,
			Contribution: contribution,
		})
	}

	add_factor("Lead Source", weights["lead_source"], bool_to_float(lead.Email != ""))
	add_factor("Valid Email", weights["has_valid_email"], bool_to_float(is_valid_email(lead.Email)))
	add_factor("Company Match", weights["has_company_match"], bool_to_float(lead.Company != ""))
	add_factor("Industry Match", weights["industry_match"], bool_to_float(lead.Industry != ""))

	if lead.Create_date != "" {
		if create_time, err := parse_date(lead.Create_date); err == nil {
			days := int(time.Since(create_time).Hours() / 24)
			value := math.Max(0, math.Min(1, 1-float64(days)/90))
			add_factor("Recency", weights["days_since_created"], value)
		}
	}

	if lead.Lead_status != "" {
		status_values := map[string]float64{
			"new":         0.3,
			"open":        0.5,
			"in_progress": 0.7,
			"qualified":   1.0,
			"unqualified": 0.1,
		}
		value := 0.5
		if v, ok := status_values[strings.ToLower(lead.Lead_status)]; ok {
			value = v
		}
		add_factor("Lead Status", weights["lead_status"], value)
	}

	opens := float64(lead.Email_open_count)
	clicks := float64(lead.Email_click_count)
	engagement_value := math.Min(1, (opens/10)*0.4+(clicks/3)*0.6)
	add_factor("Engagement", weights["engagement_score"], engagement_value)

	profile_fields := []string{
		lead.Email, lead.Firstname, lead.Lastname,
		lead.Company, lead.Jobtitle, lead.Phone,
		lead.City, lead.Country, lead.Industry,
	}
	filled_count := 0
	for _, field := range profile_fields {
		if field != "" {
			filled_count++
		}
	}
	profile_value := float64(filled_count) / float64(len(profile_fields))
	add_factor("Profile Complete", weights["profile_completeness"], profile_value)

	if lead.Jobtitle != "" {
		title := strings.ToLower(lead.Jobtitle)
		title_value := 0.3
		switch {
		case contains_any(title, "ceo", "founder", "owner"):
			title_value = 1.0
		case contains_any(title, "director", "vp", "chief"):
			title_value = 0.8
		case contains_any(title, "manager", "head"):
			title_value = 0.6
		}
		add_factor("Company Size", weights["company_size_bucket"], title_value)
	}

	recency_value := 0.2
	if lead.Num_deals > 0 {
		recency_value = 0.7
	} else if lead.Notes_last_update != "" {
		if notes_time, err := parse_date(lead.Notes_last_update); err == nil {
			if time.Since(notes_time).Hours() < 30*24 {
				recency_value = 0.5
			}
		}
	}
	add_factor("Activity Recency", weights["recency_score"], recency_value)

	score := int(math.Round(clamp(total_score*100, 0, 100)))

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

func fetch_config(email, api_key string) (*Config_response, error) {
	body, _ := json.Marshal(map[string]string{"email": email, "api_key": api_key})

	resp, err := http.Post(config_api_url+"/config", "application/json", bytes.NewBuffer(body))
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

var allowed_origins = []string{
	"https://conturs.com",
	"https://www.conturs.com",
	"https://app.conturs.com",
}

func cors_middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if slices.Contains(allowed_origins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// Handlers

func health_handler(w http.ResponseWriter, r *http.Request) {
	write_json(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "scoring-service",
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

	if req.Api_key == "" || req.Email == "" {
		write_error(w, http.StatusUnauthorized, "api_key and email required")
		return
	}

	if len(req.Leads) == 0 {
		write_error(w, http.StatusBadRequest, "No leads provided")
		return
	}

	config_email := req.Email
	if req.Client_id != "" {
		config_email = req.Client_id
	}

	config, err := fetch_config(config_email, req.Api_key)
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
	http.HandleFunc("/health", cors_middleware(health_handler))
	http.HandleFunc("/leads", cors_middleware(leads_handler))

	addr := ":" + port
	log.Printf("Scoring service starting on %s", addr)
	log.Printf("Config API: %s", config_api_url)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}
