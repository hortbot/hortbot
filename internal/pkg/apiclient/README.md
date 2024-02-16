## Adding a new API package

1. Create a new package for the API.
2. Define an interface and an implementation.
3. Add a moq go:generate directive.
4. Add to bot.Config and bot.sharedDeps.
5. Add fake to bot_test.scriptTester, and set scriptTester.bc with the fake.
6. Add directives for specifying the output of the API based on the parameters.
7. Add the real API to main.go.
