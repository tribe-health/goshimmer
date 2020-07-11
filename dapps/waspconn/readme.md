# WaspConn dApp (plugin) for Goshimmer

## Purpose

The _WaspConn_ dApp handles connection with Wasp nodes, it is the Wasp proxy
in Goshimmer. 

One or several Wasp nodes can be connected to one Goshimmer node. The Wasp node
can only be connected to one Goshimmer node (this may change in the future).

WaspConn is used for testing purposes. It allows to test many (most)
of Wasp features such as consensus, syncing, token handling, VM and so on
without real distributed Value Tangle (see `utxodb`). 
WaspConn implements some API endpoints just for testing purposes.   

## UTXODB
WaspConn contains `utxodb`package which
contains in-memory, centralized (non-distributed) ledger of value transactions. 
`utxodb` has genesis transaction with initial supply of IOTA tokens, 
it validates neq value transactions, rejects conflicting transactions, provides
API to retrieve address balances and so on. It also emulates confirmation delay and conflict handling
strategies (configurable).

`utxodb` allows to test most of Wasp features without having access to 
distributed Value Tangle.

## Dependency

WaspConn is not dependent on Wasp, Instead, Wasp has Goshimmer with WaspConn 
dApp (`wasp` branch of the Goshimmer) as dependency.

WaspConn and Goshimmer are unaware about smart contract transactions. They treat it as just
ordinary value transaction with payloads. 

## Protocol

WaspConn implements its part of the protocol between Wasp node and Goshimmer. 

- The protocol is completely **asynchronous messaging**: neither party is waiting for the response or confirmation
after sending message to another party, even if most of messages are requests which results in responses. 
It also means, that messages may be lost without notification.

- the transport between Goshimmer and Wasp is using `BufferedConnection` provided by `hive.go`. 
Protocol can handle practically unlimited message sizes.

### Posting a transaction
Wasp node may post the transaction to Goshimmer for confirmation just like any other agent. In case when 
`utxodb` is enabled, the transaction goes right into the confirmation emulation mechanism of `utxodb`. 

### Subscription
Wasp node subscribes to transaction it wants to receive. It sends a list of addresses of smart contracts 
it is running and WaspConn is sending to Wasp any new confirmed transaction which has subscribed address among it 
outputs.

### Requesting a transaction
Wasp may request a transaction by hash. WaspConn plugin send the confirmed transaction to Wasp (if found).

### Requesting address balances
Wasp may request address balances from Goshimmer. WaspConn sends confirmed outputs which belongs to the address.  

### Sending request backlog to Wasp

Upon request, WaspCon sends not only unspent outputs contained in the address, but also analyses colored 
tokens and sends origin transactions of corresponding colors if they contain unspent outputs.

## Configuration

All configuration values for the WaspConn plugin are in the `waspconn` portion of the `config.json` file.
```
  "waspconn": {
    "port": 5000,
    "utxodbconfirmseconds": 0,
    "utxodbconfirmrandomize": false,
    "utxodbconfirmfirst": true
  }
```
- `waspconn.port` specifies port where WaspCon is listening for new Wasp connections.
- `waspconn.utxodbconfirmseconds` specifies emulated confirmation time. When new transaction is posted, 
specified amount of seconds it is in `pending` state and only after confirmation time it is included 
into the `utxodb` ledger.
`0` seconds means it is included immediately and result is known to the posting call synchronously.
- `waspconn.utxodbconfirmrandomize` if `false`, the emulated confirmation time is fixed, if `true` it is 
uniformly distributed around the confirmation time parameter.
-  `waspconn.utxodbconfirmfirst` determines behavior in case of conflicting transactions. If `true`, 
eventually the first out of all conflicting transactions will be included into `utxodb` ledger. If `false` 
all conflicting transactions will be rejected within duration period between posting first of them and 
the supposed confirmation deadline.
   
