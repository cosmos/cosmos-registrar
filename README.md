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
cd cosmos-registrar && make install
```

after running the command above you should find the binary at: `$GOPATH/bin`


**Prebuilt Binaries**

[Binaries are auto-generated](https://github.com/cosmos/cosmos-registrar/releases) upon each Git tag.

**Removal**

In case you want to un-install the `registrar` after running the steps above,
the files and folder to be removed are:

- `/usr/local/bin/registrar` - the executable binary
- `$HOME/.config/cosmos/registry` - (linux only) the folder where config files and the workspace is stored by default
- `$HOME/Library/Application Support/cosmos/registry` - (macOs only) the folder where config files and the workspace is stored by default


## Usage

You can run `registrar` for an interactive walkthrough:

```sh
$ registrar -c private/config.yaml
D[2021-05-21|16:11:11.056] Using config file at                         config=private/config.yaml
? what shall we do today? Register a new ChainID

Good choice, following this process you will submit a
pull request to the chain IDs registry hosted on GitHub:

https://github.com/cosmos/registry

The first step is to create a fork of the registry using this link:

https://github.com/cosmos/registry/fork

? Go ahead and confirm when you have done so Yes

Next enter the rpc url for a node of your network
eg http://10.0.0.1:26657

? rpc address http://localhost:26657
checking out branch  localnet
...
```

Or non-interactively:
`registrar claim http://localhost:26657`
`registrar update` (your PR to claim the chain id must be accepted first)

Usually you will run the claim process interactively and run the update subcommand with a cronjob.

### Claiming a chain ID (interactive)

The procedure to claim a chain id only requires to run the command `registrar` and select the *Register a new ChainID* option from the proposed menu.

At this point you can follow the instructions that the tool will display and at the end of the
process you should have successfully submitted a request (PR) to claim your chain id.

### cosmoshub-4 special notes
The `cosmoshub-4` `genesis.json` is too large to be downloaded from a node's Tendermint RPC (should be fixed with Tendermint 0.35) and too large to be uploaded to a Github repository.

Therefore when you are claiming `cosmoshub-4` ensure that `genesis.cosmoshub-4.json.gz` is in your current directory. It is taken from [cosmos/mainnet](https://github.com/cosmos/mainnet/) and has a md5sum of `a4216a3cae68e9190d0757c90bcb1f1b`.

### Publish updates

Once the claim has been successful you can run the command:

```sh
registrar update
```

the command will read your configuration and submit updates to the main registry on your behalf.

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

## Troubleshooting
If a command should fail, it is possible the workspace and the remote git repo is dirty and needs to be cleaned up manually for now.

1. Ensure that your configuration directory only  contains `config.yaml`
2. In your own fork of the `registry` repo, ensure that the branch with the `chain_id` that you were trying to register is deleted.