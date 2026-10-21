# system builds for Fostrom OS

This repo builds the base system for Fostrom OS.
The kernel comes from [fostrom/kernel-builds](https://github.com/fostrom/kernel-builds).

## Building

Requirements: Docker and [just](https://just.systems)

```bash
# build all targets:
just build

# build one target:
just build <amd64 | arm64 | rpi>
```

Outputs are in `.release/`

## License

Apache 2.0. The packages in a release are under their own licenses.
