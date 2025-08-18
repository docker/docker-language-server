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
	Name string `json:"name"`
}

type TagsResponse struct {
	Next    string      `json:"next"`
	Results []TagResult `json:"results"`
}

type HubClientImpl struct {
	client http.Client
}

type HubFetcherImpl struct {
	hubClient *HubClientImpl
}

const getTagsUrl = "https://hub.docker.com/v2/namespaces/%v/repositories/%v/tags?page_size=100"

func NewHubTagsFetcher(hubClient *HubClientImpl) cache.Fetcher[[]string] {
	return &HubFetcherImpl{hubClient: hubClient}
}

func (f *HubFetcherImpl) Fetch(key cache.Key) ([]string, error) {
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

func (c *HubClientImpl) GetTags(ctx context.Context, repository, image string) ([]string, error) {
	results, err := c.GetTagsFromURL(ctx, fmt.Sprintf(getTagsUrl, repository, image))
	if err != nil {
		return nil, err
	}

	tags := make([]string, len(results))
	for i := range results {
		tags[i] = results[i].Name
	}
	return tags, nil
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

	defer res.Body.Close()
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
