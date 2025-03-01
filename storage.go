package nanoproto

import "fmt"

type NanoDataStorage struct {
	rpc        *RPC
	address    *string
	privateKey *string
}

func NewNanoDataStorage(rpc *RPC, address *string, privateKey *string) *NanoDataStorage {
	return &NanoDataStorage{rpc, address, privateKey}
}

// Your RPC must provide raw account history retrieval abilities for this method (rpc.nano.to won't work :/)
func (s *NanoDataStorage) GetData(address *string) ([][]byte, error) {
	received, _ := s.rpc.History(*address)
	bytes := []string{}

	for _, ahi := range received {
		pubKey, err := nanoAddressToPublicKey(ahi.Representative)
		if err != nil {
			return nil, err
		}

		bytes = append(bytes, pubKey)
	}

	data := getBuffers(bytes)

	return data, nil
}

// Your RPC must provide account info, work generation, and block processing abilities for this method (rpc.nano.to may work, untested)
func (s *NanoDataStorage) PutData(data []byte) error {
	addresses := CreateMessage(data)

	for _, address := range addresses {
		accountInfo, err := s.rpc.AccountInfo(*s.address)
		if err != nil {
			fmt.Errorf(err.Error())
			break
		}

		// fmt.Printf("%+v\n", accountInfo)

		work, err := s.rpc.WorkGenerate(accountInfo.Frontier)
		if err != nil {
			fmt.Errorf(err.Error())
			break
		}

		// fmt.Println(work)

		block, err := s.rpc.ChangeRepresentativeBlock(*s.privateKey, *s.address, address, work, accountInfo.Frontier, accountInfo.Balance)
		if err != nil {
			fmt.Errorf(err.Error())
			break
		}

		// fmt.Printf("%+v\n", block)

		_, err = s.rpc.ProcessChangeRepBlock(block)
		if err != nil {
			fmt.Errorf(err.Error())
			break
		}

		// fmt.Println(hash)
	}

	return nil
}
