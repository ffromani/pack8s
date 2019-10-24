# pack8s - a tiny tool which helps managing the containerized clusters. 

`pack8s` is meant to be a [podman compatible](https://podman.io/) drop-in replacement for [gocli](https://github.com/kubevirt/kubevirtci/tree/master/cluster-provision#using-gocli).

## motivation
We want to add podman support to [kubevirtci](https://github.com/kubevirt/kubevirtci), but changing
`gocli` makes not much sense since the tool is simple and very stable. The quicker and safer way is
to provide a drop-in podman-compatible replacement, hence `pack8s`

## license
Apache v2 (same as `kubevirtci`)

## build
just run
```bash
make
```

Or fetch one of the releases.

## how to try it out?
1. make sure your `kubevirtci` checkout includes [this PR](https://github.com/kubevirt/kubevirtci/pull/168).
2. build (see above) `pack8s` and put it anywhere on your PATH
3. use kubevirtci as usual (`make cluster-up`, `make cluster-down`...)

## container image
No available. `pack8s` is meant to be a single, self contained, statically linked executable, so benefits of a container image are unclear.
Contributions welcome, though.
