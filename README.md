# Cosmos Registrar

This is a config driven process for writing data programatically to a git repo. To run this:

> *NOTE:* If you would like to run this you will need a [Github Personal Access Token](https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token).


```bash
# install the cosmos-registrar
make install

# create the config file
registrar config init

# See the config file, there are a number of 
# fields that must be configured for your usecase
registrar config show

# edit the config using the binary
registrar config edit github-access-token {token}
registrar config edit git-name {name}
# etc...

# then, update your configured repo
registrar update

# this is designed to be run via a cron job
# below is the crontab for every hour updates
# 0 * * * * /path/to/registrar update --config /path/to/config > /path/to/update.log

# with small modification, this could also run as a daemon 
# via systemd and would take args for frequency
```

See https://github.com/jackzampolin/registry for example output

### Next Steps:
- [ ] Goreleaser integration
- [ ] release this