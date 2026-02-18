# Identity Provider (IdP) Setup

## Overview

We selected [Thunder](https://github.com/asgardeo/thunder) as the Identity Provider for this project. Thunder is a lightweight, developer-friendly identity and access management solution.

## Getting Started

To start the IdP server:

```bash
docker compose up
```

### Developer Console Access

Once the services are running, you can access the Thunder developer console at `http://localhost:8090/develop` with the following credentials:

- **Username:** `admin`
- **Password:** `1234`

## Bootstrap Scripts

The docker-compose setup mounts two bootstrap scripts into the Thunder container:

```yaml
- ./01-default-resources.sh:/opt/thunder/bootstrap/01-default-resources.sh
- ./02-sample-resources.sh:/opt/thunder/bootstrap/02-sample-resources.sh
```

These scripts automatically configure Thunder on first startup:

- **`01-default-resources.sh`**: Creates the default organization unit, user schema (Person), admin user, system resource server, admin role, and default authentication/registration flows
- **`02-sample-resources.sh`**: Sets up sample resources including:
  - **Traders Organization Unit** - for trader accounts
  - **Trader User Type** - user schema with custom fields (username, email, given_name, family_name)
  - **Trader Portal App** - Single Page React application with OAuth2/OIDC configuration

## Current Setup

The following resources have been configured:

- ✅ Default organization unit
- ✅ Traders organization unit
- ✅ Trader user type (schema)
- ✅ Trader Portal application (React SPA with client ID: `TRADER_PORTAL_APP`)
- ✅ OAuth2 configuration with PKCE for public clients

## Notes

- The Trader Portal React app is configured to run on `http://localhost:5173`
- Client ID will be displayed in the logs after successful creation
- All data is persisted in the `thunder-db` Docker volume
