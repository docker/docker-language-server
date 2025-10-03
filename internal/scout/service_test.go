package scout

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/docker/docker-language-server/internal/configuration"
	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/uri"
)

func mapScoutConfig(config configuration.Scout) map[string]bool {
	m := make(map[string]bool, 4)
	if config.CriticalHighVulnerabilities {
		m["critical_high_vulnerabilities"] = true
	}
	if config.NotPinnedDigest {
		m["not_pinned_digest"] = true
	}
	if config.RecommendedTag {
		m["recommended_tag"] = true
	}
	if config.Vulnerabilites {
		m["vulnerabilities"] = true
	}
	return m
}

func TestCalculateDiagnostics(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		diagnostics []protocol.Diagnostic
	}{
		{
			name:    "outdated alpine:3.16.1",
			content: "FROM alpine:3.16.1",
			diagnostics: []protocol.Diagnostic{
				{
					Message: "The image contains 1 critical and 3 high vulnerabilities",
					Code:    &protocol.IntegerOrString{Value: "critical_high_vulnerabilities"},
					Source:  types.CreateStringPointer("scout-testing-source"),
					CodeDescription: &protocol.CodeDescription{
						HRef: "https://hub.docker.com/layers/library/alpine/3.16.1/images/sha256-9b2a28eb47540823042a2ba401386845089bb7b62a9637d55816132c4c3c36eb",
					},
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 18},
					},
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityWarning),
				},
			},
		},
		{
			name:    "outdated alpine:3.16.1 with --platform flag",
			content: "FROM --platform=$BUILDPLATFORM alpine:3.16.1",
			diagnostics: []protocol.Diagnostic{
				{
					Message: "The image contains 1 critical and 3 high vulnerabilities",
					Code:    &protocol.IntegerOrString{Value: "critical_high_vulnerabilities"},
					Source:  types.CreateStringPointer("scout-testing-source"),
					CodeDescription: &protocol.CodeDescription{
						HRef: "https://hub.docker.com/layers/library/alpine/3.16.1/images/sha256-9b2a28eb47540823042a2ba401386845089bb7b62a9637d55816132c4c3c36eb",
					},
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 44},
					},
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityWarning),
				},
			},
		},
	}

	c := NewService()
	for _, tc := range testCases {
		uri := uri.URI("uri:///Dockerfile")
		doc := document.NewDocument(document.NewDocumentManager(), uri, protocol.DockerfileLanguage, 1, []byte(tc.content))
		testConfigs := []struct {
			description   string
			shouldScan    bool
			setupFunction func()
		}{
			{
				description: "removed",
				shouldScan:  true,
				setupFunction: func() {
					configuration.Remove(protocol.DocumentUri(uri))
				},
			},
			{
				description: "scan=true",
				shouldScan:  true,
				setupFunction: func() {
					configuration.Store(
						protocol.DocumentUri(uri),
						configuration.Configuration{
							Experimental: configuration.Experimental{
								VulnerabilityScanning: true,
								Scout:                 configuration.Get("/tmp/non-existent-file-to-get-default-config").Experimental.Scout,
							},
						},
					)
				},
			},
			{
				description: "scan=false",
				shouldScan:  false,
				setupFunction: func() {
					configuration.Store(
						protocol.DocumentUri(uri),
						configuration.Configuration{Experimental: configuration.Experimental{VulnerabilityScanning: false}},
					)
				},
			},
		}

		for _, testConfig := range testConfigs {
			t.Run(fmt.Sprintf("%v (%v)", tc.name, testConfig.description), func(t *testing.T) {
				defer configuration.Remove(protocol.DocumentUri(uri))
				testConfig.setupFunction()
				diagnostics := c.CollectDiagnostics("scout-testing-source", "", doc, tc.content)
				if os.Getenv("DOCKER_NETWORK_NONE") == "true" || !testConfig.shouldScan {
					require.Len(t, diagnostics, 0)
					return
				}

				for _, diagnostic := range diagnostics {
					found := false
					for _, expectedDiagnostic := range tc.diagnostics {
						if diagnostic.Message == expectedDiagnostic.Message {
							require.Equal(t, expectedDiagnostic, diagnostic)
							found = true
							break
						}
					}

					if !found {
						t.Errorf("expected diagnostic could not be found: %v", diagnostic.Message)
					}
				}
				require.Equal(t, len(tc.diagnostics), len(diagnostics))
			})
		}
	}
}

func TestCalculateDiagnostics_IgnoresSpecifics(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		codes   []string
	}{
		{
			name:    "alpine:3.16.1",
			content: "FROM alpine:3.16.1",
			codes: []string{
				"critical_high_vulnerabilities",
			},
		},
		{
			name:    "ubuntu:24.04",
			content: "FROM ubuntu:24.04",
			codes: []string{
				"vulnerabilities",
			},
		},
	}

	c := NewService()
	for _, tc := range testCases {
		uri := uri.URI("uri:///Dockerfile")
		doc := document.NewDocument(document.NewDocumentManager(), uri, protocol.DockerfileLanguage, 1, []byte(tc.content))
		testConfigs := []struct {
			description        string
			shouldScan         bool
			scoutConfiguration configuration.Scout
		}{
			{
				description: "CriticalHighVulnerabilities=false",
				scoutConfiguration: configuration.Scout{
					CriticalHighVulnerabilities: false,
					NotPinnedDigest:             true,
					RecommendedTag:              true,
					Vulnerabilites:              true,
				},
			},
			{
				description: "NotPinnedDigest=false",
				scoutConfiguration: configuration.Scout{
					CriticalHighVulnerabilities: true,
					NotPinnedDigest:             true,
					RecommendedTag:              false,
					Vulnerabilites:              true,
				},
			},
			{
				description: "RecommendedTag=false",
				scoutConfiguration: configuration.Scout{
					CriticalHighVulnerabilities: true,
					NotPinnedDigest:             true,
					RecommendedTag:              false,
					Vulnerabilites:              true,
				},
			},
			{
				description: "Vulnerabilites=false",
				scoutConfiguration: configuration.Scout{
					CriticalHighVulnerabilities: true,
					NotPinnedDigest:             true,
					RecommendedTag:              true,
					Vulnerabilites:              false,
				},
			},
		}

		for _, testConfig := range testConfigs {
			t.Run(fmt.Sprintf("%v (%v)", tc.name, testConfig.description), func(t *testing.T) {
				defer configuration.Remove(protocol.DocumentUri(uri))
				configuration.Store(
					protocol.DocumentUri(uri),
					configuration.Configuration{Experimental: configuration.Experimental{
						VulnerabilityScanning: true,
						Scout:                 configuration.Get("/tmp/non-existent-file-to-get-default-config").Experimental.Scout,
					}},
				)

				diagnostics := c.CollectDiagnostics("scout-testing-source", "", doc, tc.content)
				for _, code := range tc.codes {
					found := false
					for _, diagnostic := range diagnostics {
						if diagnostic.Code.Value.(string) == code {
							found = true
							break
						}
					}
					if !found {
						require.Fail(t, "Diagnostic not found", "Expected diagnostic code: %v", code)
					}
				}

				configuration.Store(
					protocol.DocumentUri(uri),
					configuration.Configuration{Experimental: configuration.Experimental{
						VulnerabilityScanning: true,
						Scout:                 testConfig.scoutConfiguration,
					}},
				)

				m := mapScoutConfig(testConfig.scoutConfiguration)
				diagnostics = c.CollectDiagnostics("scout-testing-source", "", doc, tc.content)
				for _, diagnostic := range diagnostics {
					code := diagnostic.Code.Value.(string)
					if !m[code] {
						require.Fail(t, "Diagnostic should have been filtered out")
					}
				}
			})
		}
	}
}

func TestGetHovers(t *testing.T) {
	testCases := []struct {
		image  string
		result *protocol.Hover
	}{
		{
			image: "alpine:3.16.1",
			result: &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: "Current image vulnerabilities:   1C   3H   9M   0L ",
				},
			},
		},
	}

	u := "file:///tmp/Dockerfile"
	s := NewService()
	for _, tc := range testCases {
		t.Run(tc.image, func(t *testing.T) {
			hover, err := s.Hover(context.Background(), u, tc.image)
			if os.Getenv("DOCKER_NETWORK_NONE") == "true" {
				var dns *net.DNSError
				require.True(t, errors.As(err, &dns))
				return
			}

			require.Nil(t, err)
			require.Equal(t, tc.result, hover)
		})
	}

	for _, tc := range testCases {
		t.Run(tc.image, func(t *testing.T) {
			defer configuration.Remove(u)
			configuration.Store(u, configuration.Configuration{Experimental: configuration.Experimental{
				VulnerabilityScanning: false,
				Scout: configuration.Scout{
					CriticalHighVulnerabilities: true,
					NotPinnedDigest:             true,
					RecommendedTag:              true,
					Vulnerabilites:              true,
				},
			}})
			hover, err := s.Hover(context.Background(), u, tc.image)
			if os.Getenv("DOCKER_NETWORK_NONE") == "true" {
				var dns *net.DNSError
				require.True(t, errors.As(err, &dns))
				return
			}

			require.Nil(t, err)
			require.Nil(t, hover)
		})
	}
}

func TestGetHovers_IgnoresSpecifics(t *testing.T) {
	testCases := []struct {
		name   string
		image  string
		config configuration.Scout
		result *protocol.Hover
	}{
		{
			name:  "alpine:3.16.1 (all)",
			image: "alpine:3.16.1",
			config: configuration.Scout{
				CriticalHighVulnerabilities: true,
				NotPinnedDigest:             true,
				RecommendedTag:              true,
				Vulnerabilites:              true,
			},
			result: &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: "Current image vulnerabilities:   1C   3H   9M   0L \r\n\r\nRecommended tags:\n\n<table>\n<tr><td><code>3.21.4</code></td><td align=\"right\">  0C</td><td align=\"right\">  1H</td><td align=\"right\">  2M</td><td align=\"right\">  0L</td><td align=\"right\"></td></tr>\n<tr><td><code>3.22.1</code></td><td align=\"right\">  0C</td><td align=\"right\">  1H</td><td align=\"right\">  2M</td><td align=\"right\">  0L</td><td align=\"right\"></td></tr>\n<tr><td><code>3.20.7</code></td><td align=\"right\">  0C</td><td align=\"right\">  1H</td><td align=\"right\">  2M</td><td align=\"right\">  2L</td><td align=\"right\"></td></tr>\n<tr><td><code>3.19.8</code></td><td align=\"right\">  0C</td><td align=\"right\">  1H</td><td align=\"right\">  1M</td><td align=\"right\">  2L</td><td align=\"right\"></td></tr>\n</table>\n",
				},
			},
		},
		{
			name:  "alpine:3.16.1 (CriticalHighVulnerabilities=false)",
			image: "alpine:3.16.1",
			config: configuration.Scout{
				CriticalHighVulnerabilities: false,
				NotPinnedDigest:             true,
				RecommendedTag:              true,
				Vulnerabilites:              true,
			},
			result: &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: "Recommended tags:\n\n<table>\n<tr><td><code>3.21.4</code></td><td align=\"right\">  0C</td><td align=\"right\">  1H</td><td align=\"right\">  2M</td><td align=\"right\">  0L</td><td align=\"right\"></td></tr>\n<tr><td><code>3.22.1</code></td><td align=\"right\">  0C</td><td align=\"right\">  1H</td><td align=\"right\">  2M</td><td align=\"right\">  0L</td><td align=\"right\"></td></tr>\n<tr><td><code>3.20.7</code></td><td align=\"right\">  0C</td><td align=\"right\">  1H</td><td align=\"right\">  2M</td><td align=\"right\">  2L</td><td align=\"right\"></td></tr>\n<tr><td><code>3.19.8</code></td><td align=\"right\">  0C</td><td align=\"right\">  1H</td><td align=\"right\">  1M</td><td align=\"right\">  2L</td><td align=\"right\"></td></tr>\n</table>\n",
				},
			},
		},
		{
			name:  "alpine:3.16.1 (RecommendedTag=false)",
			image: "alpine:3.16.1",
			config: configuration.Scout{
				CriticalHighVulnerabilities: true,
				NotPinnedDigest:             true,
				RecommendedTag:              false,
				Vulnerabilites:              true,
			},
			result: &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: "Current image vulnerabilities:   1C   3H   9M   0L ",
				},
			},
		},
		{
			name:  "ubuntu:24.04 (all)",
			image: "ubuntu:24.04",
			config: configuration.Scout{
				CriticalHighVulnerabilities: true,
				NotPinnedDigest:             true,
				RecommendedTag:              true,
				Vulnerabilites:              true,
			},
			result: &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: "Image vulnerabilities:   0C   0H   3M   5L \r\n\r\nRecommended tags:\n\n<table>\n<tr><td><code>25.10</code></td><td align=\"right\">  0C</td><td align=\"right\">  0H</td><td align=\"right\">  0M</td><td align=\"right\">  0L</td><td align=\"right\"></td></tr>\n<tr><td><code>25.04</code></td><td align=\"right\">  0C</td><td align=\"right\">  0H</td><td align=\"right\">  5M</td><td align=\"right\">  5L</td><td align=\"right\"></td></tr>\n</table>\n",
				},
			},
		},
		{
			name:  "ubuntu:24.04 (RecommendedTag=false)",
			image: "ubuntu:24.04",
			config: configuration.Scout{
				CriticalHighVulnerabilities: true,
				NotPinnedDigest:             true,
				RecommendedTag:              true,
				Vulnerabilites:              false,
			},
			result: &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: "Recommended tags:\n\n<table>\n<tr><td><code>25.10</code></td><td align=\"right\">  0C</td><td align=\"right\">  0H</td><td align=\"right\">  0M</td><td align=\"right\">  0L</td><td align=\"right\"></td></tr>\n<tr><td><code>25.04</code></td><td align=\"right\">  0C</td><td align=\"right\">  0H</td><td align=\"right\">  5M</td><td align=\"right\">  5L</td><td align=\"right\"></td></tr>\n</table>\n",
				},
			},
		},
		{
			name:  "ubuntu:24.04 (Vulnerabilites=false)",
			image: "ubuntu:24.04",
			config: configuration.Scout{
				CriticalHighVulnerabilities: true,
				NotPinnedDigest:             true,
				RecommendedTag:              false,
				Vulnerabilites:              true,
			},
			result: &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: "Image vulnerabilities:   0C   0H   3M   5L ",
				},
			},
		},
	}

	u := "file:///tmp/Dockerfile"
	s := NewService()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer configuration.Remove(u)
			configuration.Store(u, configuration.Configuration{Experimental: configuration.Experimental{
				VulnerabilityScanning: true,
				Scout:                 tc.config,
			}})
			hover, err := s.Hover(context.Background(), u, tc.image)
			if os.Getenv("DOCKER_NETWORK_NONE") == "true" {
				var dns *net.DNSError
				require.True(t, errors.As(err, &dns))
				return
			}

			require.Nil(t, err)
			require.Equal(t, tc.result, hover)
		})
	}

	for _, tc := range testCases {
		t.Run(tc.image, func(t *testing.T) {
			defer configuration.Remove(u)
			configuration.Store(u, configuration.Configuration{Experimental: configuration.Experimental{
				VulnerabilityScanning: false,
				Scout: configuration.Scout{
					CriticalHighVulnerabilities: true,
					NotPinnedDigest:             true,
					RecommendedTag:              true,
					Vulnerabilites:              true,
				},
			}})
			hover, err := s.Hover(context.Background(), u, tc.image)
			if os.Getenv("DOCKER_NETWORK_NONE") == "true" {
				var dns *net.DNSError
				require.True(t, errors.As(err, &dns))
				return
			}

			require.Nil(t, err)
			require.Nil(t, hover)
		})
	}
}
