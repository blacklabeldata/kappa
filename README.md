# kappa

Kappa is an implementation of the Kappa Architecture written in Go.


## Installation

`kappa` uses [godep](https://github.com/tools/godep) to manage dependencies. So we need to install that first:

```
go get github.com/tools/godep
```

Next, we need to actually install the code and dependencies:

```
godep get github.com/blacklabeldata/kappa
```

Finally we can build the code:

```
$ make build
```

## Generating PKI

`kappa` requires a public key infrastructure in order to run. Luckily, `kappa` makes it easy to get started.

```
$ make setup
```

This will generate a certificate authority, a private for the SSH server and an admin private key. Alternatively, you can generate them individually.

####  Generate a new CA

```
# ./kappa init-ca
```

####  Generate another certificate

```
# ./kappa new-cert --name=<CERT NAME>
```

## Running Kappa

Running `kappa` is also simple. It's one command:

```
$ make run
```

## Command Line Access

Command line access is through ssh and using the admin key we generated earlier in setup:

```
$ chmod 600 pki/private/admin.key
$ ssh -i pki/private/admin.key admin@127.0.0.1 -p 9022
```
