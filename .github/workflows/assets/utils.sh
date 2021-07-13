#!/bin/sh

# Setup runner environment
setup() {
  # Configure git to access private Go modules using the
  # provided personal access token.
  # https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token
  git config --global \
  url."https://${GITHUB_USER}:${ACCESS_TOKEN}@github.com".insteadOf "https://github.com"
}

case $1 in
  "setup")
  setup
  ;;

  *)
  echo "Invalid target: $1"
  ;;
esac
