# 2026-09-21 — PS0019 env quotes

Nine lines: strip one matching pair of surrounding quotes in ReadEnvFile.
godotenv declined — `${VAR}` expansion in a secrets file is not wanted and
the only real complaint was quotes. The bug was real: the TOTP issuer
showed `"Proto App"` in authenticator apps.
