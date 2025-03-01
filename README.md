# nanoproto

**nanoproto** is a Golang library designed to save and read protocol buffers (and other byte data) on the NANO ($XNO) cryptocurrency network. It leverages NANO's feeless and fast transactions to enable decentralized data storage and retrieval.

## Features
- Save and retrieve byte data (including protocol buffers) on the NANO network.
- Simple integration with NANO RPC nodes.
- Supports encoding and decoding of protocol buffers for structured data storage.

## How it works
**nanoproto** uses the NANO network's representative system to store and retrieve data. The library creates a new block for each data entry (~1 entry per 32 bytes of data), with the data stored in the block's `representative` field.

### How does this project differ from other NANO data storage projects?
While there aren't many projects that focus on storing data on the NANO network (for obvious reasons), **nanoproto** aims to provide the simplest and most storage efficient way to store and retrieve data on the NANO network. It uses protocol buffers to encode and decode data, allowing for structured data storage and retrieval and representative changes instead of small NANO amounts being sent from address to address (which is less efficient and takes more storage + computing power).

## Installation

To use **nanoproto**, ensure you have Go installed, then run:

```bash
go get github.com/nextu1337/go-nanoproto
```

## Usage

Below is an example of how to use **nanoproto** with **go-raw-protobuf** to store and retrieve data on the NANO network:

```go
package main

import (
	"encoding/hex"
	"fmt"

	"github.com/nextu1337/go-nanoproto"
	protobuf "github.com/nextu1337/go-raw-protobuf"
)

func main() {
	// Initialize the RPC client
	rpc := nanoproto.NewRPC("https://rainstorm.city/api")
	address := "nano_3amxrw4dbmjizkuuqo44pt1rbq49o5rsibhc9tjb6mxcxyr13yc5cir3wxc4"
	privateKey := "AAFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"

	// Create a new NanoDataStorage instance
	storage := nanoproto.NanoDataStorage{rpc, &address, &privateKey}

	// Define the storage address
	storageAddress := "nano_1zaxxczbjn1o5imqrxek8m7b1ji1zactwy5yit19xqni68i6beo9a5a1an6x"

	// Retrieve data from the NANO network
	fmt.Println("Getting storage")
	data, err := storage.GetData(&storageAddress)
	if err != nil {
		fmt.Println(err)
	}

	// Decode and print the retrieved data
	for _, chunk := range data {
		decoded := protobuf.DecodeProto(chunk)
		fmt.Println("--------------------")
		fmt.Println(hex.EncodeToString(chunk))
		fmt.Println("Decoded data", protobuf.ProtoPartsToArray(decoded.Parts))
		if len(decoded.LeftOver) > 0 {
			fmt.Println("LeftOver bytes", decoded.LeftOver)
		}
	}

	// Encode and store new data on the NANO network
    fmt.Println("--------------------")
	fmt.Println("Setting storage")

	var buf []interface{} = []interface{}{1, 2, 3, "This works!"}

    // Encode the data as protobuf
	bytes := protobuf.EncodeProto(protobuf.ArrayToProtoParts(buf))
    
	fmt.Println(hex.EncodeToString(bytes))

	storage.PutData(bytes)
}
```

## Documentation

### `NewRPC(url string) *RPC`
Initializes a new RPC client with the specified NANO node URL.

### `NanoDataStorage`
A struct that holds the RPC client, address, and private key for interacting with the NANO network.

#### Methods:
- `GetData(reps *string) ([][]byte, error)`: Retrieves data from the specified NANO address.
- `PutData(data []byte) error`: Stores the provided byte data on the NANO network.

## Contributing
Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

---

**Disclaimer**: This library is experimental and should be used with caution. Ensure you understand the risks of storing data on the NANO network before using it in production.