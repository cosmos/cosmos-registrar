# Cosmos Registrar

This is a config driven process for writing data programatically to a git repo. To run this:

```bash
# install the cosmos-registrar
make install

# create the config file
registrar config init
# See the config file, there are a number of 
# fields that must be added by users
registrar config show
# edit the config using the binary
# https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token
registrar config edit github-access-token {token}
registrar config edit git-name {name}
# etc...
```

See https://github.com/jackzampolin/registry for example output

### Next Steps:
- [ ] Goreleaser integration
- [ ] release this