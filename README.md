# research

## Simple system using Proof of Work

Start using
```
go build
```
then
```
./go-blockchain-example-main
```


After starting, you will see
```
[  info ] 20XX/XX/XX XX:XX:XX main.go:XX: node address: /ip4/192.168.XX.XX/tcp/XX/p2p/XX
```


If you would like to add another node, please input
```
./go-blockchain-example-main /ip4/192.168.XX.XX/tcp/XX/p2p/XX
```
then, the second node will connect with the first.


In each client, you can enter the following commands:
- `ls p` - list peers
- `ls c` - list blockchain
- `create b $data` - $data is just a string here - this creates a new block with the data
