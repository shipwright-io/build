#! /bin/bash
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
sudo apt-get -y install wget curl git
curl -L https://github.com/github/hub/releases/download/v2.14.2/hub-linux-amd64-2.14.2.tgz | tar xzf -
PWD="$(pwd)"
export PATH=$PWD/hub-linux-amd64-2.14.2/bin:$PATH
git fetch --all --tags --prune --force
echo "# Draft Release changes since ${PREVIOUS_TAG}" > Changes.md
echo > Features.md
echo "## Features" >> Features.md
echo > Fixes.md
echo "## Fixes" >> Fixes.md
echo > API.md
echo "## API Changes" >> API.md
echo > Docs.md
echo "## Docs" >> Docs.md
echo > Misc.md
echo "## Misc" >> Misc.md

# this effectively gets the commit associated with github.event.inputs.tags
COMMON_ANCESTOR=$(git merge-base $PREVIOUS_TAG HEAD)
echo "COMMON_ANCESTOR is ${COMMON_ANCESTOR}"
# in theory the new tag has not been created yet; do we want another input that specifies the existing
# commit desired for drafting the release? for now, we are using HEAD in the above git merge-base call
# and PR cross referencing below

# use of 'hub', which is an extension of the 'git' CLI, allows for pulling of PRs, though we can't search based on commits
# associated with those PRs, so we grab a super big number, 300, which should guarantee grabbing all the PRs back to
# github.events.inputs.tags; we use grep -v to filter out release-note-none and release-note-action-required.
# NOTE: investigated using the new 'gh' cli command, but its 'gh pr list' does not currently support the -f option so
# staying with 'hub' for now.
hub pr list --state merged -L 300 -f "%sm;%au;%i;%t;%L%n" | grep -E ", release-note|release-note," | grep -v release-note-none | grep -v release-note-action-required > last-300-prs-with-release-note.txt
# this is for debug while we sort out env differences between Gabe's fedora and GitHub Actions' ubuntu
echo "start dump last-300-prs-with-release-note.txt for potential debug"
cat last-300-prs-with-release-note.txt
echo "end dump last-300-prs-with-release-note.txt for potential debug"
# now we cylce through last-300-prs-with-release-note.txt, filtering out stuff that is too old or other anomalies,
# and update Changes.md with the release note.
while IFS= read -r pr; do
   SHA=$(echo $pr | cut -d';' -f1)

   # skip the common ancestor, which in essences is the commit associated with the tag github.event.inputs.tags
   if [ "$SHA" == "$COMMON_ANCESTOR" ]; then
      continue
   fi

   # stylistic clarification, purposefully avoiding slicker / cleverer / more compact scripting conventions

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
   echo "Examining from ${AUTHOR} PR ${PR_NUM}"
   PR_BODY=$(wget -q -O- https://api.github.com/repos/shipwright-io/build/issues/${PR_NUM:1})
   echo $PR_BODY | grep -oPz '(?s)(?<=```release-note..)(.+?)(?=```)' > /dev/null 2>&1
   rc=$?
   if [ ${rc} -eq 1 ]; then
      echo "First validation:  the release-note field for PR ${PR_NUM} was not properly formatted.  Until it is fixed, it will be skipped for release note inclusion."
      echo "See the PR template at https://raw.githubusercontent.com/shipwright-io/build/master/.github/pull_request_template.md for verification steps"
      continue
   fi
   PR_BODY_FILTER_ONE=$(echo $PR_BODY | grep -oPz '(?s)(?<=```release-note..)(.+?)(?=```)')
   echo $PR_BODY_FILTER_ONE | grep -avP '\W*(Your release note here|action required: your release note here|NONE)\W*' > /dev/null 2>&1
   rc=$?
   if [ ${rc} -eq 1 ]; then
      echo "Second validation:  the release-note field for PR ${PR_NUM} was not properly formatted.  Until it is fixed, it will be skipped for release note inclusion."
      echo "See the PR template at https://raw.githubusercontent.com/shipwright-io/build/master/.github/pull_request_template.md for verification steps"
      continue
   fi
   PR_RELEASE_NOTE=$(echo $PR_BODY_FILTER_ONE | grep -avP '\W*(Your release note here|action required: your release note here|NONE)\W*')
   PR_RELEASE_NOTE_NO_NEWLINES=$(echo $PR_RELEASE_NOTE | sed 's/\\n//g' | sed 's/\\r//g')
   MISC=yes
   echo $pr | grep 'kind/bug'
   rc=$?
   if [ ${rc} -eq 0 ]; then
      echo >> Fixes.md
      echo "$PR_NUM by $AUTHOR: $PR_RELEASE_NOTE_NO_NEWLINES" >> Fixes.md
      MISC=no
   fi
   echo $pr | grep 'kind/api-change'
   rc=$?
   if [ ${rc} -eq 0 ]; then
      echo >> API.md
      echo "$PR_NUM by $AUTHOR: $PR_RELEASE_NOTE_NO_NEWLINES" >> API.md
      MISC=no
   fi
   echo $pr | grep 'kind/feature'
   rc=$?
   if [ ${rc} -eq 0 ]; then
      echo >> Features.md
      echo "$PR_NUM by $AUTHOR: $PR_RELEASE_NOTE_NO_NEWLINES" >> Features.md
      MISC=no
   fi
   echo $pr | grep 'kind/documentation'
   rc=$?
   if [ ${rc} -eq 0 ]; then
      echo >> Docs.md
      echo "$PR_NUM by $AUTHOR: $PR_RELEASE_NOTE_NO_NEWLINES" >> Docs.md
      MISC=no
   fi
   if [ "$MISC" == "yes" ]; then
      echo >> Misc.md
      echo "$PR_NUM by $AUTHOR: $PR_RELEASE_NOTE_NO_NEWLINES" >> Misc.md
   fi
   # update the PR template if our greps etc. for pulling the release note changes
   #PR_RELEASE_NOTE=$(wget -q -O- https://api.github.com/repos/shipwright-io/build/issues/${PR_NUM:1} | grep -oPz '(?s)(?<=```release-note..)(.+?)(?=```)' | grep -avP '\W*(Your release note here|action required: your release note here|NONE)\W*')
   echo "Added from ${AUTHOR} PR ${PR_NUM:1} to the release note draft"
done < last-300-prs-with-release-note.txt

cat Features.md >> Changes.md
cat Fixes.md >> Changes.md
cat API.md >> Changes.md
cat Docs.md >> Changes.md
cat Misc.md >> Changes.md
