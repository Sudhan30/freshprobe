#!/bin/bash
# Basic endpoint health check with text output
freshprobe check https://api.example.com/data --stateless --output text
