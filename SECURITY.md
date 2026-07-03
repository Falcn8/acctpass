# Security Policy

## Supported versions

Only the latest commit on the default branch and the latest GitHub Release are
supported for security fixes.

## Reporting a vulnerability

Please do not open a public GitHub issue for suspected vulnerabilities.

Report security concerns privately to the repository owner. Include:

- affected version or commit
- operating system
- exact command or workflow involved
- impact and reproduction steps
- whether any vault, generated password, or master password may have been
  exposed

If GitHub private vulnerability reporting is enabled for the repository, use
that first.

## Scope

Security-sensitive areas include vault encryption, password derivation,
filesystem permissions, clipboard behavior, release artifacts, and CI security
scanning.

## Disclaimer

`acctpass` is not independently audited. It is provided without warranty and may
not be appropriate for high-value, shared, regulated, or production credential
workflows. Use it only after understanding the risks described in the README.
