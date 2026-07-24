// Commitlint configuration.
//
// This is JS (not YAML) so we can use an `ignores` function — a capability that
// only exists in the JS config format. It skips Dependabot's own commits: its
// group-update bodies list every bumped package on a single line and exceed
// body-max-line-length, and bot-authored messages can't be rewritten. Human
// commits still get the full ruleset below.
module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'body-max-line-length': [2, 'always', 200],
    'type-enum': [
      2,
      'always',
      ['chore', 'ci', 'docs', 'feat', 'fix', 'perf', 'refactor', 'revert', 'style', 'test', 'theme'],
    ],
  },
  // Every Dependabot commit carries this trailer; skip linting those.
  // `defaultIgnores` (merges, reverts, …) still applies alongside this.
  ignores: [(message) => /Signed-off-by: dependabot\[bot\]/.test(message)],
};
