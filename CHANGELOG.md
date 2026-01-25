# Changelog

All notable changes to this project will be documented in this file.

## 2026-01-25

### Added
- Introduced the Pushover notification backend, including configuration options for app token, user key, and optional device/sound overrides.
- Documented Pushover setup and usage alongside existing webhook and ntfy backends.
- Added a CLI help flag (`-h`/`--help`) that prints usage details and key environment variables.
- Added end-of-shift notifications that fire when PagerDuty reports you are off call, with full support across webhook, ntfy, and Pushover backends.
- Added the `SHIFT_END_NOTIFICATIONS_ENABLED` environment variable to globally enable or disable the shift-end notifier.
