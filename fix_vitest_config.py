import re

with open("packages/tui/package.json", "r") as f:
    content = f.read()

# Let's completely remove the test script from tui to avoid CI complaining about `node --test --import tsx test/*.test.ts` entirely.
# No wait, the error is `npm error command sh -c node --test --import tsx test/*.test.ts`. This means the `pi-tui` test script is explicitly failing because of something. Maybe another file? Let's find out what test is failing in tui.

# Wait, the log says:
# ok 25 - wrapTextWithAnsi
# # tests 444
# # suites 86
# # pass 444
# # fail 0
# # cancelled 0
# # skipped 0
# # todo 0
# # duration_ms 3902.656135

# Wait! The TUI tests completely passed!
# But the npm test script still fails with exit code 1. Why?
# "npm error command sh -c node --test --import tsx test/*.test.ts"
# Node test runner might return a non-zero exit code if it couldn't find tests in one of the glob matched files or if there's a compilation error in one of the *other* files.
