# OAuth 2.0 token client

`auth/oauth2` implements the reusable protocol portion of a confidential OAuth
client. It exchanges authorization codes and refresh tokens through a configured
token endpoint using `client_secret_basic` or `client_secret_post`.

HTTP routes, cookies, sessions, CORS, and application-specific redirects remain
the responsibility of the consuming service.

```go
client, err := oauth2.NewClient(oauth2.Config{
    TokenEndpoint:       "https://identity.example/accounts/oauth/token",
    ClientID:            os.Getenv("CLIENT_ID"),
    ClientSecret:        os.Getenv("CLIENT_SECRET"),
    ClientAuthentication: oauth2.ClientSecretPost,
})

tokens, err := client.ExchangeAuthorizationCode(ctx, oauth2.AuthorizationCodeRequest{
    Code:        code,
    RedirectURI: "https://app.example/oauth/login",
})
```

`RedirectURI` is optional. When omitted with `ClientSecretBasic`, the
authorization-code request body contains only `grant_type` and `code`. When it
is provided, it must be an absolute HTTP(S) URL and is included in the request.

`ClientSecretBasic` is the backward-compatible default. Use `ClientSecretPost`
when required by the registered OIDC client; this method sends `client_id` and
`client_secret` in the form body. For Identity V1 refresh compatibility
with TNF 8.4, add `oauth2.WithRefreshClientCredentialsInBody()` to `NewClient`.
This keeps the Basic authorization header and also sends both credentials in
the refresh form body.

The client limits response bodies, never includes client secrets or token
values in errors, and preserves the previous refresh token when the provider
does not rotate it.
