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
