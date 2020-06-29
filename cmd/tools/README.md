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
