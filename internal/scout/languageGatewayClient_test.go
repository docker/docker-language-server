package scout

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostImage(t *testing.T) {
	testCases := []struct {
		image    string
		err      error
		response ImageResponse
	}{
		{
			// this triggers a 400 which is ignored
			image: "alpine:3.16.1::",
			err:   nil,
		},
		{
			image: "alpine:3.16.1",
			err:   nil,
			response: ImageResponse{
				Image: Image{
					Registry:   "docker.io",
					Repository: "library/alpine",
					Short:      "alpine:3.16.1",
					Tag:        "3.16.1",
				},
				Diagnostics: []Diagnostic{
					{
						Kind:     "not_pinned_digest",
						Message:  "The image can be pinned to a digest",
						Severity: "hint",
					},
				},
				Edits: []Edit{
					{
						Title:      "Pin the base image digest",
						Edit:       "alpine:3.16.1@sha256:7580ece7963bfa863801466c0a488f11c86f85d9988051a9f9c68cb27f6b7872",
						Diagnostic: "not_pinned_digest",
					},
				},
				Infos: []Info{
					{
						Description: Description{Plaintext: "Current image vulnerabilities: 1C 3H 9M 0L"},
						Kind:        "critical_high_vulnerabilities",
					},
				},
			},
		},
	}

	c := NewLanguageGatewayClient()
	for _, tc := range testCases {
		response, err := c.PostImage(context.Background(), "jwt", tc.image)
		if os.Getenv("DOCKER_NETWORK_NONE") == "true" {
			var dns *net.DNSError
			require.True(t, errors.As(err, &dns))
			continue
		}

		require.Equal(t, tc.err, err)
		if tc.err != nil {
			require.Len(t, tc.response.Diagnostics, 0)
			require.Len(t, tc.response.Edits, 0)
			require.Len(t, tc.response.Infos, 0)
			continue
		}

		require.Equal(t, tc.response.Image.Registry, response.Image.Registry)
		require.Equal(t, tc.response.Image.Repository, response.Image.Repository)
		require.Equal(t, tc.response.Image.Short, response.Image.Short)
		require.Equal(t, tc.response.Image.Tag, response.Image.Tag)

		for _, expectedDiagnostic := range tc.response.Diagnostics {
			found := false
			for _, diagnostic := range response.Diagnostics {
				if expectedDiagnostic.Kind == diagnostic.Kind {
					require.Equal(t, expectedDiagnostic.Message, diagnostic.Message)
					require.Equal(t, expectedDiagnostic.Severity, diagnostic.Severity)
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected diagnostic kind not found: %v", expectedDiagnostic.Kind)
			}
		}

		for _, expectedEdit := range tc.response.Edits {
			found := false
			for _, edit := range response.Edits {
				if expectedEdit.Edit == edit.Edit {
					require.Equal(t, expectedEdit.Diagnostic, edit.Diagnostic)
					require.Equal(t, expectedEdit.Title, edit.Title)
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected edit not found: %v", expectedEdit.Edit)
			}
		}

		for _, expectedInfo := range tc.response.Infos {
			found := false
			for _, info := range response.Infos {
				if expectedInfo.Kind == info.Kind {
					require.Equal(t, expectedInfo.Description.Plaintext, info.Description.Plaintext)
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected info kind not found: %v", expectedInfo.Kind)
			}
		}
	}
}
