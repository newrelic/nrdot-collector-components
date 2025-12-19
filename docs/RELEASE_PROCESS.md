# Release and Update Process

This document explains how to update dependencies and prepare releases for the nrdot-collector-components repository.

## Overview

The repository tracks OpenTelemetry Collector dependencies and releases in sync with upstream. There are two main workflows:

1. **update-otel**: Weekly updates to track ongoing upstream development (pseudo-versions)
2. **prepare-release**: Official release preparation (release versions)

## Version Alignment

The project maintains version alignment with three upstream projects:

- **OpenTelemetry Collector Core** (stable modules): e.g., v1.47.0
- **OpenTelemetry Collector** (beta modules): e.g., v0.141.0
- **OpenTelemetry Collector Contrib**: e.g., v0.141.0

**Critical Rule**: Beta and Contrib minor versions MUST always match.

## Workflow 1: Update Development Dependencies (update-otel)

**Purpose**: Track upstream development between releases using pseudo-versions.

**When to run**:
- Automatically every Friday at 08:27 UTC
- Manually when you want to pull in latest upstream changes
- Between official releases to stay current with development

**What it does**:
- Pulls latest commits from upstream `main` branches
- Updates dependencies to pseudo-versions (e.g., `v0.141.1-0.20251219063944-48959d9e269d`)
- Creates a draft PR for review

**How to run**:

1. Navigate to: https://github.com/newrelic/nrdot-collector-components/actions/workflows/update-otel.yaml
2. Click "Run workflow"
3. No inputs required - it automatically:
   - Fetches latest commit from `open-telemetry/opentelemetry-collector`
   - Fetches latest commit from `open-telemetry/opentelemetry-collector-contrib`
   - Updates all dependencies to pseudo-versions from those commits

**Example output**:
```
Collector version: v0.141.1-0.20251219063944-48959d9e269d (minor: v0.141)
Found contrib pseudo-version: v0.141.1-0.20251219063944-48959d9e269d (minor: v0.141)
✅ Contrib and collector minor versions match: v0.141
```

**Validation**:
The workflow will **fail** if:
- Collector beta and contrib minor versions don't match
- Example: If collector is on v0.142.x but contrib hasn't released v0.142.x yet

## Workflow 2: Prepare Official Release (prepare-release)

**Purpose**: Prepare an official release with stable version tags.

**When to run**:
- After OpenTelemetry Collector releases a new version (e.g., v0.142.0)
- After OpenTelemetry Collector Contrib releases the matching version

**Prerequisites**:
1. ✅ Upstream collector must have released the target version (e.g., v0.142.0)
2. ✅ Upstream contrib must have released the matching version (e.g., v0.142.0)
3. ✅ Check that versions.yaml shows the previous version (e.g., v0.141.0)

**How to run**:

1. Navigate to: https://github.com/newrelic/nrdot-collector-components/actions/workflows/prepare-release.yml
2. Click "Run workflow"
3. Provide inputs:
   - **candidate-beta**: The version you want to release (e.g., `0.142.0`)
   - **current-beta**: The version currently in versions.yaml (e.g., `0.141.0`)

**Example: Releasing v0.142.0**

Starting state:
- `versions.yaml`: v0.141.0
- Upstream collector: v0.142.0 released ✅
- Upstream contrib: v0.142.0 released ✅

Workflow inputs:
```
candidate-beta: 0.142.0
current-beta: 0.141.0
```

**What the workflow does**:

1. **Updates module versions** (`make update-otel`):
   - Sets OTEL_VERSION="" → multimod uses versions.yaml
   - Updates stable modules to match collector's stable version (e.g., v1.47.0)
   - Updates beta modules to match collector's beta version (e.g., v0.142.0)
   - Updates contrib to v0.142.0
   - Validates that contrib and beta minor versions match

2. **Updates versions.yaml**:
   ```yaml
   version: v0.141.0  →  v0.142.0
   ```

3. **Updates builder-config.yaml files**:
   ```yaml
   # cmd/nrdotcol/builder-config.yaml
   version: 0.141.0-dev  →  0.142.0-dev

   exporters:
     - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.141.0  →  v0.142.0
   ```

4. **Regenerates collector code**:
   - Runs `make gennrdotcol` and `make genoteltestbedcol`
   - This regenerates go.mod files from the updated builder configs

5. **Updates changelog**:
   - Adds section for v0.142.0

6. **Creates PR** with all changes

## Example: Complete Release Cycle (v0.141.0 → v0.142.0)

### Phase 1: Development (Weeks 1-2)

**Week 1**: Automatic development updates
```
Friday: update-otel workflow runs automatically
Result: Dependencies updated to v0.141.1-0.20251206xxxxx (pseudo-version)
```

**Week 2**: More development
```
Friday: update-otel workflow runs automatically
Result: Dependencies updated to v0.141.1-0.20251213xxxxx (newer pseudo-version)
```

Current state:
- versions.yaml: v0.141.0 (unchanged)
- Dependencies: v0.141.1-0.20251213xxxxx (tracking development)

### Phase 2: Official Release (Week 3)

**Step 1**: Check upstream releases
```bash
# Verify collector released v0.142.0
curl -s https://api.github.com/repos/open-telemetry/opentelemetry-collector/releases/latest | jq -r .tag_name
# Output: v0.142.0 ✅

# Verify contrib released v0.142.0
curl -s https://api.github.com/repos/open-telemetry/opentelemetry-collector-contrib/releases/latest | jq -r .tag_name
# Output: v0.142.0 ✅
```

**Step 2**: Check current version
```bash
grep "version:" versions.yaml
# Output: version: v0.141.0 ✅
```

**Step 3**: Run prepare-release workflow
- Navigate to Actions → Automation - Prepare Release
- Click "Run workflow"
- Inputs:
  - `candidate-beta`: `0.142.0`
  - `current-beta`: `0.141.0`

**Step 4**: Review and merge PR
- Review the prepare-release PR (e.g., #104)
- Verify changes:
  - ✅ versions.yaml updated to v0.142.0
  - ✅ builder-config.yaml files updated
  - ✅ CHANGELOG updated
  - ✅ testbed/go.mod shows v0.142.0
- Merge the PR

**Step 5**: Create and push tags
- The push-release-tags workflow runs automatically after merge
- Tags all modules with v0.142.0

Final state:
- versions.yaml: v0.142.0 ✅
- All dependencies: v0.142.0 ✅
- Git tags: receiver/nopreceiver/v0.142.0, etc. ✅

### Phase 3: Back to Development (Week 4+)

Continue with weekly `update-otel` runs to track v0.142.x and v0.143.x development.

## Validation

After any update, validate versions are correct:

```bash
# Run the version validation script
bash internal/buildscripts/validate-versions.sh

# Expected output for release state:
✅ Collector beta and contrib minor versions match: v0.142
✅ Collector stable matches beta expectation: v1.47.0
✅ All versions aligned - release state
✅ Version validation passed
```

## Common Scenarios

### Scenario 1: Urgent fix needed from upstream main

**Problem**: Upstream fixed a critical bug in main, not yet released.

**Solution**: Use `update-otel`
```
Run: update-otel workflow (no inputs)
Result: Get pseudo-version with the fix
Example: v0.141.1-0.20251219063944-48959d9e269d
```

### Scenario 2: Collector released v0.142.0 but contrib hasn't

**Problem**: Can't release v0.142.0 yet.

**What happens**:
```bash
make update-otel OTEL_VERSION="" OTEL_STABLE_VERSION="" CONTRIB_VERSION=""

Output:
Collector version: v0.142.0 (minor: v0.142)
Trying stable contrib release for minor version: v0.142
No matching contrib version found
❌ ERROR: Contrib minor version (v0.141) doesn't match collector minor version (v0.142)
Cannot proceed - contrib hasn't released v0.142 yet!
```

**Solution**: Wait for contrib to release v0.142.0, then run prepare-release.

### Scenario 3: Need to regenerate local cmd/nrdotcol/go.mod

**Problem**: Local go.mod is stale after pulling main.

**Solution**:
```bash
make gennrdotcol
# Regenerates from builder-config.yaml (source of truth)
```

### Scenario 4: Validation fails after update

**Problem**:
```
❌ CRITICAL: Collector stable (v1.48.0) doesn't match beta v0.141.0 expectation (v1.47.0)!
```

**Diagnosis**: You have v1.48.0 (core) but v0.141.0 (beta). Core v1.48.0 requires beta v0.142.0.

**Solution**: Either:
- Downgrade core to v1.47.0 (to match beta v0.141.0), OR
- Upgrade beta to v0.142.0 (to match core v1.48.0)

## Files Modified by Each Workflow

### update-otel modifies:
- All `go.mod` files (via multimod)
- All `go.sum` files
- `cmd/nrdotcol/builder-config.yaml` (indirect, via updatehelper)
- `cmd/oteltestbedcol/builder-config.yaml` (indirect, via updatehelper)

### prepare-release modifies:
- `versions.yaml`
- `CHANGELOG.md`
- `CHANGELOG-API.md`
- `cmd/nrdotcol/builder-config.yaml`
- `cmd/oteltestbedcol/builder-config.yaml`
- All `go.mod` files (via make update-otel)
- All `go.sum` files
- `internal/buildscripts/modules`

## Important Notes

1. **cmd/nrdotcol/go.mod is gitignored**: This file is generated locally from builder-config.yaml and never committed. Always run `make gennrdotcol` after pulling changes.

2. **versions.yaml is the source of truth**: For releases, this file determines what version gets tagged and released.

3. **Draft PRs**: update-otel creates draft PRs for review before merging.

4. **Validation is critical**: Always run `validate-versions.sh` to ensure version alignment.

5. **Wait for contrib**: Never release before contrib has released the matching version.

## Troubleshooting

### Issue: "Could not resolve contrib pseudo-version from commit"

**Cause**: The specified commit doesn't exist or isn't accessible.

**Fix**: Verify commit hash is correct from https://github.com/open-telemetry/opentelemetry-collector-contrib/commits/main

### Issue: "Minor version mismatch"

**Cause**: Collector and contrib are on different minor versions.

**Fix**: This is expected during development. The workflow will fall back to finding a matching stable release.

### Issue: "go.mod validation failed"

**Cause**: Local go.mod is stale or wasn't regenerated.

**Fix**:
```bash
make gennrdotcol
make genoteltestbedcol
```

## Reference Links

- OpenTelemetry Collector: https://github.com/open-telemetry/opentelemetry-collector
- OpenTelemetry Collector Contrib: https://github.com/open-telemetry/opentelemetry-collector-contrib
- Version validation script: `internal/buildscripts/validate-versions.sh`
- Update workflow: `.github/workflows/update-otel.yaml`
- Release workflow: `.github/workflows/prepare-release.yml`
