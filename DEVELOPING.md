# Developing Popple

## Persistence layer

Popple uses a SQLite database by default for persistence. The schema for
the database is stored in `/internal/sqlite3/schema.sql`. The queries
that Popple uses are found in `/internal/sqlite3/queries.sql`. `sqlc`
uses these files to generate a lot of boilerplate SQL code.

If there are any changes to either of those two files, run `sqlc
generate` from the root of the project to re-generate the boilerplate
SQL code found under `/internal/sqlite3/data`.

Note that `sqlc` does *not* generate `/internal/sqlite3/schema.go`,
since that is a manually-created file. Its only purpose is to re-use
`/internal/sqlite3/schema.sql` so that we can ensure the required tables
exist during startup.

## The karma application layer

The entrypoint for anything related to counting karma or operating on
karma levels is in the top-level `popple.go` file. New functionality is
added as a method that hangs off the Popple struct.

If the implementation for the new feature is non-trivial, try to
decompose it and implement the heavy-lifting in
`/internal/popple/popple.go`.

Anything related to bot/chat-level interactions goes in
`/cmd/discord/popple`.

This divide makes it easy to plug Popple's karma-counting functionality
into any chat-specific implementation. Another cool side-effect of this
is that the karma application layer doesn't impose *any* decisions over
the chat-specific implementation. So, for example, it wouldn't get in
the way of a localization effort to provide a chat-implementation in
someone's native language.

This also allows the chat-specific implementation to choose how or when
the bot responds to various subcommands.

## What's the deal with the types in `/internal/popple/popple.go`?

It seems like a lot of noise at first, but this is my attempt to allow
callers to decide if they want to block on some internal logic.

This is primarily because the adapter.PersistenceLayer is pluggable.
The SQLite implementation is the only implementation used in the
repo right now, and it's safe to say that those calls do not take a long
time.

That might not always be the case though, and someone else could
take the Popple package and implement their own adapter.PersistenceLayer
with a number of complicated network calls and retries, etc.

The types ending with `Result` are usually just my way of coping with
the fact that Go doesn't have tuples that we can send down a channel,
since some of the logic returns a value that someone is interested in
as well as an error or something to that effect.

The types ending with `F` are loosely synonymous with the concept of a
`Future`. They do not fulfill *all* of the usual criteria that most
people associate with a future, so perhaps this is just bad naming on
my part.

## The chat-specific implementation

Discord is the only chat implementation at this time. All of the guts
are found in `/cmd/discord/popple`.

If your change has anything to do with how or when the bot interacts
with the chat server, the `/cmd/discord/popple` package is likely where
the change will go.

## The `adapter` package

The `adapter` package is what allows the karma-counting core to "plug
in" anywhere. It contains types that are meant to cross between the
application layer and the outside world.

These types *could have* just lived in the top-level Popple package, but
that would create an import cycle for packages under `/internal/` that
also rely on these types.
