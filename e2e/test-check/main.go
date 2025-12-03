package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <pat-token> [server-addr]", os.Args[0])
	}

	patToken := os.Args[1]
	serverAddr := "http://localhost:8123"
	if len(os.Args) > 2 {
		serverAddr = "http://localhost" + os.Args[2]
	}

	req, err := http.NewRequest("GET", serverAddr+"/oauth2/token-exchange/test", nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+patToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Authorization ALLOWED")
		fmt.Println("\nHeaders received:")

		for k, v := range resp.Header {
			if strings.HasPrefix(strings.ToLower(k), "x-") {
				fmt.Printf("  %s: %s\n", k, strings.Join(v, ", "))
			}
		}

		userJWT := resp.Header.Get("x-user-jwt")
		if userJWT != "" {
			fmt.Printf("\n✅ JWT Token received successfully!\n")
			fmt.Printf("   Token length: %d characters\n", len(userJWT))
			previewLen := 80
			if len(userJWT) < previewLen {
				previewLen = len(userJWT)
			}
			fmt.Printf("   Token preview: %s...\n", userJWT[:previewLen])

			userID := resp.Header.Get("x-user-id")
			userEmail := resp.Header.Get("x-user-email")

			fmt.Printf("\n📋 User Information:\n")
			if userID != "" {
				fmt.Printf("   User ID: %s\n", userID)
			}
			if userEmail != "" {
				fmt.Printf("   Email: %s\n", userEmail)
			}
		}
	} else {
		fmt.Printf("❌ Authorization DENIED\n")
		fmt.Printf("Status: %d\n", resp.StatusCode)
		fmt.Printf("Body: %s\n", string(body))
	}
}
