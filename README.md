# github-action-python-versioner

This action automatically updates the version number in `pyproject.toml` and
creates appropriate tags.

## Example usage

```
name: Updates version and tags
on:
  push:
    branches:
      - master
permissions:
  contents: write
jobs:
  update_version_and_tag:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - name: Install Python 3
      uses: actions/setup-python@v2
      with:
        python-version: 3.8
    - name: Update version
      uses: kurtmc/github-action-python-versioner@v1

```
