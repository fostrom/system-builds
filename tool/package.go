package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// pack packs the built system the way Nerves takes it, under the name Nerves
// asks for it by, and writes beside it the manifest, the capabilities and the
// sources. Each is the same bytes from every build of one commit.
func pack(args []string) error {
	flags := flag.NewFlagSet("package", flag.ExitOnError)
	name := flags.String("target", "", "amd64, arm64 or rpi")
	commit := flags.String("commit", "unknown", "the commit built")
	root := flags.String("root", "/", "where src/ and work/ are")
	parse(flags, args)
	t, err := lookup(*name)
	if err != nil {
		return err
	}
	l := under(*root)

	epoch := os.Getenv("SOURCE_DATE_EPOCH")
	if epoch == "" {
		return fmt.Errorf("no SOURCE_DATE_EPOCH")
	}
	versionFile, err := os.ReadFile(filepath.Join(l.src, "VERSION"))
	if err != nil {
		return err
	}
	version := strings.TrimSpace(string(versionFile))
	if err := os.MkdirAll(l.release, 0o755); err != nil {
		return err
	}

	system := "fostrom_system_" + t.name
	unpacked := filepath.Join(filepath.Dir(l.output), "system")
	if err := os.RemoveAll(unpacked); err != nil {
		return err
	}
	if err := os.MkdirAll(unpacked, 0o755); err != nil {
		return err
	}
	if err := l.make("system"); err != nil {
		return err
	}
	made := filepath.Join(l.output, system+".tar.gz")
	if err := run("", nil, "tar", "-xzf", made, "-C", unpacked); err != nil {
		return err
	}
	artifact := filepath.Join(
		l.release,
		fmt.Sprintf("%s-portable-%s-%s.tar.gz", system, version, nervesChecksum(versionFile)),
	)
	if err := tarball(epoch, artifact, unpacked, system, true); err != nil {
		return err
	}
	if err := os.RemoveAll(unpacked); err != nil {
		return err
	}

	prefix := filepath.Join(l.release, fmt.Sprintf("system-%s-%s", t.name, version))
	capabilities := filepath.Join(l.output, "target/etc/fostrom/capabilities.json")
	if err := copyFile(capabilities, prefix+".capabilities.json"); err != nil {
		return err
	}
	legal := filepath.Join(l.output, "legal-info")
	err = tarball(epoch, prefix+"-sources.tar", legal, "sources", false, notSources...)
	if err != nil {
		return err
	}
	return writeManifest(l, t, version, *commit, artifact, prefix+".manifest.json")
}

// Downloads that are builds rather than sources: the kernel's sources are in
// its kernel-builds release, the toolchain is Nerves' build, and the Pi's GPU
// firmware is published only as binaries.
var notSources = []string{
	"sources/fostrom-kernel-*",
	"sources/toolchain-external-custom",
	"sources/rpi-firmware-*",
}

// nervesChecksum is the checksum Nerves names a system artifact by: the first
// seven hex digits of the SHA-256 of each checksum file's SHA-256, and a
// system's only checksum file is VERSION.
func nervesChecksum(version []byte) string {
	inner := sha256.Sum256(version)
	outer := sha256.Sum256(inner[:])
	return strings.ToUpper(hex.EncodeToString(outer[:]))[:7]
}

// tarball writes dir, under parent, as a tar that is the same bytes from the
// same tree: names in order, every time the build's, no owners, modes as
// Buildroot normalises its overlays, and gzip, where used, without a name or
// time.
func tarball(epoch, out, parent, dir string, gzipped bool, exclude ...string) error {
	args := []string{
		"--create", "--format=gnu", "--sort=name", "--mtime=@" + epoch,
		"--owner=0", "--group=0", "--numeric-owner", "--mode=u+rwX,go=rX",
		"--file", out, "--directory", parent,
	}
	if gzipped {
		args = append(args, "--use-compress-program=gzip -n")
	}
	for _, e := range exclude {
		args = append(args, "--exclude="+e)
	}
	return run("", nil, "tar", append(args, dir)...)
}
