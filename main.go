package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/nknorg/nkn/v2/api/httpjson/client"
	"github.com/nknorg/nkn/v2/common"
	"github.com/nknorg/nkn/v2/pb"
	"github.com/nknorg/nkn/v2/program"
	"google.golang.org/protobuf/proto"
)

type GetBlockResp struct {
	Result struct {
		Header struct {
			SignerPk string `json:"signerPk"`
		} `json:"header"`
		Transactions []struct {
			PayloadData string `json:"payloadData"`
			Programs    []struct {
				Code      string `json:"code"`
				Parameter string `json:"parameter"`
			} `json:"programs"`
			TxType string `json:"txType"`
		} `json:"transactions"`
	} `json:"result"`
}

func main() {
	startHeight := flag.Int("s", 0, "Start block height (inclusive)")
	endHeight := flag.Int("e", 0, "End block height (inclusive)")
	rpcAddr := flag.String("rpc", "http://127.0.0.1:30003", "RPC address")
	verbose := flag.Bool("v", false, "Verbose: print transaction count of each address")

	flag.Parse()

	counter := make(map[string]int, 0)
	for height := *startHeight; height < *endHeight; height++ {
		if *verbose {
			log.Println(height)
		}
		err := countActiveAddrAtHeight(height, *rpcAddr, counter)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *verbose {
		for addr, n := range counter {
			fmt.Printf("%v\t%v\n", addr, n)
		}
	}

	fmt.Printf("Active address: %v\n", len(counter))
}

func countActiveAddrAtHeight(height int, rpcAddr string, counter map[string]int) error {
	resp, err := client.Call(rpcAddr, "getblock", 0, map[string]interface{}{"height": height})
	if err != nil {
		return err
	}

	v := &GetBlockResp{}
	err = json.Unmarshal(resp, v)
	if err != nil {
		return err
	}

	if len(v.Result.Header.SignerPk) == 0 {
		return errors.New("no signer")
	}

	pk, err := hex.DecodeString(v.Result.Header.SignerPk)
	if err != nil {
		return err
	}

	ph, err := program.CreateProgramHash(pk)
	if err != nil {
		return err
	}

	blockDigner, err := ph.ToAddress()
	if err != nil {
		return err
	}

	counter[blockDigner]++

	addrs := make([][]byte, 0)
	for _, tx := range v.Result.Transactions {
		buf, err := hex.DecodeString(tx.PayloadData)
		if err != nil {
			return err
		}

		switch tx.TxType {
		case pb.PayloadType_name[int32(pb.PayloadType_SIG_CHAIN_TXN_TYPE)]:
			payload := &pb.SigChainTxn{}
			err = proto.Unmarshal(buf, payload)
			if err != nil {
				return err
			}
			addrs = append(addrs, payload.Submitter)
		case pb.PayloadType_name[int32(pb.PayloadType_TRANSFER_ASSET_TYPE)]:
			payload := &pb.TransferAsset{}
			err = proto.Unmarshal(buf, payload)
			if err != nil {
				return err
			}
			addrs = append(addrs, payload.Sender, payload.Recipient)
		case pb.PayloadType_name[int32(pb.PayloadType_COINBASE_TYPE)]:
			payload := &pb.Coinbase{}
			err = proto.Unmarshal(buf, payload)
			if err != nil {
				return err
			}
			addrs = append(addrs, payload.Sender, payload.Recipient)
		case pb.PayloadType_name[int32(pb.PayloadType_REGISTER_NAME_TYPE)]:
			payload := &pb.RegisterName{}
			err = proto.Unmarshal(buf, payload)
			if err != nil {
				return err
			}
			addrs = append(addrs, payload.Registrant)
		case pb.PayloadType_name[int32(pb.PayloadType_TRANSFER_NAME_TYPE)]:
			payload := &pb.TransferName{}
			err = proto.Unmarshal(buf, payload)
			if err != nil {
				return err
			}
			addrs = append(addrs, payload.Registrant)
		case pb.PayloadType_name[int32(pb.PayloadType_DELETE_NAME_TYPE)]:
			payload := &pb.DeleteName{}
			err = proto.Unmarshal(buf, payload)
			if err != nil {
				return err
			}
			addrs = append(addrs, payload.Registrant)
		case pb.PayloadType_name[int32(pb.PayloadType_SUBSCRIBE_TYPE)]:
			payload := &pb.Subscribe{}
			err = proto.Unmarshal(buf, payload)
			if err != nil {
				return err
			}
			addrs = append(addrs, payload.Subscriber)
		case pb.PayloadType_name[int32(pb.PayloadType_UNSUBSCRIBE_TYPE)]:
			payload := &pb.Unsubscribe{}
			err = proto.Unmarshal(buf, payload)
			if err != nil {
				return err
			}
			addrs = append(addrs, payload.Subscriber)
		case pb.PayloadType_name[int32(pb.PayloadType_GENERATE_ID_TYPE)]:
			payload := &pb.GenerateID{}
			err = proto.Unmarshal(buf, payload)
			if err != nil {
				return err
			}
			if len(payload.Sender) > 0 {
				addrs = append(addrs, payload.Sender)
			}
			addrs = append(addrs, payload.PublicKey)
		case pb.PayloadType_name[int32(pb.PayloadType_NANO_PAY_TYPE)]:
			payload := &pb.NanoPay{}
			err = proto.Unmarshal(buf, payload)
			if err != nil {
				return err
			}
			if len(payload.Sender) > 0 {
				addrs = append(addrs, payload.Sender)
			}
			addrs = append(addrs, payload.Sender, payload.Recipient)
		default:
			log.Printf("Unknown txn type: %v", tx.TxType)
			continue
		}
	}

	for _, v := range addrs {
		var ph common.Uint160
		if len(v) == common.UINT160SIZE {
			ph = common.BytesToUint160(v)
		} else if len(v) == ed25519.PublicKeySize {
			ph, err = program.CreateProgramHash(v)
			if err != nil {
				return err
			}
		} else {
			log.Printf("Unknown size: %v", len(v))
			continue
		}
		addr, err := ph.ToAddress()
		if err != nil {
			return err
		}
		counter[addr]++
	}

	return nil
}
