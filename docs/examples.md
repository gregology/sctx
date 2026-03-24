---
title: Examples
description: Real-world AGENTS.yaml files for dbt, API, React, Terraform, and monorepo projects
---

# Examples

Complete `AGENTS.yaml` files for real project types. Each example shows context entries and decisions working together.

If you need to understand individual fields first, see [Context entries](context.md) and [Decisions](decisions.md).

## dbt project

A dbt `models/` directory contains SQL models, YAML schema files, and markdown docs. An agent editing a SQL model needs completely different context than one editing a YAML schema.

```
models/
  AGENTS.yaml
  staging/
    stg_orders.sql
    stg_orders.yml
    stg_orders.md
    stg_customers.sql
    stg_customers.yml
```

```yaml
# models/AGENTS.yaml

context:
  - content: |
      SQL models use the following conventions:
      - CTEs over subqueries, always
      - snake_case for all identifiers
      - Prefix staging models with stg_, intermediate with int_, mart with fct_ or dim_
      - Use the incremental materialization with merge strategy for large tables
      - Reference other models with {{ ref('model_name') }}, never hardcode table names
    match: ["**/*.sql"]
    on: [edit, create]
    when: before

  - content: |
      Schema YAML files define tests and documentation for each model.
      Every model must have:
      - A description
      - A unique test on the primary key
      - not_null tests on required columns
      - accepted_values tests on status/type columns
      Column descriptions should be written for business users, not engineers.
    match: ["**/*.yml", "**/*.yaml"]
    on: [edit, create]
    when: before

  - content: |
      Documentation files follow this template:
      ## Overview (what this model represents)
      ## Source (where the data comes from)
      ## Key columns (the important fields and what they mean)
      ## Business rules (any transformations or filters applied)
      These are read by analysts and PMs. Write for that audience.
    match: ["**/*.md"]
    on: [edit, create]
    when: before

decisions:
  - decision: "Incremental models use merge strategy, not delete+insert"
    rationale: "Merge handles late-arriving data correctly without duplicates"
    alternatives:
      - option: "delete+insert"
        reason_rejected: "Creates a window where rows are missing during refresh"
      - option: "insert_overwrite"
        reason_rejected: "Only works with partition-based models"
    date: 2025-08-15
    match: ["**/*.sql"]
```

You could put all of this in an `AGENTS.md`, but every paragraph would start with "if you're editing a SQL file..." and the agent would parse all of it every time. With `AGENTS.yaml`, each entry is scoped to exactly the files it applies to.

## API service

A typical API directory has route handlers, middleware, tests, and OpenAPI specs. Each needs different guidance.

```
src/api/
  AGENTS.yaml
  handlers/
    users.py
    users_test.py
    orders.py
    orders_test.py
  middleware/
    auth.py
    rate_limit.py
  openapi/
    spec.yaml
```

```yaml
# src/api/AGENTS.yaml

context:
  - content: |
      Handlers follow this pattern:
      1. Validate input with a Pydantic model
      2. Call the service layer (never access the database directly)
      3. Return a typed response model
      4. Raise HTTPException for error cases, don't return error dicts
    match: ["handlers/**/*.py"]
    exclude: ["**/*_test.py"]
    on: [edit, create]
    when: before

  - content: |
      Tests use pytest with the test client fixture. Each handler test file
      should test: happy path, validation errors, auth failures, and not-found
      cases. Use factory functions for test data, not raw dicts.
    match: ["**/*_test.py"]
    on: [edit, create]
    when: before

  - content: |
      Middleware must be stateless. No database calls, no file I/O.
      Configuration comes from environment variables loaded at startup.
      Always call `await call_next(request)` even in error paths.
    match: ["middleware/**/*.py"]
    on: [edit, create]
    when: before

  - content: "The OpenAPI spec is the source of truth for the API contract. Update it before changing handler signatures, not after."
    match: ["openapi/**"]
    on: edit
    when: after

decisions:
  - decision: "Pydantic for request/response validation, not marshmallow or attrs"
    rationale: "Native FastAPI integration, better type inference, JSON Schema generation for free"
    alternatives:
      - option: "marshmallow"
        reason_rejected: "Requires separate schema definitions, no FastAPI integration"
      - option: "attrs + cattrs"
        reason_rejected: "No JSON Schema output, manual validation code"
    date: 2025-09-01
```

## React component library

Component directories mix implementation, tests, stories, and styles. The conventions for each are different.

```
src/components/
  AGENTS.yaml
  Button/
    Button.tsx
    Button.test.tsx
    Button.stories.tsx
    Button.module.css
```

```yaml
# src/components/AGENTS.yaml

context:
  - content: |
      Components are functional, using hooks. Props interfaces are defined
      in the same file, exported, and named ComponentNameProps.
      Use forwardRef for any component that wraps a native HTML element.
    match: ["**/*.tsx"]
    exclude: ["**/*.test.tsx", "**/*.stories.tsx"]
    on: [edit, create]
    when: before

  - content: |
      Tests use React Testing Library. Test behavior, not implementation.
      Query by role or label, never by class name or test ID unless there's
      no accessible alternative. Every component needs: render test,
      interaction test, accessibility check with axe.
    match: ["**/*.test.tsx"]
    on: [edit, create]
    when: before

  - content: |
      Stories follow the CSF3 format. Every component needs: a Default
      story, one story per significant prop variation, and an interactive
      story with args. Use the autodocs tag.
    match: ["**/*.stories.tsx"]
    on: [edit, create]
    when: before

  - content: "CSS modules only. No global styles, no inline styles, no styled-components. Use design tokens from tokens.css for colors, spacing, and typography."
    match: ["**/*.css"]
    on: [edit, create]
    when: before
```

## Terraform infrastructure

Infrastructure-as-code directories mix resource definitions, variable files, and documentation.

```
infra/
  AGENTS.yaml
  main.tf
  variables.tf
  terraform.tfvars
  README.md
  modules/
    networking/
      main.tf
      outputs.tf
```

```yaml
# infra/AGENTS.yaml

context:
  - content: |
      All resources must have: a Name tag, an Environment tag, and a
      ManagedBy tag set to "terraform". Use locals for any value
      referenced more than once. Never hardcode AWS account IDs.
    match: ["**/*.tf"]
    exclude: ["**/*.tfvars"]
    on: [edit, create]
    when: before

  - content: "Variable files contain only variable declarations. No resources, no data sources, no locals. Every variable needs a description and a type constraint."
    match: ["**/variables.tf"]
    on: [edit, create]
    when: after

  - content: "tfvars files are environment-specific configuration. Never commit secrets here. Use SSM Parameter Store references for sensitive values."
    match: ["**/*.tfvars"]
    on: [edit, create]
    when: before

decisions:
  - decision: "Modules over inline resources for anything used more than once"
    rationale: "Consistent patterns, testable units, version-pinned interfaces"
    alternatives:
      - option: "Copy-paste resources across environments"
        reason_rejected: "Drift between environments, maintenance burden"
      - option: "Terragrunt wrappers"
        reason_rejected: "Extra tool, extra abstraction layer, team doesn't know it"
    date: 2025-07-10
    match: ["**/*.tf"]
```

## Monorepo with shared conventions

Some context applies everywhere. Some only applies to specific packages. Structured Context handles this through directory inheritance.

```
monorepo/
  AGENTS.yaml              <- shared conventions
  packages/
    auth/
      AGENTS.yaml          <- auth-specific
      src/
    payments/
      AGENTS.yaml          <- payments-specific
      src/
    shared/
      src/
```

```yaml
# monorepo/AGENTS.yaml (root)

context:
  - content: "All packages use ESM imports. No require() calls. No default exports."
    match: ["**/*.ts", "**/*.tsx"]
    on: [edit, create]
    when: before

  - content: "Error classes extend BaseError from @monorepo/shared. Never throw plain strings or generic Error."
    match: ["**/*.ts"]
    on: [edit, create]
    when: before

decisions:
  - decision: "pnpm over npm or yarn"
    rationale: "Strict dependency resolution, disk efficiency, workspace support"
    alternatives:
      - option: "npm"
        reason_rejected: "Flat node_modules causes phantom dependency issues"
      - option: "yarn berry"
        reason_rejected: "PnP mode breaks too many tools"
    date: 2025-06-01
```

```yaml
# monorepo/packages/payments/AGENTS.yaml

context:
  - content: "All monetary values are integers in cents. Never use floats for money. The Money type from @monorepo/shared handles formatting."
    match: ["**/*.ts"]
    on: [edit, create]
    when: after

decisions:
  - decision: "Stripe over Braintree for payment processing"
    rationale: "Better API design, better docs, team has experience"
    alternatives:
      - option: "Braintree"
        reason_rejected: "Worse developer experience, PayPal ownership concerns"
      - option: "Adyen"
        reason_rejected: "Overkill for our volume, enterprise-focused onboarding"
    revisit_when: "Stripe pricing becomes prohibitive or we need multi-PSP"
    date: 2025-08-20
    match: ["**/*.ts"]
```

When an agent edits `packages/payments/src/checkout.ts`, it gets the shared monorepo conventions *and* the payments-specific context. The payments context appears last, giving it stronger influence.

## Structured Context

- This is a Go project. Follow idiomatic Go conventions: short variable names, table-driven tests, no assertion libraries.
- Both AGENTS.yaml and AGENTS.yml are recognized. If both exist in the same directory, they are loaded and merged.
