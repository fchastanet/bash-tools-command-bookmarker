#!/usr/bin/env bash

set -e -o pipefail -o errexit
echo >&2 "Runs deadcode ..."

tempFile=$(mktemp)
#trap 'rm -f "${tempFile}"' EXIT
echo "${tempFile}"
# run deadcode
deadcode \
  -filter "github.com/fchastanet/shell-command-bookmarker" \
  ./app/main.go >"${tempFile}"

# remove the ignored lines
sed -i -E \
  -e '/^pkg\/tui\/table\/table.go:.*: unreachable func: WithNavigation$/d' \
  -e '/^pkg\/tui\/table\/table.go:.*: unreachable func: WithAction$/d' \
  -e '/^pkg\/tui\/table\/table.go:.*: unreachable func: WithSelectable$/d' \
  -e '/^pkg\/tui\/table\/truncation.go:.*: unreachable func: TruncateLeft$/d' \
  -e '/^pkg\/components\/tabs\/tabs.go:46:6: unreachable func: defaultKeyMap$/d' \
  -e '/^pkg\/components\/tabs\/tabs.go:61:6: unreachable func: NewTabs$/d' \
  -e '/^pkg\/components\/tabs\/tabs.go:91:16: unreachable func: Tabs.GetKeyBindings$/d' \
  -e '/^pkg\/components\/tabs\/tabs.go:97:16: unreachable func: Tabs.Init$/d' \
  -e '/^pkg\/components\/tabs\/tabs.go:108:16: unreachable func: Tabs.Update$/d' \
  -e '/^pkg\/components\/tabs\/tabs.go:137:16: unreachable func: Tabs.updateActiveTab$/d' \
  -e '/^pkg\/components\/tabs\/tabs.go:168:17: unreachable func: Tab.View$/d' \
  -e '/^pkg\/components\/tabs\/tabs.go:182:16: unreachable func: Tabs.View$/d' \
  "${tempFile}"

if [[ -s "${tempFile}" ]]; then
  echo >&2 "Dead code found:"
  cat "${tempFile}"
  exit 1
else
  echo >&2 "No dead code found."
fi
