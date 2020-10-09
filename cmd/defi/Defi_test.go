package defi

import (
	"fmt"
	"math/big"
	"os"
	"testing"

	"ethereum/rpc-network/accounts/abi/bind"
	"ethereum/rpc-network/accounts/abi/bind/backends"
	"ethereum/rpc-network/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

func init() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
}

var (
	key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr    = crypto.PubkeyToAddress(key.PublicKey)
	key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	addr2   = crypto.PubkeyToAddress(key2.PublicKey)
	key3, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	fee     = crypto.PubkeyToAddress(key3.PublicKey)
)

func TestENS(t *testing.T) {
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{
		addr:  {Balance: big.NewInt(10000000000)},
		addr2: {Balance: big.NewInt(10000000)}},
		10000000)
	transactOpts := bind.NewKeyedTransactor(key)

	// Deploy the ENS registry
	ensAddr, _, _, err := DeployDefi(transactOpts, contractBackend, fee)
	if err != nil {
		t.Fatalf("can't DeployContract: %v", err)
	}
	ens, err := NewDefi(ensAddr, contractBackend)
	if err != nil {
		t.Fatalf("can't NewContract: %v", err)
	}

	contractBackend.Commit()
	feeAddr, err := ens.FeeAddr(nil)
	if err != nil {
		t.Fatalf("can't feeAddr: %v", err)
	}
	fmt.Println(feeAddr.String(), " ", fee.String())
}
