// This tool builds a target's Nerves system with Buildroot, and packs the
// system, its manifest and its sources for a release.
package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const usage = `usage: system-tool <command> [flags]

  configure   unpack and patch Buildroot, write the target's system directory, make defconfig
  download    fetch every source the target needs, through the shared download cache
  build       build the system with Buildroot, offline
  package     pack the system for Nerves, with its manifest, capabilities and sources

  Called by Buildroot:

  post-build  after packages are installed, before the root filesystem is made
  post-image  after the root filesystem is made
`

func main() {
	commands := map[string]func([]string) error{
		"configure":  configure,
		"download":   download,
		"build":      build,
		"package":    pack,
		"post-build": postBuild,
		"post-image": postImage,
	}
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	run, ok := commands[os.Args[1]]
	if !ok {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err := run(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[1], err)
		os.Exit(1)
	}
}

// parse reads a command's flags and refuses anything after them, which would
// otherwise end parsing there and leave every later flag at its default.
func parse(flags *flag.FlagSet, args []string) {
	flags.Parse(args)
	if flags.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "%s: unexpected argument %q\n", flags.Name(), flags.Arg(0))
		os.Exit(2)
	}
}

// Where a build keeps things: under / in the build container, and under a
// scratch directory in a test.
type layout struct {
	src       string // this repository's inputs
	sources   string // the pinned tarballs
	cache     string // downloads shared by every target and every build
	buildroot string // inside nerves_system_br, where its scripts look for it
	nerves    string // nerves_system_br
	output    string // Buildroot's output directory
	dl        string // this target's downloads
	release   string // what a release publishes
}

func under(root string) layout {
	at := func(p string) string { return filepath.Join(root, p) }
	return layout{
		src:       at("src"),
		sources:   at("sources"),
		cache:     at("cache"),
		buildroot: at("work/nerves_system_br/buildroot"),
		nerves:    at("work/nerves_system_br"),
		output:    at("work/o"),
		dl:        at("work/dl"),
		release:   at("work/release"),
	}
}

// system is the target's system directory: what Nerves calls the system, and
// Buildroot sees as NERVES_DEFCONFIG_DIR. Its name is the system's name.
func (l layout) system(target string) string {
	return filepath.Join(filepath.Dir(l.output), "fostrom_system_"+target)
}

// make runs Buildroot in the output directory, which remembers the external
// trees and the system directory from the defconfig.
func (l layout) make(goals ...string) error {
	env := append(os.Environ(), "BR2_DL_DIR="+l.dl)
	return run("", env, "make", append([]string{"-C", l.output}, goals...)...)
}

// run runs a command in dir, its output going to the build's.
func run(dir string, env []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = dir, env, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// one finds the single file a pattern matches.
func one(pattern string) (string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) != 1 {
		return "", fmt.Errorf("want one %s, found %v", pattern, matches)
	}
	return matches[0], nil
}

// copyTree copies every file under src into dest, keeping their paths.
func copyTree(src, dest string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		return copyFile(p, filepath.Join(dest, rel))
	})
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
