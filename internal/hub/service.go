package hub

import (
	"github.com/docker/docker-language-server/internal/cache"
)

type Service interface {
	GetTags(repository, image string) ([]TagResult, error)
}

type ServiceImpl struct {
	tagResultManager cache.CacheManager[[]TagResult]
}

func NewService() Service {
	client := NewHubClient()
	return &ServiceImpl{
		tagResultManager: cache.NewManager(NewHubTagResultsFetcher(client)),
	}
}

func (s *ServiceImpl) GetTags(repository, image string) ([]TagResult, error) {
	return s.tagResultManager.Get(HubTagsKey{Repository: repository, Image: image})
}
