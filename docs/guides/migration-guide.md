---
page_title: "Migrating to the Terraform Plugin Framework (v3.0.0)"
subcategory: ""
description: |-
  What changed in the Britive provider v3.0.0 rewrite, what is new, the risks involved, how to back up your state before upgrading, what to expect on your first plan after upgrading, and how to roll back to the legacy SDK-based provider if you hit an issue.
---



# Migrating to the Terraform Plugin Framework (v3.0.0)



Starting with **v3.0.0**, the Britive Terraform provider has been rewritten from the ground up on
[HashiCorp's Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) (protocol
version 6), replacing the legacy [Terraform Plugin SDKv2](https://developer.hashicorp.com/terraform/plugin/sdkv2)
implementation used in v2.x releases (up to and including **v2.3.x**).



This guide explains:



- Why this migration happened and what it means for you.
- What is unchanged, and what new functionality v3.0.0 adds.
- The risks involved, since this is a **full internal rewrite**, not an incremental change.
- How to **back up your Terraform state** before upgrading.
- What to expect on your **first plan and refresh** after upgrading.
- How to **roll back to provider v2.3.x** if you run into problems after upgrading.



## Why migrate to the Plugin Framework?



HashiCorp is steering the Terraform provider ecosystem toward the Plugin Framework and has placed SDKv2 in
maintenance mode — it no longer receives new features, and long term it will not support newer Terraform protocol
capabilities. Rewriting the Britive provider on the Plugin Framework lets us:



- Adopt **protocol version 6**, required for newer Terraform capabilities.
- Get stronger, framework-level **type safety** for attributes (no more generic `interface{}` handling).
- Use native **plan modifiers** and **validators**, reducing perpetual diffs and invalid-configuration errors.
- Stay aligned with HashiCorp's supported tooling, so future Terraform releases continue to work with this provider.



## What did NOT change



Your existing configuration should not need any edits to work with v3.0.0.



- **Existing schemas are unchanged.** Every resource and data source argument, attribute name, and nested
  block keeps the same name, type, and nesting as in v2.3.x. Nothing was renamed, retyped, or removed.
- **The provider surface is unchanged.** v3.0.0 registers the same **28 resources and 10 data sources** as
  v2.3.x — none added, none removed.
- **Terraform state format is compatible.** You should not need to run `terraform state mv`, re-import
  resources, or otherwise manually migrate state.
- **Provider configuration** (`tenant`, `token`, environment variables, etc.) is unchanged.



## What is new in v3.0.0



Alongside the rewrite, v3.0.0 adds new optional functionality. These additions are purely additive: existing
configurations continue to work untouched, and you do not need to adopt any of it as part of the upgrade.



- **Application-scoped permissions.** `britive_permission` gains an optional `permission_scopes` argument —
  a set of application types (for example AWS, Azure, GCP) that the permission is restricted to. It can only
  be set when `consumer` is `apps` and `resources` is `["*"]`; other combinations are rejected by the Britive
  API.
- **Prompt-at-checkout variables.** The `variables` blocks on `britive_resource_manager_profile_permission`
  gain `prompt_at_checkout` and `regex_pattern`, so a value can be supplied by the user at checkout instead
  of being stored in configuration. Password-type variables require `prompt_at_checkout = true` and never
  carry a value in configuration.



**Adopting new functionality closes the rollback path.** These attributes do not exist in v2.3.x, so a
configuration that uses them will fail validation if you pin the provider back down. If you want to keep
rollback available during your upgrade window, upgrade first and confirm everything is stable, then adopt
new functionality later as a separate change.



## ⚠️ Risk: this is a full rewrite



Even though the goal was a drop-in replacement, **every resource and data source was rewritten from scratch**.
This is materially different from a typical point release, and it carries more risk than a normal upgrade:



- Subtle behavioral differences can exist even when schemas match exactly — for example, how
  optional/computed fields are resolved, how plan modifiers suppress diffs, or how default values are
  applied.
- Provider-side validation is stricter under the Plugin Framework, so configurations that were silently
  accepted by v2.3.x could now fail validation at `terraform plan`/`apply` time.
- Edge cases in fields with historically tricky diff behavior, sensitive fields, Optional+Computed attributes
  are the most likely places to see a difference, even though they were explicitly targeted for fixes.



**Recommendation:** Do not upgrade production workspaces directly. Follow the staged rollout below.



### Recommended rollout



1. **Back up your state** for every workspace that uses this provider (see [below](#back-up-your-terraform-state-before-upgrading)).
   This step is load-bearing, not precautionary: once v3.0.0 has written state — even via a refresh with zero
   infrastructure changes — that state can no longer be read by v2.3.x, and the backup is the only rollback
   path.
2. Upgrade a **non-production** workspace/environment first.
3. Run `terraform plan` and carefully review the output:
   - ✅ Expected: **no changes** (`No changes. Your infrastructure matches the configuration.`).
   - ⚠️ Investigate: any unexpected diff, especially destroy/recreate actions, before running `terraform apply`.
     See the [next section](#what-to-expect-on-your-first-plan-after-upgrading) for one category of change
     that is expected.
4. If the plan is clean, proceed to apply in non-production, validate the managed resources in the Britive
   console, then repeat for production.
5. If at any point the plan or apply looks wrong, **stop**, do not apply, and follow the
   [rollback steps](#rolling-back-to-provider-v23x) below.



## What to expect on your first plan after upgrading



On the first plan or refresh under v3.0.0, Terraform is likely to report a block titled **"Note: Objects have
changed outside of Terraform"**. This is a one-time state normalization performed by the new provider's Read
functions, not drift in your tenant. Typical entries:



- **Computed attributes being populated** — fields that v2.3.x left unset in state are now filled in by
  v3.0.0 (for example `template_id` on response templates, or `type` inside resource-manager permission
  variables). These appear as additions, with the values you configured unchanged.
- **Empty ↔ null normalization** — SDKv2 stored empty strings and empty collections; the Plugin Framework
  normalizes these to null (for example `resource_label_color_map = []` -> `null`).
- **Read enrichment** — v3.0.0 may populate both the ID and the name of referenced objects where v2.3.x
  stored only the one you configured (for example tag owner user/tag blocks).



Accepting these into state (for example via `terraform apply -refresh-only`, or as part of a normal apply) is
safe and expected. If you see changes that do not fit these patterns — in particular value changes on
attributes you configured, or destroy/recreate actions — stop and investigate before applying.



## Back up your Terraform state before upgrading



Do this for **every** Terraform workspace/root module that uses the `britive` provider, before you change the
`version` constraint or run `terraform init -upgrade`.



### Step 1 — Identify your backend



Check the `backend` block in your Terraform configuration (or run `terraform init` and look at the output) to
confirm where your state lives: local file, Terraform Cloud/Enterprise, S3, Azure Storage, GCS, Consul, etc. The
backup method differs slightly depending on backend.



### Step 2 — Pull a local copy of the current state



Regardless of backend, you can always pull a point-in-time copy of the *current* state to a local file:



```bash
terraform state pull > backup-pre-v3-$(date +%Y%m%d%H%M%S).tfstate
```



Verify the file is not empty, looks like valid JSON, and record the serial — you may need it during rollback:



```bash
jq '.version, .terraform_version, .serial' backup-pre-v3-*.tfstate
```



Store this file somewhere safe outside of the working directory (e.g. a separate backup folder, S3 bucket, or
internal artifact storage) — do **not** rely solely on a copy sitting next to your `.tf` files.



### Step 3 — Back up the local state directory (local backend only)



If you're using the default local backend, also copy the `.terraform.lock.hcl` file and any `terraform.tfstate*`
files. Note that `terraform.tfstate.backup` only exists after at least two state writes — if it is absent on a
young workspace, that is normal and nothing is lost:



```bash
TS=$(date +%Y%m%d%H%M%S); mkdir -p ../state-backups/$TS
cp -v terraform.tfstate ../state-backups/$TS/
# terraform.tfstate.backup may legitimately not exist yet:
[ -f terraform.tfstate.backup ] && cp -v terraform.tfstate.backup ../state-backups/$TS/
cp -v .terraform.lock.hcl ../state-backups/$TS/
```



The `.terraform.lock.hcl` backup is important — it records the exact provider version/checksums currently in
use and is what you'll restore if you need to pin back to v2.3.x.



### Step 4 — Back up remote state (if applicable)



- **Terraform Cloud/Enterprise:** State versions are already kept automatically. Confirm you can see the current
  state version in the workspace's **States** tab, and note its version number/timestamp before upgrading.
- **S3 backend:** If the bucket has **versioning enabled**, the current object version is your backup — note the
  object's version ID. If versioning is *not* enabled, copy the state object explicitly:
  ```bash
  aws s3 cp s3://<your-bucket>/<key> s3://<your-bucket>/backups/<key>-$(date +%Y%m%d%H%M%S)
  ```
- **Azure Storage backend:** Ensure blob versioning/soft-delete is enabled, or copy the blob to a backup
  container/path before upgrading.
- **GCS backend:** Ensure object versioning is enabled on the bucket, or copy the object to a backup path.



### Step 5 — Confirm the current provider version



Record the provider version currently pinned, so you know exactly what to roll back to:



```bash
terraform version
grep -A 3 'britive/britive' .terraform.lock.hcl
```



Only proceed to upgrade once you have a verified state backup and know your current pinned version (expected to
be `2.3.x` or earlier for anyone following this guide).



## Rolling back to provider v2.3.x



If you upgrade to v3.0.0 and observe unexpected plan diffs, apply failures, or incorrect behavior against a
resource, roll back rather than working around the issue in a production environment.



**Expect this error if v3.0.0 has already written state.** If any v3.0.0 plan refresh, refresh-only apply, or
apply has updated your state, the v2.3.x plan in [Step 4](#step-4--validate-against-your-backed-up-state) will
fail with errors of the form:



```
Error: Resource instance managed by newer provider version

The current state of <resource> was created by a newer provider version
than is currently selected. Upgrade the britive provider to work with this state.
```



This is expected, not a sign that anything is broken on your tenant: v3.0.0 upgrades resource schema versions
in state, and v2.3.x cannot read them back. It is your cue to restore the state backup
([Step 4](#step-4--validate-against-your-backed-up-state) below) rather than trying to make the in-place state
work.



### Step 1 — Pin the provider version back down



Edit the `required_providers` block in your Terraform configuration, pinning the exact version you recorded in
backup [Step 5](#step-5--confirm-the-current-provider-version) (or a constraint such as `~> 2.3.0`; note that
Terraform requires valid constraint syntax — a literal placeholder like `"2.3.x"` will be rejected by
`terraform init`):



```hcl
terraform {
  required_providers {
    britive = {
      source  = "britive/britive"
      version = "2.3.6" # the exact version recorded in backup Step 5
    }
  }
}
```



### Step 2 — Restore the lock file (recommended) or regenerate it



If you saved `.terraform.lock.hcl` in your backup ([Step 3](#step-3--back-up-the-local-state-directory-local-backend-only) above), restore it to guarantee you get back the exact same provider build:



```bash
cp ../state-backups/<timestamp>/.terraform.lock.hcl .
```



If you don't have a saved copy, regenerate the lock entry for the pinned version instead:



```bash
rm -f .terraform.lock.hcl
terraform init -upgrade
```



### Step 3 — Reinstall the pinned provider



```bash
terraform init -upgrade
```



Terraform will download v2.3.x of the `britive/britive` provider from the registry and update
`.terraform.lock.hcl` accordingly. Confirm the version:



```bash
terraform providers
```



### Step 4 — Validate against your backed-up state



Run a plan before applying anything:



```bash
terraform plan
```



- If the plan is clean (no unexpected changes), you're safely back on v2.3.x.
- If Terraform reports state that looks inconsistent — including the "Resource instance managed by newer
  provider version" errors described above, which occur whenever a v3.0.0 refresh or apply has written state —
  restore the state backup you took earlier instead of trusting the in-place state:



  ```bash
  terraform state push backup-pre-v3-<timestamp>.tfstate
  ```



  Use `state push` with care — it overwrites remote state. Confirm you're pushing to the correct workspace, and
  consider this a last resort in a controlled maintenance window rather than a routine action.



  **If this fails with `cannot import state with serial N over newer state with serial M`:** the remote state has
  advanced (e.g. a v3.0.0 apply or refresh bumped the serial) since you took the backup, and Terraform is
  refusing to silently discard that history. Do not reach for `-force` immediately — first check what you'd be
  throwing away:



  ```bash
  terraform state pull > current-state.tfstate
  jq -c '.resources[] | {type, name, module}' backup-pre-v3-<timestamp>.tfstate \
    | sort > /tmp/backup-resources.txt
  jq -c '.resources[] | {type, name, module}' current-state.tfstate \
    | sort > /tmp/current-resources.txt
  diff /tmp/backup-resources.txt /tmp/current-resources.txt
  ```



  (The `-c` flag keeps each resource on one line so the sort and diff are meaningful.)



  - If the diff shows nothing you care about losing, bump the serial in your backup above the current one and
    push normally — this still enforces the lineage check, so you can't accidentally push a state file from the
    wrong workspace:



    ```bash
    jq '.serial = <serial_number>' backup-pre-v3-<timestamp>.tfstate > backup-pre-v3-bumped.tfstate
    terraform state push backup-pre-v3-bumped.tfstate
    ```



  - Only use `terraform state push -force` if you understand it skips *both* the serial and lineage checks. Prefer
    the serial-bump approach above unless you have a specific reason to bypass the lineage check too.



Finish with a final `terraform plan` — it should report "No changes.", confirming the rollback is complete.



## Getting help



If you're unsure whether a plan diff after upgrading is expected, stop before applying and reach out to the
Britive team rather than guessing. It's much easier to review a `plan` than to recover from an unwanted `apply`
against production infrastructure.
