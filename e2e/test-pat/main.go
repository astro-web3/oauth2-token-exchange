package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	defaultServerAddr = "http://localhost:8123"
	patServiceBase    = "/pat.v1.PATService"
	createPATPath     = patServiceBase + "/CreatePAT"
	listPATsPath      = patServiceBase + "/ListPATs"
	deletePATPath     = patServiceBase + "/DeletePAT"
)

type CreatePATRequest struct {
	ExpirationDate int64 `json:"expiration_date"`
}

type CreatePATResponse struct {
	Pat   PAT    `json:"pat"`
	Token string `json:"token"`
}

type PAT struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id"`
	ExpirationDate int64  `json:"expiration_date"`
	CreatedAt      int64  `json:"created_at"`
}

type ListPATsResponse struct {
	Pats []PAT `json:"pats"`
}

type DeletePATRequest struct {
	PatID string `json:"pat_id"`
}

type DeletePATResponse struct {
	Success bool `json:"success"`
}

func main() {
	if len(os.Args) < 4 {
		log.Fatalf("Usage: %s <user-id> <email> <preferred-username> [server-addr]", os.Args[0])
	}

	userID := os.Args[1]
	email := os.Args[2]
	preferredUsername := os.Args[3]
	serverAddr := defaultServerAddr
	if len(os.Args) > 4 {
		serverAddr = os.Args[4]
	}

	fmt.Println("🧪 Starting PAT Management E2E Tests")
	fmt.Println("=====================================")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	patID, patToken, err := testCreatePAT(client, serverAddr, userID, email, preferredUsername)
	if err != nil {
		log.Fatalf("❌ CreatePAT test failed: %v", err)
	}
	fmt.Printf("✅ CreatePAT test passed (PAT ID: %s, PAT Token: %s)\n\n", patID, patToken)

	// if err := testListPATs(client, serverAddr, userID, email, preferredUsername); err != nil {
	// 	log.Fatalf("❌ ListPATs test failed: %v", err)
	// }
	// fmt.Printf("✅ ListPATs test passed\n\n")

	// if err := testAuthorizePAT(client, serverAddr, patToken); err != nil {
	// 	log.Fatalf("❌ AuthorizePAT test failed: %v", err)
	// }
	// fmt.Printf("✅ AuthorizePAT test passed\n\n")

	// if err := testDeletePAT(client, serverAddr, userID, email, preferredUsername, patID); err != nil {
	// 	log.Fatalf("❌ DeletePAT test failed: %v", err)
	// }
	// fmt.Printf("✅ DeletePAT test passed\n\n")

	// fmt.Println("🎉 All E2E tests passed!")
}

func testCreatePAT(client *http.Client, serverAddr, userID, email, preferredUsername string) (string, string, error) {
	fmt.Println("📝 Test: CreatePAT")

	expirationDate := time.Now().Add(24 * time.Hour).Unix()
	reqBody := CreatePATRequest{
		ExpirationDate: expirationDate,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := serverAddr + createPATPath
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Auth-Request-User", userID)
	req.Header.Set("X-Auth-Request-Email", email)
	req.Header.Set("X-Auth-Request-Preferred-Username", preferredUsername)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var createResp CreatePATResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if createResp.Pat.ID == "" {
		return "", "", fmt.Errorf("PAT ID is empty")
	}
	if createResp.Token == "" {
		return "", "", fmt.Errorf("PAT token is empty")
	}
	if createResp.Pat.UserID != userID {
		return "", "", fmt.Errorf("user ID mismatch: expected %s, got %s", userID, createResp.Pat.UserID)
	}

	fmt.Printf("   Created PAT ID: %s\n", createResp.Pat.ID)
	fmt.Printf("   User ID: %s\n", createResp.Pat.UserID)
	fmt.Printf("   Expiration Date: %s\n", time.Unix(createResp.Pat.ExpirationDate, 0).Format(time.RFC3339))

	return createResp.Pat.ID, createResp.Token, nil
}

func testListPATs(client *http.Client, serverAddr, userID, email, preferredUsername string) error {
	fmt.Println("📋 Test: ListPATs")

	url := serverAddr + listPATsPath
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Auth-Request-User", userID)
	req.Header.Set("X-Auth-Request-Email", email)
	req.Header.Set("X-Auth-Request-Preferred-Username", preferredUsername)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var listResp ListPATsResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	fmt.Printf("   Found %d PAT(s)\n", len(listResp.Pats))
	for i, pat := range listResp.Pats {
		fmt.Printf("   PAT %d: ID=%s, UserID=%s, Expires=%s\n",
			i+1, pat.ID, pat.UserID, time.Unix(pat.ExpirationDate, 0).Format(time.RFC3339))
	}

	return nil
}

func testAuthorizePAT(client *http.Client, serverAddr, patToken string) error {
	fmt.Println("🔐 Test: AuthorizePAT")

	url := serverAddr + "/oauth2/token-exchange/test"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+patToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authorization denied: status %d, body: %s", resp.StatusCode, string(body))
	}

	userID := resp.Header.Get("X-Auth-Request-User")
	userEmail := resp.Header.Get("X-Auth-Request-Email")
	userGroups := resp.Header.Get("X-Auth-Request-Groups")
	userJWT := resp.Header.Get("X-Auth-Request-Access-Token")

	fmt.Printf("   Authorization: ALLOWED\n")
	if userID != "" {
		fmt.Printf("   User ID: %s\n", userID)
	}
	if userEmail != "" {
		fmt.Printf("   Email: %s\n", userEmail)
	}
	if userGroups != "" {
		fmt.Printf("   Groups: %s\n", userGroups)
	}
	if userJWT != "" {
		fmt.Printf("   Access Token: %s...\n", userJWT[:min(50, len(userJWT))])
	}

	if userID == "" {
		return fmt.Errorf("missing X-Auth-Request-User header")
	}
	if userJWT == "" {
		return fmt.Errorf("missing X-Auth-Request-Access-Token header")
	}

	return nil
}

func testDeletePAT(client *http.Client, serverAddr, userID, email, preferredUsername, patID string) error {
	fmt.Println("🗑️  Test: DeletePAT")

	reqBody := DeletePATRequest{
		PatID: patID,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := serverAddr + deletePATPath
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Auth-Request-User", userID)
	req.Header.Set("X-Auth-Request-Email", email)
	req.Header.Set("X-Auth-Request-Preferred-Username", preferredUsername)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var deleteResp DeletePATResponse
	if err := json.Unmarshal(body, &deleteResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !deleteResp.Success {
		return fmt.Errorf("delete operation returned success=false")
	}

	fmt.Printf("   Deleted PAT ID: %s\n", patID)

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
