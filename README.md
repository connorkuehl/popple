# Popple

A karma bot for Discord.

## Usage

| Command | Values | Description |
| - | - | - |
| @Popple announce | on, off, yes, no | Whether or not Popple will print a subject's karma level after it has been modified |
| @Popple karma | Something with karma | Prints the subjects' karma level. Multiple subjects' karma levels may be checked |
| @Popple bot | Integer > 0 | Prints the `n` subjects with the least karma. The default value is `10` if a value is not supplied |
| @Popple top | Integer > 0 | Prints the top `n` subjects with the most karma. The default value is `10` if a value is not supplied |
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

Popple will ignore "net-zero" operations on karma.

```txt
Person) Popple++ Popple--

*crickets*
```

A message can have any number of karma events for any number of subjects:

```txt
Person) PoeThePotatoPirate++ Popple-- HelloWorld--
Popple) PoeThePotatoPirate has 2 karma. Popple has 3 karma. HelloWorld has -2 karma.
```

Parentheses may be used for more complicated karma subjects, including those
with whitespace, ticks, or other parentheses in their name.

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

*crickets*
```

It can be turned back on with `@Popple announce yes` or
`@Popple announce on`.

## Building

Clone or otherwise download the Popple source code and run:

```console
$ go build ./...
```

The `go` toolchain should download the dependencies and build Popple.

## Running

### Pre-requisites:

1. A valid Discord bot token so that Popple can connect to and interact
with Discord.
1. A MySQL database for persisting karma counts and per-server configuration.
1. A RabbitMQ instance so that Popple components can exchange messages.

There are currently two application-layer components:

1. popplebot (./cmd/popplebot) is Popple's point-of-presence on Discord. It
reads and responds to messages in the Discord servers that it is in. If a
message requires any application logic, it submits a request to RabbitMQ
so that an instance of popplesvc can take care of it.
1. popplesvc (./cmd/popplesvc) is where the main application-layer logic takes
place. It processes requests from popplebot and persists any necessary state
to the database.

### Quickstart:

Note, the following configuration is sufficient for local development, but
will require changes in order to be secure for a production deployment.

The only place I have left a placeholder value is for the Discord bot token,
but otherwise I am using the default RabbitMQ credentials of `guest:guest` and
the default root credentials for the MySQL docker container in this example.

1. [one-time-setup] Create an env file to hold all of the necessary Popple configuration:

```console
cat > .poppleenv <<EOF
POPPLEBOT_AMQP_HOST=poppleevents
POPPLEBOT_AMQP_PORT=5672
POPPLEBOT_AMQP_USER=guest
POPPLEBOT_AMQP_PASS=guest
POPPLEBOT_DISCORD_TOKEN=<YOUR_SECRET_DISCORD_TOKEN>
POPPLE_AMQP_HOST=poppleevents
POPPLE_AMQP_PORT=5672
POPPLE_AMQP_USER=guest
POPPLE_AMQP_PASS=guest
POPPLE_DB_HOST=poppledb
POPPLE_DB_PORT=3306
POPPLE_DB_USER=root
POPPLE_DB_PASS=password
POPPLE_DB_NAME=popple
POPPLE_LISTEN_HEALTH=0.0.0.0:8080
MYSQL_ROOT_PASSWORD=password
EOF
```

2. [one-time-setup] Create a volume so that the MySQL database can persist beyond the Docker
container's lifecycle:

```console
$ docker volume create poppledata
```

(Note, the docker-compose expects it to be named "poppledata")

3. [one-time-setup] Set up the database:

```console
$ docker volume create poppledata
$ docker run -d \
    --name poppledb \
    --publish 3306:3306 \
    -v poppledata:/var/lib/mysql \
    -e MYSQL_ROOT_PASSWORD=password \
    mysql:8.0
$ mysql -h 127.0.0.1 -u root -p
mysql> CREATE DATABASE popple;
mysql> USE popple;
mysql> source ./cmd/popplesvc/.devel/schema.sql;
$ docker stop poppledb
$ docker rm poppledb
```

4. Build the popplebot and popplesvc images if needed:

```console
$ docker-compose build
```

5. Start it up:

```console
$ docker-compose up -d
```
