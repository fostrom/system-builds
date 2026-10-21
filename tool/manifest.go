package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// The manifest says what a system build produced, so it can be read without
// unpacking it: what it was built from, every package with its licence, the
// resolved Buildroot options, and the hash of every image and of the artifact.
// Nothing in it records when or where it was built, so two builds of one
// commit agree on it too.
type manifestDoc struct {
	Target         string            `json:"target"`
	Version        string            `json:"version"`
	Commit         string            `json:"commit"`
	Platform       string            `json:"platform"`
	Architecture   string            `json:"architecture"`
	Buildroot      string            `json:"buildroot"`
	NervesSystemBR string            `json:"nerves_system_br"`
	Packages       []packageInfo     `json:"packages"`
	Config         map[string]string `json:"config"`
	Images         []fileInfo        `json:"images"`
	Artifact       fileInfo          `json:"artifact"`
}

type packageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	License string `json:"license"`
}

type fileInfo struct {
	Path   string `json:"path"`
	Size   int64  `json:"size,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	Link   string `json:"link,omitempty"`
}

var buildrootVersion = regexp.MustCompile(`(?m)^export BR2_VERSION := (\S+)$`)

func writeManifest(l layout, t target, version, commit, artifact, out string) error {
	doc := manifestDoc{
		Target:       t.name,
		Version:      version,
		Commit:       commit,
		Platform:     t.platform,
		Architecture: t.arch,
		Config:       map[string]string{},
	}

	makefile, err := os.ReadFile(filepath.Join(l.buildroot, "Makefile"))
	if err != nil {
		return err
	}
	m := buildrootVersion.FindSubmatch(makefile)
	if m == nil {
		return fmt.Errorf("no BR2_VERSION in Buildroot's Makefile")
	}
	doc.Buildroot = string(m[1])
	nerves, err := os.ReadFile(filepath.Join(l.nerves, "VERSION"))
	if err != nil {
		return err
	}
	doc.NervesSystemBR = strings.TrimSpace(string(nerves))

	if doc.Packages, err = readPackages(
		filepath.Join(l.output, "legal-info/manifest.csv"),
	); err != nil {
		return err
	}

	config, err := os.ReadFile(filepath.Join(l.output, ".config"))
	if err != nil {
		return err
	}
	for line := range strings.SplitSeq(string(config), "\n") {
		if option, value, ok := strings.Cut(line, "="); ok && strings.HasPrefix(option, "BR2_") {
			doc.Config[option] = value
		}
	}

	images := filepath.Join(l.output, "images")
	err = filepath.WalkDir(images, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(images, p)
		f, err := describe(p, filepath.ToSlash(rel))
		doc.Images = append(doc.Images, f)
		return err
	})
	if err != nil {
		return err
	}
	if doc.Artifact, err = describe(artifact, filepath.Base(artifact)); err != nil {
		return err
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(out, append(data, '\n'), 0o644)
}

// readPackages reads every target package, its version and its licence, from
// Buildroot's legal information, in name order.
func readPackages(manifest string) ([]packageInfo, error) {
	f, err := os.Open(manifest)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s is empty", manifest)
	}
	packages := []packageInfo{}
	for _, row := range rows[1:] {
		packages = append(packages, packageInfo{Name: row[0], Version: row[1], License: row[2]})
	}
	slices.SortFunc(packages, func(a, b packageInfo) int { return strings.Compare(a.Name, b.Name) })
	return packages, nil
}

// describe gives a file's size and hash, or a link's target.
func describe(path, name string) (fileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return fileInfo{}, err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		to, err := os.Readlink(path)
		return fileInfo{Path: name, Link: to}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fileInfo{}, err
	}
	sum := sha256.Sum256(data)
	return fileInfo{Path: name, Size: info.Size(), SHA256: hex.EncodeToString(sum[:])}, nil
}
