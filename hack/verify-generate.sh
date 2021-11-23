#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

# Verifies if a developer has forgot to run the
# `make generate` so that all the changes in the
# clientset and CRDs should also be pushed

if [[ -n "$(git status --porcelain -- pkg/client pkg/apis deploy/crds)" ]]; then
  echo "The pkg/client, pkg/apis package or CRDs contains changes:"
  git --no-pager diff --name-only -- pkg/client pkg/apis deploy/crds
  echo
  echo "Run make generate to those commit changes!"
  exit 1
fi
