package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// build builds the configured target with Buildroot. It runs with no network:
// every source was fetched by download, and checked against its hash. Every
// package's licence files go into the root filesystem, from Buildroot's own
// legal information, before the build makes it.
func build(args []string) error {
	flags := flag.NewFlagSet("build", flag.ExitOnError)
	name := flags.String("target", "", "amd64, arm64 or rpi")
	root := flags.String("root", "/", "where work/ is")
	parse(flags, args)
	if _, err := lookup(*name); err != nil {
		return err
	}
	l := under(*root)

	if err := l.make("legal-info"); err != nil {
		return err
	}
	legal := filepath.Join(l.output, "legal-info")
	licences := filepath.Join(l.system(*name), "rootfs_overlay/usr/share/licenses")
	if err := copyLicences(legal, licences); err != nil {
		return err
	}
	return l.make()
}

// copyLicences copies each target package's licence files from Buildroot's
// legal information into dest, under the package's name. The kernel's archive
// installs its own.
func copyLicences(legal, dest string) error {
	f, err := os.Open(filepath.Join(legal, "manifest.csv"))
	if err != nil {
		return err
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("legal-info wrote an empty manifest")
	}
	for _, row := range rows[1:] {
		pkg, version := row[0], row[1]
		if pkg == "fostrom-kernel" {
			continue
		}
		dir := pkg
		if version != "" {
			dir += "-" + version
		}
		src := filepath.Join(legal, "licenses", dir)
		if _, err := os.Stat(src); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err := copyTree(src, filepath.Join(dest, pkg)); err != nil {
			return err
		}
	}
	return nil
}
