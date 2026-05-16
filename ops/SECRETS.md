# Production secret layout

Production deploys load every sensitive value (database password, JWT secret,
message-encryption key, MinIO root password, VAPID private key, FCM service
account JSON) from files mounted into the containers via Docker Compose
`secrets:`, not from environment variables. This document covers the one-time
operator setup and the migration from the previous `.env`-only deployment.

The non-sensitive bits (POSTGRES_USER, ALLOWED_ORIGINS, REGISTRY, TAG, VAPID
public key, etc.) stay in `~/nothing/.env` as before.

## Why

Before:

```
~/nothing/.env                       # mode 664, vpmuser can `cat`
   JWT_SECRET=...                    # in every `docker inspect messenger-backend`
   MESSAGE_ENCRYPTION_KEY=...        # ditto
   POSTGRES_PASSWORD=...             # ditto
   MINIO_ROOT_PASSWORD=...           # ditto
   VAPID_PRIVATE_KEY=...             # ditto
~/nothing/fcm-credentials.json       # mode 644, world-readable on the host
```

Any SSH session as `vpmuser` was a one-`cat` exfiltration of every secret.

After:

```
/etc/messenger/secrets/              # mode 750, root:docker
   postgres_password                 # mode 640
   minio_root_password               # mode 640
   jwt_secret                        # mode 640
   message_encryption_key            # mode 640
   vapid_private_key                 # mode 640
   db_url                            # mode 640
   fcm_credentials.json              # mode 640
~/nothing/.env                       # only non-sensitive values left
```

* The `secrets/` directory is `/etc/messenger/` (owned by root) — cleanly
  outside `vpmuser`'s home, no way to land there by accident.
* Mode `640` + group `docker` means the Docker daemon (root) and members of
  the `docker` group can read; anyone else (other users, scripts running
  unprivileged) cannot.
* `docker inspect messenger-backend` no longer prints values — only the
  `*_FILE` paths into `/run/secrets/`. To read the actual secret you have
  to be inside the container or on the host with at least `docker` group
  membership.

`vpmuser` today is in the `docker` group (it has to be, to run `docker compose`
without `sudo`). That means `vpmuser` can still `cat` the files via group
permission — but anyone in the `docker` group is effectively root anyway
(`docker run -v /:/host` mounts the host filesystem read-write). So this is
the equilibrium: the file permissions accurately reflect the real trust
boundary, instead of pretending the whole filesystem is sensitive.

## First-time setup (run as root on the production host)

```sh
# 0. Move into the project dir as a sanity check
cd /home/vpmuser/nothing

# 1. Create the secrets directory with correct ownership and mode
sudo install -d -m 0750 -o root -g docker /etc/messenger/secrets

# 2. Lift each value out of the current .env into its own file.
#    The grep|cut sequence strips the `KEY=` prefix and any surrounding quotes.
#    Files are written with mode 0640 root:docker.
extract() {
  local key="$1" out="$2"
  grep "^${key}=" .env | cut -d= -f2- | sed -e 's/^"//' -e 's/"$//' \
    | sudo tee "/etc/messenger/secrets/${out}" >/dev/null
  sudo chmod 0640 "/etc/messenger/secrets/${out}"
  sudo chown root:docker "/etc/messenger/secrets/${out}"
}

extract POSTGRES_PASSWORD       postgres_password
extract MINIO_ROOT_PASSWORD     minio_root_password
extract JWT_SECRET              jwt_secret
extract MESSAGE_ENCRYPTION_KEY  message_encryption_key
extract VAPID_PRIVATE_KEY       vapid_private_key

# 3. Build the DB URL from the parts already in .env + the password file.
#    `read` strips the trailing newline; we re-add one for the resulting file
#    only because `echo` is the simplest way to write it.
pgu=$(grep '^POSTGRES_USER=' .env | cut -d= -f2)
pgu=${pgu:-messenger}
pgd=$(grep '^POSTGRES_DB=' .env | cut -d= -f2)
pgd=${pgd:-messenger}
pgp=$(sudo cat /etc/messenger/secrets/postgres_password)
echo "postgres://${pgu}:${pgp}@postgres:5432/${pgd}?sslmode=disable" \
  | sudo tee /etc/messenger/secrets/db_url >/dev/null
sudo chmod 0640 /etc/messenger/secrets/db_url
sudo chown root:docker /etc/messenger/secrets/db_url

# 4. Move the FCM credentials file in. The old volume mount in compose is gone;
#    it's now a Docker secret.
sudo cp fcm-credentials.json /etc/messenger/secrets/fcm_credentials.json
sudo chmod 0640 /etc/messenger/secrets/fcm_credentials.json
sudo chown root:docker /etc/messenger/secrets/fcm_credentials.json

# 5. Verify each file looks right
sudo ls -l /etc/messenger/secrets/
# expected: drwxr-x---  root docker  and  -rw-r----- root docker for each file
```

Verify it worked from `vpmuser` without `sudo`:

```sh
ls -l /etc/messenger/secrets/   # works — directory is r-x for docker group
cat /etc/messenger/secrets/jwt_secret  # works too — vpmuser is in docker group

# Now switch identities to verify protection against non-docker users:
sudo -u nobody cat /etc/messenger/secrets/jwt_secret
# → "Permission denied" — nobody is not root and not in docker group
```

## After setup: trim the .env

The `.env` no longer needs (and **must not** contain) the sensitive values.
Comment-out or delete the following keys — they're now in `/etc/messenger/secrets/`:

```
# POSTGRES_PASSWORD       — moved to /etc/messenger/secrets/postgres_password
# MINIO_ROOT_PASSWORD     — moved to /etc/messenger/secrets/minio_root_password
# JWT_SECRET              — moved to /etc/messenger/secrets/jwt_secret
# MESSAGE_ENCRYPTION_KEY  — moved to /etc/messenger/secrets/message_encryption_key
# VAPID_PRIVATE_KEY       — moved to /etc/messenger/secrets/vapid_private_key
```

Keep the rest (POSTGRES_USER, POSTGRES_DB, ALLOWED_ORIGINS, MINIO_ROOT_USER,
MINIO_BUCKET_NAME, REGISTRY, TAG, VAPID_PUBLIC_KEY, VAPID_SUBJECT, etc.) —
those are not sensitive.

Optional but recommended: tighten the remaining `.env` since it still tells
attackers a lot about the deployment shape:

```sh
chmod 0640 ~/nothing/.env
```

## Roll out

```sh
cd ~/nothing
# Pull the latest docker-compose.prod.yml from main (already references secrets:)
git pull origin main   # or however you ship config to the host
# Restart with the new layout
docker compose -f docker-compose.prod.yml up -d
docker compose -f docker-compose.prod.yml logs --tail=50 backend
# Backend log should show normal startup. If you see "JWT_SECRET is not set" or
# similar, the secret file path doesn't match — re-check step 2 above.
```

## Rotating a secret

To rotate, e.g., the JWT secret:

```sh
NEW=$(openssl rand -base64 48 | tr -d '\n')
echo -n "$NEW" | sudo tee /etc/messenger/secrets/jwt_secret >/dev/null
sudo chmod 0640 /etc/messenger/secrets/jwt_secret
sudo chown root:docker /etc/messenger/secrets/jwt_secret
docker compose -f docker-compose.prod.yml restart backend
```

Note: rotating `JWT_SECRET` invalidates every existing user session (since
existing tokens were signed with the old secret). Rotating
`MESSAGE_ENCRYPTION_KEY` makes existing messages unreadable; do not do it
until the messages-migration plan is in place.

## Backups

`/etc/messenger/secrets/` is **not** in git, **not** in CI artifacts, and not
included in the regular `postgres_data` / `minio_data` volume backups.
Back it up manually to an encrypted location (USB stick, GPG-encrypted blob
in S3, password manager attachments) the first time you set it up. Without
it, the postgres data and message ciphertexts in `messages.text` are
irrecoverable.

## CI / autodeploy

The `deploy` job in `.github/workflows/ci.yml` rsyncs `docker-compose.prod.yml`
and `nginx-proxy/` to the host and runs `docker compose up -d`. It deliberately
does **not** touch `/etc/messenger/secrets/` — secrets never travel through
CI. The CI run will simply fail to start the backend if the secret files are
missing on the host. So: run the first-time setup above **before** the first
CI deploy of the new compose.
