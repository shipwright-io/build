<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

`waiter`: Wait Until Condition or Timeout
-----------------------------------------

In a nutshell, it waits until the lock-file is removed. When starting the application, a lock-file is created. When the file is removed, the `waiter` stops gracefully. When timeout is reached, the application exits on error.

## Usage

Please consider `--help` to see the possible flags, the possible sub-commands are:

```sh
waiter start
```

And:

```sh
waiter done
```