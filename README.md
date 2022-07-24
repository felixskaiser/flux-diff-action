# flux-diff-action

[![GitHub Super-Linter](https://github.com/felixskaiser/flux-diff-action/workflows/GitHub%20Super-Linter/badge.svg)](https://github.com/marketplace/actions/super-linter) [![Example](https://github.com/felixskaiser/flux-diff-action/workflows/Example/badge.svg)](https://github.com/felixskaiser/flux-diff-action/actions/workflows/example.yaml)

Find [Flux2](https://fluxcd.io/docs/) [Kustomizations](https://fluxcd.io/docs/components/kustomize/kustomization/) in two directories and get the diff between the built manifests.

## Usage Examples

### `on: push` with Job Summary

```yaml
name: Flux Kustomize Diff

on: push

jobs:
  diff-flux-kustomizations:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout default branch
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
          output_format: markdown

      # Use single quotes ('') to prevent command substitution because of backticks (```)
      - name: Add to job summary
        run: echo '${{ steps.diff_flux_kustomizations.outputs.diff }}' >> "$GITHUB_STEP_SUMMARY"
```

### `on: pull_request` with PR comment

```yaml
name: Flux Kustomize Diff

on: pull_request

jobs:
  diff-flux-kustomizations:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout base branch
        uses: actions/checkout@v3
        with:
          ref: ${{ github.base_ref }}
          path: base_branch

      - name: Checkout current branch
        uses: actions/checkout@v3
        with:
          path: current_branch

      - name: Diff Flux Kustomizations
        uses: felixskaiser/flux-diff-action@main
        with:
          src_path: base_branch
          dst_path: current_branch
          output_type: pr_comment
```

## Inputs

- `src_path` - (required) Path to the source directory.

- `dst_path` - (required) Path to the destination directory.

- `output_type` - (optional) How to output the diff.

  Default is Action output only. Set to `pr_comment` to comment diff on PR.

  Input `output_type: pr_comment` implies input `output_format: markdown`.

- `output_format` - (optional) The format with which to output the diff.

  Default is colored stdout. Set to `markdown` to get output without color, prefix \`\`\`diff and suffix \`\`\`.

  Input `output_type: pr_comment` implies input `output_format: markdown`.

## Outputs

- `diff` - The diff returned by this action.
