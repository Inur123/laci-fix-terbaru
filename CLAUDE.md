# Claude Code Instructions for Laci Digital

## Git Push Protocol
Whenever the user asks to push changes to git ("push ke git", "push ke repo"):
1. Verification:
   - `npm --prefix fe run typecheck`
   - `cd be && go test ./...`
2. Version Bump:
   - Update `fe/package.json` version following SemVer (`vX.Y.Z`).
   - Add release entry in `CHANGELOG.md` adhering to Keep a Changelog.
3. Commit & Tag:
   - Use Conventional Commits (`<type>(<scope>): <message> [vX.Y.Z]`).
   - Create annotated tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z: <message>"`.
4. Push:
   - `git push origin main --tags`
