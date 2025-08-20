package dockerfile

import (
	"fmt"
	"strings"

	"github.com/docker/docker-language-server/internal/hub"
	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/dromara/carbon/v2"
)

func InlayHint(hubService hub.Service, doc document.DockerfileDocument, rng protocol.Range) ([]protocol.InlayHint, error) {
	content := doc.Input()
	lines := strings.Split(string(content), "\n")
	nodes := doc.Nodes()
	hints := []protocol.InlayHint{}
	for _, node := range nodes {
		line := protocol.UInteger(node.StartLine) - 1
		if rng.Start.Line <= line && line <= rng.End.Line {
			if strings.EqualFold(node.Value, "FROM") && node.Next != nil {
				repository, image, tag := types.HubRepositoryImage(node.Next.Value)
				if repository != "" && image != "" && tag != "" {
					tags, err := hubService.GetTags(repository, image)
					if err == nil {
						for _, t := range tags {
							if t.Name == tag {
								if t.TagLastPushed != "" {
									c := carbon.Parse(t.TagLastPushed, carbon.Local)
									if c != nil && c.IsValid() {
										goTime := c.StdTime()
										localFormat := goTime.Format("2006-01-02 15:04:05 MST")
										hints = append(hints, protocol.InlayHint{
											Label:       fmt.Sprintf("(last pushed %v)", c.DiffForHumans()),
											PaddingLeft: types.CreateBoolPointer(true),
											Position:    protocol.Position{Line: line, Character: protocol.UInteger(len(lines[node.StartLine-1]))},
											Tooltip:     types.CreateAnyPointer(localFormat),
										})
									}
								}
								break
							}
						}
					}
				}
			}
		}
	}
	return hints, nil
}
