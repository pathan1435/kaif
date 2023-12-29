package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/go-github/github"
	_ "github.com/lib/pq"
	"golang.org/x/oauth2"
)

const (
	host     = "localhost"
	port     = 5433
	user     = "postgres"
	password = "1434"
	dbname   = "cms"
)

func getDB() (*sql.DB, error) {
	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func getGitHubLicense(ctx context.Context, owner, repo, accessToken string) (string, error) {
	client := github.NewClient(nil)

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/license", owner, repo), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(ctx, req, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API request failed with status code: %d", resp.StatusCode)
	}

	// Read the response body into a buffer
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var licenseResponse struct {
		SPDXID string `json:"spdx_id"`
	}

	// Unmarshal from the buffer
	err = json.Unmarshal(body, &licenseResponse)
	if err != nil {
		return "", err
	}

	return licenseResponse.SPDXID, nil
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	accessToken := "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"
	owner := "pathan1435"
	repo := "file"

	// Set up a GitHub client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Content for the new file
	content := []byte("This is the content of the new file.")

	// Create a new file in the GitHub repository
	filePath := "20decfile3.text"
	createFileOptions := &github.RepositoryContentFileOptions{
		Message:   github.String("Create new file"),
		Content:   content,
		Committer: &github.CommitAuthor{Name: github.String("kaif"), Email: github.String("pathankaif51@gmail.com")},
	}

	// CreateFile returns a repository content response
	createdFile, _, err := client.Repositories.CreateFile(ctx, owner, repo, filePath, createFileOptions)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: fmt.Sprintf("Error creating file: %s", err)}, nil
	}

	// Retrieve metadata about the created file
	fileDetails, _, _, err := client.Repositories.GetContents(ctx, owner, repo, filePath, &github.RepositoryContentGetOptions{})
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: fmt.Sprintf("Error getting file details: %s", err)}, nil
	}

	// Prepare a response payload with file metadata
	response := map[string]interface{}{
		"file_name":    *fileDetails.Name,
		"file_path":    *fileDetails.Path,
		"file_content": content,
		"download_url": *createdFile.Content.DownloadURL,
		// Include other file metadata fields as needed
	}

	// Convert the response to JSON
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: fmt.Sprintf("Error encoding JSON: %s", err)}, nil
	}

	// Get a PostgreSQL connection
	db, err := getDB()
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: fmt.Sprintf("Error connecting to the database: %s", err)}, nil
	}
	defer db.Close()

	// Insert data into PostgreSQL
	_, err = db.Exec(`
		INSERT INTO cms (details)
		VALUES ($1::jsonb)`,
		jsonResponse,
	)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: fmt.Sprintf("Error inserting data into PostgreSQL: %s", err)}, nil
	}

	// Set the Content-Type header to application/json
	headers := map[string]string{"Content-Type": "application/json"}

	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: string(jsonResponse), Headers: headers}, nil
}

func main() {
	lambda.Start(handleRequest)
}
