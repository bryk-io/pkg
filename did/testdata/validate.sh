#!/bin/sh

node jsonld-validator/main.js ${1} | jq .
