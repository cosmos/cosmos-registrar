# Cosmos Registrar

This is a config driven process for writing data programatically to a git repo. To run this:

```bash
# create the config file
go run main.go config init
# See the config file, there are a number of 
# fields that must be added by users
go run main.go config show
# edit the config using the binary
# https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token
go run main.go config edit github-access-token {token}
go run main.go config edit git-name {name}
# etc...
```

See https://github.com/jackzampolin/registry for example output

### Next Steps:
- [ ] Makefile to build binary
- [ ] Goreleaser integration
- [ ] release this