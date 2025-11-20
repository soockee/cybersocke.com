package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	firebase "firebase.google.com/go/v4"
	"github.com/soockee/cybersocke.com/cmd/set_firebase_claims/claimconfig"
	"github.com/soockee/cybersocke.com/impersonation"
	"google.golang.org/api/option"
)

// set_firebase_claims assigns custom roles/claims to a Firebase user.
// Usage:
//
//	go run ./cmd/set_firebase_claims --uid <firebase-uid> --roles writer,admin --writer --admin
//
// Flags:
//
//	--uid     (required) Firebase User UID
//	--roles   comma separated roles; stored in claims["roles"]. If exactly one role provided also sets claims["role"].
//	--writer  convenience boolean claim (claims["writer"] = true)
//	--admin   convenience boolean claim (claims["admin"] = true)
//
// Environment (required):
//
//	FIREBASE_IMPERSONATE_SERVICE_ACCOUNT  service account email to impersonate (caller must have roles/iam.serviceAccountTokenCreator)
//	GCP_PROJECT_ID                        Firebase/GCP project ID
//
// Notes:
//   - The utility uses ADC and IAM impersonation exclusively (no JSON key usage).
//   - After updating claims the user must refresh their ID token (re-login or force getIdToken(true)).
func main() {
	uid := flag.String("uid", "", "Firebase user UID")
	rolesStr := flag.String("roles", "", "Comma separated roles (e.g. writer,admin)")
	writerFlag := flag.Bool("writer", false, "Set writer boolean claim")
	adminFlag := flag.Bool("admin", false, "Set admin boolean claim")
	flag.Parse()

	if *uid == "" {
		log.Fatal("--uid is required")
	}

	cfg, err := claimconfig.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx := context.Background()

	// Impersonation required path only
	impTS, err := impersonation.ServiceAccountTokenSource(ctx, cfg.FirebaseImpersonateServiceAccount, []string{
		"https://www.googleapis.com/auth/identitytoolkit",
	})
	if err != nil {
		log.Fatalf("impersonation token source: %v", err)
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.GCPProjectID}, option.WithTokenSource(impTS))
	if err != nil {
		log.Fatalf("init firebase app (impersonated): %v", err)
	}
	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("init auth client: %v", err)
	}

	claims := map[string]any{}
	if *rolesStr != "" {
		parts := strings.Split(*rolesStr, ",")
		var roles []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				roles = append(roles, p)
			}
		}
		if len(roles) > 0 {
			claims["roles"] = roles
			// convenience: set single role if exactly one
			if len(roles) == 1 {
				claims["role"] = roles[0]
			}
		}
	}
	if *writerFlag {
		claims["writer"] = true
	}
	if *adminFlag {
		claims["admin"] = true
	}

	if len(claims) == 0 {
		log.Fatal("no claims specified; use --roles or --writer/admin flags")
	}

	if err := authClient.SetCustomUserClaims(ctx, *uid, claims); err != nil {
		log.Fatalf("set custom claims: %v", err)
	}

	fmt.Printf("Successfully updated claims for UID %s: %+v\n", *uid, claims)
	fmt.Println("User must refresh ID token (re-login or force getIdToken(true)) to see changes.")
}
