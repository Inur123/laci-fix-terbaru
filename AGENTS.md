# Agent Instructions: Laci Digital (laci-fix-terbaru)

Welcome! Whenever you are asked by the user to **"push ke git"**, **"commit dan push"**, or **"buat release"**, you MUST follow the standardized professional Git workflow below. DO NOT make bare ("polosan") commits.

---

## Mandatory Git Workflow for AI Assistants

When the user instructs you to push to git:

1. **Verify Integrity First**:
   - Run `npm --prefix fe run typecheck` to verify frontend TypeScript compilation.
   - Run `cd be && go test ./...` to verify backend tests.
   - If tests or typecheck fail, fix them before pushing.

2. **Determine Semantic Version (SemVer)**:
   - Check the current version in `fe/package.json` and `CHANGELOG.md`.
   - Determine increment:
     - `PATCH` (e.g. `1.1.0` -> `1.1.1`) for bug fixes, styling adjustments, or minor tweaks.
     - `MINOR` (e.g. `1.1.0` -> `1.2.0`) for new features, new pages, or significant enhancements.
     - `MAJOR` (e.g. `1.0.0` -> `2.0.0`) for breaking changes.
   - Update `"version"` in `fe/package.json`.

3. **Update `CHANGELOG.md`**:
   - Add a new section `## [vX.Y.Z] - YYYY-MM-DD` following [Keep a Changelog](https://keepachangelog.com/).
   - Categorize items into `### Added`, `### Changed`, `### Fixed`.

4. **Craft a Conventional Commit Message**:
   - Format: `<type>(<scope>): <summary> [vX.Y.Z]`
   - Types: `feat`, `fix`, `perf`, `refactor`, `docs`, `style`, `chore`.
   - Include bulleted details explaining what changed.
   - Example:
     ```
     feat(ui): standardize button designs and center search icons [v1.1.0]

     - Uniform action buttons to Upload and Cancel across all forms
     - Center Search icons vertically across all pages
     - Add public agenda detail page with national holiday integration
     - Migration 003_add_asal_pimpinan.sql for flexible leadership attribution
     ```

5. **Commit, Tag, and Push**:
   - `git add .`
   - `git commit -m "<structured commit message>"`
   - Create annotated tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z: <summary>"`
   - Push branch and tags: `git push origin <branch> --tags`

6. **Confirm to User**:
   - Report the committed version, tag created, and files pushed.
