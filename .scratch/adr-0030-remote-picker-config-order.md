# ADR 0030: Remote picker preserves config order

## Status

Accepted

## Context

Users may want frequently used remotes near the top of the picker. Config file order is the most direct way to express that.

## Decision

The remote picker displays remotes in the order they appear in the config file.

## Consequences

- Users can hand-order remotes by editing config.
- The config parser or config loading layer must preserve remote section order.
- Tests should cover remote ordering so future parser changes do not silently sort remotes.

