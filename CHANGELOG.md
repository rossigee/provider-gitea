# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial provider implementation
- Repository management support
- Organization management support
- User management support (admin only)
- Webhook management support
- Comprehensive documentation and examples
- Build and deployment scripts

### Features
- **Repository Management**: Create, update, and delete Git repositories
- **Organization Management**: Manage organizations and their settings
- **User Management**: User account management (requires admin privileges)
- **Webhook Management**: Configure repository and organization webhooks
- **Provider Configuration**: Flexible authentication and connection configuration
- **Examples**: Complete working examples for all resource types

### Technical Details
- Built on Crossplane Runtime v1.15.0
- Uses Gitea API v1
- Supports both user and organization repositories
- SSL verification configurable
- Comprehensive error handling and logging