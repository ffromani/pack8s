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
2. build (see above) `pack8s` and put it anywhere on your PATH.
3. set up your box as [described in this blog post](https://podman.io/blogs/2019/01/16/podman-varlink.html) or see `local box setup` below.
4. use kubevirtci as usual (`make cluster-up`, `make cluster-down`...).

## local box setup
Excerpt taken from [this blog post](https://podman.io/blogs/2019/01/16/podman-varlink.html)

### Set up Podman on the Fedora/RHEL machine
```bash
$ sudo yum install podman libvarlink-util
$ sudo groupadd podman
```

Copy /lib/tmpfiles.d/podman.conf to /etc/tmpfiles.d/podman.conf.
```
$ sudo cp /lib/tmpfiles.d/podman.conf /etc/tmpfiles.d/podman.conf
```
Edit `/etc/tmpfiles.d/podman.conf` to read like:
```
d /run/podman 0750 root podman
```
Copy /lib/systemd/system/io.podman.socket to /etc/systemd/system/io.podman.socket.
```
$ sudo cp /lib/systemd/system/io.podman.socket /etc/systemd/system/io.podman.socket
```
Edit section [Socket] of `/etc/systemd/system/io.podman.socket` to read like:
```
[Socket]
ListenStream=/run/podman/io.podman
SocketMode=0660
SocketGroup=podman
```
Then activate the changes:
```bash
$ sudo systemctl daemon-reload
$ sudo systemd-tmpfiles --create
$ sudo systemctl enable --now io.podman.socket
```
The directory and socket now belongs to the podman group
```bash
$ sudo ls -al /run/podman
drwxr-x---.  2 root podman   60 14. Jan 14:50 .
drwxr-xr-x. 51 root root   1420 14. Jan 14:36 ..
srw-rw----.  1 root podman    0 14. Jan 14:50 io.podman
```

## container image
No available. `pack8s` is meant to be a single, self contained, statically linked executable, so benefits of a container image are unclear.
Contributions welcome, though.
