
```txt
startup:
--------

func main()
    -> Open sqlite 3 database (this is the persistence layer)
    -> Establish Discord session
    -> Configure command router, so that depending on a message's prefix, it is
       routed to the corresponding command in command.go.
    -> session.AddHandler(callback)
         (This callback will enqueue a closure onto a work queue for processing
         later. The idea is to keep the session.AddHandler callback as short as
         possible.)

normal operation:
-----------------

    discordgo.Session
        -> callback()
            -> Place job on work queue, to be picked up by func worker
               ("jobs" are just closures that capture the necessary
               state for a given function in command.go; see the calls
               to router.addRoute for more info)

    func worker()
        -> Receive job from work queue
            -> Call closure
               (Closure contains the router's state that will lead it to the
                correct business logic in command.go.)
               -> {checkKarma, modKarma, top, bot, uptime, version, help, etc...}()

shutdown:
---------

func main()
    -> Receive signal
        -> Detach handler from Discord session to avoid new work during
           shutdown.
        -> Send cancel message down cancel channel to workers.
        -> Wait for acknowledgement from workers or continue to
           shut down without them after a deadline expires.
        -> Exit.

func worker()
    -> Finish current work
    -> Receive cancel message
        -> Send acknowledgement
            -> return
```
