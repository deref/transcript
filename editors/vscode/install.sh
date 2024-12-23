#!/bin/bash

npx @vscode/vsce package
code --install-extension "cmdt-$(jq -r .version package.json).vsix"
