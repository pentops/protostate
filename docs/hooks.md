State Machine Hooks
===================

[!Hook Types](./hook_types.svg)

## Status

```go
.SetStatus(...)
```

No function, just sets the status enum of the state machine.

This is the only way to set the status.

## Mutation

Callback function with access to the State Data and the incomming event.

Cannot publish events, and has no database access. Must not have any side
effects.

Combined with the PSM logic, this is considered a 'pure' function as the state
is cloned before the function is called.


## Logic

Callback function with access to the State Data and the incomming event.

- Can publish side-effects and chain events.
- No access to the database transaction.
- Runs only on RunEvent
- Must not modify state data

## Data

Callback function with access to the State Data and the incomming event.

Used to store pre-calculated objects for other queries.

- Has direct *Write Access* to the database transation
- TX must be consistent and forces high TX isolation
- Runs on Run or Follow
- Must not modify state data

## Link

Callback function with access to State Data and the incomming event.

Used to cause transitions in another state machine atomically.

- Must not modify state data
- Runs only on RunEvent
- TODO: Read Only access to the database transaction for upsert lookups


## General Logic

## General State Data

## General Event Data

## Event Publisher

## Upsert Publisher
