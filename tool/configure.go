package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// configure unpacks Buildroot and nerves_system_br, patches them as
// nerves_system_br's create-build.sh would and then with ours, writes the
// target's system directory, and makes the defconfig. create-build.sh itself
// takes one external tree, and this build has two.
func configure(args []string) error {
	flags := flag.NewFlagSet("configure", flag.ExitOnError)
	name := flags.String("target", "", "amd64, arm64 or rpi")
	root := flags.String("root", "/", "where src/ and sources/ are, and work/ goes")
	parse(flags, args)

	t, err := lookup(*name)
	if err != nil {
		return err
	}
	l := under(*root)

	buildroot, err := one(filepath.Join(l.sources, "buildroot-*.tar.gz"))
	if err != nil {
		return err
	}
	nerves, err := one(filepath.Join(l.sources, "nerves_system_br-*.tar"))
	if err != nil {
		return err
	}
	if err := unpack(buildroot, l.buildroot); err != nil {
		return err
	}
	if err := unpackHex(nerves, l.nerves); err != nil {
		return err
	}
	if err := sameBuildroot(buildroot, l.nerves); err != nil {
		return err
	}
	for _, p := range [][2]string{
		{l.buildroot, filepath.Join(l.nerves, "patches")},
		{l.buildroot, filepath.Join(l.src, "patches/buildroot")},
		{l.nerves, filepath.Join(l.src, "patches/nerves_system_br")},
	} {
		script := filepath.Join(l.buildroot, "support/scripts/apply-patches.sh")
		if err := run("", nil, script, "-s", p[0], p[1]); err != nil {
			return err
		}
	}

	system := l.system(*name)
	if err := writeSystem(l, *name, t, system); err != nil {
		return err
	}
	return run("", nil, "make", "-C", l.buildroot, "O="+l.output,
		"BR2_EXTERNAL="+l.nerves+":"+filepath.Join(l.src, "buildroot"),
		"NERVES_DEFCONFIG_DIR="+system,
		"BR2_DEFCONFIG="+filepath.Join(system, "nerves_defconfig"),
		"defconfig")
}

// writeSystem writes the target's system directory: its defconfig, the
// options common to every target followed by its own, the files the defconfig
// and the hooks name, rendered with the target's values, and the root
// filesystem overlay.
func writeSystem(l layout, name string, t target, system string) error {
	var defconfig strings.Builder
	for _, f := range []string{"common.defconfig", name + ".defconfig"} {
		data, err := os.ReadFile(filepath.Join(l.src, "config", f))
		if err != nil {
			return err
		}
		defconfig.Write(data)
		defconfig.WriteString("\n")
	}
	if err := os.MkdirAll(filepath.Join(system, "rootfs_overlay"), 0o755); err != nil {
		return err
	}
	hook := func(command string) string {
		return fmt.Sprintf("#!/bin/sh\nexec system-tool %s -target %s\n", command, name)
	}
	files := map[string]struct {
		body string
		mode os.FileMode
	}{
		"nerves_defconfig": {defconfig.String(), 0o644},
		"Config.in":        {"", 0o644},
		"post-build.sh":    {hook("post-build"), 0o755},
		"post-createfs.sh": {hook("post-image"), 0o755},
	}
	for file, f := range files {
		if err := os.WriteFile(filepath.Join(system, file), []byte(f.body), f.mode); err != nil {
			return err
		}
	}

	rendered := map[string]string{
		"VERSION":                        "VERSION",
		"busybox.fragment":               "config/busybox.fragment",
		"fwup.conf":                      "fwup/" + t.boot + ".conf",
		"fwup-ops.conf":                  "fwup/" + t.boot + "-ops.conf",
		"fwup_include/fwup-common.conf":  "fwup/" + t.boot + "-layout.conf",
		"fwup_include/provisioning.conf": "fwup/provisioning.conf",
	}
	values := map[string]any{
		"Target":       name,
		"Platform":     t.platform,
		"Architecture": t.arch,
		"Signature":    t.signature,
		"Image":        t.image,
		"Console":      t.console,
		"EFI":          t.efi,
	}
	switch t.boot {
	case "grub":
		rendered["grub.cfg"] = "boot/grub.cfg"
		rendered["grub-embedded.cfg"] = "boot/grub-embedded.cfg"
	case "rpi":
		rendered["config.txt"] = "boot/rpi-config.txt"
		rendered["cmdline-a.txt"] = "boot/rpi-cmdline-a.txt"
		rendered["cmdline-b.txt"] = "boot/rpi-cmdline-b.txt"
		rendered["ramoops-pi3.dts"] = "boot/ramoops-pi3.dts"
		rendered["ramoops-pi4.dts"] = "boot/ramoops-pi4.dts"
		files, err := readBootFiles(filepath.Join(l.src, "boot/rpi.boot-files"))
		if err != nil {
			return err
		}
		values["BootFiles"] = files
	}
	for dest, src := range rendered {
		err := render(filepath.Join(l.src, src), filepath.Join(system, dest), values)
		if err != nil {
			return err
		}
	}

	overlay := filepath.Join(system, "rootfs_overlay")
	for _, tree := range []string{"common", t.boot} {
		if err := copyTree(filepath.Join(l.src, "rootfs", tree), overlay); err != nil {
			return err
		}
	}
	return nil
}

// A file on a Pi's boot partition.
type bootFile struct {
	Name  string // on the partition
	Path  string // under images/
	Delta bool   // patched against the other slot's copy on an upgrade
}

// readBootFiles reads a boot file list: a file a line, as its name, its path,
// and delta or nothing.
func readBootFiles(path string) ([]bootFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var files []bootFile
	for n, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		switch {
		case len(f) == 0 || strings.HasPrefix(f[0], "#"):
		case len(f) == 2:
			files = append(files, bootFile{Name: f[0], Path: f[1]})
		case len(f) == 3 && f[2] == "delta":
			files = append(files, bootFile{Name: f[0], Path: f[1], Delta: true})
		default:
			return nil, fmt.Errorf("%s:%d: want a name, a path, and delta or nothing", path, n+1)
		}
	}
	return files, nil
}

// render writes a file from this repository with a target's values filled in,
// refusing any value the target does not have.
func render(src, dest string, values map[string]any) error {
	text, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmpl, err := template.New(filepath.Base(src)).
		Funcs(template.FuncMap{"upper": strings.ToUpper}).
		Option("missingkey=error").
		Parse(string(text))
	if err != nil {
		return err
	}
	var out strings.Builder
	if err := tmpl.Execute(&out, values); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, []byte(out.String()), 0o644)
}

// unpack extracts a tarball's one top directory into dir.
func unpack(archive, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return run("", nil, "tar", "-xf", archive, "-C", dir, "--strip-components=1")
}

// unpackHex extracts a Hex package, whose files are in a tarball inside the
// tarball.
func unpackHex(archive, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	contents := filepath.Join(dir, "contents.tar.gz")
	if err := run("", nil, "tar", "-xf", archive, "-C", dir, "contents.tar.gz"); err != nil {
		return err
	}
	if err := run("", nil, "tar", "-xzf", contents, "-C", dir); err != nil {
		return err
	}
	return os.Remove(contents)
}

var nervesBuildroot = regexp.MustCompile(`(?m)^NERVES_BR_VERSION=(\S+)$`)

// sameBuildroot refuses a Buildroot other than the one nerves_system_br's
// patches are written for.
func sameBuildroot(tarball, nerves string) error {
	script, err := os.ReadFile(filepath.Join(nerves, "create-build.sh"))
	if err != nil {
		return err
	}
	m := nervesBuildroot.FindSubmatch(script)
	if m == nil {
		return fmt.Errorf("no NERVES_BR_VERSION in nerves_system_br's create-build.sh")
	}
	got, want := filepath.Base(tarball), "buildroot-"+string(m[1])+".tar.gz"
	if got != want {
		return fmt.Errorf("nerves_system_br is written for %s, not %s", want, got)
	}
	return nil
}
