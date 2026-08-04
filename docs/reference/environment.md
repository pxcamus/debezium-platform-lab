# Environment variables

Configuration is entirely environment-driven. Values are read from a local environment file
first, then from the repository's shared version pins as a fallback, so a fresh checkout
runs without any local file and your own file only needs to carry secrets and host-specific
overrides.

!!! warning "Placeholder"

    This page is a skeleton.

Topics to cover: the full variable table with defaults and consumers; which file each kind
of value belongs in; and the precedence rule — including that variables already exported in
your shell win over both files, which is a common source of confusion (see
[environment shadowing](../troubleshooting/index.md#env-shadowing)).

<!-- TODO: generate the variable table from the source so it cannot drift. -->
