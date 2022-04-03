# Parsing & handling encrypted net-messages

See also [net-messages](../net-messages) for regular net-messages.

This example shows how to have the parser deal with encrypted net-messages.

For Valve MM games, the decryption key can be obtained from `.dem.info` files using `MatchInfoDecryptionKey()`.
The key then needs to be passed to `ParserConfig.NetMessageDecryptionKey`.

## Run

    go run enc_net_nsg.go -demo path/to/demo.dem -info path/to/demo.dem.info

This prints chat messages from the passed demo (assuming the `.dem.info` file contains the correct decryption key).
