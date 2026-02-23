import {StrictMode} from 'react'
import {createRoot} from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import "@radix-ui/themes/styles.css";
import {BrowserRouter} from 'react-router-dom';
import {Theme} from '@radix-ui/themes';
import { ErrorBoundary } from './components/ErrorBoundary'
import { AsgardeoProvider } from '@asgardeo/react'

const APP_URL = import.meta.env.VITE_APP_URL || window.location.origin
const CLIENT_ID = import.meta.env.VITE_IDP_CLIENT_ID || 'TRADER_PORTAL_APP'
const IDP_BASE_URL = import.meta.env.VITE_IDP_BASE_URL || 'https://localhost:8090'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ErrorBoundary>
      <AsgardeoProvider
        clientId={CLIENT_ID}
        baseUrl={IDP_BASE_URL}
        platform="AsgardeoV2"
        afterSignInUrl={APP_URL}
        afterSignOutUrl={APP_URL}
        scopes={["openid", "profile", "email"]}
      >
        <Theme scaling="110%">
          <BrowserRouter>
            <App/>
          </BrowserRouter>
        </Theme>
      </AsgardeoProvider>
    </ErrorBoundary>
  </StrictMode>,
)
