# Passkey Implementation Plan

## Overview

This document outlines the implementation plan for adding WebAuthn/Passkeys authentication to the ProtoStack-ReactGo application. Passkeys provide a more secure and user-friendly alternative to traditional password-based authentication.

## Current System Analysis

The application currently supports:
- Username/password authentication
- Optional TOTP-based MFA
- JWT-based sessions with CSRF protection
- PostgreSQL user storage
- React frontend with Mantine UI components

## Implementation Complexity

**Level: Medium-High**
**Estimated Timeline: 2-3 weeks**
**Key Dependencies:**
- Backend: `github.com/go-webauthn/webauthn v0.10.0`
- Frontend: `@simplewebauthn/browser v9.0.0`

## Phase 1: Database Schema Changes

### New Table: `webauthn_credentials`

```sql
CREATE TABLE webauthn_credentials (
    credential_id VARCHAR(255) PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES app_user(app_user_id),
    public_key BYTEA NOT NULL,
    sign_count BIGINT NOT NULL DEFAULT 0,
    attestation_object BYTEA,
    authenticator_data BYTEA,
    credential_type VARCHAR(50) NOT NULL DEFAULT 'public-key',
    transports TEXT[], -- USB, NFC, BLE, INTERNAL
    created_at NUMERIC(17,7) NOT NULL,
    last_used_at NUMERIC(17,7),
    device_name VARCHAR(100),
    aaguid BYTEA -- Authenticator Attestation GUID
);

-- Index for efficient user credential lookups
CREATE INDEX idx_webauthn_credentials_user_id ON webauthn_credentials(user_id);

-- Add schema permissions (following existing pattern)
GRANT USAGE ON SCHEMA biz TO group_global_writer;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA biz TO group_global_writer;
GRANT ALL ON ALL SEQUENCES IN SCHEMA biz TO group_global_writer;
```

### Migration Script

Create `sql/021-webauthn-credentials.sql` with the above table definition and grants.

## Phase 2: Backend Implementation

### 2.1 Dependencies and Configuration

**Add to `src/api-srv/go.mod`:**
```go
require (
    github.com/go-webauthn/webauthn v0.10.0
)
```

**Add to configuration:**
```go
// Config struct additions
WebAuthnDisplayName string // "ProtoStack App"
WebAuthnID          string // "localhost" or production domain
WebAuthnOrigin      string // "http://localhost:8080" or production URL
```

### 2.2 Repository Layer

**New File: `src/api-srv/internal/repository/webauthn_repository.go`**

```go
type WebAuthnCredential struct {
    CredentialID      string
    UserID           int
    PublicKey        []byte
    SignCount        int64
    AttestationObject []byte
    AuthenticatorData []byte
    Transports       []string
    CreatedAt        float64
    LastUsedAt       *float64
    DeviceName       *string
    AAGUID          []byte
}

type WebAuthnRepository struct {
    db *pgxpool.Pool
}

func NewWebAuthnRepository(db *pgxpool.Pool) *WebAuthnRepository {
    return &WebAuthnRepository{db: db}
}

func (r *WebAuthnRepository) StoreCredential(ctx context.Context, cred *WebAuthnCredential) error {
    // INSERT implementation
}

func (r *WebAuthnRepository) GetCredentialsByUserID(ctx context.Context, userID int) ([]*WebAuthnCredential, error) {
    // SELECT implementation
}

func (r *WebAuthnRepository) GetCredentialByID(ctx context.Context, credentialID string) (*WebAuthnCredential, error) {
    // SELECT implementation
}

func (r *WebAuthnRepository) UpdateSignCount(ctx context.Context, credentialID string, signCount int64) error {
    // UPDATE implementation
}

func (r *WebAuthnRepository) DeleteCredential(ctx context.Context, credentialID string, userID int) error {
    // DELETE implementation
}
```

### 2.3 Model Layer

**New File: `src/api-srv/internal/model/webauthn_model.go`**

```go
// WebAuthn registration request/response models
type WebAuthnRegisterBeginRequest struct {
    UserName string `json:"userName"`
}

type WebAuthnRegisterBeginResponse struct {
    Challenge        string                   `json:"challenge"`
    User             webauthn.User            `json:"user"`
    ExcludeCredentials []webauthn.Credential  `json:"excludeCredentials"`
    AuthenticatorSelection webauthn.AuthenticatorSelection `json:"authenticatorSelection"`
}

// Additional models for auth flows, credential management, etc.
```

### 2.4 Handler Layer

**New File: `src/api-srv/internal/handler/webauthn_handler.go`**

#### WebAuthn Registration Handlers

```go
func WebAuthnRegisterBeginHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository, waRepo *repository.WebAuthnRepository) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Implementation for starting passkey registration
    }
}

func WebAuthnRegisterFinishHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository, waRepo *repository.WebAuthnRepository) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Implementation for completing passkey registration
    }
}
```

#### WebAuthn Authentication Handlers

```go
func WebAuthnAuthenticateBeginHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository, waRepo *repository.WebAuthnRepository) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Implementation for starting passkey authentication
    }
}

func WebAuthnAuthenticateFinishHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository, waRepo *repository.WebAuthnRepository) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Implementation for completing passkey authentication
    }
}
```

#### Credential Management Handlers

```go
func WebAuthnListCredentialsHandler(appServices *app_lib.AppServices, waRepo *repository.WebAuthnRepository) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Implementation for listing user's passkeys
    }
}

func WebAuthnDeleteCredentialHandler(appServices *app_lib.AppServices, waRepo *repository.WebAuthnRepository) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Implementation for deleting a passkey
    }
}
```

### 2.5 Server Integration

**Update `src/api-srv/cmd/server/main.go`:**

```go
// Initialize WebAuthn
webAuthn, err := webauthn.New(&webauthn.Config{
    RPDisplayName: config.WebAuthnDisplayName,
    RPID:          config.WebAuthnID,
    RPOrigins:     []string{config.WebAuthnOrigin},
})
if err != nil {
    logger.Error("WAINIT", "Failed to initialize WebAuthn", "username", "app-root")
    os.Exit(1)
}

// Initialize repository
waRepo := repository.NewWebAuthnRepository(dbPool)

// Create handlers
waRegisterBeginHandler := handler.WebAuthnRegisterBeginHandler(appServices, userRepo, waRepo, webAuthn)
waRegisterFinishHandler := handler.WebAuthnRegisterFinishHandler(appServices, userRepo, waRepo, webAuthn)
waAuthBeginHandler := handler.WebAuthnAuthenticateBeginHandler(appServices, userRepo, waRepo, webAuthn)
waAuthFinishHandler := handler.WebAuthnAuthenticateFinishHandler(appServices, userRepo, waRepo, webAuthn)
waListHandler := handler.WebAuthnListCredentialsHandler(appServices, waRepo)
waDeleteHandler := handler.WebAuthnDeleteCredentialHandler(appServices, waRepo)

// Register routes
mux.HandleFunc("POST /webauthn/register/begin", waRegisterBeginHandler)
mux.HandleFunc("POST /webauthn/register/finish", waRegisterFinishHandler)
mux.HandleFunc("POST /webauthn/authenticate/begin", waAuthBeginHandler)
mux.HandleFunc("POST /webauthn/authenticate/finish", waAuthFinishHandler)
mux.HandleFunc("GET /webauthn/credentials", waListHandler)
mux.HandleFunc("DELETE /webauthn/credentials/{credentialID}", waDeleteHandler)
```

## Phase 3: Frontend Implementation

### 3.1 Dependencies

**Update `src/web-app/package.json`:**
```json
{
  "dependencies": {
    "@simplewebauthn/browser": "^9.0.0",
    "@simplewebauthn/typescript-types": "^9.0.0"
  }
}
```

### 3.2 Type Definitions

**Update `src/web-app/types/auth-type.ts`:**
```typescript
export interface PasskeyCredential {
  credentialId: string;
  deviceName?: string;
  createdAt: number;
  lastUsedAt?: number;
}

export interface WebAuthnRegisterBeginResponse {
  challenge: string;
  user: {
    id: string;
    name: string;
    displayName: string;
  };
  excludeCredentials: Array<{
    id: string;
    type: string;
  }>;
  authenticatorSelection: {
    authenticatorAttachment?: string;
    residentKey?: string;
    userVerification?: string;
  };
}

export interface WebAuthnAuthenticateBeginResponse {
  challenge: string;
  allowCredentials: Array<{
    id: string;
    type: string;
  }>;
}
```

### 3.3 API Utilities

**Update `src/web-app/utils/api.ts`:**
```typescript
export const webauthnApi = {
  registerBegin: (userName: string) =>
    apiRequest('/webauthn/register/begin', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ userName }),
    }),

  registerFinish: (data: any) =>
    apiRequest('/webauthn/register/finish', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    }),

  authenticateBegin: () =>
    apiRequest('/webauthn/authenticate/begin', { method: 'POST' }),

  authenticateFinish: (data: any) =>
    apiRequest('/webauthn/authenticate/finish', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    }),

  listCredentials: () =>
    apiRequest('/webauthn/credentials'),

  deleteCredential: (credentialId: string) =>
    apiRequest(`/webauthn/credentials/${credentialId}`, {
      method: 'DELETE',
    }),
};
```

### 3.4 Auth Store Updates

**Update `src/web-app/stores/auth-store.ts`:**
```typescript
interface AuthStore {
  // ... existing properties

  // Passkey methods
  registerPasskey: (userName: string) => Promise<void>;
  authenticateWithPasskey: () => Promise<void>;
  listPasskeys: () => Promise<PasskeyCredential[]>;
  deletePasskey: (credentialId: string) => Promise<void>;
}

// Implementation
registerPasskey: async (userName: string) => {
  // WebAuthn registration flow
},

authenticateWithPasskey: async () => {
  // WebAuthn authentication flow
},

listPasskeys: async () => {
  // List user's passkeys
},

deletePasskey: async (credentialId: string) => {
  // Delete a passkey
},
```

### 3.5 UI Components

#### Passkey Registration Modal
**New File: `src/web-app/components/PasskeyRegistrationModal.tsx`**
```typescript
interface PasskeyRegistrationModalProps {
  opened: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export function PasskeyRegistrationModal({
  opened,
  onClose,
  onSuccess,
}: PasskeyRegistrationModalProps) {
  // Implementation for passkey registration UI
}
```

#### Passkey List Component
**New File: `src/web-app/components/PasskeyList.tsx`**
```typescript
interface PasskeyListProps {
  credentials: PasskeyCredential[];
  onDelete: (credentialId: string) => void;
}

export function PasskeyList({ credentials, onDelete }: PasskeyListProps) {
  // Implementation for displaying and managing passkeys
}
```

### 3.6 Login Page Updates

**Update `src/web-app/pages/login.tsx`:**
```typescript
// Add passkey authentication option
const handlePasskeyLogin = async () => {
  await authenticateWithPasskey();
};

// Add passkey button to login form
<Button
  fullWidth
  variant="outline"
  leftIcon={<IconFingerprint />}
  onClick={handlePasskeyLogin}
  loading={loading}
  mb="md"
>
  Sign in with Passkey
</Button>
```

## Phase 4: User Experience Flow

### Registration Flow
1. User authenticates with password → redirected to dashboard
2. User visits user preferences → "Add Passkey" option
3. Modal guides user through WebAuthn registration
4. Passkey stored and associated with user account

### Authentication Flow
1. User visits login page → sees passkey option above password form
2. Clicks "Sign in with Passkey" → browser prompts for biometric verification
3. Successful authentication → user logged in with JWT session

### Fallback Behavior
- Passkeys are optional (users can still use passwords)
- Graceful degradation for unsupported browsers/devices
- Clear error messages for failed WebAuthn operations

## Phase 5: Security Considerations

### Credential Storage
- Public keys only (private keys never leave user device)
- Encrypted storage of public keys in database
- Proper indexing for efficient lookups

### Authentication Validation
- Challenge-response protocol prevents replay attacks
- Sign count validation prevents cloned credential attacks
- Origin validation prevents cross-site attacks

### Privacy Protection
- No biometric data stored (only public keys)
- Users can delete passkeys at any time
- No tracking of authentication attempts

### Error Handling
- User-friendly error messages for WebAuthn failures
- Automatic fallback to password authentication
- Proper cleanup of failed registration attempts

## Phase 6: Testing Strategy

### Unit Tests
- WebAuthn repository CRUD operations
- Handler validation logic
- Frontend WebAuthn API integration

### Integration Tests
- Complete registration flow
- Complete authentication flow
- Error scenarios and fallbacks
- Cross-browser compatibility

### Manual Testing
- Various authenticator types (platform, cross-platform)
- Different browsers and devices
- Error recovery scenarios

## Phase 7: Migration and Rollout

### Backward Compatibility
- Existing users continue with password authentication
- Passkeys are opt-in feature
- No breaking changes to existing API

### Gradual Rollout
1. Deploy backend changes with feature flags disabled
2. Enable passkey registration for new users
3. Add UI components with feature flags
4. Enable for existing users
5. Make passkeys the default authentication method

### Feature Flags
```go
// Configuration additions
EnablePasskeys bool // Default: false
```

## Phase 8: Documentation and Training

### User Documentation
- How to register passkeys
- How to manage passkeys
- Troubleshooting common issues
- Browser compatibility information

### Developer Documentation
- API endpoint specifications
- WebAuthn integration details
- Security considerations
- Testing procedures

## Timeline and Milestones

### Week 1: Foundation
- [ ] Database schema and migration
- [ ] Backend dependencies and basic setup
- [ ] Repository layer implementation

### Week 2: Core Implementation
- [ ] Handler layer implementation
- [ ] Frontend dependencies and types
- [ ] Basic WebAuthn integration

### Week 3: UI and Testing
- [ ] Frontend components and UI updates
- [ ] Integration testing
- [ ] Error handling and fallbacks

### Week 4: Polish and Deployment
- [ ] User experience refinements
- [ ] Documentation updates
- [ ] Production deployment preparation

## Success Criteria

- [ ] Users can register passkeys during account creation
- [ ] Users can authenticate using passkeys
- [ ] Graceful fallback to password authentication
- [ ] Proper error handling for WebAuthn failures
- [ ] Passkeys work across supported browsers and devices
- [ ] All existing functionality continues to work
- [ ] Security audit passes

## Risks and Mitigations

### Browser Compatibility
**Risk:** Not all users have WebAuthn-compatible browsers/devices
**Mitigation:** Optional feature with clear password fallback

### Security Implementation
**Risk:** Incorrect WebAuthn implementation could introduce vulnerabilities
**Mitigation:** Use established libraries, thorough testing, security review

### User Adoption
**Risk:** Users may be confused by new authentication method
**Mitigation:** Clear UI, helpful tooltips, optional adoption

## References

- [WebAuthn Specification](https://www.w3.org/TR/webauthn-2/)
- [FIDO Alliance Resources](https://fidoalliance.org/)
- [WebAuthn Guide](https://webauthn.guide/)
- [Go WebAuthn Library](https://github.com/go-webauthn/webauthn)
- [SimpleWebAuthn](https://simplewebauthn.dev/)</content>
<parameter name="filePath">doc/120-passkey-plan.md