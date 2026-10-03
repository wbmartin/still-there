package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type APIClient struct {
	baseURL   string
	client    *http.Client
	jwtToken  string
	csrfToken string
	userName  string
	password  string
}

type AuthResponse struct {
	User struct {
		AppUserID       int    `json:"appUserId"`
		UserName        string `json:"userName"`
		FirstName       string `json:"firstName"`
		LastName        string `json:"lastName"`
		SecurityContext int    `json:"securityContext"`
		GrantList       []int  `json:"grantList"`
	} `json:"user"`
	CsrfToken string `json:"csrfToken"`
}

type ExampleResponse struct {
	Data struct {
		ExampleID       int     `json:"exampleId"`
		ExampleText     string  `json:"exampleText"`
		ExampleInt      int     `json:"exampleInt"`
		ExampleFloat    float64 `json:"exampleFloat"`
		ExampleIntArray []int   `json:"exampleIntArray"`
		UpdateTs        float64 `json:"updateTs"`
	} `json:"data"`
}

func NewAPIClient(baseURL, userName, password string, insecure bool) *APIClient {
	// -insecure skips certificate verification for a self-signed local cert
	// (bin/400-make-local-dev-certs); never against a real host.
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, //nolint:gosec // opt-in flag
	}

	return &APIClient{
		userName: userName,
		password: password,
		baseURL:  baseURL,
		client: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

func (c *APIClient) Login() error {
	// Authentication request
	authData := map[string]string{
		"userName": c.userName,
		"password": c.password,
	}

	jsonData, _ := json.Marshal(authData)
	req, err := http.NewRequest(
		"POST",
		c.baseURL+"/auth",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"auth failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	// Parse response
	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to parse auth response: %w", err)
	}

	// Extract JWT from cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "auth" {
			c.jwtToken = cookie.Value
			break
		}
	}

	c.csrfToken = authResp.CsrfToken

	if c.jwtToken == "" {
		return fmt.Errorf("no JWT token received")
	}

	return nil
}

func (c *APIClient) CreateExample() (int, float64, error) {
	exampleData := map[string]interface{}{
		"exampleText":       "Load Test Example",
		"exampleInt":        42,
		"exampleFloat":      3.14,
		"exampleIntArray":   []int{1, 2, 3},
		"exampleFloatArray": []float64{1.1, 2.2, 3.3},
	}

	// Make the request and capture response
	req, err := http.NewRequest("POST", c.baseURL+"/example", nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.jwtToken != "" {
		req.AddCookie(&http.Cookie{Name: "auth", Value: c.jwtToken})
	}
	if c.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", c.csrfToken)
	}

	// Add JSON body
	jsonData, err := json.Marshal(exampleData)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal request data: %w", err)
	}
	req.Body = io.NopCloser(bytes.NewBuffer(jsonData))

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("create request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, 0, fmt.Errorf(
			"create failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	// Parse response to get example ID and update timestamp
	var response struct {
		Data struct {
			ExampleID int     `json:"exampleId"`
			UpdateTS  float64 `json:"updateTs"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, 0, fmt.Errorf("failed to parse create response: %w", err)
	}

	return response.Data.ExampleID, response.Data.UpdateTS, nil
}

func (c *APIClient) UpdateExample(exampleID int, updateTs float64) error {
	// For the update, we just exercise the update operation
	// without changing values
	// This tests the update path and optimistic locking
	exampleData := map[string]interface{}{
		"exampleId":         exampleID,
		"exampleText":       "Load Test Example",
		"exampleInt":        42,
		"exampleFloat":      3.14,
		"exampleIntArray":   []int{1, 2, 3},
		"exampleFloatArray": []float64{1.1, 2.2, 3.3},
		"updateTs":          updateTs,
	}

	return c.makeRequest("PUT", "/example", exampleData, nil)
}

func (c *APIClient) GetExample(exampleID int) error {
	return c.makeRequest(
		"GET",
		"/example?exampleId="+fmt.Sprintf("%d", exampleID),
		nil,
		nil,
	)
}

func (c *APIClient) DeleteExample(exampleID int, updateTs float64) error {
	return c.makeRequest(
		"DELETE",
		"/example?exampleId="+fmt.Sprintf("%d", exampleID)+
			"&updateTs="+fmt.Sprintf("%.7f", updateTs),
		nil,
		nil,
	)
}

func (c *APIClient) Logout() error {
	return c.makeRequest("POST", "/logoff", nil, nil)
}

func (c *APIClient) makeRequest(
	method, endpoint string,
	data interface{},
	headers map[string]string,
) error {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Add JWT cookie if we have one
	if c.jwtToken != "" {
		req.AddCookie(&http.Cookie{
			Name:  "auth",
			Value: c.jwtToken,
		})
	}

	// Add CSRF token for state-changing operations
	if method != "GET" && c.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", c.csrfToken)
	}

	// Add any additional headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for success status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"request failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	return nil
}
