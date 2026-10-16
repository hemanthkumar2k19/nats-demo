# NATS Client Connection Configuration Standard

## 1. Overview

The NATS Go client provides an `Options` structure to customize client connections to a NATS server or cluster. These options control server connectivity, reconnection behavior, connection liveness, graceful shutdown, transport security, and authentication.

For enterprise usage, application teams should follow a consistent connection configuration standard across services. This improves operational visibility, resilience, security, and consistency across application deployments.

This document identifies the essential options to standardize and proposes environment variable names for configuring them. The recommended values will be defined separately by the NATS platform team.

**Configuration principles**

- Externalize deployment-specific connection settings through environment variables.
- Use a consistent connection identity for each application or service.
- Define standard connection, reconnection, and shutdown behavior.
- Enforce the approved TLS and authentication policies.
- Retrieve credentials and private keys through the approved secrets-management mechanism.
- Map environment variables to the corresponding SDK options in application code. The Go NATS client does not automatically read these proposed environment variables.

## 2. Enterprise Connection Configuration

| Category | Option | Description | Recommended ENV Name | Recommended Value |
|---|---|---|---|---|
| Connectivity | `Servers` | List of NATS server URLs used by the client to establish connections. | `NATS_SERVERS` | |
| Connectivity | `Name` | Identifies the client connection on the NATS server. | `NATS_CONNECTION_NAME` | |
| Connectivity | `Timeout` | Maximum time allowed for a connection dial operation. | `NATS_CONNECT_TIMEOUT` | |
| Resilience | `AllowReconnect` | Enables reconnection after an established connection is disconnected. | `NATS_ALLOW_RECONNECT` | |
| Resilience | `MaxReconnect` | Maximum number of reconnection attempts; a negative value allows unlimited attempts. | `NATS_MAX_RECONNECT_ATTEMPTS` | |
| Resilience | `ReconnectWait` | Delay between reconnection attempts to a previously used server. | `NATS_RECONNECT_WAIT` | |
| Resilience | `RetryOnFailedConnect` | Enables retry behavior when the initial connection attempt fails. | `NATS_RETRY_ON_FAILED_CONNECT` | |
| Connection Liveness | `PingInterval` | Interval at which the client sends PING messages to check connection responsiveness. | `NATS_PING_INTERVAL` | |
| Connection Liveness | `MaxPingsOut` | Maximum unanswered PING messages before the client reports a stale connection. | `NATS_MAX_PINGS_OUT` | |
| Graceful Shutdown | `DrainTimeout` | Maximum time allowed for a drain operation to complete. | `NATS_DRAIN_TIMEOUT` | |
| Transport Security | `TLSConfig` | Configures TLS transport settings, including certificate and trust configuration. | `NATS_TLS_ENABLED` | |
| Transport Security | `TLSCertCB` | Callback that supplies a client TLS certificate. | `NATS_TLS_CLIENT_CERT_SOURCE` | |
| Transport Security | `RootCAsCB` | Callback that supplies trusted root certificate authorities for server certificate verification. | `NATS_TLS_ROOT_CA_SOURCE` | |
| Authentication | `UserJWT` | Callback that obtains a user JWT for NATS authentication. | `NATS_AUTH_METHOD` | |
| Authentication | `Nkey` / `SignatureCB` | Uses NKey identity and nonce signing for authentication. | `NATS_AUTH_METHOD` | |
| Authentication | `User` / `Password` | Supplies username and password credentials. | `NATS_AUTH_METHOD` | |
| Authentication | `Token` / `TokenHandler` | Supplies a token or dynamically obtains one for authentication. | `NATS_AUTH_METHOD` | |

## 3. Implementation Notes

1. **Environment variables are a platform convention.** The names above are proposed enterprise names, not native environment variables defined by the NATS Go SDK.

2. **Authentication methods are alternatives.** `NATS_AUTH_METHOD` identifies the selected mechanism; credentials and signing material must be supplied separately through the approved secrets-management integration.

3. **TLS must verify server identity.** Do not use insecure TLS configurations that skip server verification. The Go client source explicitly warns against the insecure behavior associated with its `Secure` option.

4. **Connection callbacks are application behavior.** Connection, disconnection, reconnection, closure, and asynchronous-error callbacks should be implemented where needed for logging, metrics, and operational visibility. They are not ordinary environment-variable settings and are therefore excluded from the configuration table.

5. **Advanced tuning is intentionally excluded.** Options such as `WriteBufferSize`, `SubChanLen`, `ReconnectBufSize`, `ReconnectOnFlusherError`, custom dialers, and WebSocket-specific settings should remain outside the baseline standard unless a documented requirement justifies them.

## 4. Next Steps

- Finalize the recommended values and defaults for each configuration.
- Confirm which settings are mandatory and which are optional.
- Define the approved authentication mechanism and TLS requirements.
- Implement the environment-variable mapping in the supported language SDKs.
- Publish language-specific connection examples for application teams.
