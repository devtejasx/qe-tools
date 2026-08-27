module.exports = {
  extends: ['@commitlint/config-conventional'],
  // Dependency bots write a conventional header with a `deps` scope, but their
  // bodies carry release notes and changelog excerpts that run well past
  // body-max-line-length, so every one of their PRs fails the check. Skip those
  // commits wholesale.
  //
  // Only the header is tested: `ignores` receives the whole message, so an
  // unanchored match would also skip a hand-written commit that happened to
  // quote a bot subject somewhere in its body.
  ignores: [(message) => /^\w+\(deps(-dev)?\)!?: /.test(message.split('\n')[0])],
}
