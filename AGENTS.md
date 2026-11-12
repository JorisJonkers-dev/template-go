# Agents

Guidance for any coding agent working in this repository. It is the estate's Go service template.

The repo's agent contract is [CLAUDE.md](CLAUDE.md); the estate-wide conventions are at https://github.com/JorisJonkers-dev/workspace/blob/main/CLAUDE.md.

## Build and test

Tasks are defined in `Taskfile.yml`; `task --list` shows them. Tool versions are pinned in `mise.toml`.

- `task check` runs lint, generation check, unit and integration tests, build and the secrets scan
- `task test` and `task test:integration` run the tests alone
- `task gen` regenerates the generated code
