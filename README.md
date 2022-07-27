# Diff Flux Kustomizations

[![GitHub Super-Linter](https://github.com/felixskaiser/flux-diff-action/workflows/GitHub%20Super-Linter/badge.svg)](https://github.com/marketplace/actions/super-linter)
[![Example](https://github.com/felixskaiser/flux-diff-action/workflows/Example/badge.svg)](https://github.com/felixskaiser/flux-diff-action/actions/workflows/example.yaml)

A [GitHub Action](https://docs.github.com/en/actions) to find [Flux2](https://fluxcd.io/docs/) [Kustomizations](https://fluxcd.io/docs/components/kustomize/kustomization/) in two directories and get the diff between the built manifests.

## Usage Examples

Check out the live [example workflow](https://github.com/felixskaiser/flux-diff-action/actions/workflows/example.yaml).

### `on: push` with Job Summary

```yaml
name: Flux Kustomize Diff

on:
  push:
    branches-ignore:
      - 'main'

jobs:
  flux-diff:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout main branch
        uses: actions/checkout@v3
        with:
          ref: main
          path: main_branch

      - name: Checkout current branch
        uses: actions/checkout@v3
        with:
          path: current_branch

      - id: diff_flux_kustomizations
        uses: felixskaiser/flux-diff-action@main
        with:
          src_path: main_branch
          dst_path: current_branch
          output_type: job_summary
```

### `on: pull_request` with PR comment

```yaml
name: Flux Kustomize Diff

on:
  pull_request:
    branches: [main]

jobs:
  flux-diff:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout main branch
        uses: actions/checkout@v3
        with:
          ref: main
          path: main_branch

      # defaults to PR branch
      - name: Checkout PR branch
        uses: actions/checkout@v3
        with:
          path: pr_branch

      - name: Diff Flux Kustomizations
        uses: felixskaiser/flux-diff-action@main
        with:
          src_path: main_branch
          dst_path: pr_branch
          output_type: pr_comment
```

### Compare directories

```yaml
name: Flux Kustomize Diff

on: push

jobs:
  flux-diff:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout base branch
        uses: actions/checkout@v3

      - id: diff_flux_kustomizations
        uses: felixskaiser/flux-diff-action@main
        with:
          src_path: clusters/production
          dst_path: clusters/staging

      - name: Do something with output
        run: echo "${{ steps.diff_flux_kustomizations.outputs.diff }}"
```

## Inputs

- `src_path` - (required) Path to the source directory.

- `dst_path` - (required) Path to the destination directory.

- `output_type` - (optional) How to output the diff.  Default is Action output only.

    Set to `pr_comment` to comment diff on PR in markdown format. The PR to comment is derived from workflow context.

    Set to `job_summary` to add diff to the workflow job summary in markdown format.

## Outputs

- `diff` - The diff returned by this action.
