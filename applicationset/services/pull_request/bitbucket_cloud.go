package pull_request

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	bitbucket "github.com/ktrysmt/go-bitbucket"
)

type BitbucketCloudService struct {
	client         	*bitbucket.Client
	owner     	string
	repositorySlug 	string
}

type BitbucketCloudPullRequest struct {
	ID	int				`json:"id"`
	Source 	BitbucketCloudPullRequestSource `json:"source"`
}

type BitbucketCloudPullRequestSource struct {
	Branch BitbucketCloudPullRequestSourceBranch `json:"branch"`
	Commit BitbucketCloudPullRequestSourceCommit `json:"commit"`
}

type BitbucketCloudPullRequestSourceBranch struct {
	Name string `json:"name"`
}

type BitbucketCloudPullRequestSourceCommit struct {
	Hash string `json:"hash"`
}

type PullRequestResponse struct {
	Page		int32		`json:"page"`
	Size 		int32 		`json:"size"`
	Pagelen 	int32 		`json:"pagelen"`
	Next 		string 		`json:"next"`
	Previous 	string 		`json:"previous"`
	Items 		[]PullRequest 	`json:"values"`
}

var _ PullRequestService = (*BitbucketCloudService)(nil)

func parseUrl(uri string) (*url.URL, error) {
	if uri == "" {
		uri = "https://api.bitbucket.org/2.0"
	}

	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	return url, nil
}

func NewBitbucketCloudServiceBasicAuth(username, password, baseUrl, owner, repositorySlug string) (PullRequestService, error) {
	url, err := parseUrl(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("error parsing base url of %s for %s/%s: %v", baseUrl, owner, repositorySlug, err)
	}

	bitbucketClient := bitbucket.NewBasicAuth(username, password)
	bitbucketClient.SetApiBaseURL(*url)

	return &BitbucketCloudService{
		client:		bitbucketClient,
		owner:		owner,
		repositorySlug:	repositorySlug,
	}, nil
}

func NewBitbucketCloudServiceNoAuth(baseUrl, owner, repositorySlug string) (PullRequestService, error) {
	url, err := parseUrl(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("error parsing base url of %s for %s/%s: %v", baseUrl, owner, repositorySlug, err)
	}

	// There is currently no method to explicitly not require auth
	bitbucketClient := bitbucket.NewOAuthbearerToken("")
	bitbucketClient.SetApiBaseURL(*url)

	return &BitbucketCloudService{
		client:         bitbucketClient,
		owner:     	owner,
		repositorySlug: repositorySlug,
	}, nil
}

func (b *BitbucketCloudService) List(_ context.Context) ([]*PullRequest, error) {
	opts := &bitbucket.PullRequestsOptions{
		Owner:		b.owner,
		RepoSlug: 	b.repositorySlug,
	}

	response, err := b.client.Repositories.PullRequests.Gets(opts)
	if err != nil {
		return nil, fmt.Errorf("error listing pull requests for %s/%s: %v", b.owner, b.repositorySlug, err)
	}

	resp, ok := response.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Not a valid format")
	}

	repoArray, ok := resp["values"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("Not a valid format")
	}

	jsonStr, err := json.Marshal(repoArray)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert response body to json")
	}

	var pulls []BitbucketCloudPullRequest
	if err := json.Unmarshal(jsonStr, &pulls); err != nil {
		return nil, fmt.Errorf("Failed to convert json to type '[]BitbucketCloudPullRequest'")
	}

	pullRequests := []*PullRequest{}
	for _, pull := range pulls {
		pullRequests = append(pullRequests, &PullRequest{
			Number:  pull.ID,
			Branch:  pull.Source.Branch.Name,
			HeadSHA: pull.Source.Commit.Hash,
		})
	}

	return pullRequests, nil
}