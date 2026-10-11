Optional PostgreSQL CA certificates, never private keys.

If the configured database endpoint requires a Supabase CA, download it from
Dashboard > Database > Settings > SSL and save it here as `supabase-ca.crt`.
Compose mounts this directory read-only at `/etc/portway/certs`.
Append `&sslrootcert=/etc/portway/certs/supabase-ca.crt` to the PostgreSQL DSN
while keeping `sslmode=verify-full`. Do not disable certificate verification.
The current project required the Supabase Root 2021 CA. The checked-in public
certificate was retrieved over HTTPS from the URL referenced by the official
Supabase Dashboard source:
https://github.com/supabase/supabase/blob/master/apps/studio/hooks/custom-content/custom-content.json
https://supabase-downloads.s3-ap-southeast-1.amazonaws.com/prod/ssl/prod-ca-2021.crt

SHA-256 certificate fingerprint:
80:70:25:AD:50:D4:ED:21:9D:2C:9C:7D:29:9C:00:4F:82:4E:B0:0C:F7:F6:5A:FE:F6:07:D0:7B:72:E6:CA:FA
Valid until 2031-04-26. This is a public CA, not a private key or credential.
