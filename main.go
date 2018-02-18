package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/scottrigby/gcb-pr/git"
	"github.com/scottrigby/trigger-gcp-cloudbuild/cloudbuild"
	cb "google.golang.org/api/cloudbuild/v1"
	"gopkg.in/go-playground/webhooks.v3"
	"gopkg.in/go-playground/webhooks.v3/github"
)

const (
	path = "/webhooks"
	port = 3016
)

func main() {
	hook := github.New(&github.Config{Secret: os.Getenv("GITHUB_WEBHOOK_SECRET")})
	hook.RegisterEvents(HandlePullRequest, github.PullRequestEvent)

	err := webhooks.Run(hook, ":"+strconv.Itoa(port), path)
	if err != nil {
		fmt.Println(err)
	}
}

// HandlePullRequest handles GitHub pull_request events:
func HandlePullRequest(payload interface{}, header webhooks.Header) {
	pl := payload.(github.PullRequestPayload)
	path := "clonedir"

	err := os.RemoveAll(path)
	if err != nil {
		fmt.Println(err)
	}

	url, err := cloneURL(pl)
	if err != nil {
		fmt.Println(err)
	}
	ref := pl.PullRequest.Head.Ref
	gitResp, err := git.ShallowClone(path, url, ref)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(gitResp)

	file := "clonedir/cloudbuild.yaml"
	// We mirror the name and value of GCR's built-in supported variables:
	// - ${_PROJECT_ID}: Same as built-in $PROJECT_ID: [GCP project ID].
	// - ${_REPO_NAME}: Same as built-in $RELEASE_NAME: [GIT_PROJECT_ACCOUNT]-[GIT_PROJECT_NAME].
	// - ${_COMMIT_SHA}: Same as built-in $COMMIT_SHA.
	projectID := os.Getenv("GCP_PROJECT_ID")
	repoName := pl.Repository.Owner.Login + "-" + pl.Repository.Name
	commitSHA := pl.PullRequest.Head.Sha
	// And include the new one:
	// - ${_PR_NUMBER}: The webhook-triggering PR number.
	prNumber := strconv.Itoa(int(pl.Number))
	err2 := triggerCloudBuild(file, projectID, repoName, commitSHA, prNumber)
	if err2 != nil {
		fmt.Println(err2)
	}
}

func cloneURL(pl github.PullRequestPayload) (string, error) {
	if pl.PullRequest.Head.Repo.Private == true {
		if value, ok := os.LookupEnv("GITHUB_ACCESS_TOKEN"); ok {
			fullName := pl.PullRequest.Head.Repo.FullName
			url := "https://x-access-token:" + value + "@github.com/" + fullName
			return url, nil
		}
		return "", errors.New("env: GITHUB_ACCESS_TOKEN is required for private repos")
	}
	return pl.PullRequest.Head.Repo.CloneURL, nil
}

func triggerCloudBuild(file string, projectID string, repoName string, commitSHA string, prNumber string) error {
	y, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		return err
	}
	substitutions := map[string]string{
		"_PROJECT_ID": projectID,
		"_REPO_NAME":  repoName,
		"_COMMIT_SHA": commitSHA,
		"_PR_NUMBER":  prNumber}
	build, err := getBuild(j, substitutions, commitSHA, repoName)
	if err != nil {
		return err
	}
	operation, err := cloudbuild.TriggerCloudBuild(projectID, build)
	if err != nil {
		return err
	}
	fmt.Printf("cloudbuild operation: %s\n", operation)
	return nil
}

func getBuild(j []byte, substitutions map[string]string, commitSHA string, repoName string) (*cb.Build, error) {
	build := &cb.Build{
		Substitutions: substitutions,
		Source: &cb.Source{
			RepoSource: &cb.RepoSource{
				CommitSha: commitSHA,
				RepoName:  repoName}}}
	err := json.Unmarshal(j, build)
	if err != nil {
		return nil, err
	}
	return build, nil
}
