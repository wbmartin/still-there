# To do

Still There: a Raspberry Pi base station in an elderly person's home
watches a BLE wristband and checks in with this server; the server texts
a contact list when a check-in is missed, the band is absent, or the band
looks taken off. This file is the checklist until the app has a Radley
House tracker (`etc/.radley-house`, DERIVING.md section 2).

Proof of concept first. Branches: `feature/*` -> `dev` -> `main`, and
`main` deploys to prd-wbm; no UAT (CLAUDE.md, Branches).

## Decided

- Pi: Raspberry Pi 3 Model B, Raspbian 10 (buster) 32-bit, wired to the
  home router. Agent is a Go binary (`GOOS=linux GOARCH=arm GOARM=7`)
  run by systemd.
- Band: Minew B7 (firmware "Beacon Plus" 2.3.17), tracked by its static
  address `C3:00:00:98:F3:19` for the PoC. ACC (accelerometer) slot to be
  enabled in BeaconSET+ for stillness. A Minew B6 was ordered; revisit
  when it arrives. The scanner matches by address or ID, not by model.
- Band powered off reads as absent: same alarm as the wearer leaving.
- SMS: Telnyx, behind a sender interface. Start toll-free verification
  early; it takes days to weeks (AWS End User Messaging is up to 15
  business days too, so no gain there).
- Alarm settings live on the server, per device, and are pushed to the Pi.
  Starting values: check-in every 60 s, missed-check-in alarm after 3,
  band-absent alarm after 10 min unseen.
- Tamper is in scope: band still while present (off-wrist), Pi missed
  check-ins (unplugged, offline). Pi case switch later.

## Waiting on staff

- [ ] Enable the B7 ACC slot in BeaconSET+ (keep address tracking).
- [ ] Open a Telnyx account; buy a toll-free number; submit toll-free
      verification.
- [ ] Check whether BeaconSET+ can disable power-off by button.
- [ ] B6: does it have off-wrist / strap-cut detection? Need the listing
      or datasheet.

## 1. Pi agent (`src/pi-agent/`, own Go module)

- [x] BLE scan via `tinygo.org/x/bluetooth` (BlueZ over D-Bus); per
      configured address: last seen, smoothed RSSI, present/absent.
- [x] Local config file (JSON): server URL, device token, interval,
      thresholds, band list, config version.
- [x] Check-in loop: POST state; immediate check-in on a present/absent
      change.
- [x] systemd unit; `bin/` script to cross-compile and copy to the Pi.
- [x] Verify on the Pi: band on/off shows present/absent in the log
      (2026-10-24: present 09:17:44, absent 09:19:32, present 09:20:52;
      address unchanged across power cycles). Service installed on the
      Pi by bin/630, running as user stillthere; back by itself after a
      Pi reboot (2026-10-24 09:26).

## 2. Server

Design agreed 2026-10-24 (staff):

- Device = the Pi. It holds the token (shown once at issue, stored as a
  SHA-256 hash, reissue kills the old one). Bands carry no credential;
  the Pi names them by address.
- Contacts are per device (one household). Added by an admin in the
  tenant; the contact gets an invite text and is active only after
  replying Y. STOP opts out of everything; the admin UI shows it, and
  only the contact texting START re-enables (carriers block sends after
  STOP). HELP answers with who we are.
- Tenant settings: time zone, reminder interval (minutes), daily
  reminder time.
- Alarm life: raised -> text all; repeat every reminder interval until
  acknowledged. Acknowledged (reply OK, or in the UI) -> text all "<who>
  acknowledged"; then one reminder a day at the tenant's time while
  still open. Admin turns it off in the UI -> text all "<who> turned
  off"; no more reminders. Band seen again -> resolved; text all
  "alerting re-activated for <band label>" with who acknowledged or
  turned it off. A band absence raises again only after the band was
  seen.
- Reply OK acknowledges every open alarm on the households that number
  is a contact for.
- Per-contact notification preferences per event: raised, reminder,
  acknowledged, turned off, re-activated, daily reminder, device
  offline.
- Check-in history kept in full for the PoC; pruning later.
- Until Telnyx can reach a public server: SMS_PROVIDER=log and a
  dev-only "simulate reply" endpoint. Telnyx goes live on the server
  `prd-wbm`.

Commits on `feature/server-checkin`, each passing `bin/300-run-tests all`:

- [x] 1. Schema and grants (migrations 0008, 0009).
- [x] 2. Device API: CRUD, `/deviceToken`, `/deviceCheckin`.
- [x] 3. Watchdog, SMS sender (log), inbound replies, simulate endpoint.
- [x] 4. Admin pages: devices, bands, contacts, alerts, tenant settings
      (with page help).
- [x] Telnyx sender and signed inbound webhook (code and unit tests;
      not yet live: needs the Telnyx keys and a public URL on prd-wbm).

## 3. Config push

- [ ] Check-in reply carries config version, and the config when it
      changed; Pi applies, saves, acks the version next check-in.

## 4. Tamper

- [ ] Parse B7 ACC frames; stillness alarm while present; quiet-hours
      threshold so sleep does not alarm.
- [ ] Later: Pi case switch on GPIO.

## PoC done when

- [ ] Band off -> text within about 10 min.
- [ ] Pi unplugged -> text within about 3 min.
- [ ] Band still -> text after the stillness threshold.
- [ ] Admin config change reaches the Pi within one check-in.
- [ ] Security doc written before `dev` -> `main`.

## Known issues

- [ ] e2e flake (2026-10-24): twice in five runs the e2e API on :8082
      stopped answering about a minute in (ECONNREFUSED from Vite's
      proxy, no panic in the Playwright log), failing most of the suite;
      a rerun passed all 111. Cause not found yet.

## Host work

The one list of hand steps on a host.

- [ ] **H1. prd-wbm, API env file: the monitoring keys.** No separate
  step any more: `SMS_PROVIDER=log` and `WATCHDOG_TICK_SECONDS=30` ship
  in `infra/prd-wbm/etc/prd-stillthere-api.env`, installed by
  `doc/320-prd-wbm-deploy.md` step 8 (H2). Live texts (the four
  `TELNYX_*` keys, set on the host only, and Telnyx's webhook URL
  `https://webhook.teamtaggle.com/telnyx-inbound`) are doc/320 step 17,
  with its check and rollback.
- [ ] **H2. prd-wbm, first install of still-there** beside gsd-notes:
  `doc/320-prd-wbm-deploy.md`, First install, steps 1-16 (DNS A records
  for `stillthere.teamtaggle.com` and `webhook.teamtaggle.com`, user
  `svc-stillthere`, group `stillthere-deploy`, `/srv/prd-stillthere`,
  database `prd_stillthere_db`, unit, webroot certificates, the two
  nginx sites, first release, first admin, smoke checks) and 19. Needs
  H3 first. Check and rollback: each step's own. Optional after it:
  step 17 (Telnyx) and step 18 (fail2ban jail, once gsd-notes has
  installed fail2ban).
- [ ] **H3. prd-wbm, prerequisite (gsd-notes' host work):** gsd's
  certificate renews by webroot, with no nginx stop hook under
  `/etc/letsencrypt/renewal-hooks/pre` or `post` and a `deploy` hook
  that reloads nginx (gsd-notes doc/320, Certificates). Otherwise this
  app's webroot renewals fail. Check: doc/320 step 4.
- Backups: not set up yet (gsd-notes doc/380 is the pattern).

Each entry: host and change, command, check, rollback. Agents add
entries and never run or tick them; only a staff member's report ticks one.
