package main

import "fmt"

// What differs between the targets. The rest of a system is the same for all.
type target struct {
	name      string // amd64, arm64 or rpi
	boot      string // grub on amd64 and arm64, rpi on every 64 bit Raspberry Pi
	platform  string // the platform an upgrade checks the device against
	arch      string // the architecture an upgrade checks the device against
	signature string // the MBR signature, which root=PARTUUID= names
	image     string // the kernel image in the kernel-builds archive
	console   string // console arguments for the kernel, on GRUB targets
	efi       string // GRUB's EFI image, on GRUB targets
}

var targets = map[string]target{
	// x86-64 machines that boot through UEFI.
	"amd64": {
		boot:      "grub",
		platform:  "fostrom-amd64",
		arch:      "x86_64",
		signature: "04030201",
		image:     "bzImage",
		console:   "console=tty1 console=ttyS0",
		efi:       "bootx64.efi",
	},

	// arm64 servers and SystemReady or EBBR boards, booted through UEFI.
	"arm64": {
		boot:      "grub",
		platform:  "fostrom-arm64",
		arch:      "aarch64",
		signature: "46524f41",
		image:     "Image",
		efi:       "bootaa64.efi",
	},

	// Every 64 bit Raspberry Pi.
	"rpi": {
		boot:      "rpi",
		platform:  "fostrom-rpi-arm64",
		arch:      "aarch64",
		signature: "46524f53",
		image:     "Image",
	},
}

func lookup(name string) (target, error) {
	t, ok := targets[name]
	if !ok {
		return t, fmt.Errorf("no target %q: amd64, arm64 or rpi", name)
	}
	t.name = name
	return t, nil
}
