#!/bin/bash
# Batch check multiple endpoints concurrently
freshprobe batch \
  https://api.example.com/data \
  https://cdn.example.com/assets \
  https://auth.example.com/health \
  --stateless --concurrency 5
