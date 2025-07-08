# DFMailbox compliance tests
This is **the** way to test compliance with DFMailbox's protocol.

# Running the tests
Run `ginkgo` in the project root
```sh
ginkgo -vrp # Verbose, recursive, and parallel
```
Although it is recommended to use ginkgo, technically `go test` works too.

# Setup
To test your implementation against this test suite, you must first create a `Dockerfile` and a `compliance-docker-compose.yml` for your app.

The `compliance-docker-compose.yml` must have the DFMabilbox service all the required services.
The DFMailbox service must meet these requirements:
- All exposed ports are to be automatically assigned. Only the first port will be recognized as the DFMailbox instance port.
    ```yaml
        ports:
          - 8080
    ```
- Takes in all the docker compose env vars that will be provided by the test suite.
    - `DFMC_ADDRESS` - the primary address of the instance
    - `DFMC_PRIVATE_KEY` - the base64 encoded ED25519 private key with the seed only
        - example value: `TESTING0KEYTESTING0KEYTESTING0KEYTESTING000=`
    ```yaml
    environment:
      HOST: ${DFMC_ADDRESS}
      SECRET_KEY: ${DFMC_PRIVATE_KEY}
      PORT: 8080
      ```
- Has the extra_hosts section contain
    ```yaml
    extra_hosts:
      - "host.docker.internal:${DFMC_HOST_GATEWAY:-host-gateway}"
      - "alt-host.docker.internal:${DFMC_HOST_GATEWAY:-host-gateway}"
  ```

# Environment variables
Environment files are NOT read from `.env`

## `DFMC_COMPOSE_FILE`
Specifies the path to lookup the file. This file path is relative to `/test`.
The default is  `../../compliance-docker-compose.yml`

## `DFMC_HOST_GATEWAY`
Specifies the IP that the host machine has. Usually this values shouldn't be modified.
Defaults to `""` (which then should be replaced by compose file to be `host-gateway`).
This value is passed into the `compliance-docker-compose.yml`.

