#!/usr/bin/env python
import os
import shutil
import argparse
import mod_util

# [goos, goarch, target_name, archive_fmt]
ALL_TARGETS = [
    ["windows", "amd64", "windows-x86_64", "zip"],
    ["windows", "arm64", "windows-arm64", "zip"],
    ["darwin", "amd64", "macos-x86_64", "gztar"],
    ["darwin", "arm64", "macos-arm64", "gztar"],
    ["linux", "amd64", "linux-x86_64", "gztar"],
    ["linux", "arm64", "linux-arm64", "gztar"],
    ["linux", "386", "linux-i386", "gztar"],
    ["freebsd", "amd64", "freebsd-x86_64", "gztar"],
    ["freebsd", "arm64", "freebsd-arm64", "gztar"],
    ["freebsd", "386", "freebsd-i386", "gztar"],
]

mod_util.cd_root()

arg_parser = argparse.ArgumentParser()
arg_parser.add_argument("--tidy", action="store_true", help="Run `go mod tidy` before building")
arg_parser.add_argument("--locked", action="store_true", help="Use locked package deps from `vendor/`")
arg_parser.add_argument("--ref", default="", help="Ref tag for CD (if empty, VERSION file is used)")

args = arg_parser.parse_args()

ref_tag = args.ref
if ref_tag == "":
    with open("VERSION", "r") as f:
        ref_tag = f.read()

ref_tag = ref_tag.replace("/", "_")

################################################################################

if args.tidy:
    mod_util.run(["go", "mod", "tidy"])

go_params = ["go", "build", "-v"]
if args.locked:
    go_params += ["-mod", "vendor"]

for t in ALL_TARGETS:
    goos, goarch, target_name, archive_fmt = t[0], t[1], t[2], t[3]
    env = { "GOOS": goos, "GOARCH": goarch }

    target_dir = f"target/{target_name}/"
    os.makedirs(target_dir, exist_ok=True)

    print(f"> Building for target {target_name}")
    mod_util.run(go_params + ["-o", target_dir, "."], env)
    print()

    print(f"> Creating archive for \"{target_dir}\".. ")
    shutil.make_archive(f"artifacts/vince-{ref_tag}-{target_name}", format=archive_fmt, root_dir=target_dir, base_dir="")
    print()
