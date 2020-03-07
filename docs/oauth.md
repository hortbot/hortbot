# OAuth flow

The bot must be authorized to perform certain actions, such as updating a stream title or game.
Authoriziation must be obtained from the user, and maintained over time. This is done using an
OAuth 2 flow.

1. The user starts at the website, accessing an auth page.
1. The HTTP server generates a random UUID, and stores it in redis.
   This UUID is used to generate an auth link on the server (as its state),
   and the user is redirected to it. Redis stores some extra metadata, including
   the host that's handling the request.
1. The user authenticates on Twitch.
1. Twitch redirects back to the callback endpoint on the website with the state and a code.
1. The server verifies the state by checking redis, removing it atomically.
    1. If the host of the request does not match the host of the state, then the state is
       stored back into redis, and the request is redirected to the same URL at the correct
       host.
1. The server exchanges the code for a token with Twitch.
1. The server queries Twitch using the token to ask which user authenticated.
1. The server stores the token and the user ID into the Postgres database.

The flow for authenticating the bot user itself is the same, but via another endpoint with more
permissions.

When the bot needs to make a request, it fetches the token it needs from the database, then builds
an HTTP client. This HTTP client will automatically refresh the token if needed, and return the new
token for later storage.
