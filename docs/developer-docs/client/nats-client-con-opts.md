# NATS Client Connection Configuration Standard

## 1. Overview

The NATS Go client provides an `Options` structure to customize client connections to a NATS server or cluster. These options control server connectivity, reconnection behavior, connection liveness, graceful shutdown, transport security, and authentication.

For enterprise usage, application teams should follow a consistent connection configuration standard across services. This improves operational visibility, resilience, security, and consistency across application deployments.

This document identifies the essential options to standardize and proposes environment variable names for configuring them. The recommended values will be defined separately by the NATS platform team.

**Configuration principles**

- Externalize deployment-specific connection settings through environment variables.
- Use a consistent client identity for each application or service.
- Define standard connection, reconnection, and shutdown behavior.
- Enforce the approved TLS and authentication policies.
- Retrieve credentials and private keys through the approved secrets-management mechanism.
- Map environment variables to the corresponding SDK options in application code. The Go NATS client does not automatically read these proposed environment variables.

## 2. Enterprise Connection Configuration

| Category | Option | Description | Recommended ENV Name | Recommended Value |
|---|---|---|---|---|
| Connectivity | `Servers` | List of NATS server URLs used by the client to establish connections. | `NATS_SERVERS` | |
| Connectivity | `Name` | Identifies the client on the NATS server. | `NATS_CLIENT_NAME` | |
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

## 3. Language Configuration Entities & SDK Connection Examples

The following reference implementations demonstrate how to map the externalized enterprise environment variables into a strongly-typed configuration entity, and use that entity to establish the NATS client connection across Go, Java, and Python.

### 3.1 Go (Struct Tag Binding with caarlos0/env)

This pattern uses struct tags with `github.com/caarlos0/env/v10` to bind environment variables directly to configuration fields and converts them into `nats.Option` values for `nats.Connect`.

```go
package natsconfig

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/nats-io/nats.go"
)

// NATSClientConfig maps externalized enterprise connection variables.
type NATSClientConfig struct {
	Servers              []string      `env:"NATS_SERVERS" envSeparator:"," envDefault:"nats://localhost:4222"`
	ClientName           string        `env:"NATS_CLIENT_NAME" envDefault:"order-service"`
	ConnectTimeout       time.Duration `env:"NATS_CONNECT_TIMEOUT" envDefault:"2s"`
	AllowReconnect       bool          `env:"NATS_ALLOW_RECONNECT" envDefault:"true"`
	MaxReconnectAttempts int           `env:"NATS_MAX_RECONNECT_ATTEMPTS" envDefault:"10"`
	ReconnectWait        time.Duration `env:"NATS_RECONNECT_WAIT" envDefault:"2s"`
	RetryOnFailedConnect bool          `env:"NATS_RETRY_ON_FAILED_CONNECT" envDefault:"true"`
	PingInterval         time.Duration `env:"NATS_PING_INTERVAL" envDefault:"20s"`
	MaxPingsOut          int           `env:"NATS_MAX_PINGS_OUT" envDefault:"3"`
	DrainTimeout         time.Duration `env:"NATS_DRAIN_TIMEOUT" envDefault:"30s"`
	TLSEnabled           bool          `env:"NATS_TLS_ENABLED" envDefault:"false"`
	TLSRootCASource      string        `env:"NATS_TLS_ROOT_CA_SOURCE"`
	TLSClientCertSource  string        `env:"NATS_TLS_CLIENT_CERT_SOURCE"`
	TLSClientKeySource   string        `env:"NATS_TLS_CLIENT_KEY_SOURCE"`
	AuthMethod           string        `env:"NATS_AUTH_METHOD"`
	AuthToken            string        `env:"NATS_AUTH_TOKEN"`
	AuthUser             string        `env:"NATS_AUTH_USER"`
	AuthPassword         string        `env:"NATS_AUTH_PASSWORD"`
	AuthCredentialsFile  string        `env:"NATS_AUTH_CREDENTIALS_FILE"`
}

// LoadFromEnv loads and parses environment variables.
func LoadFromEnv() (*NATSClientConfig, error) {
	var cfg NATSClientConfig
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse NATS configuration: %w", err)
	}
	return &cfg, nil
}

// BuildOptions translates the configuration entity into a nats.Option slice.
func (c *NATSClientConfig) BuildOptions() ([]nats.Option, error) {
	opts := []nats.Option{
		nats.Name(c.ClientName),
		nats.Timeout(c.ConnectTimeout),
		nats.ReconnectWait(c.ReconnectWait),
		nats.PingInterval(c.PingInterval),
		nats.MaxPingsOutstanding(c.MaxPingsOut),
		nats.DrainTimeout(c.DrainTimeout),
	}

	if !c.AllowReconnect {
		opts = append(opts, nats.NoReconnect())
	} else {
		opts = append(opts, nats.MaxReconnects(c.MaxReconnectAttempts))
	}

	if c.RetryOnFailedConnect {
		opts = append(opts, nats.RetryOnFailedConnect(true))
	}

	// TLS Transport Security
	if c.TLSEnabled {
		if c.TLSRootCASource != "" {
			opts = append(opts, nats.RootCAs(c.TLSRootCASource))
		}
		if c.TLSClientCertSource != "" && c.TLSClientKeySource != "" {
			opts = append(opts, nats.ClientCert(c.TLSClientCertSource, c.TLSClientKeySource))
		}
	}

	// Authentication Methods
	switch strings.ToLower(c.AuthMethod) {
	case "token":
		if c.AuthToken != "" {
			opts = append(opts, nats.Token(c.AuthToken))
		}
	case "user_password":
		if c.AuthUser != "" {
			opts = append(opts, nats.UserInfo(c.AuthUser, c.AuthPassword))
		}
	case "jwt", "credentials":
		if c.AuthCredentialsFile != "" {
			opts = append(opts, nats.UserCredentials(c.AuthCredentialsFile))
		}
	}

	return opts, nil
}

// Connect establishes the NATS connection using the mapped entity.
func Connect(cfg *NATSClientConfig) (*nats.Conn, error) {
	opts, err := cfg.BuildOptions()
	if err != nil {
		return nil, err
	}
	serverList := strings.Join(cfg.Servers, ",")
	return nats.Connect(serverList, opts...)
}
```

### 3.2 Java (Spring Boot @ConfigurationProperties)

In Spring Boot, `@ConfigurationProperties` binds environment variables using standard relaxed binding rules (e.g., `NATS_SERVERS` to `servers`, `NATS_CONNECT_TIMEOUT` to `connectTimeout`). The class exposes a `@Bean` method producing the connected `io.nats.client.Connection`.

```java
package com.example.nats.config;

import io.nats.client.Connection;
import io.nats.client.Nats;
import io.nats.client.Options;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import javax.net.ssl.SSLContext;
import java.io.IOException;
import java.security.GeneralSecurityException;
import java.time.Duration;
import java.util.List;

@Configuration
@ConfigurationProperties(prefix = "nats")
public class NatsConnectionConfig {

    private List<String> servers = List.of("nats://localhost:4222");
    private String clientName = "order-service";
    private Duration connectTimeout = Duration.ofSeconds(2);
    private boolean allowReconnect = true;
    private int maxReconnectAttempts = 10;
    private Duration reconnectWait = Duration.ofSeconds(2);
    private boolean retryOnFailedConnect = true;
    private Duration pingInterval = Duration.ofSeconds(20);
    private int maxPingsOut = 3;
    private Duration drainTimeout = Duration.ofSeconds(30);

    private boolean tlsEnabled = false;
    private String tlsRootCaSource;
    private String tlsClientCertSource;
    private String tlsClientKeySource;

    private String authMethod;
    private String authToken;
    private String authUser;
    private String authPassword;
    private String authCredentialsFile;

    public List<String> getServers() { return servers; }
    public void setServers(List<String> servers) { this.servers = servers; }
    public String getClientName() { return clientName; }
    public void setClientName(String clientName) { this.clientName = clientName; }
    public Duration getConnectTimeout() { return connectTimeout; }
    public void setConnectTimeout(Duration connectTimeout) { this.connectTimeout = connectTimeout; }
    public boolean isAllowReconnect() { return allowReconnect; }
    public void setAllowReconnect(boolean allowReconnect) { this.allowReconnect = allowReconnect; }
    public int getMaxReconnectAttempts() { return maxReconnectAttempts; }
    public void setMaxReconnectAttempts(int maxReconnectAttempts) { this.maxReconnectAttempts = maxReconnectAttempts; }
    public Duration getReconnectWait() { return reconnectWait; }
    public void setReconnectWait(Duration reconnectWait) { this.reconnectWait = reconnectWait; }
    public boolean isRetryOnFailedConnect() { return retryOnFailedConnect; }
    public void setRetryOnFailedConnect(boolean retryOnFailedConnect) { this.retryOnFailedConnect = retryOnFailedConnect; }
    public Duration getPingInterval() { return pingInterval; }
    public void setPingInterval(Duration pingInterval) { this.pingInterval = pingInterval; }
    public int getMaxPingsOut() { return maxPingsOut; }
    public void setMaxPingsOut(int maxPingsOut) { this.maxPingsOut = maxPingsOut; }
    public Duration getDrainTimeout() { return drainTimeout; }
    public void setDrainTimeout(Duration drainTimeout) { this.drainTimeout = drainTimeout; }
    public boolean isTlsEnabled() { return tlsEnabled; }
    public void setTlsEnabled(boolean tlsEnabled) { this.tlsEnabled = tlsEnabled; }
    public String getTlsRootCaSource() { return tlsRootCaSource; }
    public void setTlsRootCaSource(String tlsRootCaSource) { this.tlsRootCaSource = tlsRootCaSource; }
    public String getTlsClientCertSource() { return tlsClientCertSource; }
    public void setTlsClientCertSource(String tlsClientCertSource) { this.tlsClientCertSource = tlsClientCertSource; }
    public String getTlsClientKeySource() { return tlsClientKeySource; }
    public void setTlsClientKeySource(String tlsClientKeySource) { this.tlsClientKeySource = tlsClientKeySource; }
    public String getAuthMethod() { return authMethod; }
    public void setAuthMethod(String authMethod) { this.authMethod = authMethod; }
    public String getAuthToken() { return authToken; }
    public void setAuthToken(String authToken) { this.authToken = authToken; }
    public String getAuthUser() { return authUser; }
    public void setAuthUser(String authUser) { this.authUser = authUser; }
    public String getAuthPassword() { return authPassword; }
    public void setAuthPassword(String authPassword) { this.authPassword = authPassword; }
    public String getAuthCredentialsFile() { return authCredentialsFile; }
    public void setAuthCredentialsFile(String authCredentialsFile) { this.authCredentialsFile = authCredentialsFile; }

    /**
     * Converts configuration properties into jnats Options and creates the Connection bean.
     */
    @Bean(destroyMethod = "close")
    public Connection natsConnection() throws IOException, InterruptedException, GeneralSecurityException {
        Options.Builder builder = new Options.Builder()
                .servers(servers.toArray(new String[0]))
                .connectionName(clientName)
                .connectionTimeout(connectTimeout)
                .reconnectWait(reconnectWait)
                .pingInterval(pingInterval)
                .maxPingsOut(maxPingsOut);

        if (!allowReconnect) {
            builder.noReconnect();
        } else {
            builder.maxReconnects(maxReconnectAttempts);
        }

        // TLS Configuration
        if (tlsEnabled && tlsRootCaSource != null) {
            // Configures SSLContext from truststore/cert source
            builder.sslContext(SSLContext.getDefault());
        }

        // Authentication Methods
        if (authMethod != null) {
            switch (authMethod.toLowerCase()) {
                case "token":
                    if (authToken != null) {
                        builder.token(authToken.toCharArray());
                    }
                    break;
                case "user_password":
                    if (authUser != null && authPassword != null) {
                        builder.userInfo(authUser, authPassword);
                    }
                    break;
                case "jwt":
                case "credentials":
                    if (authCredentialsFile != null) {
                        builder.authHandler(Nats.credentials(authCredentialsFile));
                    }
                    break;
            }
        }

        return Nats.connect(builder.build());
    }
}
```

### 3.3 Python (pydantic-settings BaseSettings)

In Python services, `pydantic-settings` provides strongly-typed environment variable parsing with validation, converting settings into keyword arguments for `nats.connect`.

```python
import ssl
from typing import List, Optional
import nats
from nats.aio.client import Client as NATS
from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class NATSClientConfig(BaseSettings):
    """Maps externalized environment variables into a validated typed model."""

    model_config = SettingsConfigDict(case_sensitive=True)

    servers: List[str] = Field(
        default=["nats://localhost:4222"],
        alias="NATS_SERVERS",
    )
    client_name: str = Field(
        default="order-service",
        alias="NATS_CLIENT_NAME",
    )
    connect_timeout: float = Field(
        default=2.0,
        alias="NATS_CONNECT_TIMEOUT",
    )
    allow_reconnect: bool = Field(
        default=True,
        alias="NATS_ALLOW_RECONNECT",
    )
    max_reconnect_attempts: int = Field(
        default=10,
        alias="NATS_MAX_RECONNECT_ATTEMPTS",
    )
    reconnect_wait: float = Field(
        default=2.0,
        alias="NATS_RECONNECT_WAIT",
    )
    retry_on_failed_connect: bool = Field(
        default=True,
        alias="NATS_RETRY_ON_FAILED_CONNECT",
    )
    ping_interval: float = Field(
        default=20.0,
        alias="NATS_PING_INTERVAL",
    )
    max_pings_out: int = Field(
        default=3,
        alias="NATS_MAX_PINGS_OUT",
    )
    drain_timeout: float = Field(
        default=30.0,
        alias="NATS_DRAIN_TIMEOUT",
    )
    tls_enabled: bool = Field(
        default=False,
        alias="NATS_TLS_ENABLED",
    )
    tls_root_ca_source: Optional[str] = Field(
        default=None,
        alias="NATS_TLS_ROOT_CA_SOURCE",
    )
    tls_client_cert_source: Optional[str] = Field(
        default=None,
        alias="NATS_TLS_CLIENT_CERT_SOURCE",
    )
    tls_client_key_source: Optional[str] = Field(
        default=None,
        alias="NATS_TLS_CLIENT_KEY_SOURCE",
    )
    auth_method: Optional[str] = Field(
        default=None,
        alias="NATS_AUTH_METHOD",
    )
    auth_token: Optional[str] = Field(
        default=None,
        alias="NATS_AUTH_TOKEN",
    )
    auth_user: Optional[str] = Field(
        default=None,
        alias="NATS_AUTH_USER",
    )
    auth_password: Optional[str] = Field(
        default=None,
        alias="NATS_AUTH_PASSWORD",
    )
    auth_credentials_file: Optional[str] = Field(
        default=None,
        alias="NATS_AUTH_CREDENTIALS_FILE",
    )

    def to_connect_kwargs(self) -> dict:
        """Translates the configuration entity into nats-py connect kwargs."""
        kwargs = {
            "servers": self.servers,
            "name": self.client_name,
            "connect_timeout": self.connect_timeout,
            "allow_reconnect": self.allow_reconnect,
            "max_reconnect_attempts": self.max_reconnect_attempts,
            "reconnect_time_wait": self.reconnect_wait,
            "ping_interval": int(self.ping_interval),
            "max_outstanding_pings": self.max_pings_out,
            "drain_timeout": self.drain_timeout,
        }

        # TLS Transport Security
        if self.tls_enabled and self.tls_root_ca_source:
            ssl_ctx = ssl.create_default_context(cafile=self.tls_root_ca_source)
            if self.tls_client_cert_source and self.tls_client_key_source:
                ssl_ctx.load_cert_chain(
                    certfile=self.tls_client_cert_source,
                    keyfile=self.tls_client_key_source,
                )
            kwargs["tls"] = ssl_ctx

        # Authentication Methods
        if self.auth_method:
            method = self.auth_method.lower()
            if method == "token" and self.auth_token:
                kwargs["token"] = self.auth_token
            elif method == "user_password" and self.auth_user and self.auth_password:
                kwargs["user"] = self.auth_user
                kwargs["password"] = self.auth_password
            elif method in ("jwt", "credentials") and self.auth_credentials_file:
                kwargs["user_credentials"] = self.auth_credentials_file

        return kwargs


async def connect_nats(cfg: NATSClientConfig) -> NATS:
    """Connects to NATS using the mapped configuration entity."""
    nc = NATS()
    await nc.connect(**cfg.to_connect_kwargs())
    return nc
```

## 4. Implementation Notes

1. **Environment variables are a platform convention.** The names above are proposed enterprise names, not native environment variables defined by the NATS Go SDK.

2. **Authentication methods are alternatives.** `NATS_AUTH_METHOD` identifies the selected mechanism; credentials and signing material must be supplied separately through the approved secrets-management integration.

3. **TLS must verify server identity.** Do not use insecure TLS configurations that skip server verification. The Go client source explicitly warns against the insecure behavior associated with its `Secure` option.

4. **Connection callbacks are application behavior.** Connection, disconnection, reconnection, closure, and asynchronous-error callbacks should be implemented where needed for logging, metrics, and operational visibility. They are not ordinary environment-variable settings and are therefore excluded from the configuration table.

5. **Advanced tuning is intentionally excluded.** Options such as `WriteBufferSize`, `SubChanLen`, `ReconnectBufSize`, `ReconnectOnFlusherError`, custom dialers, and WebSocket-specific settings should remain outside the baseline standard unless a documented requirement justifies them.

## 5. Next Steps

- Finalize the recommended values and defaults for each configuration.
- Confirm which settings are mandatory and which are optional.
- Define the approved authentication mechanism and TLS requirements.
- Publish language-specific connection templates for application teams.

