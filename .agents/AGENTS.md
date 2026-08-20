# AGENTS.md

Shared contract for every agent and person working on Quadis Hotels. Read it
before starting. Update it as you go.

Replaces the separate per-agent context files. Two of those existed, they
disagreed about what was done, and the disagreement caused real damage — see
§7.

**Everything here is something someone checked, not something someone intended.**
If you write a claim into this file, verify it first.

---

## 1. Claim board — write here BEFORE you start

Say who you are and what you are touching. Two collisions have already happened
because nobody did.

| Agent | Working on | Files / resources | Since | Status |
|---|---|---|---|---|
| _(free)_ | | | | |

Rules:
- Claim before the first edit, not after the last one.
- Claim the **resource**, not the intention. "`public/images/**` + anything
  referencing an image path" beats "image optimisation".
- Clear your row when you stop, even if unfinished — say what you left half-done.
- If a row is occupied and you need those files, say so here rather than editing
  around them.

---

## 2. Rules that are not negotiable

1. **Never commit or upload `client-assets/`.** It holds the client's live
   GoDaddy and Razorpay passwords in plaintext. The `client-assets/` line in
   `.gitignore` is the only thing preventing that. Do not narrow it.

   > ⚠️ **`.gitignore` is not preventing it for one file.**
   > `client-assets/property-data.md` was committed in `c4dbefb` *before* the
   > ignore rule existed, and **git does not un-track a file just because a
   > pattern later matches it** — it is on `origin/main` today, in a public
   > repo. Checked 3 Aug: it is lat/lng, metro and landmark data with **no
   > credentials**, so this is untidiness, not a leak. Left in place
   > deliberately; purging it rewrites public history. **The lesson is the
   > general one — `git ls-files | grep client-assets` before you trust the
   > ignore rule.**

1b. **Never put a real credential in a test file, a fixture or a doc.** This
   repo is **public** (§3b). Verify with the parameter itself, not by eye:
   ```
   REAL=$(aws ssm get-parameter --profile quadis-client --region ap-south-1 \
            --name /quadis/admin-pin --with-decryption \
            --query 'Parameter.Value' --output text)
   grep -rlF "$REAL" backend/ src/ docs/ .agents/ deploy/    # must print nothing
   ```
   > **This is not hypothetical. Caught 3 Aug, one command before the push:**
   > `backend/__tests__/adminPin.test.ts` had `BOOTSTRAP_PIN` set to the **live
   > production admin PIN** from `/quadis/admin-pin`. The test sets
   > `process.env.ADMIN_PIN` to that constant itself, so *any* six digits work —
   > the real one bought nothing and cost everything. Git history was checked
   > and it had **never been committed**, so no rotation was needed; the file
   > was untracked and the push would have been the exposure event. A test file
   > is exactly where this hides, because nobody rereads a passing test.
2. **Never touch DNS.** The domain is registered at GoDaddy but its nameservers
   are at theserverindia, so the zone — including the Google MX records — is
   served by the old web host. Repointing without recreating those records takes
   the client's email down. See `docs/dns-cutover.md`.
3. **Do not change occupancy pricing without a written client answer.** It
   decides what guests are billed. Current rule, from 27 Jul: extra adult 30%,
   under-8 free, 8–12 at 20%, 13+ as adult.
4. **Do not build further into booking, payments or the admin panel.** Two
   things are unanswered and either one can discard that work:
   - Whether "Book Now" stays on our site or redirects to the client's PMS. If
     it redirects, CheckoutModal, RazorpayService, InvoiceService and the
     soft-hold engine are all unused.
   - **The client already has a working hotel management system** at
     `adminweb.quadishotels.com` — hotels, bookings, coupons, occupancy, users,
     SEO fields, image upload — holding their real inventory. See §5. We may be
     rebuilding what they already have.
5. **`Ready` is not `working`.** Finish every deploy with a browser check. §6.

---

## 3. Our own account — the OLD stack. ⚠️ STILL RUNNING, and it is a hazard

> **This is no longer where the site lives. §3b is.** Read this section as the
> description of a *second, separate* deployment in **our** AWS account that
> nobody turned off — verified answering on 3 Aug: it serves the whole site and
> its `/api/properties` comes out of its own RDS in a different order from the
> live box. Razorpay no longer points here. It is still a duplicate indexable
> copy of the site and still billing, so it should be wound down deliberately —
> but do **not** delete it as cleanup: its RDS is the only record of anything
> before 1 Aug. See §4.

| Thing | Value |
|---|---|
| AWS account / region | `093650262440` / `us-east-1` |
| Site (**old, still up**) | `https://djqj43186y3yh.cloudfront.net` — HTTPS, `/api/*` proxied, deep links fixed. Guests do not reach it; **Razorpay may still be pointed at it** |
| CloudFront distribution | `E1ZV1EQ1QRKH08` — invalidate `/index.html` and `/` after a frontend upload |
| Deployed version | Frontend `4fa8b1c`, 28 Jul (bundle `index-DvPrWQGR.js`). Backend `co3-84a3e86` — unchanged since, no backend code in `b44ad78`/`4fa8b1c` |
| EB app / environment | `quadis-backend` / `quadis-backend-live` |
| API origin | `http://quadis-backend-live.eba-ekdyt4m3.us-east-1.elasticbeanstalk.com` (HTTP; reach it through CloudFront) |
| RDS | `quadis-db-live`, Postgres 18.3 — seeded: 9 properties, 20 room types, 197 keys |
| **EB instance SG** | **`sg-07e7ba582065ed9e8`** — see the trap below |
| RDS SG | `sg-03e1c65d4487ac04a` — 5432 open only to the instance SG |
| Frontend bucket | `quadis-hotels-test-co3` |
| Photo bucket | `quadis-hotel-images` |
| Frontend build flag | `VITE_API_URL=https://djqj43186y3yh.cloudfront.net/api` |

> **Security-group trap.** `describe-configuration-settings` returns *two*
> values for `SecurityGroups`. The obvious one, `sg-0f03fb094dfbada9c`, is the
> **load balancer**. Granting RDS access to it instead of the instance group
> cuts the application off from its own database — health checks keep passing
> while every data call hangs for forty seconds. This has happened once already.
> Resolve it from the instance:
> ```
> aws ec2 describe-instances --instance-ids \
>   $(aws elasticbeanstalk describe-environment-resources \
>       --environment-name quadis-backend-live \
>       --query 'EnvironmentResources.Instances[].Id' --output text) \
>   --query 'Reservations[].Instances[].SecurityGroups[].[GroupId,GroupName]'
> ```

---

## 3b. The client's own account — built 30 Jul 2026

> **Which deployment doc is current.** Four of them describe four different
> architectures, so read them in this order and no other:
>
> | File | Status |
> |---|---|
> | **AGENTS.md §3b + §4 (here)** | **Production. The only current description.** |
> | `deploy/` | The actual scripts. `build-artifact.sh` → `push.sh` → `cutover.sh` |
> | `docs/dns-zone-live-capture.txt` | Her complete zone, record by record |
> | `docs/aws-account-migration.md` | **DNS, mail and env-vars only** — its build steps are superseded |
> | `docs/DEPLOYMENT.md` | Keep for env-var and occupancy-pricing reference |
> | `docs/HANDOFF-aws-deploy.md` | History. Had a wrong security-group ID; corrected in place |
> | `docs/AWS_DEPLOYMENT_GUIDE.md` | History, and local-only (gitignored) |

Her account, `266877689020`, `ap-south-1`. Everything below is live and was
verified, not assumed. **Nothing here touches DNS** — `quadishotels.com` still
resolves to `115.124.108.190` and the MX is still Google, confirmed after the
build.

| Thing | Value |
|---|---|
| Instance | `i-0d126c49ffdfe1668` — **`t3.medium`**, resized 31 Jul (see below) |
| Account plan | **PAID** since 31 Jul. Was `FREE`, which is what blocked the resize |
| **Elastic IP** | **`13.234.85.127`** — the address that goes in her A record at cutover |
| Security group | `sg-0b19531290e391e17` — **80 and 443 only, no port 22** |
| Shell | SSM Session Manager: `aws ssm start-session --target i-0d126c49ffdfe1668` |
| IAM role | `quadis-app-ec2` — SSM core, plus write scoped to one bucket and `/quadis/*` |
| Photo bucket | `quadis-hotel-photos` — private, public access blocked, AES256 |
| DB password | SSM Parameter Store `/quadis/db-password`, SecureString. **Not in this repo, not in the EB-style config trap.** |
| Health | `curl http://13.234.85.127/healthz` → `ok` |
| **App** | **DEPLOYED 31 Jul, last shipped 20 Aug** (mobile pass, §4). ⚠️ This line read "last shipped 3 Aug" until 20 Aug, which was **wrong** — the 5 Aug feedback round did ship, proved against production: the bank logos added in `160c78a` return 200 and the live DB carries the migrated GMB ratings (Quadis Sector 51 4.5, Downtown Sec 15 4.0). Second stale-line incident on this file after the Razorpay webhook one; **check a claim here against the box before trusting it.** Frontend + API + Postgres all live on the box. `http://13.234.85.127/` serves the site; `/api/properties` returns 9 seeded properties out of local Postgres |
| Deploy | `./deploy/build-artifact.sh && ./deploy/push.sh` — artifact to S3, installed via SSM. **No port 22, so no scp/rsync from a laptop, by design** |

### The deploy, and the four things it had to get right — 31 Jul

`deploy/` now contains the whole path: `build-artifact.sh` (assembles),
`push.sh` (S3 + SSM), `install.sh` (runs on the box), `quadis-api.service`,
and `nginx/quadis.conf`. Four traps, all hit for real and all now fixed in the
scripts rather than only on the box:

1. **`NODE_ENV=production` is set by the systemd unit, and it is load-bearing.**
   `backend/src/routes/webhooks.ts:17` gates the Razorpay simulation bypass on
   `NODE_ENV !== 'production'` — without it, anyone can confirm any booking
   with a plain POST and no payment. `backend/src/lib/auth.ts` separately falls
   back to the literal `'quadis-dev-only-session-secret'`, which is in the
   public repo. Nothing in `bootstrap.sh` set `NODE_ENV`. Do not start this app
   any other way.
2. **`pg_hba.conf` is first-match-wins.** `bootstrap.sh` *appended*
   `host quadis quadis … scram-sha-256`, which lands **below** Amazon Linux's
   stock `host all all 127.0.0.1/32 ident` and is never evaluated. Result:
   `Ident authentication failed for user "quadis"`, migrations refuse to run,
   the API dies on boot — while nginx serves the frontend perfectly. The rule
   is now *inserted above* the stock line.
3. **`node` is v18 on this box, not v20.** `bootstrap.sh` installs `nodejs20`,
   but AL2023 registers node-18 and node-20 in `alternatives` at equal
   priority and bare `node` resolves to **18.20.8**. The unit pins
   `/usr/bin/node-20` explicitly.
4. **`node_modules` is built on the box, never shipped.** `sharp` is
   ABI-specific (§9) and the build host runs a different Node major. `install.sh`
   runs `npm-20 ci --omit=dev` on the target. This also cut the artifact from
   91 MB to 62 MB.

**What ships, and what must never.** The artifact is *assembled*, never synced:
`www/` (built frontend), `api/` (compiled backend + package.json), `nginx/`,
`systemd/`. `build-artifact.sh` hard-fails if `docs/`, `.agents/`,
`client-assets/`, `.git/`, `src/` or any `.env`/`.pem` is staged. As a second
layer, `nginx/quadis.conf` denies dotfiles and `/(docs|.agents|client-assets|
backend|src|scripts|deploy|node_modules)/` outright — verified returning 404
for `/.env`, `/backend/.env`, `/docs/client-comms/README.md` and
`/.agents/AGENTS.md`. **The repo is a public GitHub repo and stays public by
the owner's decision (31 Jul); the web root is the boundary that matters.**

Verified on the box, not assumed: `/` 200, `/api/health` healthy,
`/api/properties` → 9 properties from Postgres, `/hotel-amar-inn/deluxe-room`
→ 301 `/hotels/hotel-amar-inn`, `/contactus` → 301 `/contact`,
`/banquets/we-offers` → 301 `/banquets`, all 16 redirect rules installed.

### 🟢 LIVE — cutover completed 1 Aug 2026

`quadishotels.com` now resolves to `13.234.85.127`. **This site is in front of
real guests. §2 rule 2 now means the opposite of what it used to: changes here
are visible immediately, so nothing goes out untested.**

The host made the change on 1 Aug; SOA serial went `2026072302` → `2026080101`.
Verified after the cert was issued, not assumed:

| Check | Result |
|---|---|
| `https://www.quadishotels.com/` | 200 |
| apex → www | 301 |
| `http://` → `https://` | 301 |
| Certificate | Let's Encrypt, `CN=quadishotels.com`, SAN covers apex + www, **expires 30 Oct 2026** |
| `/api/properties` | 9 properties from local Postgres |
| Legacy 301s | `/hotel-amar-inn/deluxe-room`, `/contactus`, `/banquets/we-offers` all correct |
| CORS | her two origins 201, `evil.example.com` rejected |
| **Payments** | `create-order` → **`isSimulated: false`** over HTTPS |
| `/.env`, `/.agents/`, `/docs/` | 404 |
| **Mail** | **MX Google, SPF, DKIM 417b, DMARC `p=quarantine` — ALL UNCHANGED** |
| `adminweb` `blog` `webmail` `mail` `booking` `api` | still `115.124.108.190`, untouched |
| Old host | still up and serving; nothing was destroyed |

**The certificate renews via certbot's systemd timer. It was issued with
`-a webroot -w /var/www/certbot`, so renewal depends on that webroot staying
readable by nginx and on the `location ^~ /.well-known/acme-challenge/` block
surviving. Do not remove either — renewal fails silently and the site goes
dark ~30 Oct.**

**`ftp.quadishotels.com` followed the apex** onto our box, which runs no FTP.
It was a CNAME to the apex, so this was unavoidable without a separate record.
Nobody has said whether it is used; ask before assuming it is dead.

✅ **The Razorpay webhook URL was repointed at
`https://www.quadishotels.com/api/webhooks/razorpay`** (Divyansh; stated
in-session, not verified from the dashboard). **This line previously said it was
still outstanding and was left stale for two days** — long enough that a later
pass re-derived a whole guest-loses-their-room incident from it and wrote it up
as live. See the top of §4. **If you fix something described here, fix the line
here too; a stale "outstanding" costs more than no note at all.**

Postgres listens on **loopback only**. There is no database port on the
network, which retires the whole class of problem behind incidents 1 and 3 —
no RDS security group to point at the wrong thing, no TLS CA bundle to ship.

Nightly `pg_dump` at 02:15 UTC to `s3://quadis-hotel-photos/backups/`, 30-day
expiry. This exists because the cost message promises "roz ka backup".

### RESOLVED 31 Jul — she approved `t3.medium`, the account is on the Paid plan

The box is `t3.medium` and running. Verified after the resize, not assumed:
4 GiB (`free -m` → 3839 MB), `nginx` and `postgresql` both `enabled` **and**
`active` after the reboot, Postgres still bound to `127.0.0.1:5432` only
(`listen_addresses = localhost`), EIP `13.234.85.127` still associated, CPU
credits still `standard`, `/etc/cron.d/quadis-backup` intact, `/healthz` → 200.

**The Free plan was the blocker, and upgrading was never optional.** The
earlier note here said `t3.medium` was "rejected as not eligible for Free
Tier". The real error on the resize is `FreeTierRestrictionError`: *"This
operation is not available for free plan accounts."* Upgrading fixed it:

```
aws freetier upgrade-account-plan --account-plan-type PAID
```

**Three things about the Free plan that were not in this file and change the
decision entirely:**

1. **A Free-plan account CLOSES ITSELF.** The plan ends after six months or
   when credits run out, whichever comes first, and then *"your account closes
   automatically, and you lose access to your resources and data"* — AWS holds
   the content 90 days, then deletes it. Her `accountPlanExpirationDate` was
   **29 Jan 2027**. Hosting production on that plan meant the site and its
   database disappearing in January. The field is gone from
   `get-account-plan-state` now that the plan is `PAID`.
2. **She has $99.47 of credits and they SURVIVED the upgrade** — confirmed in
   the API response after the flip. AWS applies leftover free-plan credits to
   future bills; they are only forfeited if the account closes *without*
   upgrading. That is roughly **2.4 months of the ₹3,550 bill already paid
   for**, and it expires whether or not she uses it. Tell her.
3. **Reserved Instances are Free-plan-blocked** — the plan excludes "features
   that could possibly deplete your credits… Savings Plans, Reserved
   Instances". So the 1-year RI option in §4 also needed this upgrade.

Treat the upgrade as **one-way**; AWS documents no downgrade path.

> **A dry-run is not a permission check — this cost an hour.** Before the
> resize, `run-instances --instance-type t3.medium --dry-run` returned
> *"Request would have succeeded"*. It was wrong: `DryRun` validated
> parameters and IAM, and did **not** evaluate the free-plan restriction,
> which only fired on the real `modify-instance-attribute`. Same shape as
> incident 6 and the IMDSv2 bug above — the check was green and the thing was
> blocked. Do not accept a dry-run as proof that a plan- or quota-level
> restriction is absent.

### Three bugs this build produced, all fixed — do not reintroduce them

1. **IMDSv2 vs an IMDSv1 metadata call.** The instance is launched with
   `HttpTokens=required`, and a tokenless `curl` to the metadata service
   returns an **empty string rather than failing**. That produced
   `--region ""`, `Invalid endpoint: https://ssm..amazonaws.com`, and under
   `set -e` killed the bootstrap before nginx was ever installed — a box that
   was `running`/`ok` on every EC2 check while serving nothing. Same shape as
   incident 6: green everywhere, dead in fact. The region is now pinned.
2. **`set -x` printed the generated DB password into
   `/var/log/cloud-init-output.log`**, which is world-readable and survives
   reboots. The password has been rotated, Parameter Store updated, and both
   logs scrubbed (verified: 0 occurrences). `bootstrap.sh` now wraps the secret
   in `set +x` / `set -x`.
3. **Our own nginx blocked the Let's Encrypt challenge, on both hostnames and
   for two different reasons.** Found 31 Jul while checking cutover readiness;
   it would have fired on cutover day, with DNS already moved and her domain
   on our box serving no HTTPS. Both were verified against the live box with a
   `Host:` header before DNS moved — the failure was real, not theoretical:
   - **apex → 301.** The apex→www redirect was a *server-level* `if`, and
     nginx evaluates those in the SERVER_REWRITE phase, **before it selects a
     location**. So it pre-empted every location in the block — including
     anything `certbot --nginx` inserts at runtime. Let's Encrypt would have
     followed the 301 to `https://www…` and hit a **closed port 443**, because
     443 does not exist until the certificate being requested is issued.
   - **www → 404.** `/.well-known/acme-challenge/…` contains `/.`, so the
     `location ~ /\.` dotfile deny matched it and returned 404.

   Fixes, all three needed together: a `map`-computed ACME exception on the
   redirect (nginx `if` cannot AND two conditions); a
   `location ^~ /.well-known/acme-challenge/` declared **before** the deny —
   `^~` is what makes a prefix location beat a regex one, a plain prefix would
   still lose; and `cutover.sh` now authenticates with
   **`-a webroot -w /var/www/certbot -i nginx`** instead of `--nginx`, so
   validation does not depend on certbot rewriting our config at the one
   moment it must not fail. `-i nginx` still installs the cert and the 443
   blocks.

   **The lesson is the check, not the fix.** Both states return 404 once the
   webroot is empty, so a status code cannot distinguish "reachable" from
   "denied". `install.sh` now writes a canary at
   `/.well-known/acme-challenge/ping` containing `acme-ok`, and
   `cutover.sh --check` asserts that **body** on both hostnames via `Host:`
   headers — which works before DNS moves. Verified end to end by running the
   real config in an nginx container: apex and www both return `acme-ok`, the
   apex still 301s real traffic to www, the Elastic-IP catch-all still serves
   without redirecting, and `/.env`, `/.agents/…`, `/docs/…` still 404.

   ⚠️ **This fix is in the repo and NOT yet on the box.** The deployed config
   is still the broken one. `./deploy/build-artifact.sh && ./deploy/push.sh`
   has to run before cutover — and `cutover.sh --check` will keep failing the
   acme line until it does, which is the intended behaviour, not a bug.

---

## 3a. The domain we are moving onto — read off the live host 30 Jul 2026

Everything here was resolved or fetched on 30 Jul, not copied from an earlier
doc. `docs/aws-account-migration.md` has the cutover procedure; this is the
current state of the thing we are cutting over to.

| Fact | Value |
|---|---|
| Canonical host | **`https://www.quadishotels.com/`** — the apex 301s to `www`, both HTTP and HTTPS |
| Apex A | `115.124.108.190` · `www` is a CNAME to the apex |
| Stack | Microsoft-IIS/10.0, ASP.NET, **PleskWin** — Windows, not the Linux shared hosting `docs/dns-cutover.md` assumed |
| Nameservers | `yellow1/yellow2.theserverindia.com` |
| Zone serial | `2026072302` — **the zone was edited 23 Jul 2026** |
| Site last modified | 22 Jul 2026 |
| Cert (apex + www) | Let's Encrypt wildcard `*.quadishotels.com`, 24 Jun → 22 Sep 2026 |
| Registrar | GoDaddy — created 18 Jan 2017, **expires 18 Jan 2027** |
| Registrar locks | `clientTransferProhibited`, `clientUpdateProhibited`, `clientDeleteProhibited`, `clientRenewProhibited` |

The registrar locks are GoDaddy's defaults and do not block a nameserver change
made from inside her own GoDaddy account — but they do block one made any other
way. Only she can lift them. Do not discover this on cutover day.

**`www` is the canonical host, not the apex.** Both our docs describe the
opposite (`@` at CloudFront, `www CNAME` to it). Whatever we cut over to has to
keep `www` working as the primary, because that is what her backlinks, her
sitemap and Razorpay's approved domain all point at.

**Someone is still actively maintaining the old site.** Zone edited 23 Jul,
homepage modified 22 Jul. Do not plan around the old host being abandoned or the
old designer being unreachable.

**The SOA contact is `websolvo5@gmail.com`** — read straight off the zone. We
have been waiting on "the old designer" for the theserverindia panel login
without having a name; that is the zone administrator's address.

### Mail records, and two the migration doc does not list

```
MX     1 smtp.google.com                        (Google Workspace)
SPF    v=spf1 a mx include:websitewelcome.com include:_spf.google.com
       include:Yellow.theserverindia.com ~all
DKIM   default._domainkey  v=DKIM1; k=rsa; p=MIIBIjANBgkqhkiG9w0…
DMARC  v=DMARC1; p=quarantine; adkim=r; aspf=r
```

`docs/aws-account-migration.md` lists MX, SPF and the two Google verification
TXTs — it does **not** list DKIM or DMARC, and it says DKIM selectors "will not
show from outside". The `default` selector does resolve; it was found by
guessing common selector names.

This matters more than a missing record usually would: **DMARC is
`p=quarantine`.** Recreate the zone from that doc's list and you lose DKIM, so
her mail still delivers — into spam — and it fails quietly rather than bouncing.
Get the zone export and diff it; guessing selectors is not a substitute.

### Subdomains — all eight on `115.124.108.190`

`www` `adminweb` `booking` `mail` `webmail` `blog` `api` `ftp`. Every one must
survive the cutover; only the apex/`www` moves.

- **`blog.` is a live WordPress** (`wp-json` link header, `/index.php/`) on
  IIS/Plesk. Nobody has mentioned it. It is content the client may care about
  and an attack surface nobody is patching — ask before assuming it can go.
- `adminweb.` 200s, modified 22 Jul, cert renewed 18 Jul → maintained. §5.
- `booking.` 301s to `www.booking.…` and its cert expired 18 Apr 2026 → dead.
- `api.` returns 404. It exists in DNS and serves nothing.

### The cutover breaks 63 of her 74 indexed URLs

Her `sitemap.xml` lists 74 URLs. Checked against the routes in `src/App.tsx`,
**11 still resolve and 63 would 404** the moment DNS points at us:

| Count | What | Why |
|---|---|---|
| 52 | `/<hotel-slug>/<room-slug>` room pages | We have **no route at this shape at all** — they are top-level paths, not under `/hotels/`, so they hit `*` |
| 4 | `/banquets/banquet-hall-at-…` | Ours are `banquets-at-…`; hers are `banquet-hall-at-…`, and we have no Cladis 15 venue |
| 2 | `/hotels/hotel-amar-inn`, `/hotels/hotel-amby-inn` | Slug mismatch — see below |
| 5 | `/privacy-policy` `/terms-and-conditions` `/career-at-quadis` `/contactus` `/banquets/we-offers` | No route. Note hers is `/contactus`, ours is `/contact` |

The 52 room pages are one page per room **per meal plan** — `…-deluxe-room`,
`…-with-breakfast`, `…-with-breakfast-lunch-dinner`. That is her SEO surface and
it is most of the site.

Nobody has scoped redirects. Until someone does, "go live" means her Google
results lead to 404s. This is not a DNS problem and it will not be caught by any
check in §6.

**Seven of her eight hotel slugs now match ours exactly.** One does not:

| Hers (live) | Ours (seeded) |
|---|---|
| ~~`hotel-amar-inn`~~ | ~~`hotel-amar-in`~~ — **fixed 31 Jul**, our slug is now `hotel-amar-inn` |
| `hotel-amby-inn` | `hotel-amby-inn-lajpat-nagar-ii` ← still ours, still redirected |

Fixing the remaining one makes another dead URL live for the cost of a rename.
Her site is the authority on her own slugs.

We also seed a ninth hotel, `hotel-quadis-central-sector-27-noida`, which is
**not on her live site at all**. Worth asking whether it is open.

### `/hotel-amby-inn/amby-inn-executive-room` is a live, indexed page

The open question in §4 — "is Amby Inn's Executive the Super Deluxe?" — has a
partial answer. Her own site sells **Deluxe and Executive** at Amby Inn, with
no Super Deluxe anywhere, across six indexed URLs. Our seed says deluxe 20 /
super 3. So it is likelier that the *category name* is wrong in our seed than
that her photo was mislabelled. Still ask — this changes what a guest is sold —
but ask the sharper question: is "Super Deluxe" a name we invented for Amby Inn?

Also indexed: `/hotel-amar-inn/Test-Hotel-Slug`, a test page leaked into her
public sitemap. Not ours to fix, worth mentioning to her.

---

## 4. Task board

### Open

- [x] **20 Aug — mobile pass. Committed; deploy record below.**
      `src/components/media.tsx`, `src/styles/chrome.css`,
      `src/styles/components.css`, `src/styles/pages.css`.

      **The finding worth keeping even if the code is thrown away: the home page
      was sending every phone a 5.2 MB video.** `HeroVideoShowcase` renders
      `<video autoPlay>` on `/videos/Quadis.mp4`, so the fetch starts
      immediately, for a decorative background above the fold. It is ~30x the
      weight of the entire rest of the page and it is the whole of the mobile
      performance problem in one file. The component already had the fallback it
      needed — a 176 KB poster — reached only on `onError` or reduced-motion.
      It is now also taken on `(max-width: 768px)`, Save-Data, and 2g/3g.

      > **Two details that are the difference between this working and merely
      > looking like it works.**
      > 1. The decision is made in the `useState` **initialiser**, not an
      >    effect. The pre-existing reduced-motion check started `false` and
      >    flipped in `useEffect` — which runs after the first commit, by which
      >    time `<video autoPlay>` is mounted and the 5.2 MB is already in
      >    flight. That fallback had never saved a byte for anyone.
      > 2. The resize listener is a **one-way latch**. A phone in landscape is
      >    ~844px wide, so re-testing the width query on rotation would start
      >    exactly the download this exists to prevent.

      Verified in a browser at a 375px viewport: **0 `<video>` elements, 0 mp4
      requests**, poster served with `fetchpriority=high` as the LCP element. At
      1265px the `<video>` and its `/videos/Quadis.mp4` source are still there —
      **desktop is deliberately unchanged.**

      **Touch targets.** Everything interactive now clears the 44px floor that
      the stepper and the `tel:`/`mailto:` rule already observed. Measured, not
      guessed, at 375px: footer nav links were **25px** and the five footer
      social icons **20x20** — the smallest tappable things on the site and on
      *every* page. Also `.deal-card-fern__btn` 39px, `.offer-copy-button` 35px,
      `.back-link` 24px, `.map-link` 24px, `.tour-mode-pill` 39px. All scoped to
      `max-width: 768px`, so the signed-off desktop rendering is untouched.
      Re-audited clean on `/`, `/hotels`, `/hotels/:slug`, `/banquets`,
      `/corporate-hotel-booking`, `/virtual-tour`.

      `.dest-stamp__badge` was `0.58rem` = **9.28px**, the only type on the site
      genuinely below legible rather than merely small. 11.2px on mobile only.

      > **The layout itself was already sound — that is a real result, not a
      > gap in the check.** Every route was measured for elements crossing the
      > right edge and **none does**. `.deals-fern-bg` reports 717px wide on a
      > 375px viewport, but it is the intentional `left/right: -50vw` full-bleed
      > band and `body { overflow-x: hidden }` clips it: `scrollTo(200, 0)`
      > leaves `scrollX` at **0**, so the page cannot actually be panned. The
      > 5 Aug booking-bar fix is holding.
      >
      > ⚠️ **But that `overflow-x: hidden` means a real overflow bug would be
      > invisible to the eye and to a scrollbar.** Anything auditing this in
      > future must measure element rects against the viewport, not trust
      > `scrollWidth` or a screenshot.

      Typecheck clean; production build clean and the rules are present in the
      emitted CSS. **Not verified in a browser: `/restaurant`, `/gallery`,
      `/contact`, `/about-us`, `/login`** — the extension's renderer wedged
      partway through and did not recover. Their only finding was the shared
      footer, which is one component and is verified fixed on six other pages,
      but that is an inference and is written down here as one.

      Not done, and deliberately: **route-level code splitting.** The bundle is
      one 471 KB chunk (150 kB gzip), which is ordinary for an SPA and is not a
      defect. Splitting it would save perhaps 10-15 kB gzip for a guest, while
      adding a chunk-fetch that can fail on exactly the flaky mobile connection
      this pass is about — and there is no error boundary in the tree to catch
      it. Not worth that trade on a live site without being asked.


- [x] **RESOLVED — the Razorpay webhook URL was repointed at the live site.**
      Done by Divyansh; **§3b's "still outstanding" line was simply never
      updated**, which is what sent a later pass down the chain below.
      ⚠️ **Provenance: stated in-session, not verified from our side** — it
      needs the Razorpay dashboard, which is §2/§4a "the client's account".
      Treat it as done; if you need proof, get it from the dashboard.

      **Do not re-raise this from the old §3b wording alone.** That line is now
      corrected. The reasoning below is kept because the *mechanism* is real and
      would bite again if the URL were ever changed back or duplicated.

      > **Why the logs cannot confirm it either way, checked 3 Aug.** nginx on
      > the live box has **no Razorpay-originated POSTs** to
      > `/api/webhooks/razorpay` — only `curl/8.5.0` probes from one IP. That is
      > not evidence of a problem: `payments/create-order` appears **twice in
      > total** across all four access logs, both manual checks. **No real guest
      > has attempted a payment on the live site yet**, so no webhook would have
      > fired regardless. Absence of deliveries proves nothing until money moves.
      > The first real booking is the test that settles it — watch it.

      **The mechanism, for whoever reads this later.** If the webhook ever points
      anywhere but the live box: the guest books (row lands in Postgres on
      `13.234.85.127`), pays, Razorpay captures and fires `order.paid` at the
      wrong stack, which finds nothing in *its* database and returns
      `404 Target booking not found` (`webhooks.ts:69`). On the live box nothing
      arrives, `startHoldCleanupWorker` (`server.ts:21`) runs every minute, and
      `cleanupExpiredHolds(15)` expires the hold — **room released and resold,
      after the money was taken.** Nobody is told, because the WhatsApp alerts
      below have no `META_WHATSAPP_TOKEN`.

      > **The webhook handler is not the bug and must not be "fixed".** It is
      > idempotent, it refuses to confirm an `EXPIRED` booking because that
      > would oversell the room, and it records the payment and returns 409 for
      > manual review. All correct — it just has to actually receive the event.

- [ ] ⚠️ **The retired CloudFront stack is not retired — it is alive and
      answering.** Verified directly 3 Aug, and independent of the webhook
      question above. It is a second complete copy of the site on a *different
      database*:

      | Probe, 3 Aug | `djqj43186y3yh.cloudfront.net` (old) | `www.quadishotels.com` (live) |
      |---|---|---|
      | `/` | 200 — serves a whole site | 200 |
      | `/api/properties` | 9, first ids `prop-3, prop-4, prop-5…` | 9, first ids `prop-2, prop-3, prop-4…` |
      | `/api/webhooks/razorpay` | **401 `Missing Razorpay webhook signature`** — it accepts webhooks | 401, same |

      Different id ordering off the same seed = **two different databases**. The
      old EB app and its RDS are still running in account `093650262440`, billing.

      Why it still matters now the webhook is repointed:
      - It is a **second indexable copy of the whole site** — duplicate content
        against `www.quadishotels.com`, competing for the same 74 URLs §4 spent
        a day making resolve.
      - Its `/api/webhooks/razorpay` **still accepts events**, so any stale or
        duplicated webhook config lands somewhere that answers instead of
        failing loudly.
      - Its RDS holds real rows and is the only record of anything that happened
        before 1 Aug. **Do not just delete it as cleanup.**

      **PARKED 3 Aug — Divyansh, to discuss later. Do not act on it unasked.**
      His read, and it holds: **no payment can complete there.** Nothing creates
      an order on that stack (DNS sends every guest to the live box) and Razorpay
      no longer posts to it, so both ends of the payment path are dead even
      though the endpoint still answers. What is left is cost and duplicate
      content, not a guest-facing risk.

      When it is picked up: snapshot the RDS first, then disable the
      distribution and the EB environment.

- [x] **3 Aug — the client walked the live production site and signed it off.**
      `client-assets/briefs/2026-08-03-whatsapp-post-live-signoff-and-admin-panel-question.png`.
      Her list, in her words: forms and booking both working, internal linking
      correct, registration and login working on every page, guest bookings
      showing up, chatbot working, WhatsApp button working. **This is the §6
      browser pass, done by the one person whose verdict actually settles it,
      on the real site in front of real guests.** Record it as such — it is the
      first end-to-end acceptance this project has had from outside the team.
      It does **not** clear the §7 pattern: she checked what a guest can see,
      which is exactly the half that stayed green through incidents 5 and 6.

- [x] **SENT 3 Aug — answered: where bookings and form details show.**
      `message-g-where-bookings-show-SEND.txt` went to her (Divyansh). **The
      two-systems question in it is now asked, so this moves from "we owe her an
      answer" to "we are waiting on hers" — see §5.** Detail below kept for what
      was verified and claimed.
      The answer is `https://www.quadishotels.com/admin` — verified live 3 Aug:
      the page returns 200 and `POST /api/admin/auth` with a wrong PIN returns
      `Invalid Admin PIN`, not `503 Admin access is not configured`, so
      `ADMIN_PIN` and `ADMIN_PASSWORD` **are** set on the box from SSM
      (`/quadis/admin-pin`, `/quadis/admin-password`, both present).
      It shows today's check-ins / pending holds / pending enquiries / revenue,
      the inventory switchboard, the WhatsApp payment-link generator, **Recent
      Bookings** (code, guest name, phone, check-in, rooms, amount, status) and
      **Recent Leads & RFPs**. All three forms — Contact, Corporate,
      BanquetDetail — POST to `/api/enquiries` and land in that second card.
      Drafted in `docs/client-comms/message-g-where-bookings-show-SEND.txt`.

      > **She wrote "admin panel me?" and she means `adminweb.quadishotels.com`.**
      > It is not that panel and cannot become it. Grep for `adminweb`,
      > `115.124.108.190` or `theserverindia` across `backend/src` and `src`
      > returns **zero hits** — there is no integration, no sync, no export.
      > Her guests' bookings land in the Postgres on her EC2 box and stop there.
      > **This is the §5 two-systems question, and the client has now walked
      > into it herself.** Settle it in the reply; do not let it slide again.

- [x] **DEPLOYED 3 Aug — the client can change her own admin PIN, and the
      sign-in no longer hands out a permanent credential.**
      Asked for directly. It touches the admin panel, which §2.4 restricts —
      the judgement was that this *hardens a surface that is already live and
      already in her hands*, rather than building new function into it, and it
      stays useful whichever way the PMS question lands.

      **The bug this fixes is worth stating plainly: `/api/admin/auth` used to
      return `ADMIN_PASSWORD` itself as the bearer token.** `requireAdmin`
      compared that string directly, so the token *was* the permanent admin
      secret — no expiry, valid until someone rotated SSM and redeployed. One
      intercepted response, or one read of `sessionStorage`, was total and
      permanent access. It is now a `signSession(...)` HMAC token with
      `role: 'admin'` and a **12-hour** TTL (a front-desk shift).

      - `admin_credentials` — one row, `pin_hash` only, scrypt via the existing
        `hashPassword`. **Deliberately not in `site_content`**: `GET /api/content`
        is public and unauthenticated (verified 200 on production), so a hash
        parked there would be world-readable, and `PUT /api/admin/content`
        writes arbitrary keys so it could be overwritten from inside the panel.
      - `ADMIN_PIN` is now **bootstrap only** — used when no row exists. The
        stored PIN wins the moment she sets one. The login response carries
        `mustChangePin`, and the dashboard opens the change form on sight of it,
        because the current PIN was sent over WhatsApp and is not a secret.
      - `POST /api/admin/change-pin` re-checks the current PIN even though the
        caller already holds a valid session — an unattended open dashboard must
        not be enough to lock the owner out of her own live site.
      - Weak PINs rejected: non-6-digit, all-same-digit, and straight runs both
        ways. Those are the first three guesses anyone makes.
      - `ADMIN_PASSWORD` still works as **break-glass**, so a forgotten PIN
        never locks her out. No expiry, which is why it is the fallback.

      > **The escalation this could have been, and the test that pins it.**
      > `verifySession()` validates *guest* logins too. A `requireAdmin` that
      > accepted "any valid session" would have promoted **every registered
      > guest** to full admin — re-pricing rooms and reading every guest's
      > phone number. The guard turns solely on `role === 'admin'`, which
      > `routes/auth.ts` never sets.
      >
      > `requireAdmin` short-circuits under `NODE_ENV=test`, so the guard was
      > **completely untested** and this would have shipped unseen.
      > `__tests__/adminPin.test.ts` flips the env and drives the middleware
      > directly: guest session rejected, forged claim rejected, expired token
      > rejected, break-glass accepted.

      Verified before shipping: 126 tests pass (was 109), both typechecks
      clean, production build clean.

      **Shipped to the box 3 Aug** — `build-artifact.sh && push.sh`, artifact
      `quadis-ebc78b6-dirty.tar.gz`, SSM command
      `ca5f9ddf-0c63-4ea0-a732-c7e05eef4302`, install `Success`. Deployed from
      a dirty tree: this work is **on production but not committed**, so the
      `-dirty` tag is the only version marker that exists for it. `push.sh`
      re-applied TLS and its own HTTPS check passed, so the certbot trap in
      §3b did not bite.

      Verified on production after the deploy, not assumed:

      | Check | Result |
      |---|---|
      | `POST /api/admin/auth`, wrong PIN | 401 `Invalid Admin PIN` |
      | `POST /api/admin/change-pin`, no token | 401 |
      | `GET /api/admin/dashboard`, garbage token | 401 |
      | `admin_credentials` table | **exists, zero rows** — see below |
      | `/` and `/api/properties` | 200, 9 properties |
      | `/admin` in a browser | login card renders, console clean |
      | Wrong PIN in the browser | `Invalid Admin PIN` renders, no React crash |

      > **How the table is known to exist without opening psql.** `/auth` awaits
      > `db.getAdminPinHash()` — which selects from `admin_credentials` —
      > *before* it compares the PIN. A missing table would throw there and
      > surface as a 500, not a clean 401. Getting `401 Invalid Admin PIN` back
      > proves the select ran and returned no rows: the table was created by
      > `schema.sql` on boot, and the panel is still in **bootstrap mode**.

      **Bootstrap mode is the part that is not finished.** Zero rows means she
      is still on the `ADMIN_PIN` from SSM — the one sent over WhatsApp. The
      change-PIN form exists and is one sign-in away, but *she* has to use it.
      Until she does, the WhatsApp PIN is the live credential.

      **Not verified: everything behind the login.** The dashboard render, the
      `mustChangePin` banner and the change-PIN form itself were not exercised,
      because doing so means typing the live admin PIN into the live panel.
      That check is the client's to make, or needs an explicit decision to sign
      in. §2.5 is therefore only **half** satisfied here — say so rather than
      logging this as a clean browser pass.

- [ ] **Three gaps the admin panel has, all exposed by her question.** None is a
      bug today and all three become one within weeks:
      1. **Only the last 15 of each are visible.**
         `backend/src/routes/admin.ts:45-46` calls `getAllBookings(15)` and
         `enquiries.slice(0, 15)`. No pagination, no date filter, no search, no
         CSV export. `getAllBookings` itself defaults to 50 — the 15 is the
         dashboard's own choice. The 16th booking does not disappear from the
         database, it disappears from *her view of it*, silently.
      2. **Registered users appear nowhere.** She specifically confirmed signup
         works; `admin.ts` has no users route and `AdminDashboard.tsx` has no
         users panel. If "forms ki details" includes accounts, today's honest
         answer is "nowhere". Decide whether she needs it before promising it.
      3. **`/admin` is not linked from anywhere in the site.** Deliberate or
         not, it means the panel is undiscoverable without being told the URL —
         so the URL has to be *told*, and the PIN handled separately from it.

- [x] **DEPLOYED 3 Aug — `/api/health` now probes the storage it is actually
      using, instead of saying `healthy` about nothing.**
      It returned a static object, while `db/index.ts` picks storage on one
      condition — if `DATABASE_URL` is absent the entire app runs **in memory**:
      it still serves the 9 seeded properties, still accepts bookings, still
      issues booking codes, and loses every row on restart. `install.sh`
      hard-fails when `/quadis/db-password` is missing, so a normal deploy
      cannot reach that state — but nothing *detected* it if one ever did.
      Textbook §7: green everywhere, gone in fact.

      `db.ping()` runs `SELECT 1` and reports `storage` and `database`. The
      endpoint returns **503 `degraded`** in two cases, which are not the same
      failure:
      1. Postgres configured but not answering.
      2. **Running in memory while `NODE_ENV=production`** — the dangerous one,
         because *nothing errors*. In-memory off production is ordinary (it is
         how dev and the tests run), so the production check is what makes it a
         fault rather than a mode.

      > **This gates deploys, deliberately.** `install.sh:246` and
      > `cutover.sh:101/222/240` curl this with `-f`, so a 503 now fails the
      > install instead of printing "api ok" over a box that has no database.
      > That is the point of the change; it is also the thing most likely to
      > surprise whoever runs the next deploy.
      >
      > `install.sh`'s failure branch was updated with it. It used to print
      > **"API did not answer"** for any non-2xx, which after this change would
      > be an outright lie when the API answered `503 degraded` — sending the
      > next person after a dead process when the process is fine and its
      > database is not. It now prints the health body when there is one.

      Verified: 132 tests pass (was 126), typecheck clean, `bash -n` on
      `install.sh`. `__tests__/health.test.ts` covers all four states, and the
      pre-existing `e2e_acceptance` 5.1 still passes unchanged.

      **Shipped 3 Aug**, SSM command `2db7d48e-985a-48f9-957c-174af06b94ed`,
      install `Success`. Verified on production:

      | Check | Result |
      |---|---|
      | `/api/health` | **200 `storage: postgres`, `database: ok`** — a real `SELECT 1`, not an assertion |
      | install-time gate | passed, and now means something: it could not have passed without the database |
      | `/api/properties` | 9 |
      | `/api/admin/auth`, wrong PIN | 401 — the PIN work survived the redeploy |
      | Browser, `/hotels` | cards render, `₹3,000 / night`, ratings `4.4`/`4.6` as decimals, console clean — full §6 pass |

- [ ] **WhatsApp alerts are built, wired, and silently doing nothing.** Found
      3 Aug while answering "where do bookings show". `NotificationService`
      sends an owner alert on every confirmed booking (`webhooks.ts:112`), an
      owner alert on every enquiry (`enquiries.ts:65`, plus two in `AIService`)
      and a receipt to the guest (`webhooks.ts:111`). All three need
      `META_WHATSAPP_TOKEN` and `META_PHONE_NUMBER_ID`; **neither is in her
      SSM** — the parameter list has `owner-whatsapp-phone` and nothing else.
      Absent them every send returns `isSimulated: true`, logs, and no-ops.
      **So nobody is notified of anything today**; she finds out by opening
      `/admin` and looking. This is the cheapest answer by far to "I want to
      see bookings without logging into another panel" — it is a credentials
      task, not a build. Do it before promising her any `adminweb` integration.

- [ ] **A post-live change list is inbound.** *"Baki kuch cheeje thi jo glt hui
      thi vo aapne bola tha hum live hone k bd change kr lenge to vo me apko ab
      bta dungi"* — 3 Aug. Not yet received as of this writing. She also asked
      *"kb tk free hoge"*, which is about our availability, not the site.
      **Everything on that list now ships to a live site in front of real
      guests** — §3b's warning, not §2.5's.

- [x] **DONE 30 Jul — all 74 of her indexed URLs now resolve, 0 broken.**
      Mapping lives in `src/data/legacyRoutes.ts`, which is the single source of
      truth. Two consumers, and they must agree:
      `deploy/nginx/legacy-redirects.conf` (real 301s, authoritative for SEO)
      and `src/components/LegacyRedirect.tsx` (React fallback, which works on
      the current S3/CloudFront deploy where there is no nginx). A rule present
      in one and missing from the other is invisible in testing, because the
      React half silently covers for nginx.
      - 52 room pages → the hotel page. Deliberately **not** deep-linked to a
        room: her room slugs do not map onto our room ids and a wrong guess
        sells a guest the wrong class.
      - `/privacy-policy` and `/terms-and-conditions` are real pages now, ported
        from her live site, and linked in the footer. They were the two we did
        not have.
      - Verified in a browser on the production build, not just typechecked:
        the redirect lands on the right hotel, `/restaurant/outdoor-catering-service`
        still beats the `/:a/:b` wildcard, and a nonsense two-segment path 404s
        instead of being redirected somewhere plausible.
      - Sitemap now emits **`https://www.quadishotels.com`**, not the apex —
        her canonical host is `www` and the apex 301s to it, so apex URLs would
        have pointed every entry at a redirect.

- [x] **DONE 31 Jul — the chatbot was quoting the retired "children are free"
      rule to guests.** Not a stale comment: `backend/src/services/AIService.ts`
      builds the live LLM system prompt, and it said *"A CHILD adds nothing at
      all"* and *"NEVER add a charge for a child"*, while `pricing.ts` charges
      20% for ages 8–12. A guest asking the assistant got one number and would
      have been billed another. `policyFor()` already returned `childPercent`
      and `adultFromAge` — the prompt simply never used them. Both the POLICIES
      block and the per-property line now state all three bands, and the prompt
      instructs the model to **ask the child's age before quoting**.
      Swept at the same time: `src/lib/pricing.ts` and `backend/src/db/schema.sql`
      both carried comments contradicting the code directly beneath them
      (schema.sql said `child_free_under_age` "defaults to 18" above a column
      declaring `DEFAULT 8`). `docs/DEPLOYMENT.md` §6 and `docs/change-order-3.md`
      are corrected/bannered. Verified: 108 tests pass, both typechecks clean,
      and no live file under `src/` or `backend/src/` still claims children are
      free.
- [ ] **Two things in her own legal copy need her answer** — found while
      porting the pages:
      1. **Her T&C and her cancellation policy contradict each other.** The
         T&C says advance deposits are "non-refundable unless otherwise
         stated"; the policy she sent on 27 Jul gives free cancellation up to
         24 hours before check-in. Both cannot hold, and the gap is a real
         refund to a real guest. `TermsAndConditions.tsx` currently resolves it
         with a precedence line pointing at the cancellation policy — the more
         specific and more recent document, and the one Razorpay requires
         visible. **That is a change to her legal text and she has to confirm
         it.**
      2. **Her privacy policy gives the contact address as
         `info.quadishotels.com` and `wecare.quadishotels.com`** — a dot where
         the `@` belongs, on both. We ship `info@quadishotels.com`, which is
         confirmed. `wecare@` is unconfirmed so it is omitted rather than
         guessed. Ask whether that mailbox exists.
- [x] **DONE 31 Jul — `hotel-amar-in` → `hotel-amar-inn`.** Our typo; her live
      site is the authority on her own hotel's name. Seven touchpoints moved
      together, because incident 4 is exactly the half-done version of this:
      `src/data/hotels.ts` (3), `src/components/Footer.tsx`,
      `backend/src/data/seed.ts` (2), and `git mv` on
      `public/images/hotels/hotel-amar-in/`. The image directory had to move
      with the slug — `src/data/images.ts` resolves photos through a build-time
      glob keyed on the directory name, so a renamed slug with an unrenamed
      directory is a hotel page with no photos and no error.
      The redirect entries were **deleted**, not repointed, in both consumers
      (`src/data/legacyRoutes.ts`, `deploy/nginx/legacy-redirects.conf`) —
      our slug is hers now, so the old rule would have redirected
      `/hotels/hotel-amar-inn` to itself. The room-page rule stays and now
      targets the corrected slug. Sitemap regenerated.
      Verified: typecheck clean, production build clean, all 15 images emitted
      as assets, and the bundle contains `hotel-amar-inn` with zero occurrences
      of the old typo. **Not** verified in a browser — the Bash sandbox has its
      own network namespace, so a preview server on localhost is unreachable
      from Chrome on the host. §6 still wants a browser pass on the next deploy.
      **`backend/src/data/seed.ts` changed, so the live RDS still holds the old
      slug until it is re-seeded.** Harmless today (the new box gets a fresh
      seed, §3b) but it means the current CloudFront deploy and the seed file
      disagree — do not read one as evidence for the other.
- [ ] **Decide `hotel-amby-inn-lajpat-nagar-ii` → `hotel-amby-inn`.** The other
      half of the slug task, deliberately left open: unlike the Amar Inn typo
      this is not a mistake, it is us having appended the locality to a slug she
      publishes without it. Renaming makes her indexed URL resolve natively and
      lets the last hotel-page redirect go; keeping it means one permanent 301.
      Same seven-touchpoint shape as above if it goes ahead, plus the image
      directory `public/images/hotels/hotel-amby-inn-lajpat-nagar-ii/`.
- [ ] **Ask about `blog.quadishotels.com`** — a live WordPress nobody has
      mentioned, unpatched, on the host we are migrating off.
- [ ] **Get the zone export and diff it against §3a**, specifically DKIM and
      DMARC. `docs/aws-account-migration.md` omits both and DMARC is
      `p=quarantine`.
- [x] **DONE 30 Jul — daily `pg_dump` cron is built.** `deploy/bootstrap.sh`
      installs `/usr/local/bin/quadis-backup.sh` and `/etc/cron.d/quadis-backup`
      at 02:15 UTC nightly, gzipped to `s3://quadis-hotel-photos/backups/`, with
      a 30-day prune so the bucket does not grow forever. This existed as a task
      because the cost message promises "roz ka backup" — it is now built rather
      than assumed. (The Hostinger weekly-vs-daily framing that used to be here
      is moot; we are on her own AWS account, §3b.)
- [ ] **Production shape is decided: one EC2 box, not Beanstalk.**
      **`t3.medium`** in `ap-south-1` on her account, **Elastic IP**, Postgres
      installed on the instance, S3 for photos. No load balancer, no RDS, no
      CloudFront.

      **Size: `t3.medium` (4 GiB), chosen 30 Jul over `t3.small`.** All t3 sizes
      here are 2 vCPU — this is a RAM decision only. `t3.small` looked adequate
      until you account for Postgres moving onto the same box: incident 6 was
      1 GiB running Node *alone*, with the database on a separate RDS. On
      `t3.small` the real headroom for Node is ~1.5 GiB, not 2, and `sharp`
      spikes hard per image on upload — the same shape of failure. 4 GiB is
      deliberate headroom, not luxury.

      Verified off the AWS pricing API, `ap-south-1`, 30 Jul:

      | | EC2 | + EBS/IP/S3 | Total |
      |---|---|---|---|
      | on-demand | ₹2,845 | ₹706 | **~₹3,550/mo** |
      | 1-yr RI, All Upfront | ₹1,675 | ₹706 | **~₹2,380/mo** |

      1-yr All Upfront is $231 vs $392 on-demand — **41% off**. Standard Linux
      RIs are size-flexible within the family, so a reservation is not wasted
      if the size changes later.

      **Do not buy the RI on day one.** Run on-demand a month, confirm the
      shape under real traffic, then commit. It is the same price whenever you
      buy it, and ₹20,097 up front on her card is its own decision.

      **Set the instance to `standard` CPU-credit mode, not `unlimited`.**
      Unlimited is the default and silently bills per vCPU-hour when credits
      run out, which on a box doing image processing is how an unpredicted
      bill happens.

      **Why, because this was argued in circles once already:** the reason to
      leave AWS was never AWS. It was CloudFront, which has no static IP, so
      an apex A record cannot point at it and the whole zone has to relocate —
      taking her Google MX, DKIM and DMARC with it. An **Elastic IP is a static
      IP**, so the apex is one A record, the nameservers stay at
      theserverindia, and her email is never in the blast radius. That is the
      entire advantage the Hostinger detour was chasing, and her own account
      already has it.

      Do not reopen this by comparing a VPS against *Beanstalk + ELB + RDS +
      CloudFront* — that is a managed multi-service shape at ₹3,600-5,400 and
      it is not the only way to use AWS. Compare like with like.

- [ ] **Migrating to the client's own AWS account — see
      `docs/aws-account-migration.md`.** Builder has her account details as of
      29 Jul. Two decisions first: **region** (recommend `ap-south-1`/Mumbai —
      her guests are in Delhi NCR and this is the only cheap moment to change
      it) and **instance size** (`t3.small` minimum; `t3.micro` is what wedged
      the API for 30 minutes, incident 6). The long pole is the **DNS zone
      export from theserverindia** — it depends on other people and it is the
      one step that can take her email down. Note the cutover also **unblocks
      payment for free**, because `quadishotels.com` is already approved in
      Razorpay while the CloudFront URL is not.

**From the client's 29 Jul message** (`client-assets/briefs/2026-07-29-whatsapp-experience-image-and-complaints.png`):

- [ ] **"Book a reservation" does nothing after login.** Her item 3. A dead
      button, so worth walking in a browser before assuming where it should go —
      and note that where it *should* go is the §2.4 PMS question.
- [ ] **"Quadis" is not the same font everywhere.** Her item 2, and she asks
      whether the admin panel can change it. For this site the answer is no —
      it is a CSS token, not content. Worth saying so plainly, because the
      question is really §5 again: she is assuming her existing panel drives
      this site.
- [ ] **The link does not open in PhonePe's in-app browser.** Her item 4.
      Suspect the CloudFront hostname rather than the site; in-app browsers are
      unpredictable about unfamiliar hosts. Test before spending on it — DNS
      cutover may remove it for free.
- [ ] **Decide on 8 AI-generated images now live.** The 29 Jul gallery zip
      included eight `ChatGPT Image *.png` files, six of them filed under
      Facade & Lounges. They are photorealistic — receptions, lobbies,
      corridors, two carrying "HOTEL DOWNTOWN51" signage — and a guest browsing
      the gallery cannot tell they are synthetic. They were placed as sent, and
      their filenames still start `chatgpt-`, so they are identifiable in the
      page source and removable in one command:
      `rm public/images/{rooms/deluxe,rooms/superior,facade}/chatgpt-*.webp`.
      This is the client's call, not ours — but she should be asked, because
      rooms and lobbies that do not exist are a different thing from the
      concept renders on the "expanding into three categories" band.
- [ ] **Seven photographs appear twice in the gallery.** Pre-existing, not from
      the 29 Jul batch: seven byte-identical pairs across `public/images`, so
      Vite emits one asset and two buckets both point at it. Includes
      `upcoming/noida.webp` = `hotels/hotel-cladis-sector-15-noida/hero.webp`,
      which is the "Noida photo is a building" complaint.

      > **DO NOT "fix" this by deleting the duplicate files.** An audit on
      > 31 Jul recommended deleting the four intra-directory pairs to save
      > "~0.5 MB shipped twice". Both halves of that are wrong, verified:
      >
      > - **Nothing ships twice.** Vite already emits ONE asset per set of
      >   identical bytes — `hotel-quadis-sector-51-noida/hero.webp` collapses
      >   into `facade-2-Ds3nriSc.webp` in `dist/assets`. The 488 KB is source
      >   only; the shipped cost of the duplication is **zero bytes**.
      > - **The dedupe is already deliberate.** `src/data/images.ts:50`
      >   `uniq()` collapses them on URL, and its comment says plainly that
      >   this was chosen over deletion so as not to remove a file something
      >   else resolves by name.
      >
      > The only real symptom is that `vite dev` serves real paths, so the
      > URLs differ and the doubles still show **in dev mode only**. Production
      > is correct. This item is about asking the client which photo belongs
      > where — it is not a file-deletion task.
- [x] **DONE 31 Jul — the repo root is no longer a dumping ground.** It was
      carrying **22 loose images** (`image copy 2..17.png`, four stray `.jpeg`)
      plus the client's own 8.7 MB `Website Changes.pdf`, which was **tracked**
      and therefore public. All filed, nothing deleted, all 22 checksums
      verified present at the destination:

      | Kind | Home |
      |---|---|
      | Client WhatsApp / brief material | `client-assets/briefs/` |
      | Working screenshots (dashboards, UI) | `client-assets/screenshots/` |
      | Photography not yet placed | `client-assets/UNPROCESSED/` |

      Filed names are date-stamped from the file's own mtime
      (`2026-07-29-image-copy-10.png`) because two root files collided by name
      with files already in `screenshots/` while having **different** contents —
      a plain `mv` would have silently destroyed one of each pair.

      The root catch-alls in `.gitignore` did not cover `.pdf`, which is why the
      client's document sat tracked for five days. Now `/*.pdf`, `/*.png`,
      `/*.jpg`, `/*.docx` are covered too. They are root-anchored, so
      `public/**` is unaffected — verified.

      **The catch-alls are a safety net, not the filing system.** Anything
      landing at the root still has to be filed by hand into one of the three
      directories above.
- [x] **`public/logo/` and `public/logos/` are two different things — do not
      merge or rename them.** Checked 31 Jul because the names look like a
      typo for each other. They are not:
      - `public/logo/` — **our** brand kit, 13 Quadis SVGs (wordmarks,
        monograms, favicon, app icon). 80 KB. Only 4 are referenced; the rest
        are a deliberate complete mark set, not clutter.
      - `public/logos/partners/` — **third-party** partner and client logos,
        referenced from `src/data/logos.ts`. 17 references.

      Renaming either is a bad trade: both are referenced as **absolute URL
      strings**, so a missed one produces a broken image at runtime with no
      build error and no test failure. The confusion costs a moment; a silent
      broken logo on the live site costs more.
- [ ] **`dist` is now 64 MB** (was 48 MB) — 30 MB in `dist/images`, 29 MB in
      `dist/assets`, because every image still ships twice. See "Known, not
      urgent"; the workaround there still stands, but the number is growing.
- [ ] **Two zips landed unprocessed.** `in dining and catering landing page.zip`
      (8 files, SEO-named, for the dining/catering page) and `photo gallery.zip`
      (**294 MB**, ~72 unique images across All / Deluxe room / Superior & Super
      Deluxe / Facade & Lounges / Royal Deluxe room — `All/` is a superset of
      the rest). Almost all are ~2 MB PNGs. They cannot go into `public/` as
      sent: see incident 4, and `dist` is already 48 MB.
- [ ] **Finish the Razorpay wiring — see `docs/razorpay-golive.md`** for the
      full runbook, the verification steps and the failure-mode decoder. Razorpay side is DONE
      (29 Jul): live keys generated on the **Feb '26** merchant account, webhook
      created and Enabled at
      `https://djqj43186y3yh.cloudfront.net/api/webhooks/razorpay` with
      `payment.captured` / `payment.failed` / `order.paid`, secret set. Keys
      verified against Razorpay's API (`auth: 200`). **Our side is still on the
      placeholders** — `RAZORPAY_KEY_ID=rzp_test_simulated`,
      `RAZORPAY_WEBHOOK_SECRET` absent — because the EB config update wedged the
      instance and rolled back. See the incident below. The secrets are in the
      builder's password manager, not in this repo.
- [ ] **Two Razorpay accounts exist**, both "QUADIS SERVICES PRIVATE LIMITED" —
      created Sept '21 and Feb '26. We used **Feb '26**. If the client's old
      site ever took payments on the Sept '21 one, that is a different merchant
      account and settlements will land somewhere nobody is watching. Confirm.
- [ ] **Confirm Razorpay Payment Capture = Automatic.** On Manual, guests are
      debited, `payment.authorized` fires, we do not handle it, and the booking
      never confirms — visually identical to a webhook-secret mismatch.
- [ ] **Ask: is Amby Inn's "Executive room" the Super Deluxe?** She sent a photo
      captioned Executive, but Amby Inn's seeded categories are Deluxe 20 /
      Super Deluxe 3 — no Executive. Until she says, its Super Deluxe shows the
      Deluxe photo. One rename fixes it; guessing mis-sells a room.
- [ ] **Ask for a real Noida city photo.** `upcoming/noida.webp` is
      byte-identical to Hotel Cladis Sector 15's facade, which is precisely why
      she said "the current one is a building".
- [ ] **She asked for the AWS cost twice** — "Charges bata do server ka / Aws
      server to kharidna hi hain na". Already listed in §5; she is now waiting
      on it, so it is no longer just open, it is overdue.

Her item 1 — vouchers and booking confirmation — she could not test because
booking is unfinished, and §2.4 says do not build further into it. Her own
wording agrees: *"usse phle everything is great."*

- [ ] **Stop CloudFront masking API errors.** The SPA deep-link fallback rewrites
      any `/api/*` 4xx/5xx into `index.html`, so every backend error reaches the
      frontend as `Unexpected token '<'`. That is what made incident 5 cost an
      afternoon instead of a minute. Exclude the `/api/*` behaviour from the
      custom error response.
- [ ] **Set `trust proxy`.** `express-rate-limit` throws
      `ERR_ERL_UNEXPECTED_X_FORWARDED_FOR` on every request. Behind the load
      balancer every visitor shares one key, so the 100-req/15-min limit is
      global, not per-user — the site can rate-limit itself under light traffic.
- [ ] **Guest phone numbers land in access logs.** `GET /:code/invoice` takes
      `?phone=`, so every invoice download writes a real mobile number into
      nginx `access.log` in plaintext. Move it to a header or POST body.
- [ ] **`CORS_ORIGIN` must be updated at DNS cutover.** It is now
      `https://djqj43186y3yh.cloudfront.net`. The day `quadishotels.com` goes
      live it needs that origin too, or checkout breaks again — same failure as
      incident 5, on the day it is least welcome.
- [ ] **Confirm the 13–17 band with the client.** They gave 0–7 free and 8–12 at
      20% and stopped. Code reads 13+ as adult. Worth ₹600/night on a ₹2,000
      room. One constant — `DEFAULT_ADULT_FROM_AGE` — in both pricing mirrors.

### Done and verified — do not redo

Numeric parsing (API returns `1500` unquoted) · RDS closed to the internet · TLS
certificate verification live (`rejectUnauthorized: true` + RDS CA) · 27
hardcoded image paths repointed to `.webp` · HTTPS + deep links via CloudFront ·
orphan CloudFront deleted, one distribution left · cancellation policy page ·
three stale buckets deleted · photo upload working end to end · child age bands
implemented, 108 tests green.

**28 Jul** — 8 commits pushed to `origin/main` (head `84a3e86`). Both halves of
`8577e4b` deployed as `co3-84a3e86`: backend on EB (Ready/Green/Ok), frontend
rebuilt and synced, CloudFront invalidated. Verified live, not just green — the
API returns `child_free_under_age:8`, `child_percent:20`, `adult_from_age:13`,
all unquoted, so `migrate.ts` applied the three-band schema and the
`WHERE child_free_under_age = 18` update ran. **Live pricing changed at this
point** — it is the first real billing change to reach production, and the
13-17 band in it is still unconfirmed.

**28 Jul — the booking flow was walked in a real browser, first time ever.**
Search → property → room → age bands → hold → payment → invoice. Found and
fixed incident 5 (CORS). Verified after the fix: hold created from the browser
(`QD-MWW6N6N5`), all three age bands price correctly on screen (6 free, 10 at
₹600, 15 at ₹900 on a ₹3,000 room), GST back-computes correctly (₹3,482.14 +
12% = ₹3,900), images lazy-load, console clean throughout, invoice endpoint
returns a valid 1-page PDF. Payment stops with "Online payment is not enabled
on this environment" — correct degradation for `rzp_test_simulated`, not a
fault, and it is blocked on the client sending live keys.

**28 Jul — corporate banner corrected and deployed (`4fa8b1c`).** The client
flagged the wrong banner on the Corporate page; she was right. Fixed via a new
`corporateBannerImage` rather than repointing `corporateStayImage`, which also
feeds the home Offerings card. Verified live, not just built: the hero serves
`corporate-hotel-booking-banner-fv5WBSdE.webp`, the home card still serves
`corporate-and-long-stays-BiiR7QBY.webp` (read off the live DOM), console clean,
`base_price` still unquoted.

**29 Jul — Quadis Experience tier image replaced, built but NOT deployed.** The
client sent a pool/facade render with "Quadis Experience" on the signage and
asked for it on that tier card, replacing the one already there ("vo vali htane
k liye khre h"). Swapped in place at
`public/images/tiers/tier-quadis-experience.webp` — same filename deliberately,
because `namedIn` matches on `startsWith`, so a second file beginning
`tier-quadis-experience` would make which one wins depend on glob order. Source
1536×1024, converted at the project's 1200px/q80 settings to 1200×800, 147 kB.
The card is `aspect-ratio: 4/3` + `object-fit: cover`, so a 3:2 source loses
~6% off each side; checked against a rendered centre crop, the signage and pool
both survive it. Typecheck and build clean, new hash
`tier-quadis-experience-C2QAx-kq.webp`. **Still needs an upload, a CloudFront
invalidation and a browser check before it counts as done — §2.5.**

~~Still unprocessed from the same 28 Jul batch: `Banquet Halls.zip`...~~
**Resolved in `8bcf756`** — all 12 photos are in `public/images/banquets/` under
the three venue slugs (4 each), verified on disk 29 Jul. The banquet cards no
longer fall back to dining photos. This paragraph claimed otherwise for a day
after it stopped being true; if you are reading a "still outstanding" note here,
check it on disk before repeating it.

**Genuinely still outstanding — per-hotel photography.** Six of the nine hotels
have exactly one image in `public/images/hotels/<slug>/`:

| Hotel | Files |
|---|---|
| `hotel-amar-inn` | 15 |
| `hotel-quadis-central-sector-27-noida` | 8 |
| `hotel-quadis-sector-51-noida` | 8 |
| `hotel-downtown-sector-51-noida` | 6 |
| the other five | **1 each** |

`images.ts` pads any property under five photos from *other* properties, so
those five currently show rooms the guest is not booking. The 29 Jul
`photo gallery.zip` does **not** fix this — it is filed by room type (Deluxe,
Superior & Super Deluxe, Royal Deluxe, Facade & Lounges), not by hotel, so it
improves the shared pools without making any single property accurate.

> **RESOLVED 29 Jul — built, NOT deployed.** The five hotels' own photos had
> been sitting in `client-assets/unpacked/Hotels/` since the `Hotels.zip` send;
> we had only ever cut the facade as `hero.webp`. 26 photos placed at
> 1200px/q80. All nine properties now resolve ≥5 of their own photos and none
> is padded from another — simulated against the real tree, then walked in a
> browser. The table above is kept as the record of what was wrong.
>
> **Naming rule, keep it: room-type keywords only where the client's own
> filename states the type.** `roomImages()` keyword-matches the room slug
> against the filename, so a wrong keyword shows a guest the wrong room class on
> a booking page. The SEO-named shots ("best hotel near sector 15 noida.png")
> are `NN-room.webp` — they fill the gallery, which was the actual bug, and
> claim nothing about category. Flat files, **not** per-room subfolders: buckets
> are keyed on the exact directory (`images.ts:22-30`), so a subfolder feeds
> `roomImages()` but not `hotelImages()` and would leave the padding in place.
>
> `Royal Deluxe Room.png` was placed as `03-royal-suite.webp` — named for the
> seed slug and deliberately without the word "deluxe", because `deluxe-room`
> matches on that substring and the suite would have been served as a Deluxe.
>
> **Two things this exposed:**
>
> 1. **`roomImages()` had deluxe and super-deluxe crossed.** `super-deluxe`
>    splits to `['super','deluxe']` and `.some()` matched `01-deluxe-room.webp`,
>    while `deluxe-room` split to `['deluxe']` and matched
>    `02-super-deluxe.webp` — each category served the other's photograph, and
>    the cheaper room was advertised with the dearer one's picture. Fixed by
>    trying a whole-slug match first, keeping the loose pass as fallback because
>    the older sets are named the other way round (`room-deluxe.webp`,
>    `05-superior-room.webp`) and only the loose pass finds those. Verified in a
>    browser on three properties.
> 2. **`upcoming/noida.webp` is byte-identical to
>    `hotels/hotel-cladis-sector-15-noida/hero.webp`.** The "Noida" destination
>    photo *is* the Hotel Cladis building — exactly the client's complaint that
>    "the current one is a building". Root cause confirmed; still needs a real
>    city photo from her.
>
> **Still owed by the client here:** Amby Inn's Super Deluxe falls back to its
> own Deluxe photo. She sent Deluxe and *Executive*, no Super Deluxe, and
> "Executive" matches no seeded category (Amby Inn is deluxe 20 / super 3). If
> Executive **is** what they call the Super Deluxe, renaming
> `02-executive-room.webp` to `02-super-deluxe.webp` fixes it — ask first. Do
> not guess which room a guest is paying for.

Also `tiers/tier-quadis-select.webp` is 395×276 beside siblings at 1200px. Not a
conversion fault — the client's own source file is that size. Needs a bigger one
from them.

Only one file under `client-assets/` is tracked: `property-data.md`
(lat/lng, copy decisions, photo categorisation). Checked at push time — it holds
no credentials. The briefs and passwords are untracked, as §2.1 requires.

### Known, not urgent

**`dist` is 48 MB because every image ships twice** — 22 MB in `dist/images`
(Vite copies `public/` verbatim) and 21 MB in `dist/assets` (the same files
hashed by `import.meta.glob('/public/images/**')`).

Worth halving, but **not** by dropping the glob for API-only image resolution.
That removes the offline fallback, so an API outage or a missing row renders
pages with no images at all — and this API has gone down twice in one afternoon.
Keep the glob as the fallback and let uploads override it, which is what
`imagesForHotel()` already does.

---

## 4a. Claims we have made to the client, and what backs them

Re-verified 30 Jul before sending the cost message. Three claims that had been
sitting in drafts were wrong. Check this list before writing her another one.

| Claim | Status |
|---|---|
| "Pehle 3-8 hazaar bola tha" | ✅ `round-1-sent.txt` message 1 item 4, exactly that |
| She asked for the cheapest option | ✅ her words, 27 Jul: *"AWS hosting k liye sir ne kaha jo sbse km price ka hoga vhi lgana"* |
| `info@quadishotels.com` is on Google | ✅ MX = `smtp.google.com`, and her own *"Mail id gmail pr chlti h"* |
| Old host cannot run the new site | ✅ but **only for the stated reason** — see below |
| Monthly billing, no lock-in | ✅ **true on AWS** — on-demand is billed monthly by usage with no commitment. It was *false* on the Hostinger plan this message used to describe, whose own page says "all plans are paid upfront". The claim survived only because the platform changed under it |
| ~~₹1,500-2,000/mo~~ | ❌ **WAS SENT TO HER, AND WAS WRONG.** Priced on `t3.small`. Superseded — see the row below. The three comms files that still *instructed* sending this number were corrected 31 Jul |
| **~₹3,550/mo on `t3.medium`** | ✅ **she agreed 31 Jul**, and the account was upgraded to the Paid plan on the strength of it. ⚠️ Provenance: relayed to us in-session, not read off a message thread — if you need her words, get them before quoting her back to herself. Her $99.47 credit covers ~2.4 months of it and **she has not been told that yet — the line is now written into `message-a-server-cost-SEND.txt`, which is drafted and waiting to send** |
| "AWS pe hi rahega" | ✅ and it is what she asked for by name — *"AWS hosting k liye sir ne kaha…"* |
| Mumbai | ✅ `ap-south-1`, verified enabled on her account |
| Card is the company's | ⚠️ **asked, not verified.** Not visible from the CLI. Rule is non-negotiable — §2, same as Razorpay |
| "Roz ka backup" | ⚠️ **now a promise we owe.** Their page says **"Free weekly backups"** and daily is an enable-it extra. Task board has the cron |
| Email will not be touched | ⚠️ true **only on the VPS path** (static IP → one A record). False on CloudFront, which forces a nameserver move |

**The claim that was outright wrong: "shared hosting can't run online booking
because it has no database."** It was in the draft for a day. Her current host
runs `adminweb.quadishotels.com` — hotels, bookings, coupons, occupancy — on
that same IP, so it plainly has a database. She, her "sir", or the old designer
could have said so in one line and we would have looked like we had not looked.

The true reason is narrower and already in §5: her stack is **Windows /
IIS / ASP.NET / Plesk** and ours is **Node + Postgres**. That box cannot run
ours. Say that, not "shared hosting is weak".

**"Sir ne kaha" — the cost decision is not hers alone.** Her 27 Jul message
attributes the cheapest-option instruction to a senior. Expect the ₹1,500-2,000
confirmation to go through him, and write anything about money so it survives
being forwarded.

Also from that same message: *"Login details purane designer se magi h"* — she
had already asked for the theserverindia login on 27 Jul. **That chase produced
nothing in four days and has now been routed around** — she is going to the host
directly instead. §5 "The hosting panel".

---

## 5. Blocked on the client

Messages are drafted in `docs/client-comms/`.

✅ **`message-g-where-bookings-show-SEND.txt` was SENT 3 Aug.** It answered
where bookings and form details show, and — the important part — it **asked the
two-systems question** that `message-existing-admin-panel.txt` had been waiting
since 27 Jul to ask. That older draft is spent as an opener; do not send it now.

**We are therefore no longer blocked on ourselves here. We are waiting on her
answer to one question: does booking and availability live in the new panel or
in `adminweb`?** Everything in the rest of §5 is downstream of that answer.
Chase it — it is the single decision that says whether the booking engine,
CheckoutModal, RazorpayService and the hold engine are load-bearing or dead
code (§2 rule 4).

### They already have an admin panel — discovered 27 Jul

`adminweb.quadishotels.com` is a live hotel management system on their existing
host: Dashboard, Hotels, Booking, Settings, Pages, Menu, Coupon, Add Occupancy,
Users. It holds their real inventory — Quadis Central 17 rooms, Downtown Sector
15 28 rooms, matching the rate sheet. It already has SEO meta fields, image
upload, occupancy pricing and a booking module: four things built here in the
last two days.

Verified: resolves to `115.124.108.190` (same server as the site), HTTPS 200,
titled "Hotel QuaDis". `booking.quadishotels.com` sits on the same IP.

Almost certainly the "Booking System connected with admin panel" from their
first reply, and possibly what the revenue manager meant by PMS.

**If it stays, rates and availability are maintained in two places** — which is
the double-source problem that causes double bookings, and the same one the PMS
question is about. It has to be one system or the other, and the client has to
say which.

Open: who built it and is it supported · is there data to migrate (bookings,
coupons, users) · does the new site replace it or run alongside.

**29 Jul — "is it live and taking bookings" is largely answered, by TLS.**
Certificates on the three subdomains, read off the live hosts:

| Host | Cert validity | State |
|---|---|---|
| `www.quadishotels.com` | 24 Jun → 22 Sep 2026 | live, auto-renewing |
| `adminweb.quadishotels.com` | 18 Jul → 16 Oct 2026 | live, renewed 18 Jul |
| `booking.quadishotels.com` | 18 Jan → **18 Apr 2026** | **expired 3+ months** |

These are 90-day certs on auto-renewal. Two keep renewing; `booking.` stopped in
April, which is what happens when a vhost is disabled — not when a site is in
use. An expired cert would have shown every guest a full-page security warning
since 18 April, so **their guest-facing online booking is not running.**

So the §2.4 fear — that we are rebuilding what they already have — does not hold
for *booking*. They have a maintained **admin/PMS** and a **dead booking front
end**; only the second is what CheckoutModal replaces. It also matches the
client's own 29 Jul complaint, "abhi booking nhi bn pa rhi".

Still confirm with her before betting on it: the panel may take bookings that
staff key in by hand, which the dead subdomain says nothing about.

Also verified 29 Jul: the existing site is an **Angular SPA on IIS / ASP.NET /
Plesk for Windows**, not the Linux shared hosting `docs/dns-cutover.md` assumes.
The existing stack cannot host the Node backend — long term this is
replace-or-run-alongside, never merge.

#### 3 Aug — she asked the two-systems question herself. Do not waste it.

*"forms details and booking details kaha show hungi **admin panel me?**"* — her
words, 3 Aug, after signing off the live site (§4). She is not asking where we
put a feature. She is asking, without knowing it, which of two systems is the
one her staff will work in.

The factual answer, verified 3 Aug and not assumed:

| | Her existing panel | The new site |
|---|---|---|
| Where | `adminweb.quadishotels.com` on `115.124.108.190`, Windows/IIS/ASP.NET/Plesk | `https://www.quadishotels.com/admin`, Node on her EC2 box |
| Storage | its own database on the old host | Postgres on `13.234.85.127`, loopback-only |
| Holds | their real historic inventory, coupons, users | `bookings`, `enquiries`, `users`, `room_night_holds`, `chat_logs` |
| Connection between them | **none** | **none** |

`grep -rniE "adminweb|115\.124\.108\.190|theserverindia"` over `backend/src` and
`src` returns **nothing**. There is no integration, no sync job, no export, and
no plan for one. A guest booking on the new site is invisible to `adminweb`, and
an availability change keyed into `adminweb` is invisible to the new site.

**That is the double-booking mechanism, stated plainly, and it is live now.**
While `booking.quadishotels.com` was dead the risk was theoretical — nobody was
taking online bookings but us. It stopped being theoretical on 1 Aug, because
staff keying a walk-in into `adminweb` are now editing inventory that our site
is simultaneously selling.

It has to be one system or the other, and she is the only one who can say which.
Asking as part of answering her own question costs nothing; raising it cold in a
month costs a guest their room.

### The hosting panel — ✅ resolved 1 Aug, the A record moved. History below

> **Everything in this subsection is history.** The record moved on 1 Aug and
> the site is live (§3b). Kept for one reason: the route it describes — going
> to the host directly instead of through the old designer — is what broke a
> four-day stall, and that shape will recur. Read the present tense below as
> "as of 31 Jul".

The app is built, deployed and healthy on her own account (§3b). **The only
thing left is one A record**, and it is behind a control panel we cannot reach.
The thread has moved twice in one day, so do not re-litigate it from §3a alone:

1. `message-b-dns-access-SEND.txt` **was sent**. It deliberately gave her a
   route that does not depend on the old designer — she has been chasing him
   since 27 Jul with nothing back, and she is theserverindia's customer while
   he is a middleman.
2. **She answered by drafting the support email herself, and it is right.** It
   asks for a panel reset *or* the A record moved to `13.234.85.127`, and it
   names MX, SPF, DKIM and DMARC as must-not-touch. Do not rewrite it — that
   is the message we would have written. Screenshot:
   `client-assets/briefs/2026-07-31-whatsapp-support-email-draft-and-addresses.png`.
3. She asked only **where to send it**, forwarding two addresses.
   `message-c-which-support-address-SEND.txt` answers that — **sent 31 Jul**.
4. **She then appears to have mailed the host. Believed, not confirmed** — it
   reached us as "ig she sent mail to them", with no sent copy, no reply and
   no ticket number seen. Treat it as unverified: if the mail was never sent,
   the symptom is identical to the host ignoring it, and both look like
   silence. Ask her to forward it. Provenance discipline, §4a.

**Do not wait on being told.** `./deploy/cutover.sh --check` answers "have they
acted yet" with no client contact at all — it is read-only, safe to run any
time, and the apex-A line flips the moment they touch the record. As of the
last run the apex is still `115.124.108.190` and everything else is green.

**`theserverindia` is now `host.co.in` (ESDS Ltd Group)** — the domain
301-redirects there. Re-verified 31 Jul, and it is worth re-checking before
quoting, because it is the address she will actually mail:

| Check | Result |
|---|---|
| `theserverindia.com` | 301 → `https://www.host.co.in/` |
| `yellow1.theserverindia.com` | `115.124.108.190` — **her site's exact IP**, which is how support locates her account with no customer ID |
| `indianservers.com` | Cloudflare, a different network, unmentioned by `host.co.in`. Almost certainly an unrelated company — the other address she was given |

Send to the **reseller first with the host CC'd**: if she bought through the
Gurgaon agent, that agent holds the panel and `host.co.in` answers "you are not
our customer". CC covers both cases without her needing to know which is true.
Push the phone number (`+91 966 576 0700`) alongside the mail — this blocker has
a third party's response time inside it, and Indian hosting support moves faster
on a call than on a ticket.

The 13–17 age band · ~~the theserverindia hosting login~~ **→ now the
host.co.in support request above** · ~~live Razorpay keys~~ **→ WIRED 31 Jul.
All three secrets are in SSM as SecureString, the box is redeployed, and
`/api/payments/create-order` returns `"isSimulated": false` against the live
account. Two things remain and neither is hers: the webhook URL still points at
the dead CloudFront endpoint, and checkout cannot be exercised until DNS moves,
because Razorpay matches the origin against the approved website —
`docs/razorpay-golive.md`** · photo storage choice,
Cloudinary or S3 — `ImageStore` is an interface precisely so this stays open, do
not hard-wire a vendor · ~~AWS cost confirmation~~ **agreed 31 Jul at ~₹3,550,
§4a** · **the PMS decision in §2.4, which decides the most.**

---

## 6. How to verify anything here

Both serious incidents on this project passed every automated check while
production was down. Neither was catchable without a real database and a real
browser.

```bash
# 1. Numbers must be unquoted.
curl -s https://djqj43186y3yh.cloudfront.net/api/properties \
  | grep -o '"base_price":[^,]*' | head -2
#    want "base_price":1500        NOT "base_price":"1500.00"

# 2. Then open a browser. Cards must render, console must be clean,
#    prices must read ₹3,000 / night.
```

`curl` cannot see a React crash, and CORS failures only happen in a browser —
requests without an `Origin` header pass regardless.

---

## 7. Incidents — read these before assuming a green check means anything

1. **TLS refusal.** RDS Postgres 16+ forces SSL; node-postgres connects
   plaintext. The app crash-looped. `psql` connected fine with the identical
   URL, because psql negotiates TLS by default — so every manual check passed
   and the error named `pg_hba.conf`, which reads like a firewall problem.
2. **NUMERIC as string.** Postgres returns `NUMERIC` as text.
   `rating.toFixed(1)` threw, React unmounted, every page rendered blank. Worse
   quietly: `price + offset` concatenated, so a ₹1,500 room with a ₹1,000
   upgrade quoted `"15001000"`. Local dev, CI and 102 tests all passed.
3. **Wrong security group.** A handoff named the load balancer group instead of
   the instance group. The agent applied it faithfully and the database went
   unreachable — 40-second hangs, health checks still green.
4. **Image rename without a reference sweep.** `public/images` was converted to
   WebP and the originals deleted, but only `src/data/images.ts` was updated.
   27 paths across 9 other files kept pointing at `.png` and `.jpg` and 404'd
   live, including the whole Destinations grid. Typecheck, build and tests all
   passed — the paths are strings.

5. **CORS blocked every write, and only in a browser.** `CORS_ORIGIN` on EB was
   still the pre-CloudFront S3 website URL, so the CloudFront origin was not on
   the allowlist. Reads were unaffected and the site looked completely healthy —
   because browsers omit `Origin` on same-origin GETs but **send it on every
   non-GET**, even same-origin. So `POST /api/bookings/initiate` carried an
   `Origin` the allowlist rejected, the cors middleware threw, and Express's
   default handler returned its HTML error page. The frontend called
   `res.json()` on that and surfaced `Unexpected token '<'`, which reads like a
   routing bug and is not one. `curl` sends no `Origin`, so the same POST
   returned 201 from both the origin and CloudFront. Booking had never once
   worked from a browser; the only end-to-end test on record was a curl POST.
   Found 28 Jul by walking the flow in a real browser — §6 predicted exactly
   this and named the reason.

6. **A single env-var change took the API down for 30 minutes.** 29 Jul, setting
   the three Razorpay variables. EB restarts the app for any config change; on
   this box the restart wedged the instance — it stopped sending health data
   ~90 seconds in, 50% of ELB requests went 5xx, EB burned its **15-minute**
   timeout, **reverted the configuration**, then sat in `Updating` another 20
   minutes. That state also blocks every EB operation, so `restart-app-server`
   returned `InvalidParameterValue: Must be Ready`. What fixed it was going
   around EB entirely: `aws ec2 reboot-instances`. API answered 200 five minutes
   later.
   - EC2 said `running`/`ok` throughout and CPU credits were untouched (270 of
     288), so it was neither a dead VM nor throttling. CPU sat at ~50% — one of
     the two vCPUs pegged — from the moment the update began.
   - **The box is a `t3.micro`: 2 vCPU but only 1 GiB RAM**, and RDS is
     `db.t3.micro`/20 GB. That is the **Free Tier** shape, and the July bill is
     effectively ₹0 — which is *why* it was chosen. `t3.micro` is the only free
     EC2 size, so there is no free way up.
   - It had been up 7 days. A fresh reboot may be enough for a retry; the
     durable fix is a swap file via `.ebextensions`, which costs nothing.
   - Health had also been flapping `Ok → Warning → Ok` roughly hourly all that
     day *before* anyone touched it. The instance was already at its limit.
   - **When this moves to the client's own account, size it `t3.small`
     minimum.** There will be no free-tier reason to accept 1 GiB.
   - The frontend stayed up the whole time — it is served from S3, so only
     `/api/*` died. Guests could browse; nothing could book.

The pattern: **the automated suite cannot see the failures this project
actually has.** Three of the six above were invisible to it. Assume a seventh
exists.

---

## 8. Where the numbers came from

Rates, room categories and inventory are the client's rate sheet of 27 Jul 2026,
in `client-assets/briefs/`. Read `client-assets/briefs/INDEX.md` before changing
any of them — it records which of three conflicting sources the code follows and
why. Totals reconcile exactly: 125 Noida + 72 Delhi = 197 keys.

---

## 9. Conventions

- Comments explain **why**, not what. If a line looks odd, the comment says what
  broke without it.
- No hex outside `src/styles/tokens.css`. There is no Tailwind — class names
  like `py-12` resolve to nothing.
- `tsconfig` sets `noUnusedLocals`; orphaned imports fail the build.
- `npm run typecheck` and `cd backend && npm test` before declaring anything done.
- `src/lib/pricing.ts` mirrors `backend/src/lib/pricing.ts`. Change one, change
  the other — the quote a guest sees and the amount they are charged come from
  these two files and must agree.
- Do not zip `node_modules` for a Beanstalk deploy. `sharp` ships
  platform-specific binaries and must be built on Amazon Linux.
