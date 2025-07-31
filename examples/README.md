# Examples for tf-oracle-helper

This directory contains self-contained Go programs demonstrating how to use the `tf-oracle-helper` module.

## Prerequisites

To run these examples, you need:

- Go (version 1.18 or higher)
- Access to an Oracle database.
- Set the following environment variables for database connection:
    - `ORACLE_USERNAME`
    - `ORACLE_PASSWORD`
    - `ORACLE_DB_HOST`
    - `ORACLE_DB_PORT`
    - `ORACLE_DB_SERVICE`

## Running Examples

Each example is in its own subdirectory. To run an example:

1.  Navigate to the example's directory:
    ```bash
    cd examples/connect_and_read_parameter
    # or cd examples/create_and_drop_user
    # or cd examples/create_user_with_grants
    ```

2.  Run the Go program:
    ```bash
    go run main.go
    ```

### Example: `connect_and_read_parameter`

This example demonstrates how to establish a connection to an Oracle database and read a database parameter (`undo_retention`).

### Example: `create_and_drop_user`

This example demonstrates how to create a new Oracle user and then drop that user.

### Example: `create_user_with_grants`

This example demonstrates how to create a new Oracle user and grant them system and object privileges.