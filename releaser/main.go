package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const localChangelogPath = "./CHANGELOG.md"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <update-changelog|generate-release-notes> [args...]\n", os.Args[0])
		os.Exit(1)
	}

	command := strings.ToLower(os.Args[1])

	switch command {
	case "update-changelog":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s update-changelog <major|minor|patch>\n", os.Args[0])
			os.Exit(1)
		}

		versionType := strings.ToLower(os.Args[2])
		if versionType != "major" && versionType != "minor" && versionType != "patch" {
			fmt.Fprintf(os.Stderr, "Error: version type must be 'major', 'minor', or 'patch'\n")
			os.Exit(1)
		}

		err := updateChangelog(localChangelogPath, localChangelogPath, versionType)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully updated CHANGELOG.md with %s version bump\n", versionType)

	case "generate-release-notes":
		content, err := generateReleaseNotes(localChangelogPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(content)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'. Use 'update-changelog' or 'generate-release-notes'\n", command)
		os.Exit(1)
	}
}

func updateChangelog(changelogPath, modifiedChangelogPath, versionType string) error {
	lines, err := readFileLines(changelogPath)
	if err != nil {
		return fmt.Errorf("failed to read changelog: %w", err)
	}

	// Find the Unreleased section and the next version
	unreleasedIndex := -1
	nextVersionIndex := -1
	linksStartIndex := -1
	var currentVersion string

	for i, line := range lines {
		if strings.Contains(line, "## [Unreleased]") {
			unreleasedIndex = i
		} else if nextVersionIndex == -1 && unreleasedIndex != -1 && strings.HasPrefix(line, "## [") && !strings.Contains(line, "Unreleased") {
			nextVersionIndex = i
			// Extract current version from the line like "## [0.15.0] - 2025-08-06"
			re := regexp.MustCompile(`^## \[([0-9]+\.[0-9]+\.[0-9]+)\]`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				currentVersion = matches[1]
			}
		} else if linksStartIndex == -1 && strings.HasPrefix(lines[i], "[") && strings.Contains(lines[i], "]: https://github.com/") {
			linksStartIndex = i
			break
		}
	}

	if unreleasedIndex == -1 {
		return fmt.Errorf("could not find 'Unreleased' section in changelog")
	}
	if nextVersionIndex == -1 || currentVersion == "" {
		return fmt.Errorf("could not find current version section after 'Unreleased'")
	}
	if linksStartIndex == -1 {
		return fmt.Errorf("could not find links section in changelog")
	}

	newVersion, err := bumpVersion(currentVersion, versionType)
	if err != nil {
		return fmt.Errorf("failed to bump version: %w", err)
	}

	// Update the Unreleased section to the new version
	lines[unreleasedIndex] = fmt.Sprintf("## [%s] - %s", newVersion, today())

	// Update the links section
	updatedLinks := updateLinks(lines[linksStartIndex:], newVersion, currentVersion)
	lines = append(lines[:linksStartIndex], updatedLinks...)

	// Write the updated changelog back to the output file
	outFile, err := os.Create(modifiedChangelogPath)
	if err != nil {
		return fmt.Errorf("failed to create changelog file: %w", err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}
	return writer.Flush()
}

func bumpVersion(version, versionType string) (string, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version format: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid patch version: %s", parts[2])
	}

	switch versionType {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

func today() string {
	return time.Now().UTC().Format(time.DateOnly)
}

func updateLinks(linkLines []string, newVersion, previousVersion string) []string {
	updatedLinks := make([]string, len(linkLines)+1)

	// Add the new Unreleased link
	updatedLinks[0] = fmt.Sprintf("[Unreleased]: https://github.com/docker/docker-language-server/compare/v%s...main", newVersion)

	// Add the new version link
	updatedLinks[1] = fmt.Sprintf("[%s]: https://github.com/docker/docker-language-server/compare/v%s...v%s", newVersion, previousVersion, newVersion)

	// Add all existing links except the old Unreleased link
	for i := 1; i < len(linkLines); i++ {
		updatedLinks[i+1] = linkLines[i]
	}

	return updatedLinks
}

func generateReleaseNotes(changelogPath string) (string, error) {
	lines, err := readFileLines(changelogPath)
	if err != nil {
		return "", fmt.Errorf("failed to read changelog: %w", err)
	}

	// Find the first two ## headers
	firstHeaderIndex := -1
	secondHeaderIndex := -1

	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if firstHeaderIndex == -1 {
				firstHeaderIndex = i
			} else if secondHeaderIndex == -1 {
				secondHeaderIndex = i
				break
			}
		}
	}

	if secondHeaderIndex == -1 {
		return "", fmt.Errorf("could not find two ## headers in the changelog")
	}

	// Extract the content between the two headers
	content := []string{}
	for i := firstHeaderIndex + 1; i < secondHeaderIndex; i++ {
		content = append(content, lines[i])
	}

	// Strip leading and trailing whitespace-only lines
	start := 0
	end := len(content) - 1

	// Find first non-whitespace line
	for start < len(content) && strings.TrimSpace(content[start]) == "" {
		start++
	}

	// Find last non-whitespace line
	for end >= 0 && strings.TrimSpace(content[end]) == "" {
		end--
	}

	// Build the result string
	var result strings.Builder
	for i := start; i <= end; i++ {
		result.WriteString(content[i])
		if i < end {
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}

func readFileLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}
