# CI

| Event | Workflow | What it does |
| --- | --- | --- |
| Pull request into `dev` or `main` | [Test](workflows/test.yml) | `go test ./...` |
| Push of tag `v*` (e.g. `v0.0.1`) | [Release](workflows/build.yml) | Build Linux / macOS / Windows, create a GitHub Release with changelog and binaries |

No extra secrets. `GITHUB_TOKEN` is enough. Workflow permissions must allow **Read and write** (Settings → Actions → General) so the Release job can publish.

## GitHub repository setup

Remote: `git@github.com:S-404/ladno.git`

1. **Branches.** Keep `main` as the default branch and `dev` as the integration branch.
2. **Actions.** GitHub → **Settings** → **Actions** → **General**:
   - Actions permissions: **Allow all actions and reusable workflows** (or at least GitHub-owned actions).
   - Workflow permissions: **Read and write** (needed to create Releases).
3. **Branch protection** (Settings → **Rules** → **Rulesets**, or classic **Branches**):

   **`main`**
   - Restrict updates: no direct pushes; changes only via pull request.
   - Require a pull request before merging.
   - Require status checks to pass: **test**.

   **`dev`**
   - Require a pull request before merging.
   - Require status checks to pass: **test**.

4. **Flow.** Feature branch → PR into `dev` (tests) → PR from `dev` into `main` (tests). Then tag `main` to publish a Release.

## Version and release

Current version is in [`internal/buildinfo/VERSION`](../internal/buildinfo/VERSION). It is compiled into the app (window title and Settings → General). The git tag must match: file `0.0.1` → tag `v0.0.1`.

Bump `VERSION` in the PR that should become the next release:

| Change | Example |
| --- | --- |
| Bugfix | `0.0.1` → `0.0.2` |
| Feature | `0.0.1` → `0.1.0` |
| Breaking | `0.1.0` → `1.0.0` |

After that PR is on `main`:

```bash
git checkout main
git pull
git tag v0.0.1
git push origin v0.0.1
```

The Release workflow builds three binaries, opens a GitHub Release, attaches `ladno-0.0.1-linux`, `ladno-0.0.1-macos`, `ladno-0.0.1-windows.exe`, and fills notes from PRs merged since the previous tag (`gh release create --generate-notes`).
