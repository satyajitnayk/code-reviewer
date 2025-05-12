package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/go-github/v55/github"
	"github.com/joho/godotenv"
)

var (
	messageForNewPRs = "Thanks for opening a new PR! Please follow our contributing guidelines to make your PR easier to review."
	webhookSecret    []byte
	appID            int64
	privateKey       []byte
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Parse APP_ID
	appIDParsed, err := strconv.ParseInt(os.Getenv("APP_ID"), 10, 64)
	if err != nil {
		log.Fatalf("Invalid APP_ID: %v", err)
	}
	appID = appIDParsed

	// Read private key
	privateKeyPEM, err := os.ReadFile(os.Getenv("PRIVATE_KEY_PATH"))
	if err != nil {
		log.Fatalf("Failed to read private key: %v", err)
	}

	privateKey = privateKeyPEM

	webhookSecret = []byte(os.Getenv("WEBHOOK_SECRET"))

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/api/webhook", webhookHandler)

	// Start server
	fmt.Println("Server is listening at http://localhost:3000/api/webhook")
	log.Fatal(http.ListenAndServe(":3000", r))
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, webhookSecret)
	if err != nil {
		http.Error(w, "Invalid payload signature", http.StatusUnauthorized)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		http.Error(w, "Could not parse webhook", http.StatusBadRequest)
		return
	}

	switch e := event.(type) {
	case *github.PullRequestEvent:
		if *e.Action == "opened" {
			go handlePullRequestOpened(e)
		}
	default:
		fmt.Fprintf(w, "Event type %T not handled\n", e)
	}

	w.WriteHeader(http.StatusOK)
}

func handlePullRequestOpened(event *github.PullRequestEvent) {
	log.Printf("Received a pull request event for #%d\n", event.GetNumber())

	// Create transport with GitHub App authentication
	tr, err := ghinstallation.New(http.DefaultTransport, appID, event.GetInstallation().GetID(), privateKey)
	if err != nil {
		log.Printf("Error creating GitHub App transport: %v\n", err)
		return
	}

	client := github.NewClient(&http.Client{Transport: tr})

	// Create comment
	_, _, err = client.Issues.CreateComment(
		context.Background(),
		event.GetRepo().GetOwner().GetLogin(),
		event.GetRepo().GetName(),
		event.GetNumber(),
		&github.IssueComment{
			Body: &messageForNewPRs,
		},
	)
	if err != nil {
		log.Printf("Error creating comment: %v\n", err)
	}
}
