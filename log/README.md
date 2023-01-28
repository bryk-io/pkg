# Useful Logs

Producing good logs is important; so important in fact that there are SEVERAL
ways to do it in Go. Either with the standard library itself or using some very
good third party libraries. This package aims to:

- Provide a simple, idiomatic and friendly way to produce good/useful logs.
- Be extensible; i.e., play nice with other existing solutions.
- Be unobtrusive and simple to replace.
- Be super simple to setup and use.

## Loggers and Providers

To handle all log-related operations you first need to create a logger instance.
Loggers are interface-based utilities backed by a provider, the provider can
be implemented using any tool (or 3rd party package of your choosing). This makes
the package very easy to extend and re-use.

There are two types of loggers supported:

- `SimpleLogger`: Focusing on a simple interface to provide leveled logging.
  This loggers are compatible with the standard package logger interfaces and
  can be used as a drop-in replacement with minimal to no code changes.

- `Logger`: Logger instances extend the base functionality of the "SimpleLogger"
  and provide utilities to produce more structured/contextual log messages.

## Levels

Log messages relate to different kinds of events and some can be considered
more relevant/urgent than others, and as such, can be recorded and managed
in different ways. The simplest mechanism to specify the nature of a given
log message is by setting a specific `Level` to it. The `Logger` API provides
convenient methods to do this automatically. The standard level values supported
by this package are (in order of relevance):

- `Debug`: Should be use for information broadly interesting to developers
  and system administrators. Might include minor (recoverable) failures and
  issues indicating potential performance problems.

- `Info`: Should be used for informational messages that might make sense
  to end users and system administrators, and highlight the progress of the
  application.

- `Warning`: Should be used for potentially harmful situations of interest
  to end users or system managers that indicate potential problems.

- `Error`: Error events of considerable importance that will prevent normal
  program execution, but might still allow the application to continue running.

- `Panic`: Panic level should be used for very severe error events that might
  cause the application to terminate. Usually by calling `panic()` after logging.

- `Fatal`: Fatal level should be used for very severe error events that WILL
  cause the application to terminate. Usually by calling `os.Exit(1)` after logging.

Loggers can also be adjusted to ignore events "below" a certain level; hence
adjusting the verbosity level of the produced output.

```go
var log Logger

// By setting the level to `Warning`; all `Debug` and `Info` events
// will be automatically discarded by the logger.
log.SetLevel(Warning)
```

## Contextual Information

Sometimes is useful to add additional contextual data to log messages in the
form of key/value pairs. These `Fields` can be used, for example, to provide
further details about your environment, the task at hand, the user performing
the operation, etc.

```go
log.WithFields(Fields{
  "app.env": "dev",
  "app.live": false,
  "app.purpose": "demo",
  "app.version": "0.1.0",
}).Info("application is starting")
```

### Sub-Loggers

There might be some fields that you need to include in all logged messages.
Having to continuously add those will get repetitive and wasteful. An alternative
is to provide these common fields when initializing the logger instance with
`Sub`. You can use it regularly and add additional fields at a per-message
level as required.

The following example initialize a logger using the popular [ZeroLog](https://github.com/rs/zerolog) library as provider.

```go
// these fields will be "inherited" by all messages produced by
// the logger instance
commonFields := Fields{
  "app.env": "dev",
  "app.live": false,
  "app.purpose": "demo",
  "app.version": "0.1.0",
}

// setup a logger instance using "zero" as provider
log := WithZero(ZeroOptions{
  PrettyPrint: true,
  ErrorField:  "error",
}).Sub(commonFields)

// use the logger
log.WithField("stamp", time.Now().Unix()).Debug("starting...")
log.Info("application is ready")
```

## Composites

Logs are usually required at different places and in different formats. Having
readable console output can useful to quickly "eyeball" simple details; while at
the same time having a fully indexed collection of JSON events can be helpful as
observability, record-keeping and advanced analysis tool.

You can setup both (or as many as required) loggers and use `Composite` to manage
your application's logging requirements through the same simple `Logger` interface.

```go
// common application details
 appDetails := Fields{
  "app.env":     "dev",
  "app.purpose": "demo",
  "app.version": "0.1.0",
  "app.live":    false,
 }

 // sample file to collect logs in JSON format
 logFile, _ := os.OpenFile("my-logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
 defer logFile.Close()

 // these logger instance will "pretty" print output to the console
 consoleOutput := WithZero(ZeroOptions{PrettyPrint: true}).Sub(appDetails)

 // these logger instance will append logs in JSON format to a file.
 // Note that `Sink` can be a network connection, database, or anything
 // else conforming to the `io.Write` interface.
 fileOutput := WithZero(ZeroOptions{Sink: logFile}).Sub(appDetails)

 // using `Composite` we can "combine" both (or more) individual loggers
 // and keep the same simple to use interface.
 log := Composite(consoleOutput, fileOutput)

 // we can then just use the logger as usual
 log.Debug("initial message")
```

Log messages will be pretty printed to console output, while simultaneously
appended to the "my-logs.txt" file as one JSON object per-line.

```json
// formatted for readability
{
 "level": "debug",
 "app.env": "dev",
 "app.live": false,
 "app.purpose": "demo",
 "app.version": "0.1.0",
 "time": "2023-01-28T11:23:46-06:00",
 "message": "initial message"
}
```
