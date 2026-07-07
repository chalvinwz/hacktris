# Security Policy

## Supported versions

hacktris is pre-1.0 and ships from `main`. Security fixes land on the latest
release and the `main` branch.

| Version             | Supported |
|---------------------|-----------|
| Latest release      | ✅        |
| `main`              | ✅        |
| Older tagged builds | ❌        |

## Reporting a vulnerability

**Do not open a public issue for security problems.**

Report privately through GitHub:

1. Go to the repository's **Security** tab → **Advisories** → **Report a
   vulnerability** (direct link:
   https://github.com/chalvinwz/hacktris/security/advisories/new).
2. Describe the issue, affected version (`hacktris --version`), and steps to
   reproduce.

This routes the report privately to the maintainer via GitHub Private
Vulnerability Reporting — no public disclosure until a fix is ready.

## What to expect

hacktris is maintained by a single person on a best-effort basis. Expect an
initial acknowledgement within a few days. Once a fix is ready it ships as a new
tag, and the advisory is published crediting the reporter (unless you prefer to
remain anonymous).

## Scope

`hacktris-server` accepts network connections and decodes untrusted,
newline-delimited JSON from clients. In scope: crashes, panics, or unbounded
resource use triggered by malformed or hostile protocol messages; and any way a
client can corrupt or hijack another player's session. hacktris handles no
credentials or secrets and stores nothing on disk. Denial-of-service that merely
requires flooding a public server with well-formed traffic is expected for an
unauthenticated game relay and generally out of scope.
