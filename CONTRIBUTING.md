# Contributing

## Defects

Feel free to file a GitHub issue for any bug, big or small. If you find that the
fix is trivial and you'd like to fix it yourself, it's up to you as to whether
or not you want to file an issue before submitting the PR.

If the fix is not trivial or obvious, or you suspect it might be a lot of work,
definitely file an issue beforehand so that we can discuss possible solutions.

## Features/enhancements

Please file a GitHub issue before spending substantial amounts of time working
on a new feature to ensure it's a good fit for the project.

## Testing, running, deploying

In order to run the entire application, you'll need a valid Discord token
from [the Discord developer portal](https://discord.com/developers).
**Make sure "Message Content Intent" is enabled on the Bot settings page.**

Popple is configured via the process environment. I like to use `direnv`
to automate my development settings like this:

```console
$ cat > .envrc <<EOF
export POPPLE_DISCORD_TOKEN=YOUR_SECRET_BOT_TOKEN
export POPPLE_SQLITE_DB_PATH=popple.sqlite
EOF
$ direnv allow # or source .envrc if direnv is not installed
```

Finally, you'll need to set up the SQLite database:

```console
$ sqlite3 popple.sqlite '.read internal/database/sqlite/migrations/000001_create_tables.up.sql'
```

### Testing

The code base should be compatible with a simple `go test ./...`, however, the
BDD-style tests for the bot logic are built on top of [ginkgo](
https://github.com/onsi/ginkgo). The test output is much nicer when using
the ginkgo command line tool:

```console
$ ginkgo run internal/bot
```
