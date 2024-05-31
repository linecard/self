# Environment Variables

Self doesn't actually manage secrets or config at all. It is optional to use this feature via the symbiotic tool [entry](https://github.com/linecard/entry).

## Entry

Entry is a small, purpose built tool and it's purpose is simple:
1. Pull encrypted environment variables from SSM.
1. Execute a command with those environment variables set.

In fact, It need not be used with Self. It can be used for executing local tools with team-level secrets or it can be used in container entrypoints in general, not just in the context of Self.

Here is a simple example of how one would use entry:

```Dockerfile
ENTRYPOINT ["node", "webserver.js"]
```

becomes...

```Dockerfile
ENTRYPOINT ["entry", "-p", "/path/in/ssm/env", "--", "node", "webserver.js"]
```

To fully understand usage and placing your environment in SSM itself, please refer to the [entry documentation](https://github.com/linecard/entry).

> **Note:** A good example of using `entry` outside of containers is during development flow. You can use `entry` to execute your process, which helps verify your application will have what it needs at execution time. Borrowing from the example above you would simply run `entry -p /path/in/ssm/env -- node webserver.js` in your terminal.