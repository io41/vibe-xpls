package schemagen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"
)

const defaultGitHubBaseURL = "https://api.github.com"

type DriftOptions struct {
	GitHubBaseURL string
	HTTPClient    *http.Client
	Token         string
	RequireToken  bool
}

type driftChecker struct {
	baseURL string
	client  *http.Client
	token   string
}

type githubRefResponse struct {
	Object struct {
		SHA string `json:"sha"`
	} `json:"object"`
}

type githubTagResponse struct {
	Name string `json:"name"`
}

type stableVersion struct {
	major int
	minor int
	patch int
	tag   string
}

type driftCRDDocument struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Group string `yaml:"group"`
		Names struct {
			Plural string `yaml:"plural"`
		} `yaml:"names"`
	} `yaml:"spec"`
}

func CheckDrift(cfg Config, opts DriftOptions) error {
	if opts.RequireToken && strings.TrimSpace(opts.Token) == "" {
		return errors.New("GITHUB_TOKEN is required for schema drift check")
	}

	checker := newDriftChecker(opts)
	problems := []string{}
	for _, release := range cfg.Releases {
		sha, err := checker.fetchTagSHA(release.Tag)
		if err != nil {
			return fmt.Errorf("query upstream drift for %s: %w", release.Tag, err)
		}
		if !strings.EqualFold(sha, release.Commit) {
			problems = append(problems, fmt.Sprintf("pinned release %s resolves to different commit %s, want %s", release.Tag, sha, release.Commit))
		}
	}

	if len(cfg.Releases) > 0 {
		tags, err := checker.fetchTags()
		if err != nil {
			return fmt.Errorf("query upstream drift: %w", err)
		}
		if problem := newerStableReleaseProblem(cfg.Releases, tags); problem != "" {
			problems = append(problems, problem)
		}
	}
	if len(problems) > 0 {
		return driftProblemsError(problems)
	}

	for _, release := range cfg.Releases {
		releaseProblems, err := checker.checkReleaseCRDContent(release)
		if err != nil {
			return fmt.Errorf("query upstream drift for %s: %w", release.Tag, err)
		}
		problems = append(problems, releaseProblems...)
	}
	if len(problems) > 0 {
		return driftProblemsError(problems)
	}
	return nil
}

func newDriftChecker(opts DriftOptions) driftChecker {
	baseURL := opts.GitHubBaseURL
	if baseURL == "" {
		baseURL = defaultGitHubBaseURL
	}
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return driftChecker{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
		token:   opts.Token,
	}
}

func (c driftChecker) fetchTagSHA(tag string) (string, error) {
	var ref githubRefResponse
	path := "/repos/crossplane/crossplane/git/ref/tags/" + url.PathEscape(tag)
	if err := c.getJSON(path, &ref); err != nil {
		return "", err
	}
	if ref.Object.SHA == "" {
		return "", errors.New("tag ref response missing object.sha")
	}
	return ref.Object.SHA, nil
}

func (c driftChecker) fetchTags() ([]string, error) {
	var response []githubTagResponse
	if err := c.getJSON("/repos/crossplane/crossplane/tags?per_page=100", &response); err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(response))
	for _, tag := range response {
		if tag.Name != "" {
			tags = append(tags, tag.Name)
		}
	}
	return tags, nil
}

func (c driftChecker) checkReleaseCRDContent(release ReleaseConfig) ([]string, error) {
	crdFiles, err := yamlFiles(release.RawCRDDir)
	if err != nil {
		return nil, err
	}
	problems := []string{}
	for _, localPath := range crdFiles {
		docs, localSHA, err := readDriftCRDFile(localPath)
		if err != nil {
			return nil, err
		}
		if !containsCRD(docs) {
			continue
		}
		upstreamPath := upstreamCRDPath(release, localPath, docs)
		raw, err := c.getRaw("/repos/crossplane/crossplane/contents/" + escapeGitHubPath(upstreamPath) + "?ref=" + url.QueryEscape(release.Tag))
		if err != nil {
			return nil, fmt.Errorf("fetch CRD %s: %w", upstreamPath, err)
		}
		sum := sha256.Sum256(raw)
		upstreamSHA := hex.EncodeToString(sum[:])
		if upstreamSHA != localSHA {
			problems = append(problems, fmt.Sprintf("CRD content drift for %s %s: upstream SHA-256 %s, want committed %s", release.Tag, upstreamPath, upstreamSHA, localSHA))
		}
	}
	return problems, nil
}

func (c driftChecker) getJSON(path string, out any) error {
	raw, err := c.get(path, "application/vnd.github+json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("parse GitHub response: %w", err)
	}
	return nil
}

func (c driftChecker) getRaw(path string) ([]byte, error) {
	return c.get(path, "application/vnd.github.raw")
}

func (c driftChecker) get(path, accept string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "vibe-xpls-schema-gen")
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body := strings.TrimSpace(string(raw))
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		if body == "" {
			return nil, fmt.Errorf("GET %s: GitHub returned %s", path, resp.Status)
		}
		return nil, fmt.Errorf("GET %s: GitHub returned %s: %s", path, resp.Status, body)
	}
	return raw, nil
}

func newerStableReleaseProblem(configured []ReleaseConfig, upstreamTags []string) string {
	latestConfigured, ok := latestConfiguredRelease(configured)
	if !ok {
		return ""
	}
	latestUpstream := latestConfigured
	for _, tag := range upstreamTags {
		version, ok := parseStableSemverTag(tag)
		if !ok {
			continue
		}
		if compareStableVersion(version, latestUpstream) > 0 {
			latestUpstream = version
		}
	}
	if compareStableVersion(latestUpstream, latestConfigured) <= 0 {
		return ""
	}
	return fmt.Sprintf("newer stable Crossplane release %s is available, latest configured release is %s", latestUpstream.tag, latestConfigured.tag)
}

func latestConfiguredRelease(releases []ReleaseConfig) (stableVersion, bool) {
	var latest stableVersion
	ok := false
	for _, release := range releases {
		version, stable := parseStableSemverTag(release.Tag)
		if !stable {
			continue
		}
		if !ok || compareStableVersion(version, latest) > 0 {
			latest = version
			ok = true
		}
	}
	return latest, ok
}

func parseStableSemverTag(tag string) (stableVersion, bool) {
	if !strings.HasPrefix(tag, "v") {
		return stableVersion{}, false
	}
	parts := strings.Split(strings.TrimPrefix(tag, "v"), ".")
	if len(parts) != 3 {
		return stableVersion{}, false
	}
	values := [3]int{}
	for i, part := range parts {
		if part == "" || !isASCIIInteger(part) {
			return stableVersion{}, false
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return stableVersion{}, false
		}
		values[i] = value
	}
	return stableVersion{major: values[0], minor: values[1], patch: values[2], tag: tag}, true
}

func isASCIIInteger(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func compareStableVersion(a, b stableVersion) int {
	switch {
	case a.major != b.major:
		return a.major - b.major
	case a.minor != b.minor:
		return a.minor - b.minor
	default:
		return a.patch - b.patch
	}
}

func readDriftCRDFile(path string) ([]driftCRDDocument, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read CRD %s: %w", path, err)
	}
	sum := sha256.Sum256(raw)
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	docs := []driftCRDDocument{}
	for {
		var doc driftCRDDocument
		if err := dec.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, "", fmt.Errorf("parse CRD %s: %w", path, err)
		}
		if doc.APIVersion == "" && doc.Kind == "" {
			continue
		}
		docs = append(docs, doc)
	}
	return docs, hex.EncodeToString(sum[:]), nil
}

func containsCRD(docs []driftCRDDocument) bool {
	for _, doc := range docs {
		if doc.APIVersion == "apiextensions.k8s.io/v1" && doc.Kind == "CustomResourceDefinition" {
			return true
		}
	}
	return false
}

func upstreamCRDPath(release ReleaseConfig, localPath string, docs []driftCRDDocument) string {
	rel := relativeSourcePath(release.RawCRDDir, localPath)
	if prefix, ok := upstreamPathPrefix(release); ok {
		return pathpkg.Join(prefix, rel)
	}
	if strings.HasPrefix(rel, "cluster/crds/") {
		return rel
	}
	if inferred, ok := inferCRDPathFromDocument(docs); ok {
		return inferred
	}
	return pathpkg.Join("cluster/crds", rel)
}

func upstreamPathPrefix(release ReleaseConfig) (string, bool) {
	clean := filepath.Clean(release.RawCRDDir)
	parts := strings.Split(clean, string(filepath.Separator))
	for i, part := range parts {
		if part != release.Tag {
			continue
		}
		if i+1 >= len(parts) {
			return "", false
		}
		return pathpkg.Join(parts[i+1:]...), true
	}
	return "", false
}

func inferCRDPathFromDocument(docs []driftCRDDocument) (string, bool) {
	for _, doc := range docs {
		if doc.APIVersion != "apiextensions.k8s.io/v1" || doc.Kind != "CustomResourceDefinition" {
			continue
		}
		if group, plural, ok := crdGroupPlural(doc); ok {
			return pathpkg.Join("cluster/crds", group+"_"+plural+".yaml"), true
		}
	}
	return "", false
}

func crdGroupPlural(doc driftCRDDocument) (string, string, bool) {
	if doc.Metadata.Name != "" {
		plural, group, ok := strings.Cut(doc.Metadata.Name, ".")
		if ok && plural != "" && group != "" {
			return group, plural, true
		}
	}
	if doc.Spec.Group != "" && doc.Spec.Names.Plural != "" {
		return doc.Spec.Group, doc.Spec.Names.Plural, true
	}
	return "", "", false
}

func escapeGitHubPath(value string) string {
	parts := strings.Split(value, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func driftProblemsError(problems []string) error {
	var b strings.Builder
	b.WriteString("schema upstream drift detected:")
	for _, problem := range problems {
		b.WriteString("\n- ")
		b.WriteString(problem)
	}
	return errors.New(b.String())
}
