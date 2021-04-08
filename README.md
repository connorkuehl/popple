# Popple

A karma bot for Discord.

## Building

Clone or otherwise download the Popple source code and run:

```console
$ go build
```

The `go` toolchain should download the dependencies and build Popple.

## Running

Popple requires a valid Discord bot token in order to interact with the
Discord API.

```console
$ echo -n "my super secret token" > popple_token
$ chmod 0600 popple_token
```

Then, once that one-time setup is complete:

```console
$ ./popple -token /path/to/popple_token
```

Popple has some additional configuration options. Run `./popple -help`
for more info. These are optional, but could be helpful for adapting Popple
to better suit your needs.

## Usage

| Command | Values | Description |
| - | - | - |
| @Popple announce | on, off, yes, no | Whether or not Popple will print a subject's karma level after it has been modified |
| @Popple karma | Something with karma | Prints the subjects' karma level. Multiple subjects' karma levels may be checked |
| Subject++ | N/A | Increases Subject's karma |
| Subject-- | N/A | Decreases Subject's karma |
| (Subject with space or - +) | N/A | Parentheses may be used for complicated subjects with whitespace or special symbols |

Once Popple has joined a Discord server, it will watch for karma events in
the chat. Increase or decrease karma by suffixing the subject with a `++`
or a `--`, respectively.

For example,

```txt
Person) Thanks for being so neat, Popple++!
Popple) Popple has 1 karma.
```

You can tack on as many `++` or `--` onto the end of a subject as you want and
Popple will count them out and calculate the "net karma" from that operation.

```txt
Person) Popple++--++--++++
Popple) Popple has 2 karma.
```

Popple won't react to karma events that net to zero:

```txt
Person) Popple++--

*crickets*
```

A message can have any number of karma events for any number of subjects:

```txt
Person) PoeThePotatoPirate++++ Popple--++++ HelloWorld----
Popple) PoeThePotatoPirate has 2 karma. Popple has 3 karma. HelloWorld has -2 karma.
```

Parentheses may be used for more complicated karma subjects, including those
with whitespace or even `+` or `-` in their names.

```txt
Person) (Poe the Potato Pirate)++ (meme-bot)++
Popple) Poe the Potato Pirate has 2 karma. meme-bot has 2 karma.
```

Karma levels can be checked without requiring any karma events:

```txt
Person) @Popple karma Popple
Popple) Popple has 3 karma.
Person) @Popple karma DoesNotExist
Popple) DoesNotExist has 0 karma.
```

The above could be combined into one command like so:

```txt
Person) @Popple karma Popple DoesNotExist
Popple) Popple has 3 karma. DoesNotExist has 0 karma.
```

By default, Popple will announce a subject's karma level after it is modified.
This behavior can be disabled. Karma levels may still be checked with the
`karma` command.

```txt
Person) @Popple announce off
Person) Person++

*crickets*
```

Or

```txt
Person) @Popple announce no
Person) Person++

*crickets)
```

It can be turned back on with `@Popple announce yes` or
`@Popple announce on`.

## Deploying

Example systemd unit:

```
/etc/systemd/system/popple.service
```

```systemd
[Unit]
Description=Popple, a karma bot for Discord
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=popple
Group=popple
ExecStart=/srv/popple/go/bin/popple -token /srv/popple/.popple_token -db /srv/popple/db.sqlite
ProtectSystem=yes
ProtectHome=yes
NoNewPrivileges=yes
PrivateTmp=yes
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```console
$ systemctl enable popple
$ systemctl start popple
```
