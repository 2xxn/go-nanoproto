package nanoproto

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"slices"
)

const (
	BEGIN_PROTOBUF = "626567696E6461746100"
	FORCE_END      = "0000656E646461746100"
)

func CreateMessage(data []byte) []string {
	beginProtoMark, err := hex.DecodeString(BEGIN_PROTOBUF)
	endProtoMark, err := hex.DecodeString(FORCE_END)
	if err != nil {
		panic(err)
	}

	var buffer bytes.Buffer

	buffer.Write(beginProtoMark)
	buffer.Write(data)
	buffer.Write(endProtoMark)

	bytes := buffer.Bytes()
	if len(bytes)%32 != 0 {
		padding := 32 - len(bytes)%32
		bytes = append(bytes, make([]byte, padding)...)
	}

	var messages []string

	for i := 0; i < len(bytes); i += 32 {
		chunk := bytes[i : i+32]
		nanoAddress, err := publicKeyToNanoAddress(chunk)
		if err != nil {
			panic(err)
		}

		messages = append(messages, nanoAddress)
	}

	return messages
}

func getBuffers(history []string) [][]byte {
	beginProtoMark, err := hex.DecodeString(BEGIN_PROTOBUF)
	endProtoMark, err := hex.DecodeString(FORCE_END)
	if err != nil {
		panic(err)
	}

	var buffers [][]byte
	var longChunk string = ""

	for _, chunk := range history {
		longChunk += chunk
	}

	longChunkBytes, err := hex.DecodeString(longChunk)
	if err != nil {
		panic(err)
	}

	buffer := bytes.NewBuffer(longChunkBytes)
	reader := bufio.NewReaderSize(buffer, len(longChunkBytes))
	currentlyReading := false
	var data []byte

	for reader.Size() > 0 {
		// If we are not currently reading a protobuf message, check for the beginning of a new one
		if !currentlyReading {
			_, err := reader.ReadBytes(beginProtoMark[0])
			if err != nil {
				break // No more 0x62 bytes
			}

			reader.UnreadByte()
			bytes, err := reader.Peek(len(beginProtoMark))
			if err != nil {
				fmt.Println("Error peeking bytes", err)
				break
			}

			if slices.Equal(bytes, beginProtoMark) {
				currentlyReading = true
				reader.Discard(len(beginProtoMark))
				data = []byte{}
			}

			continue
		}

		// If we are currently reading a protobuf message, read until the end of the message
		byteRead, err := reader.ReadByte()
		if err != nil {
			fmt.Println("Error reading byte", err)
			break
		}

		if byteRead == endProtoMark[0] {
			bytes, err := reader.Peek(len(endProtoMark) - 1)
			if err != nil {
				fmt.Println("Error peeking bytes", err)
				break
			}

			if slices.Equal(bytes, endProtoMark[1:]) {
				currentlyReading = false
				buffers = append(buffers, data)
				continue
			}
		}
		data = append(data, byteRead)
	}

	return buffers
}
