package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// The names a bundle may put in requires.features: an API, the same on every
// target, and one that only grows. devicekit's Fostrom.OS.Capabilities holds
// the same list.
var features = []string{"can", "gpio", "i2c", "spi", "usb-gadget", "video"}

// What the kernel can do, each name with the options it needs built.
var inventory = []struct {
	name    string
	options []string
}{
	{"erofs", []string{"CONFIG_EROFS_FS"}},
	{"squashfs", []string{"CONFIG_SQUASHFS"}},
	{"tpm", []string{"CONFIG_TCG_TPM"}},
	{"verity", []string{"CONFIG_DM_VERITY"}},
	{"encryption", []string{"CONFIG_DM_CRYPT"}},
	{"wireguard", []string{"CONFIG_WIREGUARD"}},
	{"sandbox", []string{
		"CONFIG_NAMESPACES", "CONFIG_PID_NS", "CONFIG_IPC_NS", "CONFIG_UTS_NS",
		"CONFIG_NET_NS", "CONFIG_CGROUPS",
	}},
	{"limits", []string{"CONFIG_MEMCG", "CONFIG_CGROUP_PIDS", "CONFIG_CGROUP_SCHED"}},
}

// The container architecture a bundle runs as, by the target's architecture.
var bundleArch = map[string]string{"x86_64": "amd64", "aarch64": "arm64"}

// devicekit's capabilities.json, schema 2.
type capabilities struct {
	Schema        int      `json:"schema"`
	Target        string   `json:"target"`
	SystemVersion string   `json:"system_version"`
	Arch          string   `json:"arch"`
	System        []string `json:"system"`
	Features      []string `json:"features"`
}

// writeCapabilities writes /etc/fostrom/capabilities.json: the inventory, from
// the kernel config, and the features.
func writeCapabilities(e hookEnv, t target) error {
	config, err := os.ReadFile(e.toImages("kernel.config"))
	if err != nil {
		return err
	}
	built := map[string]bool{}
	for line := range strings.SplitSeq(string(config), "\n") {
		option, value, _ := strings.Cut(line, "=")
		built[option] = value == "y" || value == "m"
	}
	system := []string{}
	for _, c := range inventory {
		if !slices.ContainsFunc(c.options, func(o string) bool { return !built[o] }) {
			system = append(system, c.name)
		}
	}
	version, err := os.ReadFile(e.fromSystem("VERSION"))
	if err != nil {
		return err
	}

	doc := capabilities{
		Schema:        2,
		Target:        t.name,
		SystemVersion: strings.TrimSpace(string(version)),
		Arch:          bundleArch[t.arch],
		System:        system,
		Features:      features,
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(e.target, "etc/fostrom/capabilities.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}
