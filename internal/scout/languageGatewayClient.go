package scout

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/docker/docker-language-server/internal/cache"
	"github.com/docker/docker-language-server/internal/pkg/cli/metadata"
)

type LanguageGatewayClient interface {
	PostImage(ctx context.Context, jwt, image string) (ImageResponse, error)
	Fetch(key cache.Key) (ImageResponse, error)
}

type LanguageGatewayClientImpl struct {
	client http.Client
}

const languageGatewayImageUrl = "https://api.scout.docker.com/v1/language-gateway/image"

func NewLanguageGatewayClient() LanguageGatewayClient {
	return &LanguageGatewayClientImpl{
		client: http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c LanguageGatewayClientImpl) Fetch(key cache.Key) (ImageResponse, error) {
	scoutKey, ok := key.(*ScoutImageKey)
	if ok {
		return c.PostImage(context.Background(), "", scoutKey.Image)
	}
	return ImageResponse{}, errors.New("unrecognized key provided")
}

// PostImage sends an HTTP POST request to /v1/language-gateway/image to
// retrieve infromation from the Scout Language Gateway. This
// information can be used for providing diagnostics about the given
// image.
func (c LanguageGatewayClientImpl) PostImage(ctx context.Context, jwt, image string) (ImageResponse, error) {
	imageRequestBody := &ImageRequest{Image: image}
	b, err := json.Marshal(imageRequestBody)
	if err != nil {
		err := fmt.Errorf("failed to marshal image request: %w", err)
		return ImageResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, languageGatewayImageUrl, bytes.NewBuffer(b))
	if err != nil {
		err := fmt.Errorf("failed to create http request: %w", err)
		return ImageResponse{}, err
	}

	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("User-Agent", fmt.Sprintf("docker-language-server/v%v", metadata.Version))
	res, err := c.client.Do(req)
	if err != nil {
		err := fmt.Errorf("failed to send HTTP request: %w", err)
		return ImageResponse{}, err
	}

	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != 200 {
		err := fmt.Errorf("http request failed (%v status code)", res.StatusCode)
		return ImageResponse{}, err
	}

	var imageResponse ImageResponse
	_ = json.NewDecoder(res.Body).Decode(&imageResponse)
	return imageResponse, nil
}
