package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	patv1 "github.com/astro-web3/oauth2-token-exchange/pb/gen/go/pat/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	defaultServerAddr = "http://localhost:8123"
	patServiceBase    = "/pat.v1.PATService"
	createPATPath     = patServiceBase + "/CreatePAT"
	listPATsPath      = patServiceBase + "/ListPATs"
	deletePATPath     = patServiceBase + "/DeletePAT"
)

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

	if err := testListPATs(client, serverAddr, userID, email, preferredUsername); err != nil {
		log.Fatalf("❌ ListPATs test failed: %v", err)
	}
	fmt.Printf("✅ ListPATs test passed\n\n")

	if err := testAuthorizePAT(client, serverAddr, patToken); err != nil {
		log.Fatalf("❌ AuthorizePAT test failed: %v", err)
	}
	fmt.Printf("✅ AuthorizePAT test passed\n\n")

	if err := testDeletePAT(client, serverAddr, userID, email, preferredUsername, patID); err != nil {
		log.Fatalf("❌ DeletePAT test failed: %v", err)
	}
	fmt.Printf("✅ DeletePAT test passed\n\n")

	fmt.Println("🎉 All E2E tests passed!")
}

func testCreatePAT(client *http.Client, serverAddr, userID, email, preferredUsername string) (string, string, error) {
	fmt.Println("📝 Test: CreatePAT")

	expirationDate := time.Now().Add(24 * time.Hour).Unix()
	reqBody := &patv1.CreatePATRequest{
		ExpirationDate: expirationDate,
	}

	jsonBody, err := protojson.Marshal(reqBody)
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

	fmt.Printf("   CreatePAT response body: %s\n", string(body))

	var createResp patv1.CreatePATResponse
	if err := protojson.Unmarshal(body, &createResp); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	fmt.Printf("   CreatePAT response: Pat=%+v, Token=%s\n", createResp.Pat, createResp.GetToken())

	if createResp.Pat == nil {
		return "", "", fmt.Errorf("PAT is nil")
	}
	if createResp.Pat.GetId() == "" {
		return "", "", fmt.Errorf("PAT ID is empty")
	}
	if createResp.GetToken() == "" {
		return "", "", fmt.Errorf("PAT token is empty")
	}
	if createResp.Pat.GetHumanUserId() != userID {
		return "", "", fmt.Errorf("human user ID mismatch: expected %s, got %s", userID, createResp.Pat.GetHumanUserId())
	}

	fmt.Printf("   Created PAT ID: %s\n", createResp.Pat.GetId())
	fmt.Printf("   Machine User ID: %s\n", createResp.Pat.GetMachineUserId())
	fmt.Printf("   Human User ID: %s\n", createResp.Pat.GetHumanUserId())
	fmt.Printf("   Expiration Date: %s\n", time.Unix(createResp.Pat.GetExpirationDate(), 0).Format(time.RFC3339))

	return createResp.Pat.GetId(), createResp.GetToken(), nil
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

	var listResp patv1.ListPATsResponse
	if err := protojson.Unmarshal(body, &listResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	fmt.Printf("   Found %d PAT(s)\n", len(listResp.Pats))
	for i, pat := range listResp.Pats {
		fmt.Printf("   PAT %d: ID=%s, MachineUserID=%s, HumanUserID=%s, Expires=%s\n",
			i+1, pat.GetId(), pat.GetMachineUserId(), pat.GetHumanUserId(), time.Unix(pat.GetExpirationDate(), 0).Format(time.RFC3339))
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
	userPreferredUsername := resp.Header.Get("X-Auth-Request-Preferred-Username")
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
	if userPreferredUsername != "" {
		fmt.Printf("   Preferred Username: %s\n", userPreferredUsername)
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

	reqBody := &patv1.DeletePATRequest{
		PatId: patID,
	}

	jsonBody, err := protojson.Marshal(reqBody)
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

	var deleteResp patv1.DeletePATResponse
	if err := protojson.Unmarshal(body, &deleteResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !deleteResp.GetSuccess() {
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
