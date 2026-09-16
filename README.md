# fatima-cmd #

provide useful cli commands in $FATIMA_HOME/bin<br>
process with "ro" prefix means remote operating and "lc" means local.
## lcproc interactive guide

Run `lcproc` in a terminal to choose a local process, inspect/switch revisions, or duplicate it under a new name. `FATIMA_HOME` selects the local environment; no Jupiter connection is required.

- Arrow keys and Enter select; Esc goes back; q exits (Ctrl+C while entering a name).
- `lcproc sample version` opens revisions; `lcproc sample dup copy` pre-fills the duplicate name.
- Changes require an explicit confirmation. Revision switching requires the process to be stopped and does not start it automatically.
- Duplication copies matching executable/configuration files, without subdirectories, and requires subsequent registration using `roproc add`.
- Use `lcproc --plain sample version` for the existing text workflow. Non-terminal invocation also retains that workflow.

## Package selection

For `rostart`, `rostop`, `roproc`, `rocron`, `rolog`, `rohis`, `rodis`, `roclip`, and `roclric`, an explicit `-p host:package` selects that package. Without `-p`, Jupiter first resolves the package matching the client IP. A unique match is used automatically; multiple matching packages are offered for selection. If no IP matches, the only registered package is selected automatically, or the full package list is offered when several are registered. No registered packages produces guidance to check `ropack`.

The same selection applies before reports such as `rocron -l`, `rohis PROCESS`, and direct `rolog PROCESS LEVEL` changes, as well as the legacy HTTP paths. Process/group/action arguments remain intact after selection. `--plain`, `--json` (where supported), and non-terminal execution never prompt: IP-based automatic selection still applies; specify `-p` only when the target remains ambiguous.

`ropack` continues to show all packages, and `rodeploy` retains its deployment target selection flow.

`lcproc` treats selecting the currently linked revision as an informational no-op, without requesting a stop or rewriting the link. This also applies to revision arguments and `--plain`.

## rodeploy startup

`rodeploy` and `rodeploy FILE.far` start on Upload. Existing deployment history is checked in the background and shown as a count with an `l` shortcut, without switching screens or blocking FAR selection. Use `u` for Upload, `a` for uploaded artifacts, and `l` for existing deployments. The left column shows deployment progress; Tab switches list/detail focus. Explicit `rodeploy rollouts` and `rodeploy watch ID` still open their requested views.
