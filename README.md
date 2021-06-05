## Database schema

![alt text](schema.png)

## Dev setup

### Commands

```shell
## copy and modify your local env
cp .env.example .env
## run base infra (postgres), this command just create a pg host and an empty database
make init 
## run server, database will be migrated when running server by using gorm.AutoMigrate in cmd/server.go
make run 
```

### Application

Server should be available at http://localhost:8080 after `make run`, and you should use `ngrok http 8080` to establish a public URL from localhost to subscribe Slack events & commands

### Fixtures

User could be created or updated when he sends a msg to Slack channel where Slack bot is invited

