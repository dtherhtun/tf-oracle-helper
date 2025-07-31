# Makefile for tf-oracle-helper examples

.PHONY: all clean connect_and_read_parameter create_and_drop_user create_user_with_grants

# Define example directories
EXAMPLES_DIR := examples
CONNECT_EXAMPLE := $(EXAMPLES_DIR)/connect_and_read_parameter
CREATE_DROP_USER_EXAMPLE := $(EXAMPLES_DIR)/create_and_drop_user
CREATE_USER_WITH_GRANTS_EXAMPLE := $(EXAMPLES_DIR)/create_user_with_grants

# Environment variables required for Oracle connection
# Set these in your shell before running make, e.g.:
# export ORACLE_USERNAME=system
# export ORACLE_PASSWORD=your_password
# export ORACLE_DB_HOST=localhost
# export ORACLE_DB_PORT=1521
# export ORACLE_DB_SERVICE=orclpdb1

all: connect_and_read_parameter create_and_drop_user create_user_with_grants

connect_and_read_parameter:
	@echo "\n--- Running connect_and_read_parameter example ---"
	@cd $(CONNECT_EXAMPLE) && go run main.go

create_and_drop_user:
	@echo "\n--- Running create_and_drop_user example ---"
	@cd $(CREATE_DROP_USER_EXAMPLE) && go run main.go

create_user_with_grants:
	@echo "\n--- Running create_user_with_grants example ---"
	@echo "NOTE: This example requires SYSDBA privileges for the ORACLE_USERNAME."
	@cd $(CREATE_USER_WITH_GRANTS_EXAMPLE) && go run main.go

clean:
	@echo "Cleaning up Go modules in examples..."
	@cd $(CONNECT_EXAMPLE) && go clean -modcache
	@cd $(CREATE_DROP_USER_EXAMPLE) && go clean -modcache
	@cd $(CREATE_USER_WITH_GRANTS_EXAMPLE) && go clean -modcache
	@echo "Done."

