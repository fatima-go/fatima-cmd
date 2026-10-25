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

## rodeploy profile assembly

A FAR built by gofar with a project config (`.gofar/gofar.yaml`) records a per-binary install policy (`binaries`) in `deployment.json`. Jupiter reports its environment profile (`FATIMA_PROFILE`) on login, and `rodeploy` (both the default v2 flow and `--legacy`) removes the binaries that profile does not install before uploading. One Jupiter is assumed to serve one profile.

- The original FAR is never changed, so the same FAR can be deployed to several environments. The assembled copy is created under `$TMPDIR/rodeploy-*` and removed after the upload (v2) or deploy request (legacy).
- The assembled FAR keeps the original file name and every other entry byte-for-byte, and adds `assembly` (profile, removed binaries, source SHA-256) to `deployment.json`. Assembling the same FAR for the same profile always produces identical bytes, so `--request-id` retries remain valid.
- An already assembled FAR is refused for a different profile; deploy the original FAR instead.
- The original FAR is uploaded unchanged when it has no `binaries`, when the profile installs every binary, or when Jupiter does not report a profile (a warning is shown when the FAR has a profile-dependent policy).

## roclip binary transfer

`roclip` prints the juno clipboard. `roclip -b FILE` downloads `FILE` from the juno data folder into the current directory, and `roclip -u PATH` uploads a local file to the juno data folder under its base name (up to 1 GiB, any role). An upload never replaces an existing file unless `-f` is given; interrupted transfers leave the existing file intact. Juno versions without upload support are reported as such.

## lccrypto

`lccrypto enc VALUE...` encrypts with `native` by default; use `-s b64` for Base64 encoding. Quote values containing spaces. Leading/trailing whitespace and empty values are preserved.

```sh
lccrypto enc 'hello'
lccrypto enc -s b64 'hello'
lccrypto enc -f ./secret.txt -o ./secret.enc
lccrypto dec -f ./secret.enc -o ./restored.txt
```

Both commands accept `-f/--file PATH` instead of positional values and `-o/--output PATH` to save only the result without displaying plaintext or ciphertext. `enc -f` reads the whole file as one value, including all whitespace and final newlines. Encrypted output contains one value per line, without an extra final newline; `dec -f` reads this format (LF or CRLF, with an optional final line ending). `dec -o` accepts exactly one encrypted value and writes the original bytes without adding a newline.

Output files are always newly created with owner-only permissions (`0600`); existing files, including symbolic links, are rejected. There is no overwrite or append mode. Failed commands report errors on stderr and exit nonzero.

`native` retains compatibility with Fatima's existing AES-CBC format. Encryption and decryption require the same `FATIMA_PROFILE` (case-insensitive; defaults to `local`). This legacy format derives its key and fixed IV from the profile and has no authentication; it cannot reliably detect every wrong profile or altered ciphertext. Base64 provides encoding only, without confidentiality. This command validates native block lengths and padding locally; other applications using `fatima-core`'s `ResolveSecret` are unchanged.
