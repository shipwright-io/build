#! /bin/sh
# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

# this script assumes the GITHUB_TOKEN and PREVIOUS_TAG environment variables have been set;
# it produces a 'Changes.md' file as its final output;
# the file 'last-300-prs-with-release-note.txt' that is produces is intermediate data; it is not
# pruned for now to assist development of the release notes process (we are still curating all this)

if [ -z ${GITHUB_TOKEN+x} ]; then
  echo "Error: GITHUB_TOKEN is not set"
fi
if [ -z ${PREVIOUS_TAG+x} ]; then
  echo "Error: PREVIOUS_TAG is not set"
fi

sudo apt-get -y update
sudo apt-get -y install jq wget
curl -L https://github.com/github/hub/releases/download/v2.14.2/hub-linux-amd64-2.14.2.tgz | tar xzf -
PWD="$(pwd)"
export PATH=$PWD/hub-linux-amd64-2.14.2/bin:$PATH
git fetch --all --tags --prune --force
echo -e "# Insert Title\n" > Changes.md
echo -e "## Features\n\n## Fixes\n\n## Backwards incompatible changes\n\n## Docs\n\n## Misc\n\n## Thanks" >> Changes.md    - name: Draft Release
# this effectively gets the commit associated with github.event.inputs.tags
COMMON_ANCESTOR=$(git merge-base $PREVIOUS_TAG HEAD)
# in theory the new tag has not been created yet; do we want another input that specifies the existing
# commit desired for drafting the release? for now, we are using HEAD in the above git merge-base call
# and PR cross referencing below

# use of 'hub', which is an extension of the 'git' CLI, allows for pulling of PRs, though we can't search based on commits
# associated with those PRs, so we grab a super big number, 300, which should guarantee grabbing all the PRs back to
# github.events.inputs.tags; we use grep -v to filter out release-note-none and release-note-action-required.
# NOTE: investigated using the new 'gh' cli command, but its 'gh pr list' does not currently support the -f option so
# staying with 'hub' for now.
hub pr list --state merged -L 300 -f "%sm;%au;%i;%t;%L%n" | grep -E ", release-note|release-note," | grep -v release-note-none | grep -v release-note-action-required > last-300-prs-with-release-note.txt
# now we cylce through last-300-prs-with-release-note.txt, filtering out stuff that is too old or other anomalies,
# and update Changes.md with the release note.
while IFS= read -r pr; do
   SHA=$(echo $pr | cut -d';' -f1)
   # skip the common ancestor, which in essences is the commit associated with the tag github.event.inputs.tags
   if [ "$SHA" == "$COMMON_ANCESTOR" ]; then
      continue
   fi

   # styllistic clarification, purposefully avoiding slicker / cleverer / more compact scripting conventions

   # this makes sure that this PR has merged
   git merge-base --is-ancestor $SHA HEAD
   rc=$?
   if [ ${rc} -eq 1 ]; then
      continue
   fi
   # otherwise, if the current commit from the last 300 PRs is not an ancestor of github.event.inputs.tags, we have gone too far, so skip
   git merge-base --is-ancestor $COMMON_ANCESTOR $SHA
   rc=$?
   if [ ${rc} -eq 1 ]; then
      continue
   fi
   # if we are at this point, we have a PR with a release note to add
   AUTHOR=$(echo $pr | cut -d';' -f2)
   PR_NUM=$(echo $pr | cut -d';' -f3)
   PR_RELEASE_NOTE=$(wget -q -O- https://api.github.com/repos/shipwright-io/build/issues/${PR_NUM:1} | jq .body -r | grep -oPz '(?s)(?<=```release-note..)(.+?)(?=```)' | grep -avP '\W*(Your release note here|action required: your release note here|NONE)\W*')
   echo -e "$PR_NUM by $AUTHOR: $PR_RELEASE_NOTE" >> Changes.md
done < last-300-prs-with-release-note.txt
