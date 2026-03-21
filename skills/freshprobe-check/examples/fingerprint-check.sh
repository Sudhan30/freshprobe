#!/bin/bash
# Content change detection with repeat probes
freshprobe check https://api.example.com/quotes \
  --repeat 3 --interval 2s \
  --stateless --output json
