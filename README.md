# tf-oracle-helper

`tf-oracle-helper` is a Go module designed to simplify interactions with Oracle databases, primarily intended for use with Terraform providers. It provides a set of services to manage various Oracle database objects and configurations.

## Features

- **Database Client:** Establishes and manages connections to Oracle databases.
- **User Management:** Create, read, modify, and drop database users.
- **Role Management:** Create, read, modify, and drop database roles.
- **Profile Management:** Manage user profiles.
- **Grant Management:** Grant and revoke object, system, and role privileges.
- **Parameter Management:** Read and set database parameters.
- **Scheduler Window Management:** Manage Oracle scheduler windows.
- **Autotask Management:** Enable and disable autotasks.
- **Block Change Tracking:** Manage block change tracking.
- **Audit User Management:** Configure and read user audit options.
- **Statistics Preferences:** Manage global, schema, and table-level statistics preferences.

## Installation

To use `tf-oracle-helper` in your Go project, you can fetch it using `go get`:

```bash
go get github.com/dtherhtun/tf-oracle-helper/oraclehelper
```

## Usage

Here's a basic example of how to use the `oraclehelper` module to connect to an Oracle database and perform a simple operation:

```go
package main

import (
	"fmt"
	"log"
	"github.com/dtherhtun/tf-oracle-helper/oraclehelper"
)

func main() {
	cfg := oraclehelper.Cfg{
		Username:  "system",
		Password:  "your_password",
		DbHost:    "localhost",
		DbPort:    "1521",
		DbService: "your_db_service",
		SysDBA:    false,
	}

	client, err := oraclehelper.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Oracle client: %v", err)
	}
	defer client.Close()

	// Example: Read a database parameter
	param, err := client.ParameterService.Read(oraclehelper.ResourceParameter{Name: "undo_retention"})
	if err != nil {
		log.Fatalf("Failed to read parameter: %v", err)
	}
	fmt.Printf("Undo Retention: %s\n", param.Value)
}
```

For more detailed usage and examples, please refer to the GoDoc documentation for each service and the `_test.go` files within the `oraclehelper` directory.

## Contributing

Contributions are welcome! Please feel free to open issues or submit pull requests.

## Inspiration & Key Advantages

> Note: This module is inspired by [dbgeek/terraform-oracle-rdbms-helper](https://github.com/dbgeek/terraform-oracle-rdbms-helper) and builds upon its concepts.
One of the key benefits of tf-oracle-helper is that it eliminates the need for Oracle Instant Client dynamic libraries, making deployment simpler and reducing external dependencies.

## License

This project is licensed under the MIT License.
