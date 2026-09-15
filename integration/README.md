# Four-repository lifecycle integration tests

Place fatima-core, juno, jupiter and fatima-cmd next to each other. From this
folder, run `go test ./...` (or `go test -race ./...`). No external server or
credentials are needed. All network listeners bind to localhost, storage is in
temporary test folders, and deployment executors are controlled test functions.

This module is separate so production CLI dependencies do not import the server
implementations. Its replace directives deliberately resolve the four checkouts.
