#!/bin/bash
# Check endpoint against a freshness policy
freshprobe check https://api.trading.com/quotes \
  --policy-dir ${CLAUDE_PLUGIN_ROOT}/policies \
  --policy financial-data \
  --stateless --output text
