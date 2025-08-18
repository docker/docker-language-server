package hub

import (
	"github.com/docker/docker-language-server/internal/cache"
)

type Service interface {
	GetTags(repository, image string) ([]string, error)
}

type ServiceImpl struct {
	tagsManager cache.CacheManager[[]string]
}

func NewService() Service {
	client := NewHubClient()
	tf := NewHubTagsFetcher(client)
	return &ServiceImpl{
		tagsManager: cache.NewManager(tf),
	}
}

func (s *ServiceImpl) GetTags(repository, image string) ([]string, error) {
	return s.tagsManager.Get(HubTagsKey{Repository: repository, Image: image})
}
