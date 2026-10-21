package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// download fetches every source the configured target needs into its own
// download directory, taking what the shared cache already has and adding
// everything it fetched to the cache. Buildroot checks every file against its
// hash on use, so the cache can be shared by every target and kept forever.
func download(args []string) error {
	flags := flag.NewFlagSet("download", flag.ExitOnError)
	root := flags.String("root", "/", "where work/ and cache/ are")
	parse(flags, args)
	l := under(*root)

	needed, err := downloads(l)
	if err != nil {
		return err
	}
	cached := 0
	for _, f := range needed {
		from, to := filepath.Join(l.cache, f), filepath.Join(l.dl, f)
		if _, err := os.Stat(from); err != nil {
			continue
		}
		if err := copyFile(from, to); err != nil {
			return err
		}
		cached++
	}

	if err := l.make("source"); err != nil {
		return err
	}

	fetched := 0
	err = filepath.WalkDir(l.dl, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return err
		}
		rel, _ := filepath.Rel(l.dl, p)
		to := filepath.Join(l.cache, rel)
		if _, err := os.Stat(to); err == nil {
			return nil
		}
		fetched++
		return copyFile(p, to)
	})
	if err != nil {
		return err
	}
	fmt.Printf("download: %d files, %d from the cache, %d fetched\n", len(needed), cached, fetched)
	return nil
}

// downloads lists every file the target's packages download, as paths under
// a download directory, from Buildroot's own description of them.
func downloads(l layout) ([]string, error) {
	cmd := exec.Command("make", "-s", "-C", l.output, "show-info")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("make show-info: %w", err)
	}
	var info map[string]struct {
		DLDir     string `json:"dl_dir"`
		Downloads []struct {
			Source string `json:"source"`
		} `json:"downloads"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("make show-info: %w", err)
	}
	var files []string
	for _, pkg := range info {
		for _, d := range pkg.Downloads {
			files = append(files, filepath.Join(pkg.DLDir, d.Source))
		}
	}
	slices.Sort(files)
	return slices.Compact(files), nil
}
