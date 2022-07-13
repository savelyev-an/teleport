---
authors: Noah Stride <noah@goteleport.com>
state: draft
---

# RFD 79 - OIDC JWT Joining

## Required Approvers

* Engineering @zmb3 && @nklaassen
* Security @reedloden
* Product: (@xinding33 || @klizhentas)

## Terminology

- OIDC: OpenID Connect. A federated authentication protocol built on top of OAuth 2.0.
- OP: OpenID Provider.
- Issuer: an OP that has issued a specific token.

## What

Teleport should support trusting a third party OP and the JWTs that it issues when authenticating a client for the cluster join process. This is similar to the support for IAM joining, and will allow joining a Teleport cluster without the need to distribute a token on several platforms.

Users will need to be able to configure trust in an OP, and rules that determine what identities are allowed to join the cluster.

## Why

This feature allows us to reduce the friction involved in adding many new nodes to Teleport on a variety of platforms. This is also more secure, as the user does not need to distribute a token which is liable to exfilitration.

Whilst multiple providers offer OIDC identities to workloads running on their platform, we will start by targetting GCP GCE since this represents a large portion of the market. However, the work towards this feature will also enable us to simply add other providers that support OIDC such as:

- GitHub Actions: a key platform for growing usage of Machine ID.
- GitLab CI/CD
- GCP GCB

## Details

### Configuration

### Security Considerations

#### Ease of misconfiguration

It is possible for users to create a trust configuration that would allow a malicious actor to craft an identity that would be able to join their cluster.

For example, if the trust was configured with an allow rule such as:

```go
claims.google.compute_engine.instance_name == "an-instance"
```

Then a malicious actor would be able to create their own GCP project, create an instance with that name, and use it to obtain a jwt that would be trusted by Teleport.

It is imperative that users include some rule that filters it to their own project e.g:

```go
claims.compute_engine.project_id == "my-project" && claims.google.compute_engine.instance_name == "an-instance
```

#### MITM of the connection to the issuer

In order to verify the JWT, we have to fetch the public key from the issuers's JWKS endpoint. If this connection is not secured (e.g HTTPS), it would be possible for a malicious actor to return a public key they've used to sign the JWT with.

We should require that the configured issuer URL is HTTPS to mitigate this.

## References and Resources
