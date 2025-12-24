# mTLS Troubleshooting Guide

This document provides command-line tools to verify the integrity and relationships of mTLS certificates for shrmpl-vault.

## Prerequisites

Ensure you have OpenSSL installed on your system. All commands assume you're in the project root directory.

## Certificate Locations

- Client Certificate: `etc/crt/shrmpl_vault_mtls_client_cert.pem`
- Client Private Key: `etc/crt/shrmpl_vault_mtls_client_privkey.pem`
- CA Certificate: `etc/crt/shrmpl_vault_mtls_ca.pem`
- CA Private Key: `etc/crt/shrmpl_vault_mtls_ca_privkey.pem`

## 1. Verify Client Private Key Matches Client Certificate

This checks if the private key corresponds to the public key in the certificate.

```bash
# Get modulus from private key
openssl rsa -in etc/crt/shrmpl_vault_mtls_client_privkey.pem -noout -modulus

# Get modulus from certificate
openssl x509 -in etc/crt/shrmpl_vault_mtls_client_cert.pem -noout -modulus
```

**Expected Result:** Both modulus values should be identical.

## 2. Verify CA Private Key Matches CA Certificate

This checks if the CA private key corresponds to the public key in the CA certificate.

```bash
# Get modulus from CA private key
openssl rsa -in etc/crt/shrmpl_vault_mtls_ca_privkey.pem -noout -modulus

# Get modulus from CA certificate
openssl x509 -in etc/crt/shrmpl_vault_mtls_ca.pem -noout -modulus
```

**Expected Result:** Both modulus values should be identical.

## 3. Verify CA Signed the Client Certificate

This verifies that the client certificate was signed by the CA and checks the certificate chain.

```bash
# Verify certificate against CA
openssl verify -CAfile etc/crt/shrmpl_vault_mtls_ca.pem etc/crt/shrmpl_vault_mtls_client_cert.pem
```

**Expected Result:** Should output `etc/crt/shrmpl_vault_mtls_client_cert.pem: OK`

### Additional Checks

```bash
# Check issuer of client certificate
openssl x509 -in etc/crt/shrmpl_vault_mtls_client_cert.pem -noout -issuer

# Check subject of CA certificate
openssl x509 -in etc/crt/shrmpl_vault_mtls_ca.pem -noout -subject
```

**Expected Result:** The issuer of the client cert should match the subject of the CA cert.

## Troubleshooting Common Issues

- **Modulus mismatch**: The private key doesn't match the certificate. Regenerate the keypair.
- **Verification fails**: The certificate wasn't signed by the CA or the CA is invalid. Regenerate certificates using the mTLS script.
- **CA extensions missing**: Ensure the CA certificate has `basicConstraints=CA:TRUE`.

## Regeneration

If certificates are invalid, regenerate them:

```bash
./bin/410-regenerate-vault-mtls-certs
```

Then restart the infrastructure:

```bash
./bin/211-stop-shrmpl-app-infra
./bin/210-run-shrmpl-app-infra
```