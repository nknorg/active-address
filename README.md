Count the number of active addresses within a block height range.

An address is defined as active if it has made at least one transaction or received any token within the block height range.

## Usage

Build:

```shell
go build .
```

Count the active address:

```shell
./active-address -s <start block height> -e <end block height> -rpc <rpc address>
```

Add `-v` if you also want to print the number of transactions of each address.
