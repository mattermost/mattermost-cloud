# Mattermost Cloud Tools

A collection of optional tools that support working with the cloud server.

## CWL - Cloud Webhook Listener

A small listener that pretty prints cloud webhooks. Cloud webhooks are fired off from many state changes to cloud resources along with other important events. Run `cloud webhook -h` to get info on managing these.

Example output from an installation being created:

```
2020/06/29 09:58:34 Starting cloud webhook listener on port 8065
2020/06/29 10:06:08 [ INST | zf8a ] n/a -> creation-requested
2020/06/29 10:06:10 [ CLIN | 86i7 ] n/a -> creation-requested
2020/06/29 10:06:17 [ INST | zf8a ] creation-requested -> creation-in-progress
2020/06/29 10:06:20 [ CLIN | 86i7 ] creation-requested -> reconciling
2020/06/29 10:06:44 [ CLIN | 86i7 ] reconciling -> stable
2020/06/29 10:06:54 [ INST | zf8a ] creation-in-progress -> stable
```

### Installation

From the repo root:

```
go install ./cmd/tools/cwl
```

### Running CWL

Assuming `GOPATH/bin` is in your `PATH`, then run `cwl`.

### Configuration

The default `cwl` listening port is `8065`. This is the default Mattermost server port as well, which allows you to quickly switch between testing the Mattermost cloud plugin and using `cwl` without having to delete your webhook in the cloud server.

To use another port, set `CWL_PORT`.

Example:

```
export CWL_PORT=9001
```


## Event Listener

Event listener is similar to CWL but instead of webhooks it listens for Events.

Event listener registers subscription if it does not already exist and starts to listen for events.

Events are printed to stdout in JSON format.

### Running event listener

```bash
go run ./cmd/tools/event-listener
```

### Configuration

If you want to run multiple event listeners configure them to use different ports 
and subscription owners with environment variables.
- `EVENTS_PORT` - configures listener port (default `8099`)
- `SUB_OWNER` - owner of the subscription (default `local-event-listener`)

By default, subscription will be deleted when you stop the event listener.
You can keep it around by setting `CLEANUP_SUB` to `false`.
