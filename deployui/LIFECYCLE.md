# Managed deployment UX

`rodeploy` first shows existing work when there is any; select a row with arrows
and Enter, then Enter opens the action menu. The first row prepares a new FAR.
Existing Upload, Artifacts and Rollouts shortcuts remain available.

A conflict opens the blocking rollout with actions to inspect it, or cancel its
remaining targets and return to the saved new-deployment confirmation. It keeps
the selected artifact and target list. Releasing a reservation is confirmed by
the server before another Create is offered. Cancel does not roll back installed
versions or kill an executing process.

The creating CLI opens a DeploymentManagement session, heartbeats every 10s and
retries a failed heartbeat after 1s. Jupiter closes ownership after 20s without a
received heartbeat. The credential survives gRPC reconnects and screen changes
in the same CLI process; it is not written to logs or public rollout snapshots.
A new CLI is an observer, not a silent ownership takeover. It discovers saved
work through the startup screen even after a hard process exit.

Normal exit sends Detach with a bounded timeout; hard exits and lost Detach
messages are covered by heartbeat expiry. Unstarted work is cancelled. Running
work finishes and its success/failure is preserved. The TUI displays cancellation
intent, per-server results, last/next server checks and connection loss separately.
Create retries an uncertain response with the identical request ID and credential.
Explicit rejection is never described as accepted.

New managed submissions require an updated Jupiter and all selected Junos with
cancellation support. Existing legacy HTTP execution remains explicitly available
through --legacy and does not claim the new owner lifecycle guarantees.

JSON create emits one JSON snapshot per line and stays alive to own the rollout
until it finishes. WAITING can be approved with an authorized second CLI using
`act`; interrupting the creating CLI requests cleanup. Upload/list/watch/act
commands alone are observers and do not acquire ownership.

## Build and verification

Juno, Jupiter and fatima-cmd use the released fatima-core/v2 v2.0.0 and fatima-opm v1.0.0 modules without
a local replace. Standalone builds do not require a sibling fatima-core checkout.

From fatima-cmd:

    go test ./...
    cd integration
    go test ./...

The nested integration module uses sibling jupiter, juno and this CLI, together
with the released fatima-core/v2 v2.0.0 and fatima-opm v1.0.0 modules. It starts real TCP/gRPC servers, uploads a valid test FAR, validates scoped
Jupiter tickets in Juno and checks owner/observer exit, partial success/failure,
no subsequent target execution and starting a new rollout after cleanup. Test
executors do not touch production programs.

The heartbeat test intentionally waits for the production 10s interval, injects
a failed heartbeat, and checks reconnect with the same credential and one Detach.

## Between-server countdown

Jupiter reports `next_target_start_at` (Unix milliseconds). The existing one-second TUI tick renders the remaining time and next server; after the deadline it shows that start confirmation is pending. Cancellation and terminal states hide the countdown. JSON snapshots expose the same field. The server controls execution timing; the displayed countdown uses the CLI clock.
