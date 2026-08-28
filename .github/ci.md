# CI

| Event | Workflow | What it does |
| --- | --- | --- |
| Pull request into `dev` or `main` | [Test](workflows/test.yml) | `go test ./...` |
| Push to `main` (merge) | [Build](workflows/build.yml) | Linux / macOS / Windows binaries as Actions artifacts |

No repository secrets are required. `GITHUB_TOKEN` is enough.

## GitHub repository setup

Remote: `git@github.com:S-404/ladno.git`

1. **Branches.** Keep `main` as the default branch and `dev` as the integration branch.
2. **Actions.** GitHub → **Settings** → **Actions** → **General**:
   - Actions permissions: **Allow all actions and reusable workflows** (or at least GitHub-owned actions).
   - Workflow permissions: **Read repository contents and packages permissions** is enough.
3. **Branch protection** (Settings → **Rules** → **Rulesets**, or classic **Branches**):

   **`main`**
   - Restrict updates: no direct pushes; changes only via pull request.
   - Require a pull request before merging.
   - Require status checks to pass: **test**.
   - Optionally require the branch to be up to date before merging.

   **`dev`**
   - Require a pull request before merging.
   - Require status checks to pass: **test**.

   The **test** check appears in the list after the Test workflow has run at least once.

4. **Flow.** Feature branch → PR into `dev` (tests) → PR from `dev` into `main` (tests, then build artifacts on merge).
5. **Artifacts.** After a merge to `main`, open the **Build** run under **Actions** and download `ladno-linux-0.0.1` (version comes from `internal/buildinfo/VERSION`).

## Version

Current version is in [`internal/buildinfo/VERSION`](../internal/buildinfo/VERSION). It is compiled into the app (window title and Settings → General) and used in CI artifact names.

Bump it in the PR that should become the next `main` build:

| Change | Example |
| --- | --- |
| Bugfix | `0.0.1` → `0.0.2` |
| Feature | `0.0.1` → `0.1.0` |
| Breaking | `0.1.0` → `1.0.0` |

Git tags are optional. Use them if you want a named point on `main` (`git tag v0.0.1 && git push origin v0.0.1`). CI does not require tags.
