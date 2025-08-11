package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUpdateChangelog(t *testing.T) {
	// relative to releaser directory, going up to repository root)
	inputPath := "../testdata/releaser/CHANGELOG.md"
	resultPath := "../testdata/releaser/CHANGELOG.result.md"
	expectedPath := "../testdata/releaser/CHANGELOG.expected.md"

	err := updateChangelog(inputPath, resultPath, "minor")
	require.NoError(t, err, "updateChangeLog failed")

	resultLines, err := readFileLines(resultPath)
	require.NoError(t, err, "failed to read expected file: %v", resultPath)

	expectedLines, err := readFileLines(expectedPath)
	require.NoError(t, err, "failed to read expected file: %v", expectedPath)

	today := time.Now().Format("2006-01-02")
	expectedVersionLine := fmt.Sprintf("## [0.16.0] - %s", today)

	for i := range resultLines {
		// Special handling for the version header line with date
		if strings.HasPrefix(expectedLines[i], "## [0.16.0] - 2025-08-08") {
			require.Equal(t, expectedVersionLine, resultLines[i])
		} else {
			require.Equal(t, expectedLines[i], resultLines[i])
		}
	}

	require.Equal(t, len(expectedLines), len(resultLines), "files have different number of lines")
}

func TestGenerateReleaseNotes(t *testing.T) {
	// relative to releaser directory, going up to repository root)
	changelogPath := "../CHANGELOG.md"
	expectedPath := "../testdata/releaser/RELEASE.md"

	expectedLines, err := readFileLines(expectedPath)
	require.NoError(t, err, "failed to read expected file: %v", expectedPath)

	result, err := generateReleaseNotes(changelogPath)
	require.NoError(t, err, "generateReleaseNotes failed")

	resultLines := strings.Split(result, "\n")
	for i := range resultLines {
		require.Equal(t, expectedLines[i], resultLines[i])
	}
	require.Equal(t, len(expectedLines), len(resultLines), "files have different number of lines")
}
