package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/astro-web3/oauth2-token-exchange/internal/config"
	authv3 "github.com/astro-web3/oauth2-token-exchange/pb/gen/go/envoy/service/auth/v3"
	authv3connect "github.com/astro-web3/oauth2-token-exchange/pb/gen/go/envoy/service/auth/v3/authv3connect"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <pat-token> [server-addr]", os.Args[0])
	}

	cfg := config.MustLoad()

	patToken := os.Args[1]
	serverAddr := ":8123"
	if len(os.Args) > 2 {
		serverAddr = os.Args[2]
	}

	client := authv3connect.NewAuthorizationClient(
		&http.Client{},
		"http://localhost"+serverAddr,
		connect.WithGRPC(),
	)

	req := &authv3.CheckRequest{
		Attributes: &authv3.CheckRequest_Attributes{
			Request: &authv3.CheckRequest_Attributes_Request{
				Http: &authv3.HttpRequest{
					Id:     "test-request-1",
					Method: "GET",
					Headers: map[string]string{
						"authorization": "Bearer " + patToken,
						"host":          "example.com",
						"path":          "/test",
					},
					Path:     "/test",
					Host:     "example.com",
					Scheme:   "https",
					Protocol: "HTTP/1.1",
				},
			},
		},
	}

	ctx := context.Background()
	resp, err := client.Check(ctx, connect.NewRequest(req))
	if err != nil {
		log.Fatalf("Check failed: %v", err)
	}

	httpResp := resp.Msg.GetHttpResponse()
	if httpResp == nil {
		log.Fatalf("No HTTP response in CheckResponse")
	}

	if okResp := httpResp.GetOkResponse(); okResp != nil {
		fmt.Println("✅ Authorization ALLOWED")
		fmt.Println("\nHeaders to be injected:")

		for _, hvo := range okResp.GetHeaders() {
			header := hvo.GetHeader()
			if header != nil {
				fmt.Printf("  %s: %s\n", header.GetKey(), header.GetValue())
			}
		}

		userJWT := ""
		userJWTHeaderKey := cfg.Auth.HeaderKeys.UserJWT
		for _, hvo := range okResp.GetHeaders() {
			header := hvo.GetHeader()
			if header != nil && header.GetKey() == userJWTHeaderKey {
				userJWT = header.GetValue()
				break
			}
		}

		if userJWT != "" {
			fmt.Printf("\n✅ JWT Token received successfully!\n")
			fmt.Printf("   Token length: %d characters\n", len(userJWT))
			fmt.Printf("   Token preview: %s...\n", userJWT[:min(80, len(userJWT))])

			userID := ""
			userEmail := ""
			userIDHeaderKey := cfg.Auth.HeaderKeys.UserID
			userEmailHeaderKey := cfg.Auth.HeaderKeys.UserEmail
			for _, hvo := range okResp.GetHeaders() {
				header := hvo.GetHeader()
				if header != nil {
					switch header.GetKey() {
					case userIDHeaderKey:
						userID = header.GetValue()
					case userEmailHeaderKey:
						userEmail = header.GetValue()
					}
				}
			}

			fmt.Printf("\n📋 User Information:\n")
			if userID != "" {
				fmt.Printf("   User ID: %s\n", userID)
			}
			if userEmail != "" {
				fmt.Printf("   Email: %s\n", userEmail)
			}
		}
	} else if deniedResp := httpResp.GetDeniedResponse(); deniedResp != nil {
		fmt.Printf("❌ Authorization DENIED\n")
		fmt.Printf("Status: %d\n", deniedResp.GetStatus())
		fmt.Printf("Body: %s\n", deniedResp.GetBody())
	} else {
		fmt.Println("⚠️  Unknown response type")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
