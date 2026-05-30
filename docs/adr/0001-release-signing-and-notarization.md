# ADR 0001: Release Signing And Notarization Strategy

Status: accepted
Date: 2026-05-30

## Context

NexusDesk needs a trustworthy release path for Windows, macOS, and Linux. The app already generates release manifests, SHA-256 hashes, SBOM files, provenance files, and release trust diagnostics, but public distribution still depends on platform trust mechanisms that require external certificates, accounts, and clean-machine validation.

Private beta builds may be unsigned while the signing accounts are being acquired, but they must not be presented as production-ready public releases.

## Decision

Windows public releases will use an organization-owned code-signing certificate for the executable and installer. The preferred path is an OV or EV certificate with hardware-backed key storage when available. Windows packages must be signed with `signtool`, include an RFC 3161 timestamp, and keep the signing identity, timestamp server, artifact SHA-256, manifest, SBOM, provenance, and smoke evidence together in release evidence.

Windows private beta zip artifacts may remain unsigned only when release notes clearly say they are private test artifacts and users are given the manifest/SBOM/provenance verification path. Unsigned beta artifacts must not be advertised as production-ready.

macOS public releases will use an Apple Developer ID Application certificate for the app bundle and Developer ID Installer certificate when a `.pkg` is produced. Public macOS artifacts must be notarized through Apple notary service, stapled where applicable, and smoke-tested from a clean account with quarantine behavior recorded.

macOS private beta artifacts may be unsigned or ad-hoc signed only for internal testers who understand the trust prompt state. Release notes must document signing/notarization status, Gatekeeper/quarantine expectations, SHA-256 verification, and clean-machine smoke status.

Linux beta and public packages will use the documented Linux trust path until a signed package repository exists: artifact SHA-256 manifest, SBOM, provenance, package format notes, dependency notes, and clean-machine smoke evidence. If deb/rpm repository distribution is introduced later, repository signing becomes part of the release gate.

## Consequences

Signing implementation remains blocked until the Windows certificate and Apple Developer account/certificates are available.

Release readiness checks should continue to block production release when signed/notarized evidence is absent for platforms that require it.

The Windows installer, Windows executable, and macOS signing/notarization tracker rows remain open until signed artifacts are produced and verified.

Private beta release notes must keep trust limitations visible and point users to release evidence rather than antivirus or platform bypass instructions.
