package main

import (
	"context"
	"encoding/base64"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	owner       = "pathan1435"
	repo        = "kaif"
	accessToken = "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"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	client := createGitHubClient()

	// Specify the file path in the repository
	filePath := "path/to/your/file.txt"

	// Specify the content of the file
	fileContent := "Hello, GitHub!"

	// Create or update the file
	err := createOrUpdateFile(client, filePath, fileContent)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Error creating or updating file"}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200, Body: "File created or updated successfully"}, nil
}

func createOrUpdateFile(client *github.Client, path, content string) error {
	fileContentBytes := []byte(content)

	// Encode the file content in base64
	contentBase64 := base64.StdEncoding.EncodeToString(fileContentBytes)

	// Get the current commit SHA
	ref, _, err := client.Git.GetRef(context.Background(), owner, repo, "heads/master")
	if err != nil {
		return err
	}
	latestCommitSHA := *ref.Object.SHA

	// Get the latest tree SHA
	latestTree, _, err := client.Git.GetTree(context.Background(), owner, repo, latestCommitSHA, true)
	if err != nil {
		return err
	}
	latestTreeSHA := *latestTree.SHA

	// Create a new tree with the new file
	newTreeEntry := github.TreeEntry{
		Path:    github.String(path),
		Mode:    github.String("100644"), // File mode
		Type:    github.String("blob"),   // Blob type
		Content: github.String(contentBase64),
	}
	newTree := []github.TreeEntry{newTreeEntry}

	// Create a new tree
	newTreeResult, _, err := client.Git.CreateTree(context.Background(), owner, repo, latestTreeSHA, newTree)
	if err != nil {
		return err
	}

	// Create a new commit
	newCommit, _, err := client.Git.CreateCommit(context.Background(), owner, repo, &github.Commit{
		Message: github.String("Create or update file"),
		Tree:    newTreeResult,
		Parents: []github.Commit{{SHA: &latestCommitSHA}},
	})
	if err != nil {
		return err
	}

	// Update the reference (branch) to point to the new commit
	_, _, err = client.Git.UpdateRef(context.Background(), owner, repo, &github.Reference{
		Ref: github.String("heads/master"),
		Object: &github.GitObject{
			SHA: newCommit.SHA,
		},
	}, false)
	if err != nil {
		return err
	}

	return nil
}

func createGitHubClient() *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return client
}

func main() {
	lambda.Start(handler)
}
