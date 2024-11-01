# protoc --go_out=plugins=grpc:./  message.proto
# protoc --go_out=. --go_opt=paths=source_relative message.proto

protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       event_log.proto