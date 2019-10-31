# pack8s FAQs

### How do you pronounce "pack8s"?
It is pronounced "pak-eight-z", much like "packets".

### Why do you use the varlink API and not libpod directly?
No strong reasons. When we started experimenting, we just started with varlink.
Using or switching to libpod [is totally feasible](https://github.com/fromanirh/pack8s/issues/14), but at this point
in time we need to evaluate if it is better to fix (or wait for a fix) the outstanding varlink issues instead of doing a rewrite.

### Why doesn't `pack8s` do $SOMETHING
We are after a `gocli` replacement, we don't aim to provide a generic tool, let another a `podman` replacement.
If we added subcommands or options with respect to `gocli`, it is because it allowed us to test things more easily or for
developer convenience.

