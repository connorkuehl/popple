The Popple bot's basic organizational structure is that of an event-driven
application.

## TL;DR

```txt
func main()
    -> session.AddHandler(callback)

    discordgo.Session
        -> callback()
            -> Place job on work queue

    func worker()
        -> receives job from work queue
            -> func doWork(job)
                -> call functions in command.go
```

## Details

During startup, Popple's main thread will use the `discordgo` API to
start a Discord session and register a callback function for whenever
the bot receives a message from any channel that it is in.

```txt
func main
    -> session.AddHandler
```

The closure passed in to `session.AddHandler` encapsulates the incoming Message
and Discord session objects in a `Job` struct and places it on a work queue
where a worker goroutine (`func worker`) will remove it from the work queue
and perform the actual work.

This structure resembles a "top-half" and "bottom-half" style of processing.

The "top-half" is when the `discordgo` library calls the callback.
In order to keep the top-half's execution time short, the relevant data is
simply placed on a work queue for later processing and then the top-half
is done and can go back to waiting for more input from Discord.

The "bottom-half" is the other side of this coin. In `job.go`, `func worker`
is the entrypoint for the goroutines that were spun up when Popple was first
starting up.

The worker goroutines will idle in `func worker`. They are waiting for jobs
from the top-half.

```txt
func worker()
    -> func doWork()
```

Once the worker has work to do in `func doWork`, it will scan the message
and compare it to a command dispatch table to determine if the message is
a sub-command dedicated for the Popple bot or if it is just a regular message
that the bot must scan for karma events.

Once the command is identified, the relevant business-logic function
in `command.go` is called.

Generally, `command.go` is the terminating destination for the flow of control
in Popple. Depending on what bot action is taking place, Popple will perform
the work here and use the context included with the `Job` struct to interact
with the Discord API and the persistence layer.