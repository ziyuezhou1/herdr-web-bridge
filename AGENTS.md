# Repository Guidelines

## Project Structure & Module Organization

This checkout is currently an empty scaffold: no source files, manifests, tests, or assets are present. As implementation begins, use `src/` for production bridge and web code, `tests/` for automated tests, `assets/` for static files, and `scripts/` for repeatable developer utilities. Mirror source boundaries in tests—for example, `src/bridge/client.ts` should be covered by `tests/bridge/client.test.ts`. Keep the repository root limited to configuration, package manifests, environment examples, and top-level documentation.

## Build, Test, and Development Commands

No package manager or build system is configured yet, so there are currently no runnable project commands. When a Node-based toolchain is introduced, provide a stable root-level command interface and document any deviations:

- `npm install` installs the locked dependency set.
- `npm run dev` starts the local development server with reloads.
- `npm test` runs the complete automated test suite.
- `npm run lint` checks formatting and static-analysis rules.
- `npm run build` creates a production artifact.

Commit the lockfile produced by the selected package manager; do not mix npm, pnpm, and Yarn lockfiles.

## Coding Style & Naming Conventions

For JavaScript or TypeScript, use two-space indentation and let the configured formatter own whitespace. Adopt ESLint and Prettier with the first implementation. Use `camelCase` for variables and functions, `PascalCase` for classes and UI components, `kebab-case` for general filenames, and `UPPER_SNAKE_CASE` for environment variables. Keep protocol and transport logic separate from UI concerns, and handle asynchronous failures explicitly.

## Testing Guidelines

No testing framework or coverage threshold exists yet. Add a runner alongside the first feature and name tests `*.test.ts` or `*.test.js`. Cover successful requests, invalid input, connection failures, timeouts, and serialization boundaries. Every bug fix should include a regression test. Run the full suite before opening a pull request.

## Commit & Pull Request Guidelines

This checkout has no Git history from which to infer an established convention. Until one is documented, use concise Conventional Commit subjects such as `feat: add bridge handshake` or `fix: reject malformed payloads`. Pull requests should explain the problem and approach, link relevant issues, list validation performed, and include screenshots for visible UI changes. Keep changes focused and call out configuration or compatibility impacts.

## Security & Configuration

Never commit credentials, tokens, private endpoints, or populated `.env` files. Provide redacted defaults in `.env.example`, document required ports and permissions, and validate all data crossing the web/bridge boundary.
