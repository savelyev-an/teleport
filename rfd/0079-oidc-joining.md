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
- OP: OpenID Provider

## What

Teleport should support trusting a third party OP and the JWTs that it issues when authenticating a client for the cluster join process. This is similar to the support for IAM joining, and will allow joining a Teleport cluster without the need to distribute a token on several platforms.

Users will need to be able to configure an OP, and rules that determine what identities are allowed to join the cluster.

## Why

This feature allows us to reduce the friction involved in adding many new nodes to Teleport on a variety of platforms. This is also more secure, as the user does not need to distribute a token which is liable to exfilitration.

Whilst multiple providers offer OIDC identities to workloads running on their platform, we will start by targetting GCP GCE since this represents a large portion of the market. However, the work towards this feature will also enable us to simply add other providers that support OIDC such as:

- GitHub Actions: a key platform for Machine ID.
- GitLab CI/CD
- GCP GCB

## Details

### Security Considerations


## References and Resources
