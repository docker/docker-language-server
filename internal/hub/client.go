package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/docker/docker-language-server/internal/cache"
	"github.com/docker/docker-language-server/internal/pkg/cli/metadata"
)

type TagResult struct {
	Name          string `json:"name"`
	TagLastPushed string `json:"tag_last_pushed"`
}

type TagsResponse struct {
	Next    string      `json:"next"`
	Results []TagResult `json:"results"`
}

type HubClientImpl struct {
	client http.Client
}

type HubTagResultsFetcherImpl struct {
	hubClient *HubClientImpl
}

const getTagsUrl = "https://hub.docker.com/v2/namespaces/%v/repositories/%v/tags?page_size=100"

func NewHubTagResultsFetcher(hubClient *HubClientImpl) cache.Fetcher[[]TagResult] {
	return &HubTagResultsFetcherImpl{hubClient: hubClient}
}

func (f *HubTagResultsFetcherImpl) Fetch(key cache.Key) ([]TagResult, error) {
	if k, ok := key.(HubTagsKey); ok {
		return f.hubClient.GetTags(context.Background(), k.Repository, k.Image)
	}
	return nil, nil
}

func NewHubClient() *HubClientImpl {
	return &HubClientImpl{
		client: http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *HubClientImpl) GetTags(ctx context.Context, repository, image string) ([]TagResult, error) {
	return c.GetTagsFromURL(ctx, fmt.Sprintf(getTagsUrl, repository, image))
}

func (c *HubClientImpl) GetTagsFromURL(ctx context.Context, url string) ([]TagResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		err := fmt.Errorf("failed to create http request: %w", err)
		return nil, err
	}

	req.Header.Set("User-Agent", fmt.Sprintf("docker-language-server/v%v", metadata.Version))
	res, err := c.client.Do(req)
	if err != nil {
		err := fmt.Errorf("failed to send HTTP request: %w", err)
		return nil, err
	}

	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != 200 {
		err := fmt.Errorf("http request failed (%v status code)", res.StatusCode)
		return nil, err
	}

	var tagsResponse TagsResponse
	_ = json.NewDecoder(res.Body).Decode(&tagsResponse)
	if tagsResponse.Next != "" {
		tags, err := c.GetTagsFromURL(ctx, tagsResponse.Next)
		if err == nil {
			tagsResponse.Results = append(tagsResponse.Results, tags...)
		}
	}
	return tagsResponse.Results, nil
}
