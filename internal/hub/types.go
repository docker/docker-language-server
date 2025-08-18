package hub

import "fmt"

type HubTagsKey struct {
	Repository string
	Image      string
}

func (k HubTagsKey) CacheKey() string {
	return fmt.Sprintf("%v-%v", k.Repository, k.Image)
}
