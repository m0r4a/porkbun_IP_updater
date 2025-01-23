package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type PorkbunResponse struct {
	Status  string   `json:"status"`
	Records []Record `json:"records"`
}

type Record struct {
	Content string `json:"content"`
}

type PorkbunConfig struct {
	APIURL     string
	APIKey     string
	SecretKey  string
	RecordID   string
	Domain     string
	RecordName string
	RecordType string
}

type TwilioConfig struct {
	AccountSID string
	AuthToken  string
	FromPhone  string
	ToPhone    string
}

type APIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	// Configuring logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config := PorkbunConfig{
		APIURL:     "https://api.porkbun.com/api/json/v3/dns/edit/",
		APIKey:     os.Getenv("PORKBUN_API_KEY"),
		SecretKey:  os.Getenv("PORKBUN_SECRET_KEY"),
		RecordID:   os.Getenv("PORKBUN_RECORD_ID"),
		Domain:     os.Getenv("PORKBUN_DOMAIN"),
		RecordName: os.Getenv("PORKBUN_SUBDOMAIN"),
		RecordType: "A",
	}

	// Validate the configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("error in the configuration: %v", err)
	}

	if err := updateDNSIfNeeded(config); err != nil {
		log.Fatalf("error updating the DNS: %v", err)
	}
}

func updateDNSIfNeeded(config PorkbunConfig) error {
	currentDNSIP, err := getCurrentDNSIP(config)
	if err != nil {
		return fmt.Errorf("error getting current IP of the DNS: %w", err)
	}

	publicIP, err := getPublicIP()
	if err != nil {
		return fmt.Errorf("error getting the public IP: %w", err)
	}

	if currentDNSIP == publicIP {
		return nil
	}

	if err := updateDNSRecord(config, publicIP); err != nil {
		return fmt.Errorf("error updating DNS register: %w", err)
	}

	if err := SendSMS("Your IP has changed"); err != nil {
		log.Printf("error sending the SMS: %v", err)
	}

	return nil
}

func validateConfig(config PorkbunConfig) error {
	if config.APIKey == "" || config.SecretKey == "" || config.RecordID == "" {
		return fmt.Errorf("required API keys missing")
	}
	return nil
}

func getPublicIP() (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://api.ipify.org?format=text")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(ip)), nil
}

func getCurrentDNSIP(config PorkbunConfig) (string, error) {
	config.APIURL = "https://api.porkbun.com/api/json/v3/dns/retrieve/"
	requestBody := map[string]string{
		"secretapikey": config.SecretKey,
		"apikey":       config.APIKey,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error creating the JSON: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var fullAPIURL string = config.APIURL + config.Domain + "/" + config.RecordID
	req, err := http.NewRequest("POST", fullAPIURL, bytes.NewBuffer(jsonBody))

	if err != nil {
		return "", fmt.Errorf("error creating the request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error doing the request: %w", err)
	}
	defer response.Body.Close()

	var porkbunResp PorkbunResponse
	if err := json.NewDecoder(response.Body).Decode(&porkbunResp); err != nil {
		return "", fmt.Errorf("error decoding the answer: %w", err)
	}

	if len(porkbunResp.Records) == 0 {
		return "", fmt.Errorf("DNS registers not found")
	}

	currentIP := porkbunResp.Records[0].Content
	return currentIP, nil
}

func updateDNSRecord(config PorkbunConfig, newIP string) error {
	requestBody := map[string]string{
		"secretapikey": config.SecretKey,
		"apikey":       config.APIKey,
		"name":         config.RecordName,
		"type":         config.RecordType,
		"content":      newIP,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var fullAPIURL string = config.APIURL + config.Domain + "/" + config.RecordID
	req, err := http.NewRequest("POST", fullAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return fmt.Errorf("error decoding the answer: %w", err)
	}

	if apiResponse.Status != "SUCCESS" {
		return fmt.Errorf("API error: %s", apiResponse.Message)
	}

	return nil
}

func SendSMS(message string) error {

	config := TwilioConfig{
		AccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		AuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
		FromPhone:  os.Getenv("TWILIO_FROM_PHONE"),
		ToPhone:    os.Getenv("TWILIO_TO_PHONE"),
	}

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", url.PathEscape(config.AccountSID))

	data := url.Values{}
	data.Set("To", config.ToPhone)
	data.Set("From", config.FromPhone)
	data.Set("Body", message)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("error creating the request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(config.AccountSID, config.AuthToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending the SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error of TWILIO's API: status code %d", resp.StatusCode)
	}

	return nil
}
