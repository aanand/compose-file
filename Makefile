SCHEMA_GO := schema/bindata.go
SCHEMA_JSON := schema/data/config_schema_v2.1.json

test:
	go test ./loader ./schema

schema: $(SCHEMA_GO)

$(SCHEMA_GO): $(SCHEMA_JSON)
	go generate ./schema
