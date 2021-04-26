# Cosmos Registrar

The cosmos registrar project is a command line client that allows to claim a chain ID in the [cosmos chain id registry](https://github.com/cosmos/registry) and to publish updates about the status of the nodes for the claimed chain id.


## Installation 

**Building from source**

> this section assumes you are running linux and macOs

To build the cosmos registry binary from source you'll need 
- go v1.16 or above
- git
- gnu make

step 1. checkout the source repository:

```sh
git clone https://github.com/cosmos/cosmos-registrar.git
```

step 2. build the binary using go

```sh
cd cosmos-registrar && make build
```

after running the command above you should find the binary at: `./build/registrar`

step 3. install the binary in your execution path (Optional)

```sh
mv build/registrar /usr/local/bin
```


**Binary distribution**

TODO

**Removal**

In case you want to un-install the `registrar` after running the steps above, 
the files and folder to be removed are:

- `/usr/local/bin/registrar` - the executable binary
- `$HOME/.config/cosmos/registry` - (linux only) the folder where config files and the workspace is stored by default 
- `$HOME/Library/Application Support/cosmos/registry` - (macOs only) the folder where config files and the workspace is stored by default 


## Usage

The `registrar` command can be run interactively (recommended when claiming a chain id) or as a non-interactive command (recommended when updating an already-claimed chain id) 

### Claiming a chain ID (interactive)

Te procedure to claim a chain id only requires to run the command `registrar` and select the *Register a new ChainID* option from the proposed menu.

At this point you can follow the instructions that the tool will display and at the end of the 
process you should have successfully submitted a request (PR) to claim your chain id.
### Publish updates 

Once the claim has been successful you can run the command:

```sh
registrar update 
```

the command will read your configuration and submit updates to the main registry on your behalf

## Configurations

The default configuration is automatically created at:
- `$HOME/.config/cosmos/registry/config.yaml` on linux
- `$HOME/Library/Application Support/cosmos/registry/config.yaml` on macOs


Here is a sample configuration file:

```yaml
# used to authenticate to make push requests and create links for clone
git-email: myuser@apeunit.com
git-name: myuser
# github access token 
github-access-token: 12345678909876543210123456789
# the following are used to identify the repositories coordinates
#
# name of the registry fork for the current user 
registry-fork-name: registry
# location and branch name for the root registry (change only for testing purposes)
registry-root: https://github.com/cosmos/registry
registry-root-branch: main
```
