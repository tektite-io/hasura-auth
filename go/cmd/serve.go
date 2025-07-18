package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/nhost/hasura-auth/docs"
	"github.com/nhost/hasura-auth/go/api"
	"github.com/nhost/hasura-auth/go/controller"
	"github.com/nhost/hasura-auth/go/hibp"
	"github.com/nhost/hasura-auth/go/middleware"
	"github.com/nhost/hasura-auth/go/middleware/ratelimit"
	"github.com/nhost/hasura-auth/go/oidc"
	"github.com/nhost/hasura-auth/go/providers"
	"github.com/nhost/hasura-auth/go/sql"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/urfave/cli/v2"
)

const (
	flagAPIPrefix                        = "api-prefix"
	flagPort                             = "port"
	flagDebug                            = "debug"
	flagLogFormatTEXT                    = "log-format-text"
	flagTrustedProxies                   = "trusted-proxies"
	flagPostgresConnection               = "postgres"
	flagPostgresMigrationsConnection     = "postgres-migrations"
	flagDisableSignup                    = "disable-signup"
	flagConcealErrors                    = "conceal-errors"
	flagDefaultAllowedRoles              = "default-allowed-roles"
	flagDefaultRole                      = "default-role"
	flagDefaultLocale                    = "default-locale"
	flagAllowedLocales                   = "allowed-locales"
	flagDisableNewUsers                  = "disable-new-users"
	flagGravatarEnabled                  = "gravatar-enabled"
	flagGravatarDefault                  = "gravatar-default"
	flagGravatarRating                   = "gravatar-rating"
	flagRefreshTokenExpiresIn            = "refresh-token-expires-in"
	flagAccessTokensExpiresIn            = "access-tokens-expires-in"
	flagHasuraGraphqlJWTSecret           = "hasura-graphql-jwt-secret" //nolint:gosec
	flagEmailSigninEmailVerifiedRequired = "email-verification-required"
	flagSMTPHost                         = "smtp-host"
	flagSMTPPort                         = "smtp-port"
	flagSMTPSecure                       = "smtp-secure"
	flagSMTPUser                         = "smtp-user"
	flagSMTPPassword                     = "smtp-password"
	flagSMTPSender                       = "smtp-sender"
	flagSMTPAPIHedaer                    = "smtp-api-header"
	flagSMTPAuthMethod                   = "smtp-auth-method"
	flagClientURL                        = "client-url"
	flagServerURL                        = "server-url"
	flagAllowRedirectURLs                = "allow-redirect-urls"
	flagEnableChangeEnv                  = "enable-change-env"
	flagCustomClaims                     = "custom-claims"
	flagGraphqlURL                       = "graphql-url"
	flagHasuraAdminSecret                = "hasura-admin-secret" //nolint:gosec
	flagPasswordMinLength                = "password-min-length"
	flagPasswordHIBPEnabled              = "password-hibp-enabled"
	flagEmailTemplatesPath               = "templates-path"
	flagBlockedEmailDomains              = "block-email-domains"
	flagBlockedEmails                    = "block-emails"
	flagAllowedEmailDomains              = "allowed-email-domains"
	flagAllowedEmails                    = "allowed-emails"
	flagEmailPasswordlessEnabled         = "email-passwordless-enabled"
	flagRequireElevatedClaim             = "require-elevated-claim"
	flagWebauthnEnabled                  = "webauthn-enabled"
	flagWebauhtnRPName                   = "webauthn-rp-name"
	flagWebauthnRPID                     = "webauthn-rp-id"
	flagWebauthnRPOrigins                = "webauthn-rp-origins"
	flagWebauthnAttestationTimeout       = "webauthn-attestation-timeout"
	flagRateLimitEnable                  = "rate-limit-enable"
	flagRateLimitGlobalBurst             = "rate-limit-global-burst"
	flagRateLimitGlobalInterval          = "rate-limit-global-interval"
	flagRateLimitEmailBurst              = "rate-limit-email-burst"
	flagRateLimitEmailInterval           = "rate-limit-email-interval"
	flagRateLimitEmailIsGlobal           = "rate-limit-email-is-global"
	flagRateLimitSMSBurst                = "rate-limit-sms-burst"
	flagRateLimitSMSInterval             = "rate-limit-sms-interval"
	flagRateLimitBruteForceBurst         = "rate-limit-brute-force-burst"
	flagRateLimitBruteForceInterval      = "rate-limit-brute-force-interval"
	flagRateLimitSignupsBurst            = "rate-limit-signups-burst"
	flagRateLimitSignupsInterval         = "rate-limit-signups-interval"
	flagRateLimitMemcacheServer          = "rate-limit-memcache-server"
	flagRateLimitMemcachePrefix          = "rate-limit-memcache-prefix"
	flagTurnstileSecret                  = "turnstile-secret"
	flagAppleAudience                    = "apple-audience"
	flagGoogleAudience                   = "google-audience"
	flagOTPEmailEnabled                  = "otp-email-enabled"
	flagSMSPasswordlessEnabled           = "sms-passwordless-enabled"
	flagSMSTwilioAccountSid              = "sms-twilio-account-sid"
	flagSMSTwilioAuthToken               = "sms-twilio-auth-token" //nolint:gosec
	flagSMSTwilioMessagingServiceID      = "sms-twilio-messaging-service-id"
	flagAnonymousUsersEnabled            = "enable-anonymous-users"
	flagMfaEnabled                       = "mfa-enabled"
	flagMfaTotpIssuer                    = "mfa-totp-issuer"
	flagGithubEnabled                    = "github-enabled"
	flagGithubClientID                   = "github-client-id"
	flagGithubClientSecret               = "github-client-secret" //nolint:gosec
	flagGithubAuthorizationURL           = "github-authorization-url"
	flagGithubTokenURL                   = "github-token-url" //nolint:gosec
	flagGithubUserProfileURL             = "github-user-profile-url"
	flagGithubScope                      = "github-scope"
	flagGoogleEnabled                    = "google-enabled"
	flagGoogleClientID                   = "google-client-id"
	flagGoogleClientSecret               = "google-client-secret"
	flagGoogleScope                      = "google-scope"
	flagAppleEnabled                     = "apple-enabled"
	flagAppleClientID                    = "apple-client-id"
	flagAppleTeamID                      = "apple-team-id"
	flagAppleKeyID                       = "apple-key-id"
	flagApplePrivateKey                  = "apple-private-key"
	flagAppleScope                       = "apple-scope"
	flagLinkedInEnabled                  = "linkedin-enabled"
	flagLinkedInClientID                 = "linkedin-client-id"
	flagLinkedInClientSecret             = "linkedin-client-secret"
	flagLinkedInScope                    = "linkedin-scope"
	flagDiscordEnabled                   = "discord-enabled"
	flagDiscordClientID                  = "discord-client-id"
	flagDiscordClientSecret              = "discord-client-secret"
	flagDiscordScope                     = "discord-scope"
	flagSpotifyEnabled                   = "spotify-enabled"
	flagSpotifyClientID                  = "spotify-client-id"
	flagSpotifyClientSecret              = "spotify-client-secret" //nolint:gosec
	flagSpotifyScope                     = "spotify-scope"
	flagTwitchEnabled                    = "twitch-enabled"
	flagTwitchClientID                   = "twitch-client-id"
	flagTwitchClientSecret               = "twitch-client-secret"
	flagTwitchScope                      = "twitch-scope"
	flagGitlabEnabled                    = "gitlab-enabled"
	flagGitlabClientID                   = "gitlab-client-id"
	flagGitlabClientSecret               = "gitlab-client-secret" //nolint:gosec
	flagGitlabScope                      = "gitlab-scope"
	flagBitbucketEnabled                 = "bitbucket-enabled"
	flagBitbucketClientID                = "bitbucket-client-id"
	flagBitbucketClientSecret            = "bitbucket-client-secret"
	flagBitbucketScope                   = "bitbucket-scope"
	flagWorkosEnabled                    = "workos-enabled"
	flagWorkosClientID                   = "workos-client-id"
	flagWorkosClientSecret               = "workos-client-secret" //nolint:gosec
	flagWorkosDefaultOrganization        = "workos-default-organization"
	flagWorkosDefaultConnection          = "workos-default-connection"
	flagWorkosDefaultDomain              = "workos-default-domain"
	flagWorkosScope                      = "workos-scope"
	flagAzureadEnabled                   = "azuread-enabled"
	flagAzureadClientID                  = "azuread-client-id"
	flagAzureadClientSecret              = "azuread-client-secret" //nolint:gosec
	flagAzureadTenant                    = "azuread-tenant"
	flagAzureadScope                     = "azuread-scope"
	flagFacebookEnabled                  = "facebook-enabled"
	flagFacebookClientID                 = "facebook-client-id"
	flagFacebookClientSecret             = "facebook-client-secret"
	flagFacebookScope                    = "facebook-scope"
	flagWindowsliveEnabled               = "windowslive-enabled"
	flagWindowsliveClientID              = "windowslive-client-id"
	flagWindowsliveClientSecret          = "windowslive-client-secret"
	flagWindowsliveScope                 = "windowslive-scope"
	flagStravaEnabled                    = "strava-enabled"
	flagStravaClientID                   = "strava-client-id"
	flagStravaClientSecret               = "strava-client-secret" //nolint:gosec
	flagStravaScope                      = "strava-scope"
	flagTwitterEnabled                   = "twitter-enabled"
	flagTwitterConsumerKey               = "twitter-consumer-key"
	flagTwitterConsumerSecret            = "twitter-consumer-secret"
)

func CommandServe() *cli.Command { //nolint:funlen,maintidx
	return &cli.Command{ //nolint: exhaustruct
		Name:  "serve",
		Usage: "Serve the application",
		//nolint:lll
		Flags: []cli.Flag{
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagAPIPrefix,
				Usage:    "prefix for all routes",
				Value:    "",
				Category: "server",
				EnvVars:  []string{"AUTH_API_PREFIX"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagPort,
				Usage:    "Port to bind to",
				Value:    "4000",
				Category: "server",
				EnvVars:  []string{"AUTH_PORT"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagDebug,
				Usage:    "enable debug logging",
				Category: "general",
				EnvVars:  []string{"AUTH_DEBUG"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagLogFormatTEXT,
				Usage:    "format logs in plain text",
				Category: "general",
				EnvVars:  []string{"AUTH_LOG_FORMAT_TEXT"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagPostgresConnection,
				Usage:    "PostgreSQL connection URI: https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING",
				Value:    "postgres://postgres:postgres@localhost:5432/local?sslmode=disable",
				Category: "postgres",
				EnvVars:  []string{"POSTGRES_CONNECTION", "HASURA_GRAPHQL_DATABASE_URL"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagPostgresMigrationsConnection,
				Usage:    "PostgreSQL connection URI for running migrations: https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING. Required to inject the `auth` schema into the database. If not specied, the `postgres connection will be used",
				Category: "postgres",
				EnvVars:  []string{"POSTGRES_MIGRATIONS_CONNECTION"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagDisableSignup,
				Usage:    "If set to true, all signup methods will throw an unauthorized error",
				Value:    false,
				Category: "signup",
				EnvVars:  []string{"AUTH_DISABLE_SIGNUP"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagConcealErrors,
				Usage:    "Conceal sensitive error messages to avoid leaking information about user accounts to attackers",
				Value:    false,
				Category: "server",
				EnvVars:  []string{"AUTH_CONCEAL_ERRORS"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagDefaultAllowedRoles,
				Usage:    "Comma-separated list of default allowed user roles",
				Category: "signup",
				Value:    cli.NewStringSlice("me"),
				EnvVars:  []string{"AUTH_USER_DEFAULT_ALLOWED_ROLES"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagDefaultRole,
				Usage:    "Default user role for registered users",
				Category: "signup",
				Value:    "user",
				EnvVars:  []string{"AUTH_USER_DEFAULT_ROLE"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagDefaultLocale,
				Usage:    "Default locale",
				Category: "signup",
				Value:    "en",
				EnvVars:  []string{"AUTH_LOCALE_DEFAULT"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagAllowedLocales,
				Usage:    "Allowed locales",
				Category: "signup",
				Value:    cli.NewStringSlice("en"),
				EnvVars:  []string{"AUTH_LOCALE_ALLOWED_LOCALES"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagDisableNewUsers,
				Usage:    "If set, new users will be disabled after finishing registration and won't be able to sign in",
				Category: "signup",
				EnvVars:  []string{"AUTH_DISABLE_NEW_USERS"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagGravatarEnabled,
				Usage:    "Enable gravatar",
				Category: "signup",
				Value:    true,
				EnvVars:  []string{"AUTH_GRAVATAR_ENABLED"},
			},
			&cli.GenericFlag{ //nolint: exhaustruct
				Name: flagGravatarDefault,
				Value: &EnumValue{ //nolint: exhaustruct
					Enum: []string{
						"blank",
						"identicon",
						"monsterid",
						"wavatar",
						"retro",
						"robohash",
						"mp",
						"404",
					},
					Default: "blank",
				},
				Usage:    "Gravatar default",
				Category: "signup",
				EnvVars:  []string{"AUTH_GRAVATAR_DEFAULT"},
			},
			&cli.GenericFlag{ //nolint: exhaustruct
				Name: flagGravatarRating,
				Value: &EnumValue{ //nolint: exhaustruct
					Enum: []string{
						"g",
						"pg",
						"r",
						"x",
					},
					Default: "g",
				},
				Usage:    "Gravatar rating",
				Category: "signup",
				EnvVars:  []string{"AUTH_GRAVATAR_RATING"},
			},
			&cli.IntFlag{ //nolint: exhaustruct
				Name:     flagRefreshTokenExpiresIn,
				Usage:    "Refresh token expires in (seconds)",
				Value:    2592000, //nolint:mnd
				Category: "jwt",
				EnvVars:  []string{"AUTH_REFRESH_TOKEN_EXPIRES_IN"},
			},
			&cli.IntFlag{ //nolint: exhaustruct
				Name:     flagAccessTokensExpiresIn,
				Usage:    "Access tokens expires in (seconds)",
				Value:    900, //nolint:mnd
				Category: "jwt",
				EnvVars:  []string{"AUTH_ACCESS_TOKEN_EXPIRES_IN"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagHasuraGraphqlJWTSecret,
				Usage:    "Key used for generating JWTs. Must be `HMAC-SHA`-based and the same as configured in Hasura. More info: https://hasura.io/docs/latest/graphql/core/auth/authentication/jwt.html#running-with-jwt",
				Required: true,
				Category: "jwt",
				EnvVars:  []string{"HASURA_GRAPHQL_JWT_SECRET"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagEmailSigninEmailVerifiedRequired,
				Usage:    "Require email to be verified for email signin",
				Category: "signup",
				Value:    true,
				EnvVars:  []string{"AUTH_EMAIL_SIGNIN_EMAIL_VERIFIED_REQUIRED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagSMTPHost,
				Usage:    "SMTP Host. If the host is 'postmark' then the Postmark API will be used. Use AUTH_SMTP_PASS as the server token, other SMTP options are ignored",
				Category: "smtp",
				EnvVars:  []string{"AUTH_SMTP_HOST"},
			},
			&cli.UintFlag{ //nolint: exhaustruct
				Name:     flagSMTPPort,
				Usage:    "SMTP port",
				Category: "smtp",
				Value:    587, //nolint:mnd
				EnvVars:  []string{"AUTH_SMTP_PORT"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagSMTPSecure,
				Usage:    "Connect over TLS. Deprecated: It is recommended to use port 587 with STARTTLS instead of this option.",
				Category: "smtp",
				EnvVars:  []string{"AUTH_SMTP_SECURE"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagSMTPUser,
				Usage:    "SMTP user",
				Category: "smtp",
				EnvVars:  []string{"AUTH_SMTP_USER"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagSMTPPassword,
				Usage:    "SMTP password",
				Category: "smtp",
				EnvVars:  []string{"AUTH_SMTP_PASS"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagSMTPSender,
				Usage:    "SMTP sender",
				Category: "smtp",
				EnvVars:  []string{"AUTH_SMTP_SENDER"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagSMTPAPIHedaer,
				Usage:    "SMTP API Header. Maps to header X-SMTPAPI",
				Category: "smtp",
				EnvVars:  []string{"AUTH_SMTP_X_SMTPAPI_HEADER"},
			},
			&cli.GenericFlag{ //nolint: exhaustruct
				Name: flagSMTPAuthMethod,
				Value: &EnumValue{ //nolint: exhaustruct
					Enum: []string{
						"LOGIN",
						"PLAIN",
						"CRAM-MD5",
					},
					Default: "PLAIN",
				},
				Usage:    "SMTP Authentication method",
				Category: "smtp",
				EnvVars:  []string{"AUTH_SMTP_AUTH_METHOD"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagClientURL,
				Usage:    "URL of your frontend application. Used to redirect users to the right page once actions based on emails or OAuth succeed",
				Category: "application",
				EnvVars:  []string{"AUTH_CLIENT_URL"},
			},
			&cli.StringSliceFlag{ //nolint:exhaustruct
				Name:     flagAllowRedirectURLs,
				Usage:    "Allowed redirect URLs",
				Category: "application",
				EnvVars:  []string{"AUTH_ACCESS_CONTROL_ALLOWED_REDIRECT_URLS"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagServerURL,
				Usage:    "Server URL of where Auth service is running. This value is to used as a callback in email templates and for the OAuth authentication process",
				Category: "server",
				EnvVars:  []string{"AUTH_SERVER_URL"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagEnableChangeEnv,
				Usage:    "Enable change env. Do not do this in production!",
				Category: "server",
				EnvVars:  []string{"AUTH_ENABLE_CHANGE_ENV"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagCustomClaims,
				Usage:    "Custom claims",
				Category: "jwt",
				EnvVars:  []string{"AUTH_JWT_CUSTOM_CLAIMS"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagGraphqlURL,
				Usage:    "Hasura GraphQL endpoint. Required for custom claims",
				Category: "jwt",
				EnvVars:  []string{"HASURA_GRAPHQL_GRAPHQL_URL"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagHasuraAdminSecret,
				Usage:    "Hasura admin secret. Required for custom claims",
				Category: "jwt",
				EnvVars:  []string{"HASURA_GRAPHQL_ADMIN_SECRET"},
			},
			&cli.IntFlag{ //nolint: exhaustruct
				Name:     flagPasswordMinLength,
				Usage:    "Minimum password length",
				Value:    3, //nolint:mnd
				Category: "signup",
				EnvVars:  []string{"AUTH_PASSWORD_MIN_LENGTH"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagPasswordHIBPEnabled,
				Usage:    "Check user's password against Pwned Passwords https://haveibeenpwned.com/Passwords",
				Category: "signup",
				EnvVars:  []string{"AUTH_PASSWORD_HIBP_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagEmailTemplatesPath,
				Usage:    "Path to the email templates. Default to included ones if path isn't found",
				Value:    "/app/email-templates",
				Category: "email",
				EnvVars:  []string{"AUTH_EMAIL_TEMPLATES_PATH"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagBlockedEmailDomains,
				Usage:    "Comma-separated list of email domains that cannot register",
				Category: "signup",
				EnvVars:  []string{"AUTH_ACCESS_CONTROL_BLOCKED_EMAIL_DOMAINS"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagBlockedEmails,
				Usage:    "Comma-separated list of email domains that cannot register",
				Category: "signup",
				EnvVars:  []string{"AUTH_ACCESS_CONTROL_BLOCKED_EMAILS"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagAllowedEmailDomains,
				Usage:    "Comma-separated list of email domains that can register",
				Category: "signup",
				EnvVars:  []string{"AUTH_ACCESS_CONTROL_ALLOWED_EMAIL_DOMAINS"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagAllowedEmails,
				Usage:    "Comma-separated list of emails that can register",
				Category: "signup",
				EnvVars:  []string{"AUTH_ACCESS_CONTROL_ALLOWED_EMAILS"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagEmailPasswordlessEnabled,
				Usage:    "Enables passwordless authentication by email. SMTP must be configured",
				Value:    false,
				Category: "signin",
				EnvVars:  []string{"AUTH_EMAIL_PASSWORDLESS_ENABLED"},
			},
			&cli.GenericFlag{ //nolint: exhaustruct
				Name: flagRequireElevatedClaim,
				Value: &EnumValue{ //nolint: exhaustruct
					Enum: []string{
						"disabled",
						"recommended",
						"required",
					},
					Default: "disabled",
				},
				Usage:    "Require x-hasura-auth-elevated claim to perform certain actions: create PATs, change email and/or password, enable/disable MFA and add security keys. If set to `recommended` the claim check is only performed if the user has a security key attached. If set to `required` the only action that won't require the claim is setting a security key for the first time.",
				Category: "security",
				EnvVars:  []string{"AUTH_REQUIRE_ELEVATED_CLAIM"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagWebauthnEnabled,
				Usage:    "When enabled, passwordless Webauthn authentication can be done via device supported strong authenticators like fingerprint, Face ID, etc.",
				Value:    false,
				Category: "webauthn",
				EnvVars:  []string{"AUTH_WEBAUTHN_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagWebauhtnRPName,
				Usage:    "Relying party name. Friendly name visual to the user informing who requires the authentication. Probably your app's name",
				Category: "webauthn",
				EnvVars:  []string{"AUTH_WEBAUTHN_RP_NAME"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagWebauthnRPID,
				Usage:    "Relying party id. If not set `AUTH_CLIENT_URL` will be used as a default",
				Category: "webauthn",
				EnvVars:  []string{"AUTH_WEBAUTHN_RP_ID"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagWebauthnRPOrigins,
				Usage:    "Array of URLs where the registration is permitted and should have occurred on. `AUTH_CLIENT_URL` will be automatically added to the list of origins if is set",
				Category: "webauthn",
				EnvVars:  []string{"AUTH_WEBAUTHN_RP_ORIGINS"},
			},
			&cli.IntFlag{ //nolint: exhaustruct
				Name:     flagWebauthnAttestationTimeout,
				Usage:    "Timeout for the attestation process in milliseconds",
				Value:    60000, //nolint:mnd
				Category: "webauthn",
				EnvVars:  []string{"AUTH_WEBAUTHN_ATTESTATION_TIMEOUT"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagRateLimitEnable,
				Usage:    "Enable rate limiting",
				Value:    false,
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_ENABLE"},
			},
			&cli.IntFlag{ //nolint: exhaustruct
				Name:     flagRateLimitGlobalBurst,
				Usage:    "Global rate limit burst",
				Value:    100, //nolint:mnd
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_GLOBAL_BURST"},
			},
			&cli.DurationFlag{ //nolint: exhaustruct
				Name:     flagRateLimitGlobalInterval,
				Usage:    "Global rate limit interval",
				Value:    time.Minute,
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_GLOBAL_INTERVAL"},
			},
			&cli.IntFlag{ //nolint: exhaustruct
				Name:     flagRateLimitEmailBurst,
				Usage:    "Email rate limit burst",
				Value:    10, //nolint:mnd
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_EMAIL_BURST"},
			},
			&cli.DurationFlag{ //nolint: exhaustruct
				Name:     flagRateLimitEmailInterval,
				Usage:    "Email rate limit interval",
				Value:    time.Hour,
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_EMAIL_INTERVAL"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagRateLimitEmailIsGlobal,
				Usage:    "Email rate limit is global instead of per user",
				Value:    false,
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_EMAIL_IS_GLOBAL"},
			},
			&cli.IntFlag{ //nolint: exhaustruct
				Name:     flagRateLimitSMSBurst,
				Usage:    "SMS rate limit burst",
				Value:    10, //nolint:mnd
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_SMS_BURST"},
			},
			&cli.DurationFlag{ //nolint: exhaustruct
				Name:     flagRateLimitSMSInterval,
				Usage:    "SMS rate limit interval",
				Value:    time.Hour,
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_SMS_INTERVAL"},
			},
			&cli.IntFlag{ //nolint: exhaustruct
				Name:     flagRateLimitBruteForceBurst,
				Usage:    "Brute force rate limit burst",
				Value:    10, //nolint:mnd
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_BRUTE_FORCE_BURST"},
			},
			&cli.DurationFlag{ //nolint: exhaustruct
				Name:     flagRateLimitBruteForceInterval,
				Usage:    "Brute force rate limit interval",
				Value:    5 * time.Minute, //nolint:mnd
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_BRUTE_FORCE_INTERVAL"},
			},
			&cli.IntFlag{ //nolint: exhaustruct
				Name:     flagRateLimitSignupsBurst,
				Usage:    "Signups rate limit burst",
				Value:    10, //nolint:mnd
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_SIGNUPS_BURST"},
			},
			&cli.DurationFlag{ //nolint: exhaustruct
				Name:     flagRateLimitSignupsInterval,
				Usage:    "Signups rate limit interval",
				Value:    5 * time.Minute, //nolint:mnd
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_SIGNUPS_INTERVAL"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagRateLimitMemcacheServer,
				Usage:    "Store sliding window rate limit data in memcache",
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_MEMCACHE_SERVER"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagRateLimitMemcachePrefix,
				Usage:    "Prefix for rate limit keys in memcache",
				Category: "rate-limit",
				EnvVars:  []string{"AUTH_RATE_LIMIT_MEMCACHE_PREFIX"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagTurnstileSecret,
				Usage:    "Turnstile secret. If passed, enable Cloudflare's turnstile for signup methods. The header `X-Cf-Turnstile-Response ` will have to be included in the request for verification",
				Category: "turnstile",
				EnvVars:  []string{"AUTH_TURNSTILE_SECRET"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagAppleAudience,
				Usage:    "Apple Audience. Used to verify the audience on JWT tokens provided by Apple. Needed for idtoken validation",
				Category: "apple",
				EnvVars:  []string{"AUTH_PROVIDER_APPLE_AUDIENCE"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagGoogleAudience,
				Usage:    "Google Audience. Used to verify the audience on JWT tokens provided by Google. Needed for idtoken validation",
				Category: "google",
				EnvVars:  []string{"AUTH_PROVIDER_GOOGLE_AUDIENCE"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagOTPEmailEnabled,
				Usage:    "Enable OTP via email",
				Category: "otp",
				EnvVars:  []string{"AUTH_OTP_EMAIL_ENABLED"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagSMSPasswordlessEnabled,
				Usage:    "Enable SMS passwordless authentication",
				Category: "sms",
				EnvVars:  []string{"AUTH_SMS_PASSWORDLESS_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagSMSTwilioAccountSid,
				Usage:    "Twilio Account SID for SMS",
				Category: "sms",
				EnvVars:  []string{"AUTH_SMS_TWILIO_ACCOUNT_SID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagSMSTwilioAuthToken,
				Usage:    "Twilio Auth Token for SMS",
				Category: "sms",
				EnvVars:  []string{"AUTH_SMS_TWILIO_AUTH_TOKEN"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagSMSTwilioMessagingServiceID,
				Usage:    "Twilio Messaging Service ID for SMS",
				Category: "sms",
				EnvVars:  []string{"AUTH_SMS_TWILIO_MESSAGING_SERVICE_ID"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagAnonymousUsersEnabled,
				Usage:    "Enable anonymous users",
				Category: "signup",
				Value:    false,
				EnvVars:  []string{"AUTH_ANONYMOUS_USERS_ENABLED"},
			},
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagMfaEnabled,
				Usage:    "Enable MFA",
				Category: "mfa",
				Value:    false,
				EnvVars:  []string{"AUTH_MFA_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagMfaTotpIssuer,
				Usage:    "Issuer for MFA TOTP",
				Category: "mfa",
				Value:    "auth",
				EnvVars:  []string{"AUTH_MFA_TOTP_ISSUER"},
			},
			// GitHub provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagGithubEnabled,
				Usage:    "Enable GitHub OAuth provider",
				Category: "oauth-github",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_GITHUB_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagGithubClientID,
				Usage:    "GitHub OAuth client ID",
				Category: "oauth-github",
				EnvVars:  []string{"AUTH_PROVIDER_GITHUB_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagGithubClientSecret,
				Usage:    "GitHub OAuth client secret",
				Category: "oauth-github",
				EnvVars:  []string{"AUTH_PROVIDER_GITHUB_CLIENT_SECRET"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagGithubAuthorizationURL,
				Usage:    "GitHub OAuth authorization URL",
				Category: "oauth-github",
				Value:    "https://github.com/login/oauth/authorize",
				EnvVars:  []string{"AUTH_PROVIDER_GITHUB_AUTHORIZATION_URL"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagGithubTokenURL,
				Usage:    "GitHub OAuth token URL",
				Category: "oauth-github",
				Value:    "https://github.com/login/oauth/access_token",
				EnvVars:  []string{"AUTH_PROVIDER_GITHUB_TOKEN_URL"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagGithubUserProfileURL,
				Usage:    "GitHub OAuth user profile URL",
				Category: "oauth-github",
				Value:    "https://api.github.com/user",
				EnvVars:  []string{"AUTH_PROVIDER_GITHUB_USER_PROFILE_URL"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagGithubScope,
				Usage:    "GitHub OAuth scope",
				Category: "oauth-github",
				Value:    cli.NewStringSlice(providers.DefaultGithubScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_GITHUB_SCOPE"},
			},
			// Google provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagGoogleEnabled,
				Usage:    "Enable Google OAuth provider",
				Category: "oauth-google",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_GOOGLE_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagGoogleClientID,
				Usage:    "Google OAuth client ID",
				Category: "oauth-google",
				EnvVars:  []string{"AUTH_PROVIDER_GOOGLE_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagGoogleClientSecret,
				Usage:    "Google OAuth client secret",
				Category: "oauth-google",
				EnvVars:  []string{"AUTH_PROVIDER_GOOGLE_CLIENT_SECRET"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagGoogleScope,
				Usage:    "Google OAuth scope",
				Category: "oauth-google",
				Value:    cli.NewStringSlice(providers.DefaultGoogleScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_GOOGLE_SCOPE"},
			},
			// Apple provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagAppleEnabled,
				Usage:    "Enable Apple OAuth provider",
				Category: "oauth-apple",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_APPLE_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagAppleClientID,
				Usage:    "Apple OAuth client ID",
				Category: "oauth-apple",
				EnvVars:  []string{"AUTH_PROVIDER_APPLE_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagAppleTeamID,
				Usage:    "Apple OAuth team ID",
				Category: "oauth-apple",
				EnvVars:  []string{"AUTH_PROVIDER_APPLE_TEAM_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagAppleKeyID,
				Usage:    "Apple OAuth key ID",
				Category: "oauth-apple",
				EnvVars:  []string{"AUTH_PROVIDER_APPLE_KEY_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagApplePrivateKey,
				Usage:    "Apple OAuth private key",
				Category: "oauth-apple",
				EnvVars:  []string{"AUTH_PROVIDER_APPLE_PRIVATE_KEY"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagAppleScope,
				Usage:    "Apple OAuth scope",
				Category: "oauth-apple",
				Value:    cli.NewStringSlice(providers.DefaultAppleScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_APPLE_SCOPE"},
			},
			// LinkedIn provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagLinkedInEnabled,
				Usage:    "Enable LinkedIn OAuth provider",
				Category: "oauth-linkedin",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_LINKEDIN_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagLinkedInClientID,
				Usage:    "LinkedIn OAuth client ID",
				Category: "oauth-linkedin",
				EnvVars:  []string{"AUTH_PROVIDER_LINKEDIN_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagLinkedInClientSecret,
				Usage:    "LinkedIn OAuth client secret",
				Category: "oauth-linkedin",
				EnvVars:  []string{"AUTH_PROVIDER_LINKEDIN_CLIENT_SECRET"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagLinkedInScope,
				Usage:    "LinkedIn OAuth scope",
				Category: "oauth-linkedin",
				Value:    cli.NewStringSlice(providers.DefaultLinkedInScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_LINKEDIN_SCOPE"},
			},
			// Discord provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagDiscordEnabled,
				Usage:    "Enable Discord OAuth provider",
				Category: "oauth-discord",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_DISCORD_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagDiscordClientID,
				Usage:    "Discord OAuth client ID",
				Category: "oauth-discord",
				EnvVars:  []string{"AUTH_PROVIDER_DISCORD_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagDiscordClientSecret,
				Usage:    "Discord OAuth client secret",
				Category: "oauth-discord",
				EnvVars:  []string{"AUTH_PROVIDER_DISCORD_CLIENT_SECRET"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagDiscordScope,
				Usage:    "Discord OAuth scope",
				Category: "oauth-discord",
				Value:    cli.NewStringSlice(providers.DefaultDiscordScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_DISCORD_SCOPE"},
			},
			// Spotify provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagSpotifyEnabled,
				Usage:    "Enable Spotify OAuth provider",
				Category: "oauth-spotify",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_SPOTIFY_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagSpotifyClientID,
				Usage:    "Spotify OAuth client ID",
				Category: "oauth-spotify",
				EnvVars:  []string{"AUTH_PROVIDER_SPOTIFY_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagSpotifyClientSecret,
				Usage:    "Spotify OAuth client secret",
				Category: "oauth-spotify",
				EnvVars:  []string{"AUTH_PROVIDER_SPOTIFY_CLIENT_SECRET"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagSpotifyScope,
				Usage:    "Spotify OAuth scope",
				Category: "oauth-spotify",
				Value:    cli.NewStringSlice(providers.DefaultSpotifyScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_SPOTIFY_SCOPE"},
			},

			// Twitch provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagTwitchEnabled,
				Usage:    "Enable Twitch OAuth provider",
				Category: "oauth-twitch",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_TWITCH_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagTwitchClientID,
				Usage:    "Twitch OAuth client ID",
				Category: "oauth-twitch",
				EnvVars:  []string{"AUTH_PROVIDER_TWITCH_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagTwitchClientSecret,
				Usage:    "Twitch OAuth client secret",
				Category: "oauth-twitch",
				EnvVars:  []string{"AUTH_PROVIDER_TWITCH_CLIENT_SECRET"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagTwitchScope,
				Usage:    "Twitch OAuth scope",
				Category: "oauth-twitch",
				Value:    cli.NewStringSlice(providers.DefaultTwitchScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_TWITCH_SCOPE"},
			},

			// Gitlab provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagGitlabEnabled,
				Usage:    "Enable Gitlab OAuth provider",
				Category: "oauth-gitlab",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_GITLAB_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagGitlabClientID,
				Usage:    "Gitlab OAuth client ID",
				Category: "oauth-gitlab",
				EnvVars:  []string{"AUTH_PROVIDER_GITLAB_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagGitlabClientSecret,
				Usage:    "Gitlab OAuth client secret",
				Category: "oauth-gitlab",
				EnvVars:  []string{"AUTH_PROVIDER_GITLAB_CLIENT_SECRET"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagGitlabScope,
				Usage:    "Gitlab OAuth scope",
				Category: "oauth-gitlab",
				Value:    cli.NewStringSlice(providers.DefaultGitlabScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_GITLAB_SCOPE"},
			},

			// Bitbucket provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagBitbucketEnabled,
				Usage:    "Enable Bitbucket OAuth provider",
				Category: "oauth-bitbucket",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_BITBUCKET_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagBitbucketClientID,
				Usage:    "Bitbucket OAuth client ID",
				Category: "oauth-bitbucket",
				EnvVars:  []string{"AUTH_PROVIDER_BITBUCKET_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagBitbucketClientSecret,
				Usage:    "Bitbucket OAuth client secret",
				Category: "oauth-bitbucket",
				EnvVars:  []string{"AUTH_PROVIDER_BITBUCKET_CLIENT_SECRET"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagBitbucketScope,
				Usage:    "Bitbucket OAuth scope",
				Category: "oauth-bitbucket",
				Value:    cli.NewStringSlice(providers.DefaultBitbucketScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_BITBUCKET_SCOPE"},
			},

			// WorkOS provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagWorkosEnabled,
				Usage:    "Enable WorkOS OAuth provider",
				Category: "oauth-workos",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_WORKOS_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagWorkosClientID,
				Usage:    "WorkOS OAuth client ID",
				Category: "oauth-workos",
				EnvVars:  []string{"AUTH_PROVIDER_WORKOS_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagWorkosClientSecret,
				Usage:    "WorkOS OAuth client secret",
				Category: "oauth-workos",
				EnvVars:  []string{"AUTH_PROVIDER_WORKOS_CLIENT_SECRET"},
			},
			&cli.StringFlag{ //nolint:exhaustruct
				Name:     flagWorkosDefaultOrganization,
				Usage:    "WorkOS OAuth default organization",
				Category: "oauth-workos",
				EnvVars:  []string{"AUTH_PROVIDER_WORKOS_DEFAULT_ORGANIZATION"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagWorkosDefaultConnection,
				Usage:    "WorkOS OAuth default connection",
				Category: "oauth-workos",
				EnvVars:  []string{"AUTH_PROVIDER_WORKOS_DEFAULT_CONNECTION"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagWorkosDefaultDomain,
				Usage:    "WorkOS OAuth default domain",
				Category: "oauth-workos",
				EnvVars:  []string{"AUTH_PROVIDER_WORKOS_DEFAULT_DOMAIN"},
			},
			// AzureAD provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagAzureadEnabled,
				Usage:    "Enable Azuread OAuth provider",
				Category: "oauth-azuread",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_AZUREAD_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagAzureadClientID,
				Usage:    "AzureAD OAuth client ID",
				Category: "oauth-azuread",
				EnvVars:  []string{"AUTH_PROVIDER_AZUREAD_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagAzureadClientSecret,
				Usage:    "Azuread OAuth client secret",
				Category: "oauth-azuread",
				EnvVars:  []string{"AUTH_PROVIDER_AZUREAD_CLIENT_SECRET"},
			},
			&cli.StringFlag{ //nolint:exhaustruct
				Name:     flagAzureadTenant,
				Usage:    "Azuread Tenant",
				Category: "oauth-azuread",
				EnvVars:  []string{"AUTH_PROVIDER_AZUREAD_TENANT"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagAzureadScope,
				Usage:    "Azuread OAuth scope",
				Category: "oauth-azuread",
				Value:    cli.NewStringSlice(providers.DefaultAzureadScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_AZUREAD_SCOPE"},
			},
			// Facebook provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagFacebookEnabled,
				Usage:    "Enable Facebook OAuth provider",
				Category: "oauth-facebook",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_FACEBOOK_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagFacebookClientID,
				Usage:    "Facebook OAuth client ID",
				Category: "oauth-facebook",
				EnvVars:  []string{"AUTH_PROVIDER_FACEBOOK_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagFacebookClientSecret,
				Usage:    "Facebook OAuth client secret",
				Category: "oauth-facebook",
				EnvVars:  []string{"AUTH_PROVIDER_FACEBOOK_CLIENT_SECRET"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagFacebookScope,
				Usage:    "Facebook OAuth scope",
				Category: "oauth-facebook",
				Value:    cli.NewStringSlice(providers.DefaultFacebookScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_FACEBOOK_SCOPE"},
			},
			// Windowslive provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagWindowsliveEnabled,
				Usage:    "Enable Windowslive OAuth provider",
				Category: "oauth-windowslive",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_WINDOWS_LIVE_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagWindowsliveClientID,
				Usage:    "Windowslive OAuth client ID",
				Category: "oauth-windowslive",
				EnvVars:  []string{"AUTH_PROVIDER_WINDOWS_LIVE_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagWindowsliveClientSecret,
				Usage:    "Windows Live OAuth client secret",
				Category: "oauth-windowslive",
				EnvVars:  []string{"AUTH_PROVIDER_WINDOWS_LIVE_CLIENT_SECRET"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagWindowsliveScope,
				Usage:    "Windows Live OAuth scope",
				Category: "oauth-windowslive",
				Value:    cli.NewStringSlice(providers.DefaultWindowsliveScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_WINDOWS_LIVE_SCOPE"},
			},

			// Strava provider flags
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagStravaEnabled,
				Usage:    "Enable Strava OAuth provider",
				Category: "oauth-strava",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_STRAVA_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagStravaClientID,
				Usage:    "Strava OAuth client ID",
				Category: "oauth-strava",
				EnvVars:  []string{"AUTH_PROVIDER_STRAVA_CLIENT_ID"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagStravaClientSecret,
				Usage:    "Strava OAuth client secret",
				Category: "oauth-strava",
				EnvVars:  []string{"AUTH_PROVIDER_STRAVA_CLIENT_SECRET"},
			},
			&cli.StringSliceFlag{ //nolint: exhaustruct
				Name:     flagStravaScope,
				Usage:    "Strava OAuth scope",
				Category: "oauth-strava",
				Value:    cli.NewStringSlice(providers.DefaultStravaScopes...),
				EnvVars:  []string{"AUTH_PROVIDER_STRAVA_SCOPE"},
			},

			// twitter
			&cli.BoolFlag{ //nolint: exhaustruct
				Name:     flagTwitterEnabled,
				Usage:    "Enable Twitter OAuth provider",
				Category: "oauth-twitter",
				Value:    false,
				EnvVars:  []string{"AUTH_PROVIDER_TWITTER_ENABLED"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagTwitterConsumerKey,
				Usage:    "Twitter OAuth consumer key",
				Category: "oauth-twitter",
				EnvVars:  []string{"AUTH_PROVIDER_TWITTER_CONSUMER_KEY"},
			},
			&cli.StringFlag{ //nolint: exhaustruct
				Name:     flagTwitterConsumerSecret,
				Usage:    "Twitter OAuth consumer secret",
				Category: "oauth-twitter",
				EnvVars:  []string{"AUTH_PROVIDER_TWITTER_CONSUMER_SECRET"},
			},
		},
		Action: serve,
	}
}

func getRateLimiter(cCtx *cli.Context, logger *slog.Logger) gin.HandlerFunc {
	var store ratelimit.Store
	if cCtx.String(flagRateLimitMemcacheServer) != "" {
		store = ratelimit.NewMemcacheStore(
			memcache.New(cCtx.String(flagRateLimitMemcacheServer)),
			cCtx.String(flagRateLimitMemcachePrefix),
			logger.WithGroup("rate-limit-memcache"),
		)
	} else {
		store = ratelimit.NewInMemoryStore()
	}

	return ratelimit.RateLimit(
		cCtx.String(flagAPIPrefix),
		cCtx.Int(flagRateLimitGlobalBurst),
		cCtx.Duration(flagRateLimitGlobalInterval),
		cCtx.Int(flagRateLimitEmailBurst),
		cCtx.Duration(flagRateLimitEmailInterval),
		cCtx.Bool(flagRateLimitEmailIsGlobal),
		cCtx.Bool(flagEmailSigninEmailVerifiedRequired),
		cCtx.Int(flagRateLimitSMSBurst),
		cCtx.Duration(flagRateLimitSMSInterval),
		cCtx.Int(flagRateLimitBruteForceBurst),
		cCtx.Duration(flagRateLimitBruteForceInterval),
		cCtx.Int(flagRateLimitSignupsBurst),
		cCtx.Duration(flagRateLimitSignupsInterval),
		store,
	)
}

func getDependencies( //nolint:ireturn
	cCtx *cli.Context, db *sql.Queries, logger *slog.Logger,
) (
	controller.Emailer,
	controller.SMSer,
	*controller.JWTGetter,
	*oidc.IDTokenValidatorProviders,
	error,
) {
	emailer, templates, err := getEmailer(cCtx, logger)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("problem creating emailer: %w", err)
	}

	sms, err := getSMS(cCtx, templates, db, logger)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("problem creating SMS client: %w", err)
	}

	jwtGetter, err := getJWTGetter(cCtx, db)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("problem creating jwt getter: %w", err)
	}

	idTokenValidator, err := oidc.NewIDTokenValidatorProviders(
		cCtx.Context,
		cCtx.String(flagAppleAudience),
		cCtx.String(flagGoogleAudience),
		"",
	)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error creating id token validator: %w", err)
	}

	return emailer, sms, jwtGetter, idTokenValidator, nil
}

func getGoServer( //nolint:funlen
	cCtx *cli.Context, db *sql.Queries, logger *slog.Logger,
) (*http.Server, error) {
	router := gin.New()

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(docs.OpenAPISchema)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI schema: %w", err)
	}
	doc.AddServer(&openapi3.Server{ //nolint:exhaustruct
		URL: cCtx.String(flagAPIPrefix),
	})

	handlers := []gin.HandlerFunc{
		// ginmiddleware.OapiRequestValidator(doc),
		gin.Recovery(),
		cors(),
		middleware.Logger(logger),
	}

	if cCtx.Bool(flagRateLimitEnable) {
		handlers = append(handlers, getRateLimiter(cCtx, logger))
	}

	if cCtx.String(flagTurnstileSecret) != "" {
		handlers = append(handlers, middleware.Tunrstile(
			cCtx.String(flagTurnstileSecret), cCtx.String(flagAPIPrefix)),
		)
	}

	router.Use(handlers...)

	config, err := getConfig(cCtx)
	if err != nil {
		return nil, fmt.Errorf("problem creating config: %w", err)
	}

	emailer, smsClient, jwtGetter, idTokenValidator, err := getDependencies(cCtx, db, logger)
	if err != nil {
		return nil, err
	}

	oauthProviders, err := getOauth2Providers(cCtx)
	if err != nil {
		return nil, fmt.Errorf("problem creating oauth providers: %w", err)
	}

	ctrl, err := controller.New(
		db,
		config,
		jwtGetter,
		emailer,
		smsClient,
		hibp.NewClient(),
		oauthProviders,
		idTokenValidator,
		controller.NewTotp(cCtx.String(flagMfaTotpIssuer), time.Now),
		cCtx.App.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create controller: %w", err)
	}
	handler := api.NewStrictHandler(ctrl, []api.StrictMiddlewareFunc{})
	mw := api.MiddlewareFunc(ginmiddleware.OapiRequestValidatorWithOptions(
		doc,
		&ginmiddleware.Options{ //nolint:exhaustruct
			Options: openapi3filter.Options{ //nolint:exhaustruct
				AuthenticationFunc: jwtGetter.MiddlewareFunc,
			},
			SilenceServersWarning: true,
		},
	))
	api.RegisterHandlersWithOptions(
		router,
		handler,
		api.GinServerOptions{
			BaseURL:      cCtx.String(flagAPIPrefix),
			Middlewares:  []api.MiddlewareFunc{mw},
			ErrorHandler: nil,
		},
	)

	if cCtx.Bool(flagEnableChangeEnv) {
		router.POST(cCtx.String(flagAPIPrefix)+"/change-env", ctrl.PostChangeEnv)
	}

	// for backwards compatibility we keep these two endpoints without the prefix
	if cCtx.String(flagAPIPrefix) != "" {
		router.GET("/healthz", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		router.HEAD("/healthz", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
	}

	server := &http.Server{ //nolint:exhaustruct
		Addr:              ":" + cCtx.String(flagPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second, //nolint:mnd
	}

	return server, nil
}

func serve(cCtx *cli.Context) error {
	logger := getLogger(cCtx.Bool(flagDebug), cCtx.Bool(flagLogFormatTEXT))
	logger.Info(cCtx.App.Name + " v" + cCtx.App.Version)
	logFlags(logger, cCtx)

	ctx, cancel := context.WithCancel(cCtx.Context)
	defer cancel()

	pool, err := getDBPool(cCtx)
	if err != nil {
		return fmt.Errorf("failed to create database pool: %w", err)
	}
	defer pool.Close()

	db := sql.New(pool)
	if err := applyMigrations(ctx, cCtx, db, logger); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	server, err := getGoServer(cCtx, db, logger)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	go func() {
		defer cancel()
		logger.Info("starting server", slog.String("port", cCtx.String(flagPort)))
		if err := server.ListenAndServe(); err != nil {
			logger.Error("server failed", slog.String("error", err.Error()))
		}
	}()

	<-ctx.Done()

	logger.Info("shutting down server")
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return nil
}
