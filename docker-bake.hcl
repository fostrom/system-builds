variable "COMMIT" { default = "unknown" }
variable "SOURCE_DATE_EPOCH" { default = "0" }

group "default" {
  targets = ["amd64", "arm64", "rpi"]
}

target "system" {
  target = "release"
  output = ["type=local,dest=.release"]
  args = {
    COMMIT            = COMMIT
    SOURCE_DATE_EPOCH = SOURCE_DATE_EPOCH
  }
}

target "amd64" {
  inherits = ["system"]
  args = {
    TARGET = "amd64"
  }
}

target "arm64" {
  inherits = ["system"]
  args = {
    TARGET = "arm64"
  }
}

target "rpi" {
  inherits = ["system"]
  args = {
    TARGET = "rpi"
  }
}
