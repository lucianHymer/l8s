# MCP mim Server Protocol Error

**Date**: 2025-09-10

## Issue

The mim MCP server fails with Zod validation errors when connecting. The server appears to be sending error messages with an 'error' field, but Claude Code expects MCP messages to have 'id', 'method', and 'result' fields.

## Root Cause

Protocol mismatch where mim's error response format doesn't conform to the expected MCP message schema. This causes the connection to drop immediately after establishing.

## Impact

Cannot use the mim MCP server for knowledge management due to immediate connection drops.

## Related Files
- None documented