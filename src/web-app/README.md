# Web App

React frontend for the ProtoStack application using Vite, TypeScript, and Mantine.

## Development

### Environment Configuration

The API proxy URL is configured via environment variables to support different environments without separate builds.

#### Development
Create or update `.env.development`:
```bash
VITE_API_PROXY_URL=http://localhost:8080
```

#### Production
For production builds, create `.env.production`:
```bash
VITE_API_PROXY_URL=https://api.yourdomain.com
```

If not set, defaults to `http://localhost:8080`.

### Running the Development Server

```bash
npm run dev          # Development mode (uses .env.development)
npm run dev:prod     # Production mode (uses .env.production)
```

### Building

```bash
npm run build        # Development build
npm run build:prod   # Production build
```

### API Proxy

The Vite dev server proxies `/api/*` requests to the configured API endpoint. In production, nginx handles the proxy routing.

### Environment Variables

- `VITE_API_PROXY_URL`: API server URL for proxy configuration
- All `VITE_*` prefixed variables are available in the browser

## Project Structure

- `components/`: Reusable UI components
- `pages/`: Route components
- `stores/`: Zustand state management
- `types/`: TypeScript type definitions
- `utils/`: Utility functions
- `constants/`: API endpoints and configuration