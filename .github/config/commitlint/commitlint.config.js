// Dependency bots (Dependabot, MintMaker/Renovate) write a conventional subject
// but a body full of release-note links, which trips body-max-line-length. Their
// messages are machine-generated and cannot be reworded, so skip them entirely.
// Both bots sign off, so `[bot]` in the message distinguishes them from a human
// who happens to use the deps scope -- those stay linted.
const isBotDependencyBump = (message) =>
  /^(build|chore|ci|docs|feat|fix|refactor|test)\(deps(-dev)?\)!?: /.test(message) &&
  /\[bot\]/.test(message)

module.exports = {
  extends: ['@commitlint/config-conventional'],
  ignores: [isBotDependencyBump],
}
